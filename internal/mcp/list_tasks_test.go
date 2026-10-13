package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/ZenNotes/tui/internal/backend"
)

// A `[-]` cancelled task is abandoned, so list_tasks leaves it out of open,
// as the desktop app does (ZenNotes/zennotes#929); status "all" keeps it.
func TestListTasksLeavesCancelledTasksOutOfOpen(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "inbox"), 0o755); err != nil {
		t.Fatal(err)
	}
	note := "- [ ] Buy milk\n- [-] Cancelled plan\n- [x] Done thing\n"
	if err := os.WriteFile(filepath.Join(root, "inbox", "todo.md"), []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}
	cs := runServer(t, func() (backend.Target, error) {
		return backend.Target{Kind: backend.KindLocal, Root: root}, nil
	})
	contents := func(args map[string]any) []string {
		text, isErr := callTool(t, cs, "list_tasks", args)
		if isErr {
			t.Fatalf("list_tasks %v: %s", args, text)
		}
		var tasks []struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(text), &tasks); err != nil {
			t.Fatalf("list_tasks %v: %v\n%s", args, err, text)
		}
		out := []string{}
		for _, task := range tasks {
			out = append(out, task.Content)
		}
		return out
	}
	if got := contents(map[string]any{}); !reflect.DeepEqual(got, []string{"Buy milk"}) {
		t.Fatalf("open tasks: %v", got)
	}
	if got := contents(map[string]any{"status": "all"}); !slices.Contains(got, "Cancelled plan") {
		t.Fatalf("status all: %v", got)
	}
}
