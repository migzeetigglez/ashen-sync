//go:build !nogui

package gui

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestActivityLogCapsLines(t *testing.T) {
	var (
		mu   sync.Mutex
		last string
		n    int
	)
	log := newActivityLog(func(text string, lineCount int, _ uint64) {
		mu.Lock()
		defer mu.Unlock()
		last = text
		n = lineCount
	})
	for i := 0; i < activityLogMaxLines+50; i++ {
		log.appendLine(fmt.Sprintf("line-%d\n", i))
	}
	// Wait for debounce flush (single batch).
	time.Sleep(activityLogFlushEvery + 30*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if n != activityLogMaxLines {
		t.Fatalf("lineCount=%d want %d", n, activityLogMaxLines)
	}
	if strings.Contains(last, "line-0\n") {
		t.Fatal("oldest lines should have been trimmed")
	}
	if !strings.Contains(last, fmt.Sprintf("line-%d\n", activityLogMaxLines+49)) {
		t.Fatal("newest line missing")
	}
}

func TestActivityLogClear(t *testing.T) {
	var last string
	var n int
	log := newActivityLog(func(text string, lineCount int, _ uint64) {
		last = text
		n = lineCount
	})
	log.appendLine("a\n")
	log.flushPending()
	log.clear()
	if last != "" || n != 0 {
		t.Fatalf("clear left text=%q lines=%d", last, n)
	}
}

func BenchmarkLogAppendOldStyle(b *testing.B) {
	// Mimics previous per-line Append + strings.Count over growing buffer.
	var buf strings.Builder
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		for j := 0; j < 200; j++ {
			buf.WriteString(fmt.Sprintf("15:04:05 line %d\n", j))
			_ = strings.Count(buf.String(), "\n")
		}
	}
}

func BenchmarkLogAppendBatched(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		log := newActivityLog(func(string, int, uint64) {})
		for j := 0; j < 200; j++ {
			log.appendLine(fmt.Sprintf("15:04:05 line %d\n", j))
		}
		log.flushPending()
	}
}
