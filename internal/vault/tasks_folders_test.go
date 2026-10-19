package vault

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// #962: mirrors tasks-excluded-folders.test.ts in shared-domain and the Go
// server's tasks_exclude_test.go; the rules must stay byte-compatible.
func TestIsPathExcludedFromTasksWithTaskFolders(t *testing.T) {
	cases := []struct {
		path     string
		excluded []string
		included []string
		want     bool
	}{
		{"inbox/Idea.md", nil, nil, false},
		{"inbox/Books/dune.md", []string{"inbox/Books"}, nil, true},
		{"inbox/Bookshelf.md", []string{"inbox/Books"}, nil, false},
		{"Tasks/Today.md", nil, []string{"Tasks"}, false},
		{"Tasks/Work/Q4.md", nil, []string{"Tasks"}, false},
		{"inbox/Books.md", nil, []string{"Tasks"}, true},
		{"TasksArchive/Old.md", nil, []string{"Tasks"}, true},
		{"inbox/Projects/Site.md", nil, []string{"Tasks", "inbox/Projects"}, false},
		{"Projects/Archive/Old.md", []string{"Projects/Archive"}, []string{"Projects"}, true},
		{"Projects/Live.md", []string{"Projects/Archive"}, []string{"Projects"}, false},
	}
	for _, tc := range cases {
		if got := IsPathExcludedFromTasks(tc.path, tc.excluded, tc.included); got != tc.want {
			t.Errorf("IsPathExcludedFromTasks(%q, %v, %v) = %v, want %v", tc.path, tc.excluded, tc.included, got, tc.want)
		}
	}
}

func TestScanTasksHonorsTaskFolders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".zennotes", "vault.json"),
		`{"tasks":{"includedFolders":["inbox/Tasks"],"excludedFolders":["inbox/Tasks/Someday"]}}`)
	writeFile(t, filepath.Join(root, "inbox", "Tasks", "Today.md"), "- [ ] ship it\n")
	writeFile(t, filepath.Join(root, "inbox", "Tasks", "Someday", "Maybe.md"), "- [ ] later\n")
	writeFile(t, filepath.Join(root, "inbox", "Reading.md"), "- [ ] Excession\n")
	v, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := v.Settings().TasksIncludedFolders; !reflect.DeepEqual(got, []string{"inbox/Tasks"}) {
		t.Fatalf("TasksIncludedFolders = %v", got)
	}
	sources := func(tasks []Task) []string {
		out := []string{}
		seen := map[string]bool{}
		for _, tk := range tasks {
			if !seen[tk.SourcePath] {
				seen[tk.SourcePath] = true
				out = append(out, tk.SourcePath)
			}
		}
		sort.Strings(out)
		return out
	}

	tasks, err := v.ScanTasks(ParseTasksOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := sources(tasks); !reflect.DeepEqual(got, []string{"inbox/Tasks/Today.md"}) {
		t.Fatalf("default scan = %v, want only inbox/Tasks/Today.md", got)
	}

	all, err := v.ScanTasks(ParseTasksOptions{IncludeExcluded: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := sources(all); len(got) != 3 {
		t.Fatalf("IncludeExcluded scan = %v, want all three notes", got)
	}
}

func TestRenameFolderRewritesTaskFolders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".zennotes", "vault.json"),
		`{"favorites":["x"],"tasks":{"includedFolders":["inbox/Tasks","inbox/TasksArchive"],"excludedFolders":["inbox/Tasks/Someday"]}}`)
	writeFile(t, filepath.Join(root, "inbox", "Tasks", "Today.md"), "- [ ] ship it\n")
	writeFile(t, filepath.Join(root, "inbox", "Tasks", "Someday", "Maybe.md"), "- [ ] later\n")
	writeFile(t, filepath.Join(root, "inbox", "TasksArchive", "Old.md"), "- [x] done\n")
	v, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.RenameFolder(FolderInbox, "Tasks", "Todo"); err != nil {
		t.Fatal(err)
	}
	s := v.Settings()
	if want := []string{"inbox/Todo", "inbox/TasksArchive"}; !reflect.DeepEqual(s.TasksIncludedFolders, want) {
		t.Errorf("Task folders after rename = %v, want %v", s.TasksIncludedFolders, want)
	}
	if want := []string{"inbox/Todo/Someday"}; !reflect.DeepEqual(s.TasksExcludedFolders, want) {
		t.Errorf("excluded folders after rename = %v, want %v", s.TasksExcludedFolders, want)
	}
	if !reflect.DeepEqual(s.Favorites, []string{"x"}) {
		t.Errorf("the rewrite lost other vault.json keys: favorites = %v", s.Favorites)
	}
	tasks, err := v.ScanTasks(ParseTasksOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected Today and Old after rename, got %d tasks", len(tasks))
	}
}
