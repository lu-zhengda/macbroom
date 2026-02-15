package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// SimulatorScanner detects iOS Simulator device data, caches,
// and unavailable simulator runtimes.
type SimulatorScanner struct {
	libraryBase string

	// lookPath checks whether xcrun is installed.
	// Defaults to exec.LookPath; override in tests.
	lookPath func(file string) (string, error)

	// runCmd executes a command and returns its stdout.
	// Defaults to exec.CommandContext(...).Output(); override in tests.
	runCmd func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// NewSimulatorScanner returns a new SimulatorScanner.
// If libraryPath is empty, the default ~/Library path is used.
func NewSimulatorScanner(libraryPath string) *SimulatorScanner {
	return &SimulatorScanner{
		libraryBase: libraryPath,
		lookPath:    exec.LookPath,
		runCmd: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		},
	}
}

func (s *SimulatorScanner) Name() string        { return "iOS Simulators" }
func (s *SimulatorScanner) Description() string { return "iOS Simulator devices and caches" }
func (s *SimulatorScanner) Risk() RiskLevel     { return Moderate }

func (s *SimulatorScanner) base() string {
	if s.libraryBase != "" {
		return s.libraryBase
	}
	return utils.LibraryPath("")
}

func (s *SimulatorScanner) Scan(ctx context.Context) ([]Target, error) {
	if _, err := s.lookPath("xcrun"); err != nil {
		return nil, nil
	}

	var targets []Target

	// --- Scan device data directories ---
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	devicesDir := filepath.Join(s.base(), "Developer", "CoreSimulator", "Devices")
	devTargets, err := s.scanDir(devicesDir, "Simulator device data", Moderate)
	if err != nil {
		return nil, err
	}
	targets = append(targets, devTargets...)

	// --- Scan cache directories ---
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	cachesDir := filepath.Join(s.base(), "Developer", "CoreSimulator", "Caches")
	cacheTargets, err := s.scanDir(cachesDir, "Simulator cache", Safe)
	if err != nil {
		return nil, err
	}
	targets = append(targets, cacheTargets...)

	// --- Detect unavailable simulator runtimes ---
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	unavail, err := s.findUnavailableDevices(ctx)
	if err != nil {
		// If context was cancelled, propagate the error.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Otherwise skip silently (simctl may fail for various reasons).
	} else {
		targets = append(targets, unavail...)
	}

	return targets, nil
}

// scanDir reads entries from a directory and returns targets for each
// subdirectory with a non-zero size.
func (s *SimulatorScanner) scanDir(dir, description string, risk RiskLevel) ([]Target, error) {
	if !utils.DirExists(dir) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}

	var targets []Target
	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		var size int64
		if info.IsDir() {
			size, _ = utils.DirSize(entryPath)
		} else {
			size = info.Size()
		}

		if size == 0 {
			continue
		}

		targets = append(targets, Target{
			Path:        entryPath,
			Size:        size,
			Category:    "iOS Simulators",
			Description: description,
			Risk:        risk,
			ModTime:     info.ModTime(),
			IsDir:       info.IsDir(),
		})
	}

	return targets, nil
}

// simctlOutput represents the JSON output of `xcrun simctl list devices -j`.
type simctlOutput struct {
	Devices map[string][]simctlDeviceInfo `json:"devices"`
}

type simctlDeviceInfo struct {
	UDID        string `json:"udid"`
	Name        string `json:"name"`
	State       string `json:"state"`
	IsAvailable bool   `json:"isAvailable"`
}

// findUnavailableDevices runs `xcrun simctl list devices unavailable -j`
// and returns targets for each unavailable simulator device.
func (s *SimulatorScanner) findUnavailableDevices(ctx context.Context) ([]Target, error) {
	out, err := s.runCmd(ctx, "xcrun", "simctl", "list", "devices", "unavailable", "-j")
	if err != nil {
		return nil, fmt.Errorf("failed to list unavailable simulators: %w", err)
	}

	var result simctlOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("failed to parse simctl output: %w", err)
	}

	var targets []Target
	for runtime, devices := range result.Devices {
		for _, dev := range devices {
			targets = append(targets, Target{
				Path:        "simulator " + dev.UDID,
				Description: fmt.Sprintf("Unavailable simulator: %s (%s)", dev.Name, runtime),
				Category:    "iOS Simulators",
				Risk:        Moderate,
			})
		}
	}

	return targets, nil
}
