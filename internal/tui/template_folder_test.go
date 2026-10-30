package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// templateFolderApp opens a root-layout vault whose `Templates` folder is its
// template folder (ZenNotes/zennotes#1024).
func templateFolderApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("ZENNOTES_CONFIG_DIR", t.TempDir())
	root := t.TempDir()
	for rel, body := range map[string]string{
		".zennotes/vault.json":     `{"primaryNotesLocation":"root","templates":{"folder":"Templates"}}`,
		"Templates/Meeting.md":     "---\ntags: [work]\n---\n- [ ] Agenda item\n",
		"Templates/Work/Retro.md":  "- [ ] Action item\n",
		"Notes/Meeting minutes.md": "---\ntags: [work]\n---\n- [ ] Book the room\n",
	} {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return newSessionParityApp(t, root)
}

func noteSearchPaths(t *testing.T, a *App, query string) []string {
	t.Helper()
	a.openNoteSearch(query)
	p, ok := a.overlay.(*palette)
	if !ok {
		t.Fatalf("Search notes did not open for %q", query)
	}
	out := []string{}
	for _, it := range p.filtered {
		out = append(out, it.id)
	}
	slices.Sort(out)
	return out
}

// Search notes hides template notes, as the desktop's does, until the query
// starts with the folder's path and a slash, which is how a template is found
// to be edited.
func TestSearchNotesHidesTheTemplateFolderUntilTheQueryNamesIt(t *testing.T) {
	a := templateFolderApp(t)
	cases := []struct {
		query string
		want  []string
	}{
		{"", []string{"Notes/Meeting minutes.md"}},
		{"meeting", []string{"Notes/Meeting minutes.md"}},
		{"#work", []string{"Notes/Meeting minutes.md"}},
		{"templates", []string{}},
		{"Templates/", []string{"Templates/Meeting.md", "Templates/Work/Retro.md"}},
		{"templates/meet", []string{"Templates/Meeting.md"}},
		{"#work Templates/meet", []string{"Templates/Meeting.md"}},
	}
	for _, tc := range cases {
		if got := noteSearchPaths(t, a, tc.query); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Search notes %q = %v, want %v", tc.query, got, tc.want)
		}
	}
}

// The Tasks view scans as `zn task list` does.
func TestTasksViewLeavesTheTemplateFolderOut(t *testing.T) {
	a := templateFolderApp(t)
	msg := a.loadTasksCmd()().(tasksLoadedMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	sources := []string{}
	for _, task := range msg.tasks {
		sources = append(sources, task.SourcePath)
	}
	if !reflect.DeepEqual(sources, []string{"Notes/Meeting minutes.md"}) {
		t.Fatalf("Tasks = %v, want only Notes/Meeting minutes.md", sources)
	}
}

// Settings shows the template folder next to the Tasks folder lists, since
// it takes a folder out of Tasks as they do.
func TestSettingsShowTheTemplateFolder(t *testing.T) {
	a := templateFolderApp(t)
	v := &settingsView{}
	v.refresh(a)
	if !slices.Contains(v.lines, "vault · templates.folder = Templates") {
		t.Fatalf("settings rows:\n%s", strings.Join(v.lines, "\n"))
	}
}

// An edit writes `templates.folder` cleaned the way every runtime reads it,
// keeps whatever else is under `templates`, and an empty edit removes it.
func TestSetTemplateFolder(t *testing.T) {
	raw := map[string]any{"templates": map[string]any{"later": true}}
	setTemplateFolder(raw, " /Meta\\Templates/ ")
	want := map[string]any{"templates": map[string]any{"later": true, "folder": "Meta/Templates"}}
	if !reflect.DeepEqual(raw, want) {
		t.Fatalf("after setting: %#v", raw)
	}
	setTemplateFolder(raw, "  ")
	if want := map[string]any{"templates": map[string]any{"later": true}}; !reflect.DeepEqual(raw, want) {
		t.Fatalf("after clearing: %#v", raw)
	}
	raw = map[string]any{"templates": map[string]any{"folder": "Templates"}}
	setTemplateFolder(raw, "")
	if _, ok := raw["templates"]; ok {
		t.Fatalf("clearing the only key should drop templates, got %#v", raw)
	}
}
