package watch

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/michaelteague/ashen-sync/internal/detect"
	"github.com/michaelteague/ashen-sync/internal/processx"
	"github.com/michaelteague/ashen-sync/internal/syncer"
)

type Event struct {
	Time    time.Time
	Level   string // info|warn|error
	Message string
}

type Watcher struct {
	Engine       syncer.Engine
	QuietPeriod  time.Duration
	PullInterval time.Duration
	OnEvent      func(Event)
	OnConflict   func(*syncer.ConflictError)

	mu       sync.Mutex
	pending  map[string]time.Time // gameID -> last change
	running  bool
	cancel   context.CancelFunc
}

func (w *Watcher) emit(level, msg string) {
	if w.OnEvent != nil {
		w.OnEvent(Event{Time: time.Now(), Level: level, Message: msg})
	}
}

func (w *Watcher) Start(parent context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("watcher already running")
	}
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	w.running = true
	w.pending = map[string]time.Time{}
	w.mu.Unlock()

	if w.QuietPeriod <= 0 {
		w.QuietPeriod = 15 * time.Second
	}
	if w.PullInterval <= 0 {
		w.PullInterval = 5 * time.Minute
	}

	locations, err := detect.FindSaves(w.Engine.Cfg.GameOverrides)
	if err != nil {
		cancel()
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
		return err
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		cancel()
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
		return err
	}

	watched := map[string]detect.SaveLocation{}
	for _, loc := range locations {
		if !loc.Exists {
			continue
		}
		if err := fsWatcher.Add(loc.Path); err != nil {
			w.emit("warn", fmt.Sprintf("watch add failed for %s: %v", loc.Path, err))
			continue
		}
		watched[loc.Path] = loc
		w.emit("info", fmt.Sprintf("watching %s (%s)", loc.Game.ID, loc.Path))
	}

	go w.loop(ctx, fsWatcher, watched)
	return nil
}

func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.running = false
}

func (w *Watcher) Running() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

func (w *Watcher) loop(ctx context.Context, fsWatcher *fsnotify.Watcher, watched map[string]detect.SaveLocation) {
	defer fsWatcher.Close()
	defer func() {
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
	}()

	debounce := time.NewTicker(1 * time.Second)
	defer debounce.Stop()
	pullTick := time.NewTicker(w.PullInterval)
	defer pullTick.Stop()

	for {
		select {
		case <-ctx.Done():
			w.emit("info", "watcher stopped")
			return
		case ev, ok := <-fsWatcher.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
				continue
			}
			dir := filepath.Dir(ev.Name)
			loc, ok := watched[dir]
			if !ok {
				// sometimes event is on the directory itself
				loc, ok = watched[ev.Name]
				if !ok {
					continue
				}
			}
			w.mu.Lock()
			w.pending[loc.Game.ID] = time.Now()
			w.mu.Unlock()
		case err, ok := <-fsWatcher.Errors:
			if !ok {
				return
			}
			w.emit("error", err.Error())
		case <-debounce.C:
			w.flushPending(ctx, watched)
		case <-pullTick.C:
			w.idlePull(ctx, watched)
		}
	}
}

func (w *Watcher) flushPending(ctx context.Context, watched map[string]detect.SaveLocation) {
	now := time.Now()
	w.mu.Lock()
	var due []string
	for id, t := range w.pending {
		if now.Sub(t) >= w.QuietPeriod {
			due = append(due, id)
			delete(w.pending, id)
		}
	}
	w.mu.Unlock()

	byID := map[string]detect.SaveLocation{}
	for _, loc := range watched {
		byID[loc.Game.ID] = loc
	}

	for _, id := range due {
		loc := byID[id]
		running, err := processx.Running(loc.Game.ProcessNames)
		if err != nil {
			w.emit("error", err.Error())
			continue
		}
		if running {
			w.emit("info", fmt.Sprintf("skip push for %s: game still running", id))
			// re-queue until idle
			w.mu.Lock()
			w.pending[id] = time.Now()
			w.mu.Unlock()
			continue
		}
		w.emit("info", fmt.Sprintf("quiet period elapsed; pushing %s", id))
		if err := w.Engine.Push(ctx, id, syncer.PreferAbort); err != nil {
			if c, ok := syncer.IsConflict(err); ok {
				w.emit("warn", c.Error())
				if w.OnConflict != nil {
					w.OnConflict(c)
				}
				continue
			}
			w.emit("error", fmt.Sprintf("push %s: %v", id, err))
			continue
		}
		w.emit("info", fmt.Sprintf("pushed %s", id))
	}
}

func (w *Watcher) idlePull(ctx context.Context, watched map[string]detect.SaveLocation) {
	for _, loc := range watched {
		running, err := processx.Running(loc.Game.ProcessNames)
		if err != nil {
			w.emit("error", err.Error())
			continue
		}
		if running {
			continue
		}
		w.mu.Lock()
		_, pending := w.pending[loc.Game.ID]
		w.mu.Unlock()
		if pending {
			continue
		}
		w.emit("info", fmt.Sprintf("idle pull check for %s", loc.Game.ID))
		if err := w.Engine.Pull(ctx, loc.Game.ID, syncer.PreferAbort); err != nil {
			if c, ok := syncer.IsConflict(err); ok {
				w.emit("warn", c.Error())
				if w.OnConflict != nil {
					w.OnConflict(c)
				}
				continue
			}
			// empty remote is fine
			w.emit("info", fmt.Sprintf("pull %s: %v", loc.Game.ID, err))
			continue
		}
		w.emit("info", fmt.Sprintf("pulled %s", loc.Game.ID))
	}
}
