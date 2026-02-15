# macbroom v0.3 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add 4 new scanners, worker pool with live progress, treemap visualization for SpaceLens, and comprehensive TUI polish.

**Architecture:** The project follows a scanner-engine-TUI layered architecture. Scanners implement a 4-method interface and register with the engine. The TUI is a Bubble Tea app with view states. v0.3 adds new scanners (same pattern), upgrades the engine with a semaphore-based worker pool, replaces the SpaceLens tree view with a squarified treemap, and introduces a centralized theme system.

**Tech Stack:** Go 1.25, Bubble Tea (charmbracelet/bubbletea), Lipgloss (charmbracelet/lipgloss), Bubbles (charmbracelet/bubbles), Cobra CLI

---

### Task 1: Theme System

Create a centralized color theme to replace scattered inline styles.

**Files:**
- Create: `internal/tui/theme.go`
- Modify: `internal/tui/styles.go`

**Step 1: Create theme.go with the color palette**

```go
package tui

import "github.com/charmbracelet/lipgloss"

// --- Color palette ---

var (
	colorPrimary   = lipgloss.Color("170") // pink/magenta accent
	colorSecondary = lipgloss.Color("212") // lighter pink
	colorSuccess   = lipgloss.Color("82")  // green
	colorWarning   = lipgloss.Color("214") // orange
	colorDanger    = lipgloss.Color("196") // red
	colorDim       = lipgloss.Color("241") // gray
	colorSubtle    = lipgloss.Color("236") // dark gray bg
	colorText      = lipgloss.Color("252") // light gray text
	colorWhite     = lipgloss.Color("255") // white

	// Category-specific colors for the dashboard.
	categoryColors = map[string]lipgloss.Color{
		"System Junk":     lipgloss.Color("75"),  // blue
		"Browser Cache":   lipgloss.Color("214"), // orange
		"Xcode Junk":      lipgloss.Color("141"), // purple
		"Large & Old Files": lipgloss.Color("223"), // tan
		"Docker":          lipgloss.Color("39"),  // cyan
		"Node.js":         lipgloss.Color("119"), // green
		"Homebrew":        lipgloss.Color("208"), // amber
		"iOS Simulators":  lipgloss.Color("183"), // lavender
		"Python":          lipgloss.Color("220"), // yellow
		"Rust":            lipgloss.Color("208"), // orange
		"Go":              lipgloss.Color("75"),  // blue
		"JetBrains":       lipgloss.Color("171"), // pink
	}

	// Bar fill colors by percentage.
	barColorHigh   = lipgloss.Color("196") // red for > 66%
	barColorMedium = lipgloss.Color("214") // orange for > 33%
	barColorLow    = lipgloss.Color("82")  // green for <= 33%

	// Treemap palette — distinct colors for adjacent blocks.
	treemapColors = []lipgloss.Color{
		lipgloss.Color("75"),  // blue
		lipgloss.Color("119"), // green
		lipgloss.Color("214"), // orange
		lipgloss.Color("141"), // purple
		lipgloss.Color("39"),  // cyan
		lipgloss.Color("220"), // yellow
		lipgloss.Color("183"), // lavender
		lipgloss.Color("208"), // amber
	}
)

// categoryColor returns the color for a scanner category, with a default fallback.
func categoryColor(name string) lipgloss.Color {
	if c, ok := categoryColors[name]; ok {
		return c
	}
	return colorDim
}

// barColor returns the fill color based on the percentage filled.
func barColor(ratio float64) lipgloss.Color {
	if ratio > 0.66 {
		return barColorHigh
	}
	if ratio > 0.33 {
		return barColorMedium
	}
	return barColorLow
}
```

**Step 2: Update styles.go to use theme colors**

Replace the hardcoded colors in `styles.go`:

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorSecondary)

	dimStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	helpStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		MarginTop(1)

	statusBarStyle = lipgloss.NewStyle().
		Background(colorSubtle).
		Foreground(colorText).
		Padding(0, 1)

	successStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorSuccess)

	failStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorDanger)

	warnStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorWarning)

	dangerBannerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorDanger).
		Background(lipgloss.Color("52")).
		Padding(0, 1)

	headerBarStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorWhite).
		Background(colorPrimary).
		Padding(0, 1).
		MarginBottom(1)

	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorDim).
		Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(colorSubtle).
		MarginTop(1).
		Padding(0, 1)
)
```

**Step 3: Run tests to verify nothing breaks**

Run: `go build ./... && go test ./... -race`
Expected: All tests PASS, builds cleanly

**Step 4: Commit**

```bash
git add internal/tui/theme.go internal/tui/styles.go
git commit -m "refactor: centralize TUI color theme"
```

---

### Task 2: TUI Layout Polish

Add bordered panels, header bars with breadcrumbs, and a footer bar with context-sensitive keybinds across all views.

**Files:**
- Modify: `internal/tui/app.go` (all view functions)

**Step 1: Add helper functions for header and footer**

Add these to `app.go` (or a new `internal/tui/layout.go`):

```go
// renderHeader draws a header bar with breadcrumb navigation.
func renderHeader(parts ...string) string {
	breadcrumb := "macbroom"
	for _, p := range parts {
		breadcrumb += " > " + p
	}
	return headerBarStyle.Render(breadcrumb) + "\n"
}

// renderFooter draws a footer with keybind hints.
func renderFooter(hints string) string {
	return footerStyle.Render(hints)
}

// renderProgressBar draws a progress bar of the given width.
// ratio should be between 0.0 and 1.0.
func renderProgressBar(ratio float64, width int) string {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	if filled == 0 && ratio > 0 {
		filled = 1
	}
	empty := width - filled
	color := barColor(ratio)
	fillStyle := lipgloss.NewStyle().Foreground(color)
	return "[" + fillStyle.Render(strings.Repeat("█", filled)) + strings.Repeat("░", empty) + "]"
}
```

**Step 2: Update viewMenu to use new layout**

Replace the current `viewMenu`:

```go
func (m Model) viewMenu() string {
	s := renderHeader()

	for i, item := range menuItems {
		if i == m.cursor {
			s += selectedStyle.Render("> "+item.label) + "  " + dimStyle.Render(item.description) + "\n"
		} else {
			s += fmt.Sprintf("  %-15s %s\n", item.label, dimStyle.Render(item.description))
		}
	}

	s += renderFooter("j/k navigate | enter select | q quit")
	return s
}
```

**Step 3: Update viewDashboard to use colored category dots and panels**

Replace the dashboard rendering to use category colors:

```go
func (m Model) viewDashboard() string {
	s := renderHeader("Clean")

	if len(m.results) == 0 {
		s += "No junk found. Your Mac is clean!\n"
		return s + renderFooter("esc back | q quit")
	}

	var totalSize int64
	for i, r := range m.results {
		var catSize int64
		for _, t := range r.Targets {
			catSize += t.Size
		}
		totalSize += catSize

		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		dot := lipgloss.NewStyle().Foreground(categoryColor(r.Category)).Render("●")
		line := fmt.Sprintf("%s%s %-23s %10s  (%d items)",
			cursor, dot, r.Category, utils.FormatSize(catSize), len(r.Targets))

		if i == m.cursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	s += "\n" + statusBarStyle.Render(fmt.Sprintf(" Total reclaimable: %s ", utils.FormatSize(totalSize)))
	s += renderFooter("j/k navigate | enter view details | esc back | q quit")
	return s
}
```

**Step 4: Update remaining views to use dangerBannerStyle, successStyle, failStyle, renderHeader, renderFooter**

Apply the same pattern to: `viewCategory`, `viewConfirm`, `viewResult`, `viewSpaceLens`, `viewSpaceLensConfirm`, `viewDupes`, `viewDupesConfirm`, `viewDupesResult`, `viewUninstallInput`, `viewUninstallResults`, `viewUninstallConfirm`, `viewMaintain`, `viewMaintainResult`.

For each view:
- Replace `titleStyle.Render("macbroom -- X")` with `renderHeader("X")`
- Replace inline `dangerStyle` definitions with `dangerBannerStyle`
- Replace inline `successStyle`/`failStyle` definitions with the global ones
- Replace `helpStyle.Render("...")` with `renderFooter("...")`

**Step 5: Update renderBar to use colored fills**

Replace the existing `renderBar` function in `spacelens.go`:

```go
func renderBar(size, maxSize int64, width int) string {
	if maxSize == 0 {
		return ""
	}
	ratio := float64(size) / float64(maxSize)
	filled := int(ratio * float64(width))
	if filled == 0 && size > 0 {
		filled = 1
	}
	color := barColor(ratio)
	fillStyle := lipgloss.NewStyle().Foreground(color)
	return "[" + fillStyle.Render(strings.Repeat("█", filled)) + strings.Repeat("░", width-filled) + "]"
}
```

**Step 6: Run tests and verify**

Run: `go build ./... && go test ./... -race`
Expected: All tests PASS

**Step 7: Commit**

```bash
git add internal/tui/
git commit -m "feat: add TUI layout polish with headers, footers, and colored bars"
```

---

### Task 3: PythonScanner

**Files:**
- Create: `internal/scanner/python.go`
- Create: `internal/scanner/python_test.go`
- Modify: `internal/config/config.go` (add `Python bool` to `ScannersConfig`, add to `Default()`)
- Modify: `internal/cli/root.go` (register scanner in `buildEngine`, add to `selectedCategories`)

**Step 1: Write tests**

```go
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPythonScanner_Name(t *testing.T) {
	s := NewPythonScanner("", nil, 30*24*time.Hour)
	if s.Name() != "Python" {
		t.Errorf("expected name %q, got %q", "Python", s.Name())
	}
}

func TestPythonScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewPythonScanner("", nil, 30*24*time.Hour)
}

func TestPythonScanner_FindsPipCache(t *testing.T) {
	home := t.TempDir()
	cacheDir := filepath.Join(home, "Library", "Caches", "pip")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "data.bin"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewPythonScanner(home, nil, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == cacheDir {
			found = true
			if tgt.Category != "Python" {
				t.Errorf("expected category Python, got %q", tgt.Category)
			}
			if tgt.Risk != Safe {
				t.Errorf("expected risk Safe, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Errorf("expected to find pip cache target")
	}
}

func TestPythonScanner_FindsCondaPkgs(t *testing.T) {
	home := t.TempDir()
	condaDir := filepath.Join(home, "miniconda3", "pkgs")
	if err := os.MkdirAll(condaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(condaDir, "pkg.tar.bz2"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewPythonScanner(home, nil, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == condaDir {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find conda pkgs target")
	}
}

func TestPythonScanner_FindsStaleVenvs(t *testing.T) {
	searchDir := t.TempDir()
	venvDir := filepath.Join(searchDir, "my-project", ".venv")
	if err := os.MkdirAll(filepath.Join(venvDir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Write pyvenv.cfg marker file
	if err := os.WriteFile(filepath.Join(venvDir, "pyvenv.cfg"), []byte("home = /usr/bin"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	os.Chtimes(venvDir, oldTime, oldTime)

	s := NewPythonScanner(t.TempDir(), []string{searchDir}, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == venvDir {
			found = true
			if tgt.Risk != Moderate {
				t.Errorf("expected risk Moderate, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Errorf("expected to find stale venv target at %s", venvDir)
	}
}

func TestPythonScanner_SkipsFreshVenvs(t *testing.T) {
	searchDir := t.TempDir()
	venvDir := filepath.Join(searchDir, "my-project", ".venv")
	if err := os.MkdirAll(venvDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(venvDir, "pyvenv.cfg"), []byte("home = /usr/bin"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewPythonScanner(t.TempDir(), []string{searchDir}, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, tgt := range targets {
		if tgt.Path == venvDir {
			t.Error("fresh venv should not be reported")
		}
	}
}

func TestPythonScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	home := t.TempDir()
	cacheDir := filepath.Join(home, "Library", "Caches", "pip")
	os.MkdirAll(cacheDir, 0o755)

	s := NewPythonScanner(home, nil, 30*24*time.Hour)
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/scanner/ -run TestPython -v`
Expected: FAIL (PythonScanner not defined)

**Step 3: Implement PythonScanner**

```go
package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// PythonScanner detects pip cache, conda packages, and stale virtualenvs.
type PythonScanner struct {
	home        string
	searchPaths []string
	maxAge      time.Duration
}

func NewPythonScanner(home string, searchPaths []string, maxAge time.Duration) *PythonScanner {
	return &PythonScanner{home: home, searchPaths: searchPaths, maxAge: maxAge}
}

func (s *PythonScanner) Name() string        { return "Python" }
func (s *PythonScanner) Description() string { return "pip cache, conda packages, and stale virtualenvs" }
func (s *PythonScanner) Risk() RiskLevel     { return Safe }

func (s *PythonScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	// pip cache
	pipCache := filepath.Join(s.home, "Library", "Caches", "pip")
	if utils.DirExists(pipCache) {
		size, _ := utils.DirSize(pipCache)
		targets = append(targets, Target{
			Path:        pipCache,
			Size:        size,
			Category:    "Python",
			Description: "pip download cache",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// conda pkgs
	for _, condaRoot := range []string{"miniconda3", "anaconda3", "miniforge3"} {
		pkgsDir := filepath.Join(s.home, condaRoot, "pkgs")
		if utils.DirExists(pkgsDir) {
			size, _ := utils.DirSize(pkgsDir)
			targets = append(targets, Target{
				Path:        pkgsDir,
				Size:        size,
				Category:    "Python",
				Description: fmt.Sprintf("%s package cache", condaRoot),
				Risk:        Safe,
				IsDir:       true,
			})
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// stale virtualenvs
	now := time.Now()
	for _, searchPath := range s.searchPaths {
		if !utils.DirExists(searchPath) {
			continue
		}
		filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil || !d.IsDir() {
				return nil
			}
			name := d.Name()
			if name != ".venv" && name != "venv" {
				return nil
			}
			// Verify it's a real virtualenv (has pyvenv.cfg).
			if _, err := os.Stat(filepath.Join(path, "pyvenv.cfg")); err != nil {
				return fs.SkipDir
			}
			info, err := os.Stat(path)
			if err != nil {
				return fs.SkipDir
			}
			age := now.Sub(info.ModTime())
			if age >= s.maxAge {
				size, _ := utils.DirSize(path)
				targets = append(targets, Target{
					Path:        path,
					Size:        size,
					Category:    "Python",
					Description: fmt.Sprintf("stale virtualenv (unused for %d days)", int(age.Hours()/24)),
					Risk:        Moderate,
					ModTime:     info.ModTime(),
					IsDir:       true,
				})
			}
			return fs.SkipDir
		})
	}

	return targets, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/scanner/ -run TestPython -v -race`
Expected: All PASS

**Step 5: Add config toggle and register in engine**

In `internal/config/config.go`, add `Python bool \`yaml:"python"\`` to `ScannersConfig` and set `Python: true` in `Default()`.

In `internal/cli/root.go`, add to `buildEngine()`:
```go
if appConfig.Scanners.Python {
    home := utils.HomeDir()
    paths := expandPaths(appConfig.LargeFiles.Paths)
    minAge := config.ParseDuration(appConfig.LargeFiles.MinAge)
    e.Register(scanner.NewPythonScanner(home, paths, minAge))
}
```

Add `"Python"` case to `selectedCategories`.

**Step 6: Run full test suite**

Run: `go build ./... && go test ./... -race`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/scanner/python.go internal/scanner/python_test.go internal/config/config.go internal/cli/root.go
git commit -m "feat: add Python scanner (pip cache, conda, stale venvs)"
```

---

### Task 4: RustScanner

**Files:**
- Create: `internal/scanner/rust.go`
- Create: `internal/scanner/rust_test.go`
- Modify: `internal/config/config.go` (add `Rust bool`)
- Modify: `internal/cli/root.go` (register + selectedCategories)

**Step 1: Write tests**

```go
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRustScanner_Name(t *testing.T) {
	s := NewRustScanner("", nil, 0)
	if s.Name() != "Rust" {
		t.Errorf("expected name %q, got %q", "Rust", s.Name())
	}
}

func TestRustScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewRustScanner("", nil, 0)
}

func TestRustScanner_FindsCargoRegistryCache(t *testing.T) {
	home := t.TempDir()
	cacheDir := filepath.Join(home, ".cargo", "registry", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "crate.tar"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewRustScanner(home, nil, 0)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == cacheDir && tgt.Category == "Rust" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find cargo registry cache target")
	}
}

func TestRustScanner_FindsTargetDirs(t *testing.T) {
	searchDir := t.TempDir()
	targetDir := filepath.Join(searchDir, "my-project", "target")
	if err := os.MkdirAll(filepath.Join(targetDir, "debug"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Write Cargo.toml next to target/ to confirm it's a Rust project
	if err := os.WriteFile(filepath.Join(searchDir, "my-project", "Cargo.toml"), []byte("[package]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "debug", "binary"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	os.Chtimes(targetDir, oldTime, oldTime)

	s := NewRustScanner(t.TempDir(), []string{searchDir}, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == targetDir {
			found = true
			if tgt.Risk != Moderate {
				t.Errorf("expected risk Moderate, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Error("expected to find stale target/ dir")
	}
}

func TestRustScanner_NoCargoDir(t *testing.T) {
	home := t.TempDir()
	s := NewRustScanner(home, nil, 0)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestRustScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewRustScanner(t.TempDir(), nil, 0)
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/scanner/ -run TestRust -v`
Expected: FAIL

**Step 3: Implement RustScanner**

```go
package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// RustScanner detects cargo registry cache and stale target directories.
type RustScanner struct {
	home        string
	searchPaths []string
	maxAge      time.Duration
}

func NewRustScanner(home string, searchPaths []string, maxAge time.Duration) *RustScanner {
	return &RustScanner{home: home, searchPaths: searchPaths, maxAge: maxAge}
}

func (s *RustScanner) Name() string        { return "Rust" }
func (s *RustScanner) Description() string { return "Cargo registry cache and stale target directories" }
func (s *RustScanner) Risk() RiskLevel     { return Safe }

func (s *RustScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	// Cargo registry cache and source
	for _, sub := range []string{"registry/cache", "registry/src"} {
		dir := filepath.Join(s.home, ".cargo", sub)
		if utils.DirExists(dir) {
			size, _ := utils.DirSize(dir)
			targets = append(targets, Target{
				Path:        dir,
				Size:        size,
				Category:    "Rust",
				Description: fmt.Sprintf("Cargo %s", sub),
				Risk:        Safe,
				IsDir:       true,
			})
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Stale target/ directories
	now := time.Now()
	for _, searchPath := range s.searchPaths {
		if !utils.DirExists(searchPath) {
			continue
		}
		filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil || !d.IsDir() {
				return nil
			}
			if d.Name() != "target" {
				return nil
			}
			// Confirm Rust project: Cargo.toml must exist in parent dir.
			parent := filepath.Dir(path)
			if _, err := os.Stat(filepath.Join(parent, "Cargo.toml")); err != nil {
				return nil // Not a Rust target dir
			}
			info, err := os.Stat(path)
			if err != nil {
				return fs.SkipDir
			}
			if s.maxAge > 0 {
				age := now.Sub(info.ModTime())
				if age < s.maxAge {
					return fs.SkipDir
				}
			}
			size, _ := utils.DirSize(path)
			targets = append(targets, Target{
				Path:        path,
				Size:        size,
				Category:    "Rust",
				Description: fmt.Sprintf("Rust build artifacts (%s)", filepath.Base(parent)),
				Risk:        Moderate,
				ModTime:     info.ModTime(),
				IsDir:       true,
			})
			return fs.SkipDir
		})
	}

	return targets, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/scanner/ -run TestRust -v -race`
Expected: All PASS

**Step 5: Add config toggle and register**

Same pattern as Task 3: add `Rust bool \`yaml:"rust"\`` to `ScannersConfig`, set true in `Default()`, register in `buildEngine()`, add to `selectedCategories`.

**Step 6: Run full test suite**

Run: `go build ./... && go test ./... -race`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/scanner/rust.go internal/scanner/rust_test.go internal/config/config.go internal/cli/root.go
git commit -m "feat: add Rust scanner (cargo cache, stale target dirs)"
```

---

### Task 5: GoScanner

**Files:**
- Create: `internal/scanner/golang.go`
- Create: `internal/scanner/golang_test.go`
- Modify: `internal/config/config.go` (add `Go bool`)
- Modify: `internal/cli/root.go`

**Step 1: Write tests**

```go
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGoScanner_Name(t *testing.T) {
	s := NewGoScanner("")
	if s.Name() != "Go" {
		t.Errorf("expected name %q, got %q", "Go", s.Name())
	}
}

func TestGoScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewGoScanner("")
}

func TestGoScanner_FindsModCache(t *testing.T) {
	home := t.TempDir()
	modCache := filepath.Join(home, "go", "pkg", "mod", "cache")
	if err := os.MkdirAll(modCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modCache, "module.zip"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewGoScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == modCache && tgt.Category == "Go" {
			found = true
			if tgt.Risk != Safe {
				t.Errorf("expected risk Safe, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Error("expected to find Go module cache target")
	}
}

func TestGoScanner_FindsBuildCache(t *testing.T) {
	home := t.TempDir()
	// Simulate GOCACHE at default location
	buildCache := filepath.Join(home, "Library", "Caches", "go-build")
	if err := os.MkdirAll(buildCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildCache, "cache.bin"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewGoScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == buildCache {
			found = true
		}
	}
	if !found {
		t.Error("expected to find Go build cache target")
	}
}

func TestGoScanner_NoGoDir(t *testing.T) {
	home := t.TempDir()
	s := NewGoScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestGoScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewGoScanner(t.TempDir())
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/scanner/ -run TestGoScanner -v`
Expected: FAIL

**Step 3: Implement GoScanner**

```go
package scanner

import (
	"context"
	"path/filepath"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// GoScanner detects Go module cache and build cache.
type GoScanner struct {
	home string
}

func NewGoScanner(home string) *GoScanner {
	return &GoScanner{home: home}
}

func (s *GoScanner) Name() string        { return "Go" }
func (s *GoScanner) Description() string { return "Go module cache and build cache" }
func (s *GoScanner) Risk() RiskLevel     { return Safe }

func (s *GoScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	// Go module cache: ~/go/pkg/mod/cache
	modCache := filepath.Join(s.home, "go", "pkg", "mod", "cache")
	if utils.DirExists(modCache) {
		size, _ := utils.DirSize(modCache)
		targets = append(targets, Target{
			Path:        modCache,
			Size:        size,
			Category:    "Go",
			Description: "Go module cache",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Go build cache: ~/Library/Caches/go-build (macOS default)
	buildCache := filepath.Join(s.home, "Library", "Caches", "go-build")
	if utils.DirExists(buildCache) {
		size, _ := utils.DirSize(buildCache)
		targets = append(targets, Target{
			Path:        buildCache,
			Size:        size,
			Category:    "Go",
			Description: "Go build cache",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	return targets, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/scanner/ -run TestGoScanner -v -race`
Expected: All PASS

**Step 5: Add config toggle and register**

Add `Go bool \`yaml:"go"\`` to `ScannersConfig`, `Go: true` in `Default()`, register in `buildEngine()`:
```go
if appConfig.Scanners.Go {
    home := utils.HomeDir()
    e.Register(scanner.NewGoScanner(home))
}
```

Add `"Go"` to `selectedCategories`.

**Step 6: Run full test suite**

Run: `go build ./... && go test ./... -race`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/scanner/golang.go internal/scanner/golang_test.go internal/config/config.go internal/cli/root.go
git commit -m "feat: add Go scanner (module cache, build cache)"
```

---

### Task 6: JetBrainsScanner

**Files:**
- Create: `internal/scanner/jetbrains.go`
- Create: `internal/scanner/jetbrains_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/cli/root.go`

**Step 1: Write tests**

```go
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestJetBrainsScanner_Name(t *testing.T) {
	s := NewJetBrainsScanner("")
	if s.Name() != "JetBrains" {
		t.Errorf("expected name %q, got %q", "JetBrains", s.Name())
	}
}

func TestJetBrainsScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewJetBrainsScanner("")
}

func TestJetBrainsScanner_FindsCaches(t *testing.T) {
	home := t.TempDir()
	cacheDir := filepath.Join(home, "Library", "Caches", "JetBrains", "IntelliJIdea2024.1")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "index.dat"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewJetBrainsScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == cacheDir && tgt.Category == "JetBrains" {
			found = true
			if tgt.Risk != Safe {
				t.Errorf("expected risk Safe, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Error("expected to find JetBrains cache target")
	}
}

func TestJetBrainsScanner_FindsLogs(t *testing.T) {
	home := t.TempDir()
	logDir := filepath.Join(home, "Library", "Logs", "JetBrains", "PyCharm2024.2")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "idea.log"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewJetBrainsScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == logDir {
			found = true
		}
	}
	if !found {
		t.Error("expected to find JetBrains logs target")
	}
}

func TestJetBrainsScanner_NoJetBrainsDirs(t *testing.T) {
	s := NewJetBrainsScanner(t.TempDir())
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestJetBrainsScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewJetBrainsScanner(t.TempDir())
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/scanner/ -run TestJetBrains -v`
Expected: FAIL

**Step 3: Implement JetBrainsScanner**

```go
package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// JetBrainsScanner detects JetBrains IDE caches and logs.
type JetBrainsScanner struct {
	home string
}

func NewJetBrainsScanner(home string) *JetBrainsScanner {
	return &JetBrainsScanner{home: home}
}

func (s *JetBrainsScanner) Name() string        { return "JetBrains" }
func (s *JetBrainsScanner) Description() string { return "JetBrains IDE caches and logs" }
func (s *JetBrainsScanner) Risk() RiskLevel     { return Safe }

func (s *JetBrainsScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	// Scan JetBrains caches and logs
	dirs := []struct {
		base string
		desc string
	}{
		{filepath.Join(s.home, "Library", "Caches", "JetBrains"), "cache"},
		{filepath.Join(s.home, "Library", "Logs", "JetBrains"), "logs"},
	}

	for _, d := range dirs {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !utils.DirExists(d.base) {
			continue
		}
		entries, err := os.ReadDir(d.base)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			ideDir := filepath.Join(d.base, entry.Name())
			size, _ := utils.DirSize(ideDir)
			targets = append(targets, Target{
				Path:        ideDir,
				Size:        size,
				Category:    "JetBrains",
				Description: fmt.Sprintf("%s %s", entry.Name(), d.desc),
				Risk:        Safe,
				IsDir:       true,
			})
		}
	}

	return targets, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/scanner/ -run TestJetBrains -v -race`
Expected: All PASS

**Step 5: Add config toggle and register**

Add `JetBrains bool \`yaml:"jetbrains"\`` to `ScannersConfig`, `JetBrains: true` in `Default()`, register in `buildEngine()`:
```go
if appConfig.Scanners.JetBrains {
    home := utils.HomeDir()
    e.Register(scanner.NewJetBrainsScanner(home))
}
```

Add `"JetBrains"` to `selectedCategories`.

**Step 6: Run full test suite**

Run: `go build ./... && go test ./... -race`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/scanner/jetbrains.go internal/scanner/jetbrains_test.go internal/config/config.go internal/cli/root.go
git commit -m "feat: add JetBrains scanner (IDE caches and logs)"
```

---

### Task 7: Worker Pool Engine

Add `ScanGroupedWithProgress` to the engine with a semaphore-based worker pool.

**Files:**
- Modify: `internal/engine/engine.go`
- Create: `internal/engine/engine_test.go`

**Step 1: Write tests**

```go
package engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lu-zhengda/macbroom/internal/scanner"
)

// mockScanner is a test scanner that returns predefined targets after a delay.
type mockScanner struct {
	name    string
	targets []scanner.Target
	delay   time.Duration
	err     error
}

func (m *mockScanner) Name() string                                    { return m.name }
func (m *mockScanner) Description() string                             { return m.name }
func (m *mockScanner) Risk() scanner.RiskLevel                         { return scanner.Safe }
func (m *mockScanner) Scan(ctx context.Context) ([]scanner.Target, error) {
	select {
	case <-time.After(m.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return m.targets, m.err
}

func TestScanGroupedWithProgress_Basic(t *testing.T) {
	e := New()
	e.Register(&mockScanner{name: "A", targets: []scanner.Target{{Path: "/a"}}})
	e.Register(&mockScanner{name: "B", targets: []scanner.Target{{Path: "/b"}}})

	var mu sync.Mutex
	var events []ScanProgress
	results := e.ScanGroupedWithProgress(context.Background(), 2, func(p ScanProgress) {
		mu.Lock()
		events = append(events, p)
		mu.Unlock()
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	mu.Lock()
	defer mu.Unlock()
	// Each scanner should have a Started and Done event
	startCount, doneCount := 0, 0
	for _, ev := range events {
		if ev.Status == ScanStarted {
			startCount++
		}
		if ev.Status == ScanDone {
			doneCount++
		}
	}
	if startCount != 2 {
		t.Errorf("expected 2 Started events, got %d", startCount)
	}
	if doneCount != 2 {
		t.Errorf("expected 2 Done events, got %d", doneCount)
	}
}

func TestScanGroupedWithProgress_ConcurrencyLimit(t *testing.T) {
	e := New()
	// 4 scanners, concurrency 2: at most 2 should run simultaneously
	var mu sync.Mutex
	running := 0
	maxRunning := 0

	for i := 0; i < 4; i++ {
		e.Register(&mockScanner{name: "S", delay: 50 * time.Millisecond})
	}

	e.ScanGroupedWithProgress(context.Background(), 2, func(p ScanProgress) {
		mu.Lock()
		defer mu.Unlock()
		if p.Status == ScanStarted {
			running++
			if running > maxRunning {
				maxRunning = running
			}
		}
		if p.Status == ScanDone {
			running--
		}
	})

	if maxRunning > 2 {
		t.Errorf("expected max concurrency 2, got %d", maxRunning)
	}
}

func TestScanGroupedWithProgress_ContextCancelled(t *testing.T) {
	e := New()
	e.Register(&mockScanner{name: "Slow", delay: 5 * time.Second})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := e.ScanGroupedWithProgress(ctx, 1, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Error == nil {
		t.Error("expected error for cancelled context")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -v`
Expected: FAIL (ScanGroupedWithProgress not defined)

**Step 3: Implement ScanGroupedWithProgress**

Add to `internal/engine/engine.go`:

```go
// ScanStatus represents the state of a scanner in the progress callback.
type ScanStatus int

const (
	ScanWaiting ScanStatus = iota
	ScanStarted
	ScanDone
)

// ScanProgress is sent to the progress callback for each scanner event.
type ScanProgress struct {
	Name    string
	Status  ScanStatus
	Targets []scanner.Target
	Error   error
}

// ScanGroupedWithProgress runs scanners with a concurrency limit and calls
// onProgress for each scanner event (started, done).
func (e *Engine) ScanGroupedWithProgress(ctx context.Context, concurrency int, onProgress func(ScanProgress)) []ScanResult {
	if concurrency < 1 {
		concurrency = 1
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []ScanResult
		sem     = make(chan struct{}, concurrency)
	)

	for _, s := range e.scanners {
		wg.Add(1)
		go func(s scanner.Scanner) {
			defer wg.Done()

			sem <- struct{}{} // acquire
			if onProgress != nil {
				onProgress(ScanProgress{Name: s.Name(), Status: ScanStarted})
			}

			targets, err := s.Scan(ctx)

			<-sem // release

			if onProgress != nil {
				onProgress(ScanProgress{
					Name:    s.Name(),
					Status:  ScanDone,
					Targets: targets,
					Error:   err,
				})
			}

			mu.Lock()
			results = append(results, ScanResult{
				Category: s.Name(),
				Targets:  targets,
				Error:    err,
			})
			mu.Unlock()
		}(s)
	}

	wg.Wait()
	return results
}
```

**Step 4: Run tests**

Run: `go test ./internal/engine/ -v -race`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/engine_test.go
git commit -m "feat: add worker pool engine with per-scanner progress"
```

---

### Task 8: Scanning Progress View

New `viewScanning` state that shows per-scanner live progress during the Clean scan.

**Files:**
- Modify: `internal/tui/app.go`

**Step 1: Add new view state and messages**

Add `viewScanning` to the viewState enum (after `viewMenu`).

Add new message types:
```go
type scanProgressMsg struct {
	progress engine.ScanProgress
}
```

Add new Model fields:
```go
// Scan progress state
scanStatuses []scannerStatus // per-scanner status tracking
```

Add helper type:
```go
type scannerStatus struct {
	name    string
	status  engine.ScanStatus
	count   int
	size    int64
}
```

**Step 2: Update doScan to use ScanGroupedWithProgress**

Replace `doScan`:
```go
func (m Model) doScan() tea.Cmd {
	return func() tea.Msg {
		var mu sync.Mutex
		var lastProgress scanProgressMsg

		results := m.engine.ScanGroupedWithProgress(context.Background(), 4, func(p engine.ScanProgress) {
			mu.Lock()
			lastProgress = scanProgressMsg{progress: p}
			mu.Unlock()
		})
		return scanDoneMsg{results: results}
	}
}
```

Actually, to stream progress to the TUI we need a channel-based approach similar to SpaceLens/Dupes. Wire it so each `ScanProgress` event is sent to the TUI as a `scanProgressMsg` via a channel, then the TUI renders the multi-line per-scanner status.

**Step 3: Implement viewScanning**

```go
func (m Model) viewScanning() string {
	s := renderHeader("Clean")

	for _, ss := range m.scanStatuses {
		var icon, detail string
		switch ss.status {
		case engine.ScanWaiting:
			icon = dimStyle.Render("○")
			detail = dimStyle.Render("waiting...")
		case engine.ScanStarted:
			icon = m.spinner.View()
			detail = "scanning..."
		case engine.ScanDone:
			icon = successStyle.Render("✓")
			detail = fmt.Sprintf("%d items   %s", ss.count, utils.FormatSize(ss.size))
		}
		line := fmt.Sprintf("  %s %-20s %s", icon, ss.name, detail)
		s += line + "\n"
	}

	s += renderFooter("esc cancel | q quit")
	return s
}
```

**Step 4: Update menu handler to initialize scanStatuses**

When the user selects "Clean", populate `scanStatuses` from the engine's registered scanners, all set to `ScanWaiting`, then transition to `viewScanning`.

**Step 5: Handle scanProgressMsg in Update**

Update the scanner's status in `scanStatuses` when progress events arrive.

**Step 6: Run tests and build**

Run: `go build ./... && go test ./... -race`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/tui/
git commit -m "feat: add per-scanner progress view during scanning"
```

---

### Task 9: Treemap Layout Algorithm

Implement the squarified treemap layout algorithm.

**Files:**
- Create: `internal/tui/treemap.go`
- Create: `internal/tui/treemap_test.go`

**Step 1: Write tests**

```go
package tui

import "testing"

func TestLayoutTreemap_SingleItem(t *testing.T) {
	items := []treemapItem{{name: "foo", size: 100}}
	rects := layoutTreemap(items, rect{x: 0, y: 0, w: 80, h: 24})

	if len(rects) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(rects))
	}
	if rects[0].w != 80 || rects[0].h != 24 {
		t.Errorf("expected full rect (80x24), got (%dx%d)", rects[0].w, rects[0].h)
	}
}

func TestLayoutTreemap_TwoItems(t *testing.T) {
	items := []treemapItem{
		{name: "a", size: 75},
		{name: "b", size: 25},
	}
	rects := layoutTreemap(items, rect{x: 0, y: 0, w: 80, h: 24})

	if len(rects) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(rects))
	}
	// Total area should roughly match
	totalArea := 0
	for _, r := range rects {
		totalArea += r.w * r.h
	}
	expected := 80 * 24
	if totalArea != expected {
		t.Errorf("expected total area %d, got %d", expected, totalArea)
	}
}

func TestLayoutTreemap_Empty(t *testing.T) {
	rects := layoutTreemap(nil, rect{x: 0, y: 0, w: 80, h: 24})
	if len(rects) != 0 {
		t.Errorf("expected 0 rects, got %d", len(rects))
	}
}

func TestLayoutTreemap_AspectRatio(t *testing.T) {
	items := []treemapItem{
		{name: "a", size: 50},
		{name: "b", size: 30},
		{name: "c", size: 20},
	}
	rects := layoutTreemap(items, rect{x: 0, y: 0, w: 80, h: 24})

	// All rects should have reasonable aspect ratios (not extremely thin)
	for i, r := range rects {
		if r.w == 0 || r.h == 0 {
			t.Errorf("rect %d has zero dimension: %dx%d", i, r.w, r.h)
			continue
		}
		ratio := float64(r.w) / float64(r.h)
		if ratio > 20 || ratio < 0.05 {
			t.Errorf("rect %d has extreme aspect ratio: %dx%d (ratio=%.2f)", i, r.w, r.h, ratio)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run TestLayoutTreemap -v`
Expected: FAIL

**Step 3: Implement treemap layout**

```go
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/lu-zhengda/macbroom/internal/utils"
)

type rect struct {
	x, y, w, h int
}

type treemapItem struct {
	name     string
	size     int64
	isDir    bool
	path     string
	colorIdx int
}

type treemapRect struct {
	rect
	item treemapItem
}

// layoutTreemap computes a squarified treemap layout for the given items
// within the bounding rectangle.
func layoutTreemap(items []treemapItem, bounds rect) []treemapRect {
	if len(items) == 0 || bounds.w <= 0 || bounds.h <= 0 {
		return nil
	}

	// Sort by size descending.
	sorted := make([]treemapItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].size > sorted[j].size
	})

	var totalSize int64
	for _, item := range sorted {
		totalSize += item.size
	}
	if totalSize == 0 {
		return nil
	}

	return squarify(sorted, totalSize, bounds)
}

// squarify implements the squarified treemap algorithm.
func squarify(items []treemapItem, totalSize int64, bounds rect) []treemapRect {
	if len(items) == 0 || bounds.w <= 0 || bounds.h <= 0 {
		return nil
	}
	if len(items) == 1 {
		return []treemapRect{{rect: bounds, item: items[0]}}
	}

	area := float64(bounds.w * bounds.h)

	// Determine layout direction: split along the shorter side.
	horizontal := bounds.w >= bounds.h
	sideLen := bounds.h
	if horizontal {
		sideLen = bounds.w
	}

	// Try adding items to the current row until aspect ratio worsens.
	var row []treemapItem
	var rowSize int64
	bestWorst := float64(1e18)

	for i, item := range items {
		row = append(row, item)
		rowSize += item.size

		// Calculate width of the row (perpendicular to sideLen).
		rowFrac := float64(rowSize) / float64(totalSize)
		rowWidth := int(rowFrac * float64(sideLen))
		if rowWidth < 1 {
			rowWidth = 1
		}

		// Worst aspect ratio in this row.
		worst := worstAspectRatio(row, rowSize, totalSize, area, rowWidth, horizontal, bounds)

		if worst > bestWorst && i > 0 {
			// Remove last item; it made things worse.
			row = row[:len(row)-1]
			rowSize -= item.size

			// Layout the current row.
			rects := layoutRow(row, rowSize, totalSize, bounds, horizontal)

			// Compute remaining bounds.
			rowFrac = float64(rowSize) / float64(totalSize)
			var remaining rect
			if horizontal {
				consumed := int(rowFrac * float64(bounds.w))
				if consumed < 1 {
					consumed = 1
				}
				remaining = rect{x: bounds.x + consumed, y: bounds.y, w: bounds.w - consumed, h: bounds.h}
			} else {
				consumed := int(rowFrac * float64(bounds.h))
				if consumed < 1 {
					consumed = 1
				}
				remaining = rect{x: bounds.x, y: bounds.y + consumed, w: bounds.w, h: bounds.h - consumed}
			}

			rest := squarify(items[i:], totalSize-rowSize, remaining)
			return append(rects, rest...)
		}
		bestWorst = worst
	}

	// All items fit in one row.
	return layoutRow(row, rowSize, totalSize, bounds, horizontal)
}

func worstAspectRatio(row []treemapItem, rowSize, totalSize int64, area float64, rowWidth int, horizontal bool, bounds rect) float64 {
	worst := 0.0
	rowFrac := float64(rowSize) / float64(totalSize)
	for _, item := range row {
		itemFrac := float64(item.size) / float64(totalSize)
		var w, h int
		if horizontal {
			w = int(rowFrac * float64(bounds.w))
			h = int((itemFrac / rowFrac) * float64(bounds.h))
		} else {
			h = int(rowFrac * float64(bounds.h))
			w = int((itemFrac / rowFrac) * float64(bounds.w))
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		ratio := float64(w) / float64(h)
		if ratio < 1 {
			ratio = 1 / ratio
		}
		if ratio > worst {
			worst = ratio
		}
	}
	return worst
}

func layoutRow(row []treemapItem, rowSize, totalSize int64, bounds rect, horizontal bool) []treemapRect {
	var rects []treemapRect
	rowFrac := float64(rowSize) / float64(totalSize)

	if horizontal {
		w := int(rowFrac * float64(bounds.w))
		if w < 1 {
			w = 1
		}
		y := bounds.y
		for _, item := range row {
			itemFrac := float64(item.size) / float64(rowSize)
			h := int(itemFrac * float64(bounds.h))
			if h < 1 {
				h = 1
			}
			rects = append(rects, treemapRect{
				rect: rect{x: bounds.x, y: y, w: w, h: h},
				item: item,
			})
			y += h
		}
	} else {
		h := int(rowFrac * float64(bounds.h))
		if h < 1 {
			h = 1
		}
		x := bounds.x
		for _, item := range row {
			itemFrac := float64(item.size) / float64(rowSize)
			w := int(itemFrac * float64(bounds.w))
			if w < 1 {
				w = 1
			}
			rects = append(rects, treemapRect{
				rect: rect{x: x, y: y, w: w, h: h},
				item: item,
			})
			x += w
		}
	}

	return rects
}

// renderTreemap renders the treemap layout to a string for the terminal.
func renderTreemap(nodes []scanner.SpaceLensNode, width, height int, selectedIdx int) string {
	if len(nodes) == 0 || width < 4 || height < 2 {
		return "No data to display.\n"
	}

	// Convert nodes to treemap items.
	items := make([]treemapItem, len(nodes))
	for i, n := range nodes {
		items[i] = treemapItem{
			name:     n.Name,
			size:     n.Size,
			isDir:    n.IsDir,
			path:     n.Path,
			colorIdx: i % len(treemapColors),
		}
	}

	// Layout.
	rects := layoutTreemap(items, rect{x: 0, y: 0, w: width, h: height})

	// Render to a 2D grid.
	grid := make([][]rune, height)
	colors := make([][]int, height)
	selected := make([][]bool, height)
	for y := 0; y < height; y++ {
		grid[y] = make([]rune, width)
		colors[y] = make([]int, width)
		selected[y] = make([]bool, width)
		for x := 0; x < width; x++ {
			grid[y][x] = ' '
			colors[y][x] = -1
		}
	}

	for ri, r := range rects {
		colorIdx := r.item.colorIdx
		isSel := ri == selectedIdx

		// Fill the block.
		for y := r.y; y < r.y+r.h && y < height; y++ {
			for x := r.x; x < r.x+r.w && x < width; x++ {
				grid[y][x] = '░'
				colors[y][x] = colorIdx
				selected[y][x] = isSel
			}
		}

		// Draw border.
		for x := r.x; x < r.x+r.w && x < width; x++ {
			if r.y < height {
				grid[r.y][x] = '─'
			}
			if r.y+r.h-1 < height {
				grid[r.y+r.h-1][x] = '─'
			}
		}
		for y := r.y; y < r.y+r.h && y < height; y++ {
			if r.x < width {
				grid[y][r.x] = '│'
			}
			if r.x+r.w-1 < width {
				grid[y][r.x+r.w-1] = '│'
			}
		}

		// Write label inside block (if fits).
		label := r.item.name
		sizeTxt := utils.FormatSize(r.item.size)
		if r.w > 4 && r.h > 2 {
			// Name on line y+1
			maxLen := r.w - 3
			if len(label) > maxLen {
				label = label[:maxLen-1] + "…"
			}
			writeText(grid, r.x+1, r.y+1, label, width, height)
			// Size on line y+2
			if r.h > 3 {
				writeText(grid, r.x+1, r.y+2, sizeTxt, width, height)
			}
		}
	}

	// Render grid to string with colors.
	var sb strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ch := string(grid[y][x])
			ci := colors[y][x]
			if selected[y][x] {
				style := lipgloss.NewStyle().Bold(true).Reverse(true)
				if ci >= 0 && ci < len(treemapColors) {
					style = style.Foreground(treemapColors[ci])
				}
				sb.WriteString(style.Render(ch))
			} else if ci >= 0 && ci < len(treemapColors) {
				style := lipgloss.NewStyle().Foreground(treemapColors[ci])
				sb.WriteString(style.Render(ch))
			} else {
				sb.WriteString(ch)
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func writeText(grid [][]rune, x, y int, text string, maxW, maxH int) {
	if y >= maxH {
		return
	}
	for i, ch := range text {
		col := x + i
		if col >= maxW {
			break
		}
		grid[y][col] = ch
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/tui/ -run TestLayoutTreemap -v -race`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/tui/treemap.go internal/tui/treemap_test.go
git commit -m "feat: implement squarified treemap layout algorithm"
```

---

### Task 10: SpaceLens Treemap View

Wire the treemap renderer into the SpaceLens TUI view.

**Files:**
- Modify: `internal/tui/app.go` (update `viewSpaceLens`, `updateSpaceLens`)

**Step 1: Update viewSpaceLens to use treemap**

Replace the list-based `viewSpaceLens` with:

```go
func (m Model) viewSpaceLens() string {
	s := renderHeader("Space Lens")

	if m.slLoading {
		s += dimStyle.Render(m.slPath) + "\n\n"
		s += m.spinner.View() + " Analyzing...\n"
		if m.slScanning != "" {
			name := m.slScanning
			if len(name) > 40 {
				name = name[:37] + "..."
			}
			s += dimStyle.Render("  "+name) + "\n"
		}
		s += renderFooter("esc cancel")
		return s
	}

	var totalSize int64
	for _, node := range m.slNodes {
		totalSize += node.Size
	}
	s += dimStyle.Render(fmt.Sprintf("%s (%s)", m.slPath, utils.FormatSize(totalSize))) + "\n\n"

	if len(m.slNodes) == 0 {
		s += "Empty directory.\n"
		return s + renderFooter("esc back | q quit")
	}

	// Reserve lines for header (4) and footer (3).
	tmapHeight := m.height - 7
	tmapWidth := m.width - 2
	if tmapHeight < 4 {
		tmapHeight = 4
	}
	if tmapWidth < 20 {
		tmapWidth = 20
	}

	s += renderTreemap(m.slNodes, tmapWidth, tmapHeight, m.slCursor)

	// Show selected item info below.
	if m.slCursor < len(m.slNodes) {
		node := m.slNodes[m.slCursor]
		info := fmt.Sprintf("  %s  %s", node.Name, utils.FormatSize(node.Size))
		if node.IsDir {
			info += "  [dir]"
		}
		s += selectedStyle.Render(info) + "\n"
	}

	s += renderFooter("arrows navigate | enter drill in | d delete | h go up | esc back | q quit")
	return s
}
```

**Step 2: Update navigation in updateSpaceLens**

Replace j/k linear navigation with arrow-key spatial navigation. The treemap rects define positions — move cursor to the spatially adjacent block:

```go
func (m Model) updateSpaceLens(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.slLoading {
		if msg.String() == "esc" || msg.String() == "backspace" {
			if m.slCancel != nil {
				m.slCancel()
			}
			m.slLoading = false
			m.slScanning = ""
			m.slCancel = nil
			m.slProgressCh = nil
			m.currentView = viewMenu
			m.cursor = 1
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.slCursor > 0 {
			m.slCursor--
		}
	case "down", "j":
		if m.slCursor < len(m.slNodes)-1 {
			m.slCursor++
		}
	case "left", "h":
		// Go up one directory.
		if idx := lastSlash(m.slPath); idx > 0 {
			m.slPath = m.slPath[:idx]
			m.slLoading = true
			m.slCursor = 0
			cancel, ch, cmd := startSpaceLens(m.slPath)
			m.slCancel = cancel
			m.slProgressCh = ch
			return m, tea.Batch(cmd, m.spinner.Tick)
		}
	case "enter", "right", "l":
		if m.slCursor < len(m.slNodes) && m.slNodes[m.slCursor].IsDir {
			m.slPath = m.slNodes[m.slCursor].Path
			m.slLoading = true
			m.slCursor = 0
			cancel, ch, cmd := startSpaceLens(m.slPath)
			m.slCancel = cancel
			m.slProgressCh = ch
			return m, tea.Batch(cmd, m.spinner.Tick)
		}
	case "d":
		if m.slCursor < len(m.slNodes) {
			node := m.slNodes[m.slCursor]
			m.slDeleteTarget = &node
			m.currentView = viewSpaceLensConfirm
		}
	case "esc", "backspace":
		m.currentView = viewMenu
		m.cursor = 1
	}
	return m, nil
}
```

**Step 3: Run tests and build**

Run: `go build ./... && go test ./... -race`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: replace SpaceLens list view with treemap visualization"
```

---

### Task 11: Progress Bars for Dupes and Cleaning

Replace spinners with real progress bars where we have progress data.

**Files:**
- Modify: `internal/tui/app.go`

**Step 1: Add progress tracking to dupes scan**

Add `dupTotal` and `dupDone` fields to the Model to track file count progress. Update `startDupesScan` to count total files first (quick stat pass), then report progress as a fraction.

**Step 2: Update viewDupes loading state**

When `dupLoading` is true, show a progress bar instead of just a spinner:

```go
if m.dupLoading {
    s += "\n"
    if m.dupTotal > 0 {
        ratio := float64(m.dupDone) / float64(m.dupTotal)
        s += fmt.Sprintf("  %s %d/%d files\n", renderProgressBar(ratio, 30), m.dupDone, m.dupTotal)
    } else {
        s += m.spinner.View() + " Counting files...\n"
    }
    // ...
}
```

**Step 3: Add progress bar to cleaning operations**

Update `doClean` and `doDupesClean` to send incremental progress messages:

```go
type cleanProgressMsg struct {
    done  int
    total int
}
```

Show a progress bar in the confirm view while cleaning is in progress.

**Step 4: Run tests and build**

Run: `go build ./... && go test ./... -race`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: add progress bars for duplicate scanning and cleaning"
```

---

### Task 12: Counter Animations

Add animated counters that tick up when results appear.

**Files:**
- Modify: `internal/tui/app.go`

**Step 1: Add animation state to Model**

```go
// Animation state
animStart     time.Time
animDuration  time.Duration
animTargetSize int64
animating     bool
```

**Step 2: Add animation tick message**

```go
type animTickMsg struct{}

func animTick() tea.Cmd {
    return tea.Tick(30*time.Millisecond, func(t time.Time) tea.Msg {
        return animTickMsg{}
    })
}
```

**Step 3: Trigger animation on scan/dupes done**

When results arrive, set `animStart = time.Now()`, `animDuration = 500ms`, `animating = true`, and return `animTick()`.

**Step 4: Update views to show animated values**

```go
func (m Model) animatedSize(target int64) int64 {
    if !m.animating {
        return target
    }
    elapsed := time.Since(m.animStart)
    if elapsed >= m.animDuration {
        return target
    }
    ratio := float64(elapsed) / float64(m.animDuration)
    return int64(ratio * float64(target))
}
```

Use `animatedSize()` in `viewDashboard` total and `viewResult` freed size.

**Step 5: Run tests and build**

Run: `go build ./... && go test ./... -race`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: add counter animations for size totals"
```

---

### Task 13: Update Standalone SpaceLens TUI

Update the standalone SpaceLens model (`spacelens.go`) to also use the treemap.

**Files:**
- Modify: `internal/tui/spacelens.go`

**Step 1: Update SpaceLensModel.View() to use renderTreemap**

Apply the same treemap rendering as the integrated view.

**Step 2: Run tests and build**

Run: `go build ./... && go test ./... -race`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/tui/spacelens.go
git commit -m "feat: update standalone SpaceLens TUI with treemap"
```

---

### Task 14: Update Config Tests

Make sure the new scanner config toggles are tested.

**Files:**
- Modify: `internal/config/config_test.go`

**Step 1: Add tests for new scanner toggles**

```go
func TestDefaultConfig_NewScanners(t *testing.T) {
    cfg := Default()
    if !cfg.Scanners.Python {
        t.Error("expected Python scanner enabled by default")
    }
    if !cfg.Scanners.Rust {
        t.Error("expected Rust scanner enabled by default")
    }
    if !cfg.Scanners.Go {
        t.Error("expected Go scanner enabled by default")
    }
    if !cfg.Scanners.JetBrains {
        t.Error("expected JetBrains scanner enabled by default")
    }
}
```

**Step 2: Run tests**

Run: `go test ./internal/config/ -v -race`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/config/config_test.go
git commit -m "test: add config tests for new scanner toggles"
```

---

### Task 15: Final Integration and README

Update README with v0.3 features and run the full test suite.

**Files:**
- Modify: `README.md`

**Step 1: Run full test suite with coverage**

Run: `go test ./... -race -cover`
Expected: All PASS, coverage reported

**Step 2: Update README**

Add v0.3 features:
- New scanners: Python, Rust, Go, JetBrains
- Worker pool with per-scanner progress
- SpaceLens treemap visualization
- TUI polish (color theme, progress bars, animations, layout)

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: update README for v0.3 features"
```
