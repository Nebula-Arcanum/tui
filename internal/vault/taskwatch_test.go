package vault

import (
	"reflect"
	"testing"
)

const watchedNote = "inbox/Plan.md"

// watchScan reads a note the way `zn task watch` scans: in the app's grammar.
func watchScan(path, body string) []Task {
	return ParseTasks(path, "Plan", FolderInbox, body, ParseTasksOptions{Dialect: DialectApp})
}

type statusMove struct{ ID, From, To string }

func statusMoves(changes []TaskStatusChange) []statusMove {
	out := []statusMove{}
	for _, c := range changes {
		out = append(out, statusMove{c.Task.ID, c.PreviousStatus, c.Status})
	}
	return out
}

func TestDiffTaskStatusesReportsEveryWayAStatusMoves(t *testing.T) {
	const fileTask = watchedNote + "#task"
	const firstLine = watchedNote + "#0"
	for _, c := range []struct {
		name          string
		before, after string
		want          statusMove
	}{
		{"a task note moves column",
			"---\ntags: [task]\nstatus: backlog\n---\nBuild it.\n",
			"---\ntags: [task]\nstatus: ready\n---\nBuild it.\n",
			statusMove{fileTask, "backlog", "ready"}},
		{"a task note gets its first status",
			"---\ntags: [task]\n---\nBuild it.\n",
			"---\ntags: [task]\nstatus: backlog\n---\nBuild it.\n",
			statusMove{fileTask, "open", "backlog"}},
		{"a task note is checked off, keeping its own word for done",
			"---\ntags: [task]\nstatus: ready for review\n---\nBuild it.\n",
			"---\ntags: [task]\nstatus: completed\ncompletedDate: 2026-10-14\n---\nBuild it.\n",
			statusMove{fileTask, "ready for review", "completed"}},
		{"a line's @status changes, read in lower case",
			"- [ ] Write docs @status:backlog\n",
			"- [ ] Write docs @status:Ready\n",
			statusMove{firstLine, "backlog", "ready"}},
		{"a line is dropped into a status column",
			"- [ ] Write docs\n",
			"- [ ] Write docs @status:ready\n",
			statusMove{firstLine, "open", "ready"}},
		{"a line follows its note's status",
			"---\nstatus: backlog\n---\n- [ ] Write docs\n",
			"---\nstatus: ready\n---\n- [ ] Write docs\n",
			statusMove{firstLine, "backlog", "ready"}},
		{"a line is checked",
			"- [ ] Write docs\n",
			"- [x] Write docs\n",
			statusMove{firstLine, "open", "done"}},
		{"a line in a status column is checked, which takes it off the board",
			"- [ ] Write docs @status:review\n",
			"- [x] Write docs @status:review\n",
			statusMove{firstLine, "review", "done"}},
		{"a started line starts waiting",
			"- [/] Write docs\n",
			"- [/] Write docs @waiting\n",
			statusMove{firstLine, "in-progress", "waiting"}},
		{"a line is cancelled",
			"- [ ] Write docs\n",
			"- [-] Write docs\n",
			statusMove{firstLine, "open", "cancelled"}},
	} {
		got := statusMoves(DiffTaskStatuses(watchScan(watchedNote, c.before), watchScan(watchedNote, c.after)))
		if want := []statusMove{c.want}; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %+v, want %+v", c.name, got, want)
		}
	}
}

// Saving a task is not a status change: issue #941 asks that only a real move
// start the user's script.
func TestDiffTaskStatusesIgnoresEditsThatKeepTheStatus(t *testing.T) {
	for _, c := range []struct{ name, before, after string }{
		{"the task's words are edited",
			"- [ ] Write docs @status:backlog\n",
			"- [ ] Write the docs @status:backlog\n"},
		{"a task is added above, shifting its position",
			"- [ ] Write docs @status:ready\n",
			"- [ ] Plan first @status:backlog\n- [ ] Write docs @status:ready\n"},
		{"a task note's body is edited",
			"---\ntags: [task]\nstatus: ready\n---\nBuild it.\n",
			"---\ntags: [task]\nstatus: ready\n---\nBuild it, then ship it.\n"},
		{"other metadata changes",
			"- [ ] Write docs @status:ready due:2026-10-20\n",
			"- [ ] Write docs @status:ready due:2026-10-21 !high #docs\n"},
		{"a task is removed",
			"- [ ] Write docs\n- [ ] Ship\n",
			"- [ ] Ship\n"},
		{"the status is written in another case",
			"- [ ] Write docs @status:Ready\n",
			"- [ ] Write docs @status:ready\n"},
	} {
		if got := DiffTaskStatuses(watchScan(watchedNote, c.before), watchScan(watchedNote, c.after)); len(got) != 0 {
			t.Errorf("%s: reported %+v", c.name, statusMoves(got))
		}
	}
}

// A change is reported with the task as the newer scan has it, and tasks with
// the same words stay apart: by their order within a note, and by note.
func TestDiffTaskStatusesMatchesTasksByNoteAndWords(t *testing.T) {
	before := append(watchScan(watchedNote, "- [ ] Call Sam\n- [ ] Call Sam\n"), watchScan("inbox/Other.md", "- [ ] Call Sam\n")...)
	after := append(watchScan(watchedNote, "- [ ] New first\n- [ ] Call Sam\n- [x] Call Sam\n"), watchScan("inbox/Other.md", "- [ ] Call Sam\n")...)
	got := statusMoves(DiffTaskStatuses(before, after))
	want := []statusMove{{watchedNote + "#2", "open", "done"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
