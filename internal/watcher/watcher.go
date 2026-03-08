package watcher

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	fsWatcher *fsnotify.Watcher
	root      string
	OnTrigger func()
}

func New(root string, onTrigger func()) (*Watcher, error) {
	fsW, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher: fsW,
		root:      root,
		OnTrigger: onTrigger,
	}

	return w, nil
}

func (w *Watcher) Start() error {
	if err := w.watchRecursive(w.root); err != nil {
		return err
	}

	go w.listenEvents()
	return nil
}

func (w *Watcher) Stop() {
	if w.fsWatcher != nil {
		w.fsWatcher.Close()
	}
}

func (w *Watcher) listenEvents() {
	var (
		debounceDuration = 200 * time.Millisecond
		timer            *time.Timer
	)

	trigger := func() {
		if timer == nil {
			timer = time.AfterFunc(debounceDuration, func() {
				w.OnTrigger()
			})
		} else {
			timer.Reset(debounceDuration)
		}
	}

	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Create) {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					if !w.isIgnored(event.Name) {
						slog.Debug("Watching new directory", "dir", event.Name)
						w.watchRecursive(event.Name)
					}
				}
			} else if event.Has(fsnotify.Remove) {
				slog.Debug("Removed", "file", event.Name)
			}

			if w.shouldTriggerBuild(event) {
				slog.Debug("Detected change", "file", event.Name, "op", event.Op)
				trigger()
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			slog.Error("Watcher error", "error", err)
		}
	}
}

func (w *Watcher) watchRecursive(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if w.isIgnored(path) {
				return filepath.SkipDir
			}
			err = w.fsWatcher.Add(path)
			if err != nil {
				slog.Error("Failed to watch directory", "dir", path, "error", err)
			}
		}
		return nil
	})
}

func (w *Watcher) isIgnored(path string) bool {
	base := filepath.Base(path)

	switch base {
	case ".git", "node_modules", "bin":
		return true
	}

	return false
}

func (w *Watcher) shouldTriggerBuild(event fsnotify.Event) bool {
	base := filepath.Base(event.Name)

	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") || strings.HasPrefix(base, "#") {
		return false
	}

	if strings.HasSuffix(base, ".exe") || strings.Contains(event.Name, string(filepath.Separator)+"bin"+string(filepath.Separator)) {
		return false
	}

	if event.Op == fsnotify.Chmod {
		return false
	}

	return true
}
