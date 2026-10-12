package vault

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Agent edit records: what a note said before an assistant changed it through
// MCP, kept beside it as .zennotes/agent-edits/<note path>.json until the user
// reviews the change in the desktop app (ZenNotes/zennotes
// docs/ideas/agent-edit-review.md). This is the WRITING side only, and it is
// kept to string comparisons and JSON on purpose: the app folds in what
// someone else changed between two edits with a three-way merge, so no merge
// lives here. Mirrors beginAgentEdit / finishAgentEdit in
// packages/shared-domain/src/agent-edits.ts and the file handling in
// apps/desktop/src/main/agent-edit-files.ts; keep the three in step.

const (
	agentEditsDir          = "agent-edits"
	agentEditsSuffix       = ".json"
	AgentEditRecordVersion = 1
	// MaxAgentEditNoteBytes: larger notes are not recorded, a record holds copies.
	MaxAgentEditNoteBytes = 1024 * 1024
	// MaxAgentEditEntries: how many edits a record lists; the texts stay.
	MaxAgentEditEntries = 50
	// MaxAgentEditOutside: past this many outside changes a record starts over
	// from the note as it is, forgetting a review but changing no note.
	MaxAgentEditOutside = 20
)

// AgentEditEntry is one assistant edit.
type AgentEditEntry struct {
	At    int64  `json:"at"`
	Agent string `json:"agent"`
	Tool  string `json:"tool"`
}

// AgentEditOutsideChange is a change someone else made between two edits:
// the note after one assistant edit, and the note the next one started from.
type AgentEditOutsideChange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// AgentEditRecord is the record file. After is nil while an edit is being
// made: the record reaches disk BEFORE the edit, so a process that dies
// mid-edit leaves a record that shows more than the assistant did, never less.
type AgentEditRecord struct {
	Version int                      `json:"version"`
	Before  string                   `json:"before"`
	After   *string                  `json:"after"`
	Edits   []AgentEditEntry         `json:"edits"`
	Outside []AgentEditOutsideChange `json:"outside"`
}

func agentEditsRoot(root string) string {
	return filepath.Join(root, InternalVaultDir, agentEditsDir)
}

// AgentEditPath is the record file for a vault-relative note path.
func AgentEditPath(root, rel string) (string, error) {
	return SafeJoin(agentEditsRoot(root), ToPosix(rel)+agentEditsSuffix)
}

// AgentEditRecordable reports whether a note is small enough to record.
func AgentEditRecordable(text string) bool {
	return len(text) <= MaxAgentEditNoteBytes
}

// ReadAgentEdit is the note's record, or nil when there is none or it is not
// a record this version understands.
func ReadAgentEdit(root, rel string) *AgentEditRecord {
	abs, err := AgentEditPath(root, rel)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil
	}
	var raw struct {
		Version int     `json:"version"`
		Before  *string `json:"before"`
		After   *string `json:"after"`
		Edits   []struct {
			At    *float64 `json:"at"`
			Agent *string  `json:"agent"`
			Tool  *string  `json:"tool"`
		} `json:"edits"`
		Outside []struct {
			From *string `json:"from"`
			To   *string `json:"to"`
		} `json:"outside"`
	}
	if json.Unmarshal(data, &raw) != nil || raw.Version != AgentEditRecordVersion || raw.Before == nil {
		return nil
	}
	record := &AgentEditRecord{
		Version: AgentEditRecordVersion,
		Before:  *raw.Before,
		After:   raw.After,
		Edits:   []AgentEditEntry{},
		Outside: []AgentEditOutsideChange{},
	}
	for _, e := range raw.Edits {
		if e.At == nil {
			continue
		}
		agent, tool := "", ""
		if e.Agent != nil {
			agent = clip(strings.TrimSpace(*e.Agent), 80)
		}
		if e.Tool != nil {
			tool = clip(strings.TrimSpace(*e.Tool), 80)
		}
		if agent == "" {
			agent = "Assistant"
		}
		record.Edits = append(record.Edits, AgentEditEntry{At: int64(*e.At), Agent: agent, Tool: tool})
	}
	for _, c := range raw.Outside {
		if c.From != nil && c.To != nil {
			record.Outside = append(record.Outside, AgentEditOutsideChange{From: *c.From, To: *c.To})
		}
	}
	return record
}

func clip(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// WriteAgentEdit writes the record atomically, or removes it for nil.
func WriteAgentEdit(root, rel string, record *AgentEditRecord) error {
	abs, err := AgentEditPath(root, rel)
	if err != nil {
		return err
	}
	if record == nil {
		return removeAgentEditFile(root, abs)
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return writeFileAtomic(abs, append(data, '\n'), 0o644, 0o755)
}

func removeAgentEditFile(root, abs string) error {
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	pruneEmptyAgentEditDirs(root, filepath.Dir(abs))
	return nil
}

// pruneEmptyAgentEditDirs leaves no empty folders behind, agent-edits itself
// included: a vault whose changes have all been reviewed carries no trace.
func pruneEmptyAgentEditDirs(root, from string) {
	stop := agentEditsRoot(root)
	for dir := filepath.Clean(from); dir == stop || strings.HasPrefix(dir, stop+string(filepath.Separator)); dir = filepath.Dir(dir) {
		if os.Remove(dir) != nil {
			return
		}
	}
}

// BeginAgentEdit is the record to write BEFORE an edit, given the record on
// disk (or nil) and the note as it is now.
func BeginAgentEdit(existing *AgentEditRecord, current string) *AgentEditRecord {
	fresh := &AgentEditRecord{
		Version: AgentEditRecordVersion,
		Before:  current,
		After:   nil,
		Edits:   []AgentEditEntry{},
		Outside: []AgentEditOutsideChange{},
	}
	if existing == nil {
		return fresh
	}
	next := *existing
	next.Edits = append([]AgentEditEntry{}, existing.Edits...)
	next.Outside = append([]AgentEditOutsideChange{}, existing.Outside...)
	next.After = nil
	if existing.After == nil || *existing.After == current {
		return &next
	}
	if len(existing.Outside) >= MaxAgentEditOutside {
		return fresh
	}
	next.Outside = append(next.Outside, AgentEditOutsideChange{From: *existing.After, To: current})
	return &next
}

// FinishAgentEdit is the record to write AFTER an edit.
func FinishAgentEdit(started *AgentEditRecord, current string, entry AgentEditEntry) *AgentEditRecord {
	next := *started
	after := current
	next.After = &after
	edits := append(append([]AgentEditEntry{}, started.Edits...), entry)
	if len(edits) > MaxAgentEditEntries {
		edits = edits[len(edits)-MaxAgentEditEntries:]
	}
	next.Edits = edits
	if next.Outside == nil {
		next.Outside = []AgentEditOutsideChange{}
	}
	return &next
}

// moveAgentEdit carries a note's record along with the note. A record waiting
// at the destination belongs to no note (the one it was for moved or went
// outside ZenNotes) and is discarded, the way a stale creation date is.
func (v *Vault) moveAgentEdit(oldRel, nextRel string) error {
	oldAbs, err := AgentEditPath(v.root, oldRel)
	if err != nil {
		return err
	}
	nextAbs, err := AgentEditPath(v.root, nextRel)
	if err != nil {
		return err
	}
	if oldAbs == nextAbs {
		return nil
	}
	if _, err := os.Stat(oldAbs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return removeAgentEditFile(v.root, nextAbs)
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(nextAbs), v.dirMode); err != nil {
		return err
	}
	if err := os.Rename(oldAbs, nextAbs); err != nil {
		return err
	}
	pruneEmptyAgentEditDirs(v.root, filepath.Dir(oldAbs))
	return nil
}

// removeAgentEdit drops a deleted note's record: nothing is left to review.
func (v *Vault) removeAgentEdit(rel string) error {
	abs, err := AgentEditPath(v.root, rel)
	if err != nil {
		return err
	}
	return removeAgentEditFile(v.root, abs)
}

// removeAgentEditTree drops the record of a note or every record under a
// folder (an entry of the trash being emptied may be either).
func (v *Vault) removeAgentEditTree(rel string) error {
	_ = v.removeAgentEdit(rel)
	dir, err := SafeJoin(agentEditsRoot(v.root), ToPosix(rel))
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	pruneEmptyAgentEditDirs(v.root, filepath.Dir(dir))
	return nil
}
