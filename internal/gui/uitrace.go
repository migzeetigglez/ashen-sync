//go:build !nogui

package gui

import (
	"log"
	"os"
	"sync"
	"time"
)

// SC_UI_TRACE=1 logs UI-thread hot-path timings to stderr.
var (
	uiTraceOnce sync.Once
	uiTraceOn   bool
)

func uiTraceEnabled() bool {
	uiTraceOnce.Do(func() {
		uiTraceOn = os.Getenv("SC_UI_TRACE") == "1" || os.Getenv("SC_UI_TRACE") == "true"
	})
	return uiTraceOn
}

func uiTrace(label string, started time.Time) {
	if !uiTraceEnabled() {
		return
	}
	log.Printf("ui-trace %s: %s", label, time.Since(started).Round(time.Microsecond))
}

func uiTraceSince(label string, d time.Duration) {
	if !uiTraceEnabled() {
		return
	}
	log.Printf("ui-trace %s: %s", label, d.Round(time.Microsecond))
}
