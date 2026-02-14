# macbroom v0.2 Feature Set Design

## Goal

Evolve macbroom from a basic cleanup tool into a comprehensive macOS maintenance suite with config support, more scanners, duplicate detection, scheduled cleaning, and a polished TUI.

## Features

### 1. Config System

**Location:** `~/.config/macbroom/config.yaml`

Auto-created on first run with defaults. Controls:

- **Thresholds:** large file min size (default 100MB), min age (default 90 days)
- **Scan paths:** directories to search for large/old files
- **Exclusions:** glob patterns to never flag
- **Scanner toggles:** enable/disable individual scanners
- **Space Lens defaults:** default path, depth
- **Schedule settings:** interval, time, notifications

New package: `internal/config` using `gopkg.in/yaml.v3`. All scanners read from config instead of hardcoded values.

### 2. New Scanners

Four new scanners following the existing `Scanner` interface:

**DockerScanner** (`internal/scanner/docker.go`)
- Runs `docker system df --format json`
- Targets: dangling images, stopped containers, build cache, unused volumes
- Risk: Moderate
- Skips if Docker not installed

**NodeScanner** (`internal/scanner/node.go`)
- Scans `~/.npm/_cacache` for npm cache
- Finds stale `node_modules/` (90+ days, configurable) under common project directories
- Risk: Safe (cache), Moderate (node_modules)

**HomebrewScanner** (`internal/scanner/homebrew.go`)
- Uses `brew --cache` to locate cache directory
- Flags old `.tar.gz` and `.bottle.tar.gz` downloads, outdated cask `.dmg` files
- Risk: Safe

**SimulatorScanner** (`internal/scanner/simulator.go`)
- Scans `~/Library/Developer/CoreSimulator/Devices/` and `Caches/`
- Detects unavailable runtimes via `xcrun simctl list devices unavailable`
- Risk: Moderate
- Skips if Xcode not installed

All registered in `buildEngine()` and toggleable via config.

### 3. Space Lens Improvements

**Deletion:** Press `d` on any file/folder, confirm with `y`, moves to Trash, auto re-scans.

**Scrolling:** Use `scrollOffset` pattern (same as category view) instead of hard cap at 30 items.

**Percentage display:** Each item shows its percentage share of parent: `45%` next to the bar.

**Header total:** Show total directory size: `/Users (245 GB)`

### 4. Dry-run Mode

`--dry-run` flag on `clean` and `uninstall` commands. Scans normally, shows what would be deleted with sizes, exits without deleting. Output prefixed with `[DRY RUN]`.

### 5. Scan History & Stats

**History log:** `~/.local/share/macbroom/history.json` — append one JSON object per clean operation with timestamp, category, item count, bytes freed, method.

**New command:** `macbroom stats` — shows total freed all-time, breakdown by category, last 5 cleanups.

**Integration:** Shared `history.Record()` called from both CLI and TUI after successful deletion.

### 6. Duplicate File Finder

**New command:** `macbroom dupes`

**Algorithm (three-pass):**
1. Group files by size, skip unique sizes
2. Partial hash (first 4KB) to filter
3. Full SHA256 only if partial matches

**Scope:** `~/Downloads`, `~/Desktop`, `~/Documents` (configurable). Min file size: 1KB.

**TUI:** "Duplicates" as a main menu option. Shows groups, user selects which copies to delete (enforces keeping at least one per group).

### 7. Scheduled Cleaning

**Commands:**
- `macbroom schedule enable` — installs LaunchAgent
- `macbroom schedule disable` — removes LaunchAgent
- `macbroom schedule status` — shows state, last/next run

**LaunchAgent:** `~/Library/LaunchAgents/com.macbroom.cleanup.plist`
- Runs `macbroom clean --yes --quiet` at configured time
- `--quiet` flag: suppresses output, only logs to history
- Sends macOS notification via `osascript` on completion

**Config:**
```yaml
schedule:
  enabled: false
  interval: daily  # daily, weekly
  time: "10:00"
  notify: true
```

### 8. Shell Completions

Hidden flag `--generate-completion zsh/bash/fish` instead of a subcommand. Uses Cobra's built-in completion generation with custom completions for subcommands and flags.

### 9. Man Page

Generate `macbroom.1` at build time using Cobra's `doc` package. Installed to `$(PREFIX)/share/man/man1/` via `make install`. GoReleaser includes it in archives.

### 10. Uninstaller TUI + Orphan Scan

**TUI integration:** Main menu gains "Uninstall" option. Text input for app name, shows app + related files, select/deselect, confirm deletion.

**Extended leftover detection:** Add `LaunchAgents/`, `Application Scripts/`, `Group Containers/`, `Cookies/` to search locations.

**Orphan scan:** `macbroom scan --orphans` — cross-references bundle IDs in `~/Library/Preferences/*.plist` against installed apps in `/Applications/`. Shows orphaned prefs/caches with no matching app.

## Architecture

### New Packages
- `internal/config` — YAML config loading, defaults, validation
- `internal/history` — JSON log append, stats aggregation
- `internal/schedule` — LaunchAgent plist generation, launchctl management
- `internal/dupes` — Duplicate file detection (size → hash algorithm)

### New Dependencies
- `gopkg.in/yaml.v3` — YAML config parsing

### TUI Main Menu (updated)
1. Clean
2. Space Lens
3. Uninstall
4. Duplicates
5. Maintenance

## Implementation Order

1. Config system (foundation — everything reads from it)
2. New scanners (independent, can be parallelized)
3. Dry-run mode (small, touches clean/uninstall)
4. History & stats (needs clean to record)
5. Space Lens improvements + deletion
6. Duplicate finder
7. Uninstaller TUI + orphan scan
8. Scheduled cleaning
9. Shell completions + man page
