package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ZenNotes/tui/internal/backend"
	"github.com/ZenNotes/tui/internal/remote"
	"github.com/ZenNotes/tui/internal/vault"
	"github.com/coder/websocket"
)

// syncBuffer is output the watch writes while the test reads it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func writeVaultFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func localBackend(t *testing.T, root string) backend.Backend {
	t.Helper()
	b, err := backend.New(backend.Target{Kind: backend.KindLocal, Root: root}, backend.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func waitFor(t *testing.T, what string, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// watchTasks runs the watch loop over root with a hand-driven change signal,
// so each scan happens exactly when the test says.
func watchTasks(t *testing.T, root string, asJSON bool) (output *syncBuffer, changed chan<- struct{}, stop func()) {
	t.Helper()
	b := localBackend(t, root)
	opts := vault.ParseTasksOptions{Dialect: vault.DialectApp}
	previous, err := b.ScanTasks(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	output = &syncBuffer{}
	prevOut, prevErr := stdout, stderr
	stdout, stderr = output, output
	signals := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- followTaskStatuses(ctx, b, opts, previous, signals, asJSON) }()
	return output, signals, func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("the watch ended with %v", err)
		}
		stdout, stderr = prevOut, prevErr
	}
}

func TestTaskWatchPrintsOneJSONLinePerStatusChange(t *testing.T) {
	root := t.TempDir()
	writeVaultFile(t, root, "inbox/Ship it.md", "---\ntags: [task]\nstatus: backlog\n---\nBuild the thing.\n")
	writeVaultFile(t, root, "inbox/Plan.md", "# Plan\n\n- [ ] Write docs @status:backlog\n")
	output, changed, stop := watchTasks(t, root, true)

	// A send returns once the loop has the signal, while its scan may still
	// be reading, so a write after it can land mid-scan; the scan then sees
	// the note empty or rewritten, and either way the change is printed once.
	// Waiting for each line before the next write keeps their order.
	printed := func() int { return strings.Count(output.String(), "\n") }
	// An edit that keeps every status prints nothing.
	writeVaultFile(t, root, "inbox/Ship it.md", "---\ntags: [task]\nstatus: backlog\n---\nBuild the thing, carefully.\n")
	changed <- struct{}{}
	changed <- struct{}{}
	writeVaultFile(t, root, "inbox/Ship it.md", "---\ntags: [task]\nstatus: ready\n---\nBuild the thing, carefully.\n")
	changed <- struct{}{}
	waitFor(t, "the task note's change", func() bool { return printed() >= 1 })
	writeVaultFile(t, root, "inbox/Plan.md", "# Plan\n\n- [ ] Write docs @status:review\n")
	changed <- struct{}{}
	waitFor(t, "the checkbox task's change", func() bool { return printed() >= 2 })
	changed <- struct{}{}
	changed <- struct{}{}
	stop()

	lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d:\n%s", len(lines), output.String())
	}
	var events [2]map[string]any
	for i, line := range lines {
		if err := json.Unmarshal([]byte(line), &events[i]); err != nil {
			t.Fatalf("line %d is not one JSON object: %v\n%s", i+1, err, line)
		}
		at, _ := events[i]["at"].(string)
		if _, err := time.Parse(time.RFC3339, at); err != nil {
			t.Errorf("line %d: at %q: %v", i+1, at, err)
		}
		if link, _ := events[i]["link"].(string); !strings.HasPrefix(link, "zennotes://open?path=") {
			t.Errorf("line %d: link %q", i+1, link)
		}
		delete(events[i], "at")
		delete(events[i], "link")
	}
	want := [2]map[string]any{
		{"id": "inbox/Ship it.md#task", "path": "inbox/Ship it.md", "title": "Ship it", "kind": "file",
			"content": "Ship it", "previousStatus": "backlog", "status": "ready", "lineNumber": float64(0)},
		{"id": "inbox/Plan.md#0", "path": "inbox/Plan.md", "title": "Plan",
			"content": "Write docs", "previousStatus": "backlog", "status": "review", "lineNumber": float64(2)},
	}
	for i := range want {
		if !mapsEqual(events[i], want[i]) {
			t.Errorf("line %d:\n got %v\nwant %v", i+1, events[i], want[i])
		}
	}
}

func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestTaskWatchPrintsAReadableLineWithoutJSON(t *testing.T) {
	root := t.TempDir()
	writeVaultFile(t, root, "inbox/Plan.md", "- [ ] Write docs\n")
	output, changed, stop := watchTasks(t, root, false)
	writeVaultFile(t, root, "inbox/Plan.md", "- [x] Write docs\n")
	changed <- struct{}{}
	waitFor(t, "the change", func() bool { return strings.Contains(output.String(), "\n") })
	stop()
	if line := output.String(); !strings.HasSuffix(line, "  open → done  Write docs  inbox/Plan.md\n") {
		t.Fatalf("line: %q", line)
	}
}

func TestTaskWatchSignalsOnALocalVaultsFileChanges(t *testing.T) {
	root := t.TempDir()
	writeVaultFile(t, root, "inbox/Plan.md", "- [ ] Write docs\n")
	changed, stop, err := watchVaultChanges(context.Background(), localBackend(t, root), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	writeVaultFile(t, root, "inbox/Plan.md", "- [x] Write docs\n")
	select {
	case <-changed:
	case <-time.After(5 * time.Second):
		t.Fatal("no signal after a note changed")
	}
	stop()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case _, open := <-changed:
			if !open {
				return
			}
		case <-deadline:
			t.Fatal("the signal did not close once the watch stopped")
		}
	}
}

func TestTaskWatchFollowsAServersPushedChanges(t *testing.T) {
	push := make(chan string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/watch" {
			http.NotFound(w, r)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for path := range push {
			if conn.Write(r.Context(), websocket.MessageText, []byte(`{"kind":"change","path":"`+path+`"}`)) != nil {
				return
			}
		}
	}))
	defer server.Close()
	defer close(push)

	signals := make(chan struct{}, 8)
	ctx, cancel := context.WithCancel(context.Background())
	ended := make(chan struct{})
	go func() {
		followServerChanges(ctx, remote.NewClient(server.URL, ""), time.Hour, func() { signals <- struct{}{} })
		close(ended)
	}()
	expect := func(what string) {
		t.Helper()
		select {
		case <-signals:
		case <-time.After(5 * time.Second):
			t.Fatalf("no signal %s", what)
		}
	}
	expect("on connecting, for whatever changed before")
	push <- "inbox/Plan.md"
	push <- "inbox/Plan.md"
	expect("once the pushed changes settle")
	select {
	case <-signals:
		t.Fatal("one burst of changes signalled twice")
	case <-time.After(3 * taskWatchSettle):
	}
	cancel()
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the server watch did not end with its context")
	}
}

func TestTaskWatchPollsAServerWithoutAStream(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	signals := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	ended := make(chan struct{})
	go func() {
		followServerChanges(ctx, remote.NewClient(server.URL, ""), 20*time.Millisecond, func() {
			select {
			case signals <- struct{}{}:
			default:
			}
		})
		close(ended)
	}()
	for i := 0; i < 3; i++ {
		select {
		case <-signals:
		case <-time.After(5 * time.Second):
			t.Fatalf("poll %d never came", i+1)
		}
	}
	cancel()
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("polling did not end with its context")
	}
}

func TestTaskWatchRejectsABadInterval(t *testing.T) {
	root := isolatedCLI(t)
	for _, value := range []string{"0", "-1", "soon", "1.5"} {
		out := captureOutput(t, func() {
			if code := Main([]string{"task", "watch", "--interval", value, "--vault", root}); code == 0 {
				t.Errorf("--interval %s was accepted", value)
			}
		})
		if !strings.Contains(out, "--interval") {
			t.Errorf("--interval %s: %s", value, out)
		}
	}
}

// `zn task list --json` says which status column a task is in, as the app
// does, without changing the content or ids the desktop's CLI gives
// (ZenNotes/zennotes#941).
func TestTaskListJSONCarriesTheStatus(t *testing.T) {
	isolatedCLI(t)
	root := t.TempDir()
	writeVaultFile(t, root, "inbox/Ship it.md", "---\ntags: [task]\nstatus: Ready\n---\nBuild it.\n")
	writeVaultFile(t, root, "inbox/Plan.md", "---\nstatus: backlog\n---\n- [ ] Write docs @status:review\n- [ ] Inherits\n")
	writeVaultFile(t, root, "inbox/Loose.md", "- [ ] Plain\n")
	out := captureOutput(t, func() {
		if code := Main([]string{"task", "list", "--json", "--vault", root}); code != 0 {
			t.Errorf("exit %d", code)
		}
	})
	var tasks []struct {
		ID      string  `json:"id"`
		Content string  `json:"content"`
		Status  *string `json:"status"`
	}
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	got := map[string]string{}
	for _, task := range tasks {
		status := "(none)"
		if task.Status != nil {
			status = *task.Status
		}
		got[task.ID+" "+task.Content] = status
	}
	want := map[string]string{
		"inbox/Ship it.md#task Ship it":             "ready",
		"inbox/Plan.md#0 Write docs @status:review": "review",
		"inbox/Plan.md#1 Inherits":                  "backlog",
		"inbox/Loose.md#0 Plain":                    "(none)",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for key, status := range want {
		if got[key] != status {
			t.Errorf("%s: status %q, want %q (all: %v)", key, got[key], status, got)
		}
	}
}
