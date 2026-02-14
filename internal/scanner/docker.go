package scanner

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

// DockerScanner detects Docker cleanup opportunities including
// dangling images and build cache.
type DockerScanner struct {
	// lookPath is used to check if docker is installed.
	// Defaults to exec.LookPath; override in tests.
	lookPath func(file string) (string, error)

	// runCmd executes a command and returns its stdout.
	// Defaults to exec.CommandContext(...).Output(); override in tests.
	runCmd func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// NewDockerScanner returns a new DockerScanner with default command execution.
func NewDockerScanner() *DockerScanner {
	return &DockerScanner{
		lookPath: exec.LookPath,
		runCmd: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		},
	}
}

func (s *DockerScanner) Name() string        { return "Docker" }
func (s *DockerScanner) Description() string { return "Docker images, containers, and build cache" }
func (s *DockerScanner) Risk() RiskLevel     { return Moderate }

func (s *DockerScanner) Scan(ctx context.Context) ([]Target, error) {
	if _, err := s.lookPath("docker"); err != nil {
		return nil, nil
	}

	var targets []Target

	// Get dangling images.
	out, err := s.runCmd(ctx, "docker", "images", "-f", "dangling=true", "--format", "{{.ID}}\t{{.Size}}")
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, nil // Docker daemon not running, skip silently.
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		desc := "Dangling image"
		if len(parts) > 1 {
			desc += " (" + strings.TrimSpace(parts[1]) + ")"
		}
		targets = append(targets, Target{
			Path:        "docker image " + parts[0],
			Description: desc,
			Category:    "Docker",
			Risk:        Moderate,
		})
	}

	// Get build cache size.
	out, err = s.runCmd(ctx, "docker", "system", "df", "--format", "{{json .}}")
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line == "" {
				continue
			}
			var df struct {
				Type        string `json:"Type"`
				Reclaimable string `json:"Reclaimable"`
				Size        string `json:"Size"`
			}
			if json.Unmarshal([]byte(line), &df) == nil && df.Type == "Build Cache" {
				targets = append(targets, Target{
					Path:        "docker build cache",
					Description: "Build cache (" + df.Size + ", " + df.Reclaimable + " reclaimable)",
					Category:    "Docker",
					Risk:        Safe,
				})
			}
		}
	}

	return targets, nil
}
