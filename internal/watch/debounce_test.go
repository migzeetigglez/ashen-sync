package watch

import (
	"testing"
	"time"
)

func TestQuietPeriodLogic(t *testing.T) {
	w := &Watcher{
		QuietPeriod: 50 * time.Millisecond,
		pending:     map[string]time.Time{"ds3": time.Now()},
	}
	// not due yet
	if time.Since(w.pending["ds3"]) >= w.QuietPeriod {
		t.Fatal("should still be within quiet period")
	}
	time.Sleep(60 * time.Millisecond)
	if time.Since(w.pending["ds3"]) < w.QuietPeriod {
		t.Fatal("should be past quiet period")
	}
}
