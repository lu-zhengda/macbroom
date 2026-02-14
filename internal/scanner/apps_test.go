package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAppScanner_FindRelatedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	appSupport := filepath.Join(tmpDir, "Application Support", "FakeApp")
	if err := os.MkdirAll(appSupport, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appSupport, "config.json"), make([]byte, 256), 0o644); err != nil {
		t.Fatal(err)
	}

	prefs := filepath.Join(tmpDir, "Preferences")
	if err := os.MkdirAll(prefs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prefs, "com.fake.FakeApp.plist"), make([]byte, 128), 0o644); err != nil {
		t.Fatal(err)
	}

	caches := filepath.Join(tmpDir, "Caches", "com.fake.FakeApp")
	if err := os.MkdirAll(caches, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caches, "cache.db"), make([]byte, 512), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewAppScanner("", tmpDir)
	targets, err := s.FindRelatedFiles(context.Background(), "FakeApp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) < 2 {
		t.Fatalf("expected at least 2 related targets, got %d", len(targets))
	}
}

func TestAppScanner_ListApps(t *testing.T) {
	tmpDir := t.TempDir()

	app1 := filepath.Join(tmpDir, "TestApp.app")
	if err := os.MkdirAll(app1, 0o755); err != nil {
		t.Fatal(err)
	}

	s := NewAppScanner(tmpDir, "")
	apps := s.ListApps()
	if len(apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps))
	}
	if apps[0] != "TestApp" {
		t.Errorf("expected TestApp, got %s", apps[0])
	}
}
