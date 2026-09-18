package skill

import (
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
)

type ChangeEvent struct {
	Name string
	Type string // "added", "removed", "modified"
}

type Watcher struct {
	loader    *SkillLoader
	watcher   *fsnotify.Watcher
	Changes   chan ChangeEvent
	dirs      []string
	done      chan struct{}
	started   atomic.Bool
	closeOnce sync.Once
}

func NewWatcher(loader *SkillLoader, dirs []string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if err := w.Add(dir); err != nil {
			slog.Warn("cannot watch dir", "dir", dir, "error", err)
		}
	}

	return &Watcher{
		loader:  loader,
		watcher: w,
		Changes: make(chan ChangeEvent, 100),
		dirs:    dirs,
		done:    make(chan struct{}),
	}, nil
}

func (w *Watcher) Start() {
	if !w.started.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer close(w.done)
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				w.handleEvent(event)
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				slog.Error("watcher error", "error", err)
			}
		}
	}()
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	if event.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Write) == 0 {
		return
	}

	skillName := w.detectSkillName(event.Name)
	if skillName == "" {
		return
	}

	w.loader.InvalidateCache(skillName)

	eventType := "modified"
	if event.Op&fsnotify.Create != 0 {
		eventType = "added"
	} else if event.Op&fsnotify.Remove != 0 {
		eventType = "removed"
	}

	select {
	case w.Changes <- ChangeEvent{Name: skillName, Type: eventType}:
	default:
	}
}

func (w *Watcher) detectSkillName(path string) string {
	// Walk up from changed path to find the skill directory
	// Skill dirs are structured as: {base}/{skill_name}/{...}
	for _, dir := range w.dirs {
		if len(path) > len(dir) && path[:len(dir)] == dir {
			rel := path[len(dir)+1:]
			for i := 0; i < len(rel); i++ {
				if rel[i] == '\\' || rel[i] == '/' {
					return rel[:i]
				}
			}
			return rel
		}
	}
	return ""
}

func (w *Watcher) Close() error {
	var err error
	w.closeOnce.Do(func() {
		err = w.watcher.Close()
		if w.started.Load() {
			<-w.done
		}
		close(w.Changes)
	})
	return err
}
