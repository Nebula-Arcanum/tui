// Package fswatch batches file-system events under a vault root: the terminal
// UI's live reload and `zn task watch` (ZenNotes/zennotes#941) share it.
package fswatch

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher batches file-system events under the vault root.
type Watcher struct {
	root    string
	w       *fsnotify.Watcher
	batches chan []string
	closeCh chan struct{}
	once    sync.Once
}

// Start watches root and every folder under it, folders created later
// included, and batches changes 300 ms after the last one.
func Start(root string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	vw := &Watcher{root: root, w: w, batches: make(chan []string, 4), closeCh: make(chan struct{})}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if SkipDir(d.Name()) && path != root {
				return filepath.SkipDir
			}
			_ = w.Add(path)
		}
		return nil
	})
	go vw.loop()
	return vw, nil
}

// SkipDir names the folders never watched: tool and trash folders.
func SkipDir(name string) bool {
	return name == ".git" || name == "node_modules" || name == ".obsidian" || name == ".trash"
}

func (vw *Watcher) loop() {
	pending := map[string]bool{}
	var timer *time.Timer
	var timerCh <-chan time.Time
	flush := func() {
		if len(pending) == 0 {
			return
		}
		paths := make([]string, 0, len(pending))
		for p := range pending {
			paths = append(paths, p)
		}
		pending = map[string]bool{}
		select {
		case vw.batches <- paths:
		case <-vw.closeCh:
		}
	}
	for {
		select {
		case <-vw.closeCh:
			return
		case ev, ok := <-vw.w.Events:
			if !ok {
				return
			}
			name := ev.Name
			base := filepath.Base(name)
			if strings.HasSuffix(base, ".tmp") || strings.HasPrefix(base, ".DS_Store") {
				continue
			}
			if ev.Has(fsnotify.Create) {
				if info, err := os.Stat(name); err == nil && info.IsDir() && !SkipDir(base) {
					_ = vw.w.Add(name)
				}
			}
			rel, err := filepath.Rel(vw.root, name)
			if err != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			pending[rel] = true
			if timer == nil {
				timer = time.NewTimer(300 * time.Millisecond)
			} else {
				timer.Reset(300 * time.Millisecond)
			}
			timerCh = timer.C
		case <-timerCh:
			timerCh = nil
			flush()
		case _, ok := <-vw.w.Errors:
			if !ok {
				return
			}
		}
	}
}

// Next blocks until a batch of changed vault-relative paths arrives; false
// once the watcher is closed.
func (vw *Watcher) Next() ([]string, bool) {
	select {
	case paths, ok := <-vw.batches:
		return paths, ok
	case <-vw.closeCh:
		return nil, false
	}
}

// Close stops watching; safe to call more than once.
func (vw *Watcher) Close() {
	vw.once.Do(func() {
		close(vw.closeCh)
		_ = vw.w.Close()
	})
}
