package scanner

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSimulatorScanner_Name(t *testing.T) {
	s := NewSimulatorScanner("")
	if s.Name() != "iOS Simulators" {
		t.Errorf("expected name %q, got %q", "iOS Simulators", s.Name())
	}
}

func TestSimulatorScanner_Description(t *testing.T) {
	s := NewSimulatorScanner("")
	want := "iOS Simulator devices and caches"
	if s.Description() != want {
		t.Errorf("expected description %q, got %q", want, s.Description())
	}
}

func TestSimulatorScanner_Risk(t *testing.T) {
	s := NewSimulatorScanner("")
	if s.Risk() != Moderate {
		t.Errorf("expected risk Moderate, got %s", s.Risk())
	}
}

func TestSimulatorScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewSimulatorScanner("")
}

func TestSimulatorScanner_SkipsIfXcodeNotInstalled(t *testing.T) {
	s := NewSimulatorScanner("")
	s.lookPath = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("expected no error when Xcode not installed, got %v", err)
	}
	if targets != nil {
		t.Errorf("expected nil targets when Xcode not installed, got %v", targets)
	}
}

func TestSimulatorScanner_FindsDevices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create device directories with some data files.
	devicesDir := filepath.Join(tmpDir, "Developer", "CoreSimulator", "Devices")
	device1 := filepath.Join(devicesDir, "AAAAAAAA-1111-2222-3333-444444444444")
	device2 := filepath.Join(devicesDir, "BBBBBBBB-5555-6666-7777-888888888888")

	for _, d := range []string{device1, device2} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		// Write a file so DirSize returns > 0.
		if err := os.WriteFile(filepath.Join(d, "data.plist"), make([]byte, 4096), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := NewSimulatorScanner(tmpDir)
	s.lookPath = func(file string) (string, error) {
		return "/usr/bin/xcrun", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		// Return empty unavailable devices list.
		return []byte(`{"devices":{}}`), nil
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find 2 device directories.
	deviceTargets := filterByDescription(targets, "Simulator device data")
	if len(deviceTargets) != 2 {
		t.Fatalf("expected 2 device targets, got %d: %+v", len(deviceTargets), deviceTargets)
	}

	for _, tgt := range deviceTargets {
		if tgt.Category != "iOS Simulators" {
			t.Errorf("expected category %q, got %q", "iOS Simulators", tgt.Category)
		}
		if tgt.Risk != Moderate {
			t.Errorf("expected risk Moderate, got %s", tgt.Risk)
		}
		if tgt.Size == 0 {
			t.Errorf("expected non-zero size for device target %q", tgt.Path)
		}
		if !tgt.IsDir {
			t.Errorf("expected IsDir=true for device target %q", tgt.Path)
		}
	}
}

func TestSimulatorScanner_FindsCaches(t *testing.T) {
	tmpDir := t.TempDir()

	// Create cache directory with data.
	cachesDir := filepath.Join(tmpDir, "Developer", "CoreSimulator", "Caches")
	cacheEntry := filepath.Join(cachesDir, "com.apple.CoreSimulator.SimRuntime.iOS-17-0")
	if err := os.MkdirAll(cacheEntry, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheEntry, "cache.db"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewSimulatorScanner(tmpDir)
	s.lookPath = func(file string) (string, error) {
		return "/usr/bin/xcrun", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(`{"devices":{}}`), nil
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cacheTargets := filterByDescription(targets, "Simulator cache")
	if len(cacheTargets) != 1 {
		t.Fatalf("expected 1 cache target, got %d: %+v", len(cacheTargets), cacheTargets)
	}

	tgt := cacheTargets[0]
	if tgt.Risk != Safe {
		t.Errorf("expected risk Safe for cache, got %s", tgt.Risk)
	}
	if tgt.Category != "iOS Simulators" {
		t.Errorf("expected category %q, got %q", "iOS Simulators", tgt.Category)
	}
	if tgt.Size == 0 {
		t.Errorf("expected non-zero size for cache target")
	}
}

func TestSimulatorScanner_UnavailableDevices(t *testing.T) {
	tmpDir := t.TempDir()

	// simctl output with unavailable devices.
	simctlOutput := simctlDevicesJSON(map[string][]simctlDevice{
		"com.apple.CoreSimulator.SimRuntime.iOS-15-0": {
			{UDID: "AAA-111", Name: "iPhone 13", State: "Shutdown", IsAvailable: false},
		},
		"com.apple.CoreSimulator.SimRuntime.iOS-16-0": {
			{UDID: "BBB-222", Name: "iPhone 14", State: "Shutdown", IsAvailable: false},
			{UDID: "CCC-333", Name: "iPhone 14 Pro", State: "Shutdown", IsAvailable: false},
		},
	})

	s := NewSimulatorScanner(tmpDir)
	s.lookPath = func(file string) (string, error) {
		return "/usr/bin/xcrun", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return simctlOutput, nil
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	unavailTargets := filterByDescription(targets, "Unavailable simulator")
	if len(unavailTargets) != 3 {
		t.Fatalf("expected 3 unavailable device targets, got %d: %+v", len(unavailTargets), unavailTargets)
	}

	// All unavailable devices should have Moderate risk.
	for _, tgt := range unavailTargets {
		if tgt.Risk != Moderate {
			t.Errorf("expected risk Moderate for unavailable device, got %s", tgt.Risk)
		}
		if tgt.Category != "iOS Simulators" {
			t.Errorf("expected category %q, got %q", "iOS Simulators", tgt.Category)
		}
	}
}

func TestSimulatorScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	s := NewSimulatorScanner("")
	s.lookPath = func(file string) (string, error) {
		return "/usr/bin/xcrun", nil
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

func TestSimulatorScanner_SimctlFails(t *testing.T) {
	tmpDir := t.TempDir()

	s := NewSimulatorScanner(tmpDir)
	s.lookPath = func(file string) (string, error) {
		return "/usr/bin/xcrun", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, &exitError{msg: "simctl failed"}
	}

	// Should not return an error — just skip unavailable device detection.
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("expected no error when simctl fails, got %v", err)
	}
	// No device/cache dirs exist in tmpDir, so no targets expected.
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

// --- helpers ---

func filterByDescription(targets []Target, prefix string) []Target {
	var result []Target
	for _, t := range targets {
		if len(t.Description) >= len(prefix) && t.Description[:len(prefix)] == prefix {
			result = append(result, t)
		}
	}
	return result
}

// simctlDevice mirrors the JSON structure from xcrun simctl list.
type simctlDevice struct {
	UDID        string `json:"udid"`
	Name        string `json:"name"`
	State       string `json:"state"`
	IsAvailable bool   `json:"isAvailable"`
}

func simctlDevicesJSON(devices map[string][]simctlDevice) []byte {
	out := struct {
		Devices map[string][]simctlDevice `json:"devices"`
	}{Devices: devices}
	b, _ := json.Marshal(out)
	return b
}
