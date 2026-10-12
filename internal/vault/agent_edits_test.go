package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The writing side of agent edit records. The same cases run against
// beginAgentEdit / finishAgentEdit in ZenNotes/zennotes
// packages/shared-domain/src/agent-edits.test.ts.

func record(existing *AgentEditRecord, current, next string, at int64) *AgentEditRecord {
	return FinishAgentEdit(BeginAgentEdit(existing, current), next, AgentEditEntry{At: at, Agent: "Claude Code", Tool: "replace_in_note"})
}

func TestAgentEditKeepsTheNoteBeforeAndAfter(t *testing.T) {
	started := BeginAgentEdit(nil, "a\nb\n")
	if started.Before != "a\nb\n" || started.After != nil || len(started.Edits) != 0 || len(started.Outside) != 0 {
		t.Fatalf("started = %+v", started)
	}
	done := FinishAgentEdit(started, "a\nB\n", AgentEditEntry{At: 5, Agent: "Claude Code", Tool: "replace_in_note"})
	if done.Before != "a\nb\n" || done.After == nil || *done.After != "a\nB\n" || len(done.Edits) != 1 {
		t.Fatalf("done = %+v", done)
	}
}

func TestAgentEditStacksEditsAndNotesOutsideChanges(t *testing.T) {
	first := record(nil, "a\nb\nc\n", "a\nB\nc\n", 1)
	same := record(first, "a\nB\nc\n", "A\nB\nc\n", 2)
	if same.Before != "a\nb\nc\n" || len(same.Outside) != 0 || len(same.Edits) != 2 {
		t.Fatalf("stacked = %+v", same)
	}
	// The user changes the last line before the assistant's next edit.
	outside := record(first, "a\nB\nC\n", "A\nB\nC\n", 2)
	if outside.Before != "a\nb\nc\n" || len(outside.Outside) != 1 ||
		outside.Outside[0].From != "a\nB\nc\n" || outside.Outside[0].To != "a\nB\nC\n" {
		t.Fatalf("outside = %+v", outside)
	}
}

func TestAgentEditCapsEntriesAndStartsOverPastTheOutsideLimit(t *testing.T) {
	var current *AgentEditRecord
	text := "line 0\n"
	for i := 1; i <= MaxAgentEditEntries+5; i++ {
		next := "line " + strings.Repeat("x", i) + "\n"
		current = record(current, text, next, int64(i))
		text = next
	}
	if len(current.Edits) != MaxAgentEditEntries || current.Edits[len(current.Edits)-1].At != MaxAgentEditEntries+5 || current.Before != "line 0\n" {
		t.Fatalf("capped = %d edits, before %q", len(current.Edits), current.Before)
	}

	r := record(nil, "x\n", "x1\n", 0)
	for i := 1; i <= MaxAgentEditOutside; i++ {
		r = record(r, "user "+strings.Repeat("u", i)+"\n", "agent "+strings.Repeat("a", i)+"\n", int64(i))
	}
	fresh := record(r, "user again\n", "agent again\n", 99)
	if fresh.Before != "user again\n" || len(fresh.Outside) != 0 || len(fresh.Edits) != 1 {
		t.Fatalf("fresh = %+v", fresh)
	}
}

func TestAgentEditFilesRoundTripInTheAppsShape(t *testing.T) {
	root := t.TempDir()
	rec := record(nil, "a\n", "b\n", 7)
	if err := WriteAgentEdit(root, "inbox/Plan.md", rec); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".zennotes", "agent-edits", "inbox", "Plan.md.json"))
	if err != nil {
		t.Fatal(err)
	}
	// The desktop parses this file: same keys, empty lists as [], never null.
	var shape map[string]any
	if err := json.Unmarshal(data, &shape); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"version", "before", "after", "edits", "outside"} {
		if _, ok := shape[key]; !ok {
			t.Fatalf("missing %q in %s", key, data)
		}
	}
	if shape["outside"] == nil {
		t.Fatalf("outside must be [] not null: %s", data)
	}
	back := ReadAgentEdit(root, "inbox/Plan.md")
	if back == nil || back.Before != "a\n" || *back.After != "b\n" || back.Edits[0].Agent != "Claude Code" {
		t.Fatalf("back = %+v", back)
	}
	if err := WriteAgentEdit(root, "inbox/Plan.md", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".zennotes", "agent-edits")); !os.IsNotExist(err) {
		t.Fatalf("an emptied agent-edits folder should be gone: %v", err)
	}
}

func TestAgentEditReadsWhatTheAppWrites(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".zennotes", "agent-edits", "inbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// JSON.stringify of a record from @shared/agent-edits, after a partly done review.
	app := `{"version":1,"before":"A\nb\n","after":"A\nB\n","edits":[{"at":1791820000000,"agent":"Claude Code","tool":"replace_in_note"}],"outside":[]}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "Plan.md.json"), []byte(app), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ReadAgentEdit(root, "inbox/Plan.md")
	if got == nil || got.Before != "A\nb\n" || got.Edits[0].At != 1791820000000 {
		t.Fatalf("got = %+v", got)
	}
	if ReadAgentEdit(root, "inbox/Missing.md") != nil {
		t.Fatal("a missing record must read as nil")
	}
}

func TestAgentEditFollowsTheNoteAndGoesWithIt(t *testing.T) {
	v := newTestVault(t)
	if _, err := v.WriteNote("inbox/Plan.md", "# Plan\n"); err != nil {
		t.Fatal(err)
	}
	if err := WriteAgentEdit(v.root, "inbox/Plan.md", record(nil, "# Plan\n", "# Plan\nmore\n", 1)); err != nil {
		t.Fatal(err)
	}
	meta, err := v.RenameNote("inbox/Plan.md", "Roadmap")
	if err != nil {
		t.Fatal(err)
	}
	if ReadAgentEdit(v.root, "inbox/Plan.md") != nil || ReadAgentEdit(v.root, meta.Path) == nil {
		t.Fatalf("the record should follow the rename to %s", meta.Path)
	}
	if err := v.DeleteNote(meta.Path); err != nil {
		t.Fatal(err)
	}
	if ReadAgentEdit(v.root, meta.Path) != nil {
		t.Fatal("deleting the note should delete its record")
	}
}
