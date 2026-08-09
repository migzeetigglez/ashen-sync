//go:build !nogui

package gui

import (
	"strings"
	"sync"
	"time"
)

const (
	activityLogMaxLines   = 300
	activityLogFlushEvery = 50 * time.Millisecond
)

// activityLog buffers selectable log lines and flushes them in batches to the Entry.
type activityLog struct {
	mu        sync.Mutex
	pending   []string
	lineCount int
	gen       uint64
	flush     func(text string, lineCount int, gen uint64) // may be called off the UI thread
	timer     *time.Timer
	fullText  strings.Builder
}

func newActivityLog(flush func(text string, lineCount int, gen uint64)) *activityLog {
	return &activityLog{flush: flush}
}

func (a *activityLog) currentGen() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.gen
}

func (a *activityLog) clear() {
	a.mu.Lock()
	a.pending = nil
	a.lineCount = 0
	a.fullText.Reset()
	a.gen++
	gen := a.gen
	if a.timer != nil {
		a.timer.Stop()
		a.timer = nil
	}
	flush := a.flush
	a.mu.Unlock()
	flush("", 0, gen)
}

func (a *activityLog) appendLine(line string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pending = append(a.pending, line)
	if a.timer == nil {
		a.timer = time.AfterFunc(activityLogFlushEvery, a.flushPending)
	}
}

func (a *activityLog) flushPending() {
	a.mu.Lock()
	pending := a.pending
	a.pending = nil
	a.timer = nil
	if len(pending) == 0 {
		a.mu.Unlock()
		return
	}
	for _, line := range pending {
		a.fullText.WriteString(line)
		a.lineCount++
	}
	a.trimLocked()
	text := a.fullText.String()
	lines := a.lineCount
	gen := a.gen
	flush := a.flush
	a.mu.Unlock()

	t0 := time.Now()
	flush(text, lines, gen)
	uiTrace("activity_log.flush", t0)
}

func (a *activityLog) trimLocked() {
	for a.lineCount > activityLogMaxLines {
		s := a.fullText.String()
		idx := strings.IndexByte(s, '\n')
		if idx < 0 {
			a.fullText.Reset()
			a.lineCount = 0
			return
		}
		a.fullText.Reset()
		a.fullText.WriteString(s[idx+1:])
		a.lineCount--
	}
}
