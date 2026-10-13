package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

// A `[-]` cancelled task is abandoned, so `zn task list` leaves it out by
// default and with --unchecked, as the desktop app does
// (ZenNotes/zennotes#929); --all still lists it.
func TestTaskListLeavesCancelledTasksOut(t *testing.T) {
	isolatedCLI(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "inbox"), 0o755); err != nil {
		t.Fatal(err)
	}
	note := "- [ ] Buy milk\n- [-] Cancelled plan\n- [x] Done thing\n"
	if err := os.WriteFile(filepath.Join(root, "inbox", "todo.md"), []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}
	listed := func(flags ...string) []string {
		args := append([]string{"task", "list", "--json", "--vault", root}, flags...)
		out := captureOutput(t, func() {
			if code := Main(args); code != 0 {
				t.Errorf("zn %v exited %d", args, code)
			}
		})
		var tasks []struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(out), &tasks); err != nil {
			t.Fatalf("zn %v: %v\n%s", args, err, out)
		}
		contents := []string{}
		for _, task := range tasks {
			contents = append(contents, task.Content)
		}
		return contents
	}
	if got := listed(); !reflect.DeepEqual(got, []string{"Buy milk"}) {
		t.Fatalf("default: %v", got)
	}
	if got := listed("--unchecked"); !reflect.DeepEqual(got, []string{"Buy milk"}) {
		t.Fatalf("--unchecked: %v", got)
	}
	if got := listed("--all"); !slices.Contains(got, "Cancelled plan") {
		t.Fatalf("--all: %v", got)
	}
}
