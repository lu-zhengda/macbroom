package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	h := New(path)

	entry := Entry{
		Timestamp:  time.Date(2024, 2, 14, 10, 30, 0, 0, time.UTC),
		Category:   "System Junk",
		Items:      5,
		BytesFreed: 1024 * 1024 * 100, // 100 MB
		Method:     "trash",
	}

	if err := h.Record(entry); err != nil {
		t.Fatalf("failed to record entry: %v", err)
	}

	entries, err := h.Load()
	if err != nil {
		t.Fatalf("failed to load entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.Category != "System Junk" {
		t.Errorf("expected category %q, got %q", "System Junk", got.Category)
	}
	if got.Items != 5 {
		t.Errorf("expected 5 items, got %d", got.Items)
	}
	if got.BytesFreed != 1024*1024*100 {
		t.Errorf("expected %d bytes freed, got %d", 1024*1024*100, got.BytesFreed)
	}
	if got.Method != "trash" {
		t.Errorf("expected method %q, got %q", "trash", got.Method)
	}
}

func TestMultipleRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	h := New(path)

	entries := []Entry{
		{
			Timestamp:  time.Date(2024, 2, 13, 15, 0, 0, 0, time.UTC),
			Category:   "Browser Cache",
			Items:      3,
			BytesFreed: 1024 * 1024 * 800,
			Method:     "trash",
		},
		{
			Timestamp:  time.Date(2024, 2, 14, 10, 30, 0, 0, time.UTC),
			Category:   "System Junk",
			Items:      5,
			BytesFreed: 1024 * 1024 * 100,
			Method:     "permanent",
		},
		{
			Timestamp:  time.Date(2024, 2, 14, 12, 0, 0, 0, time.UTC),
			Category:   "System Junk",
			Items:      2,
			BytesFreed: 1024 * 1024 * 50,
			Method:     "trash",
		},
	}

	for _, e := range entries {
		if err := h.Record(e); err != nil {
			t.Fatalf("failed to record entry: %v", err)
		}
	}

	loaded, err := h.Load()
	if err != nil {
		t.Fatalf("failed to load entries: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(loaded))
	}

	// Verify entries are in order they were appended
	if loaded[0].Category != "Browser Cache" {
		t.Errorf("expected first entry category %q, got %q", "Browser Cache", loaded[0].Category)
	}
	if loaded[1].Category != "System Junk" {
		t.Errorf("expected second entry category %q, got %q", "System Junk", loaded[1].Category)
	}
	if loaded[2].Category != "System Junk" {
		t.Errorf("expected third entry category %q, got %q", "System Junk", loaded[2].Category)
	}
}

func TestStats(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	h := New(path)

	entries := []Entry{
		{
			Timestamp:  time.Date(2024, 2, 10, 10, 0, 0, 0, time.UTC),
			Category:   "System Junk",
			Items:      5,
			BytesFreed: 1024 * 1024 * 1024, // 1 GB
			Method:     "trash",
		},
		{
			Timestamp:  time.Date(2024, 2, 11, 11, 0, 0, 0, time.UTC),
			Category:   "Browser Cache",
			Items:      3,
			BytesFreed: 1024 * 1024 * 500, // 500 MB
			Method:     "trash",
		},
		{
			Timestamp:  time.Date(2024, 2, 12, 12, 0, 0, 0, time.UTC),
			Category:   "System Junk",
			Items:      8,
			BytesFreed: 1024 * 1024 * 1024 * 2, // 2 GB
			Method:     "permanent",
		},
		{
			Timestamp:  time.Date(2024, 2, 13, 13, 0, 0, 0, time.UTC),
			Category:   "Browser Cache",
			Items:      2,
			BytesFreed: 1024 * 1024 * 300, // 300 MB
			Method:     "trash",
		},
		{
			Timestamp:  time.Date(2024, 2, 14, 14, 0, 0, 0, time.UTC),
			Category:   "Large & Old Files",
			Items:      1,
			BytesFreed: 1024 * 1024 * 200, // 200 MB
			Method:     "permanent",
		},
		{
			Timestamp:  time.Date(2024, 2, 15, 15, 0, 0, 0, time.UTC),
			Category:   "System Junk",
			Items:      4,
			BytesFreed: 1024 * 1024 * 750, // 750 MB
			Method:     "trash",
		},
	}

	for _, e := range entries {
		if err := h.Record(e); err != nil {
			t.Fatalf("failed to record entry: %v", err)
		}
	}

	stats := h.Stats()

	// Total freed = 1GB + 500MB + 2GB + 300MB + 200MB + 750MB
	expectedTotal := int64(1024*1024*1024) + int64(1024*1024*500) +
		int64(1024*1024*1024*2) + int64(1024*1024*300) +
		int64(1024*1024*200) + int64(1024*1024*750)

	if stats.TotalFreed != expectedTotal {
		t.Errorf("expected total freed %d, got %d", expectedTotal, stats.TotalFreed)
	}

	if stats.TotalCleanups != 6 {
		t.Errorf("expected 6 total cleanups, got %d", stats.TotalCleanups)
	}

	// By category checks
	if len(stats.ByCategory) != 3 {
		t.Errorf("expected 3 categories, got %d", len(stats.ByCategory))
	}

	sysStats, ok := stats.ByCategory["System Junk"]
	if !ok {
		t.Fatal("expected System Junk in stats")
	}
	if sysStats.Cleanups != 3 {
		t.Errorf("expected 3 System Junk cleanups, got %d", sysStats.Cleanups)
	}

	browserStats, ok := stats.ByCategory["Browser Cache"]
	if !ok {
		t.Fatal("expected Browser Cache in stats")
	}
	if browserStats.Cleanups != 2 {
		t.Errorf("expected 2 Browser Cache cleanups, got %d", browserStats.Cleanups)
	}

	largeStats, ok := stats.ByCategory["Large & Old Files"]
	if !ok {
		t.Fatal("expected Large & Old Files in stats")
	}
	if largeStats.Cleanups != 1 {
		t.Errorf("expected 1 Large & Old Files cleanup, got %d", largeStats.Cleanups)
	}

	// Recent should be last 5 entries (the 6 total minus oldest)
	if len(stats.Recent) != 5 {
		t.Errorf("expected 5 recent entries, got %d", len(stats.Recent))
	}

	// Most recent first
	if stats.Recent[0].Category != "System Junk" {
		t.Errorf("expected most recent to be System Junk, got %q", stats.Recent[0].Category)
	}
	if stats.Recent[0].Timestamp != time.Date(2024, 2, 15, 15, 0, 0, 0, time.UTC) {
		t.Errorf("expected most recent timestamp 2024-02-15, got %v", stats.Recent[0].Timestamp)
	}
}

func TestEmptyHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	h := New(path)
	stats := h.Stats()

	if stats.TotalFreed != 0 {
		t.Errorf("expected 0 total freed, got %d", stats.TotalFreed)
	}
	if stats.TotalCleanups != 0 {
		t.Errorf("expected 0 total cleanups, got %d", stats.TotalCleanups)
	}
	if len(stats.ByCategory) != 0 {
		t.Errorf("expected empty category map, got %d entries", len(stats.ByCategory))
	}
	if len(stats.Recent) != 0 {
		t.Errorf("expected empty recent list, got %d entries", len(stats.Recent))
	}
}

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expected := filepath.Join(home, ".local", "share", "macbroom", "history.json")
	if path != expected {
		t.Errorf("expected default path %q, got %q", expected, path)
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	// Write invalid JSON
	if err := os.WriteFile(path, []byte("not json at all"), 0o644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	h := New(path)
	_, err := h.Load()
	if err == nil {
		t.Error("expected error loading corrupt file, got nil")
	}
}

func TestRecordCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "history.json")

	h := New(path)

	entry := Entry{
		Timestamp:  time.Now(),
		Category:   "System Junk",
		Items:      1,
		BytesFreed: 1024,
		Method:     "trash",
	}

	if err := h.Record(entry); err != nil {
		t.Fatalf("failed to record entry with nested path: %v", err)
	}

	entries, err := h.Load()
	if err != nil {
		t.Fatalf("failed to load entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}
