package vault

import (
	"strconv"
	"strings"
)

// TaskStatusChange is one task whose status differs between two scans of the
// vault: what `zn task watch` reports (ZenNotes/zennotes#941).
type TaskStatusChange struct {
	// Task is the task as the newer scan found it.
	Task           Task
	PreviousStatus string
	Status         string
}

// TaskStatusName is the status a task is in, read the way the app's boards
// place it. A task note's status is its frontmatter `status:` (open when it
// has none), done and cancelled included. A checkbox line is done, cancelled
// or forwarded once its box says so, since a board grouped by status drops
// those cards; until then it is its `@status:` (or the note's frontmatter
// status it inherits), the column it sits in on that board, and failing that
// waiting, in progress or open, as the built-in board has it (`@waiting`
// wins over `[/]`).
func TaskStatusName(t Task) string {
	custom := strings.ToLower(strings.TrimSpace(t.Status))
	if t.Kind == "file" && custom != "" {
		return custom
	}
	switch {
	case t.Forwarded:
		return "forwarded"
	case t.Cancelled:
		return "cancelled"
	case t.Checked:
		return "done"
	case custom != "":
		return custom
	case t.Waiting:
		return "waiting"
	case t.InProgress:
		return "in-progress"
	default:
		return "open"
	}
}

// taskIdentity says which task in one scan is which task in the next. A task
// note is its id (`path#task`). A checkbox task's id is its position in the
// note, which a line added above it shifts, so it is matched by its text
// instead: the same words in the same note, counted when they repeat. Scans
// must use the app's grammar, whose content leaves `@status:` and the other
// fields out, so that a status change never reads as a different task, and
// a change to the words does (it is an edit, not a status change).
type taskIdentity struct {
	seen map[string]int
}

func (k *taskIdentity) of(t Task) string {
	if t.Kind == "file" {
		return t.ID
	}
	base := t.SourcePath + "\x00" + strings.TrimSpace(t.Content)
	n := k.seen[base]
	k.seen[base] = n + 1
	return base + "\x00" + strconv.Itoa(n)
}

// DiffTaskStatuses lists the tasks found in both scans whose status changed,
// in the order of the newer scan. Tasks that appear, disappear, move to
// another note or have their words edited are not status changes.
func DiffTaskStatuses(previous, next []Task) []TaskStatusChange {
	before := map[string]Task{}
	keys := taskIdentity{seen: map[string]int{}}
	for _, t := range previous {
		before[keys.of(t)] = t
	}
	var changes []TaskStatusChange
	keys = taskIdentity{seen: map[string]int{}}
	for _, t := range next {
		old, ok := before[keys.of(t)]
		if !ok {
			continue
		}
		from, to := TaskStatusName(old), TaskStatusName(t)
		if from != to {
			changes = append(changes, TaskStatusChange{Task: t, PreviousStatus: from, Status: to})
		}
	}
	return changes
}
