package tui

import (
	"reflect"
	"testing"
)

// #962: the TUI writes the Tasks folder lists where every runtime reads them,
// under `tasks`, and keeps the other list as it is.
func TestSetTasksFolderList(t *testing.T) {
	raw := map[string]any{"tasks": map[string]any{"excludedFolders": []string{"inbox/Books"}}}
	setTasksFolderList(raw, "includedFolders", " Tasks , inbox/Work ,")
	want := map[string]any{"tasks": map[string]any{
		"excludedFolders": []string{"inbox/Books"},
		"includedFolders": []string{"Tasks", "inbox/Work"},
	}}
	if !reflect.DeepEqual(raw, want) {
		t.Fatalf("after adding Task folders: %#v", raw)
	}
	if _, stray := raw["tasksExcludedFolders"]; stray {
		t.Fatal("wrote the old top-level tasksExcludedFolders key")
	}

	setTasksFolderList(raw, "excludedFolders", "")
	setTasksFolderList(raw, "includedFolders", "  ")
	if _, ok := raw["tasks"]; ok {
		t.Fatalf("emptying both lists should drop tasks, got %#v", raw)
	}
}
