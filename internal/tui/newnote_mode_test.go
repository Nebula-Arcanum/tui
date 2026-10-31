package tui

import (
	"testing"

	"github.com/ZenNotes/tui/internal/vault"
)

// A freshly created note is entered in insert mode, so it must show the
// editor even when the default pane mode is preview.
func TestCreateNoteOpensInEditMode(t *testing.T) {
	a, _ := newFeatureTestApp(t)
	a.prefs.DefaultPaneMode = string(modePreview)
	a.createNote(vault.FolderInbox, "Fresh", "", nil, true)
	tb := a.activeTab()
	if tb == nil {
		t.Fatal("no active tab")
	}
	if tb.mode != modeEdit {
		t.Fatalf("new note opened in mode %q, want edit", tb.mode)
	}
}
