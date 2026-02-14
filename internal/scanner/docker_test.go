package scanner

import (
	"context"
	"os/exec"
	"testing"
)

func TestDockerScanner_Name(t *testing.T) {
	s := NewDockerScanner()
	if s.Name() != "Docker" {
		t.Errorf("expected name Docker, got %s", s.Name())
	}
}

func TestDockerScanner_Description(t *testing.T) {
	s := NewDockerScanner()
	want := "Docker images, containers, and build cache"
	if s.Description() != want {
		t.Errorf("expected description %q, got %q", want, s.Description())
	}
}

func TestDockerScanner_Risk(t *testing.T) {
	s := NewDockerScanner()
	if s.Risk() != Moderate {
		t.Errorf("expected risk Moderate, got %s", s.Risk())
	}
}

func TestDockerScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewDockerScanner()
}

func TestDockerScanner_SkipsIfNotInstalled(t *testing.T) {
	s := NewDockerScanner()
	// Override lookPath to simulate docker not being installed
	s.lookPath = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("expected no error when docker not installed, got %v", err)
	}
	if targets != nil {
		t.Errorf("expected nil targets when docker not installed, got %v", targets)
	}
}

func TestDockerScanner_SkipsIfDaemonNotRunning(t *testing.T) {
	s := NewDockerScanner()
	// Docker is "installed" but commands fail (daemon not running)
	s.lookPath = func(file string) (string, error) {
		return "/usr/local/bin/docker", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, &exitError{msg: "Cannot connect to the Docker daemon"}
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("expected no error when daemon not running, got %v", err)
	}
	if targets != nil {
		t.Errorf("expected nil targets when daemon not running, got %v", targets)
	}
}

func TestDockerScanner_ParsesDanglingImages(t *testing.T) {
	s := NewDockerScanner()
	s.lookPath = func(file string) (string, error) {
		return "/usr/local/bin/docker", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		// Match on the command arguments to return appropriate output
		for _, arg := range args {
			if arg == "dangling=true" {
				return []byte("abc123\t1.2GB\ndef456\t500MB\n"), nil
			}
			if arg == "df" {
				return []byte(`{"Type":"Build Cache","Size":"2.5GB","Reclaimable":"1.8GB (72%)"}`), nil
			}
		}
		return nil, nil
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect 2 dangling images + 1 build cache entry
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d: %+v", len(targets), targets)
	}

	// Verify dangling images
	wantDescs := []string{"Dangling image (1.2GB)", "Dangling image (500MB)"}
	for i := 0; i < 2; i++ {
		if targets[i].Category != "Docker" {
			t.Errorf("target[%d]: expected category Docker, got %s", i, targets[i].Category)
		}
		if targets[i].Description != wantDescs[i] {
			t.Errorf("target[%d]: expected description %q, got %q", i, wantDescs[i], targets[i].Description)
		}
		if targets[i].Risk != Moderate {
			t.Errorf("target[%d]: expected risk Moderate, got %s", i, targets[i].Risk)
		}
	}

	if targets[0].Path != "docker image abc123" {
		t.Errorf("target[0]: expected path 'docker image abc123', got %q", targets[0].Path)
	}
	if targets[1].Path != "docker image def456" {
		t.Errorf("target[1]: expected path 'docker image def456', got %q", targets[1].Path)
	}

	// Verify build cache
	if targets[2].Path != "docker build cache" {
		t.Errorf("target[2]: expected path 'docker build cache', got %q", targets[2].Path)
	}
	if targets[2].Risk != Safe {
		t.Errorf("target[2]: expected risk Safe, got %s", targets[2].Risk)
	}
}

func TestDockerScanner_NoDanglingImages(t *testing.T) {
	s := NewDockerScanner()
	s.lookPath = func(file string) (string, error) {
		return "/usr/local/bin/docker", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		for _, arg := range args {
			if arg == "dangling=true" {
				return []byte(""), nil // No dangling images
			}
			if arg == "df" {
				return []byte(""), nil // No build cache info
			}
		}
		return nil, nil
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestDockerScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	s := NewDockerScanner()
	s.lookPath = func(file string) (string, error) {
		return "/usr/local/bin/docker", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, ctx.Err()
	}

	targets, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
	if targets != nil {
		t.Errorf("expected nil targets on context cancellation, got %v", targets)
	}
}

// exitError simulates an exec.ExitError for testing.
type exitError struct {
	msg string
}

func (e *exitError) Error() string {
	return e.msg
}
