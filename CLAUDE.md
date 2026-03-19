# Burrow - CLAUDE.md

macOS system maintenance CLI (CleanMyMac + AppCleaner + DaisyDisk + iStat Menus in one binary).
Hybrid architecture: Bash shell scripts for orchestration, Go for performance-critical TUI components.
Fork of [tw93/Mole](https://github.com/tw93/Mole), rebranded as [jimmy-jain/burrow](https://github.com/jimmy-jain/burrow).

## Quick Reference

```bash
# Development setup
brew install shfmt shellcheck bats-core golangci-lint
go install golang.org/x/tools/cmd/goimports@latest
git config core.hooksPath .githooks

# Quality checks (format + lint, run before committing)
./scripts/check.sh

# Tests (BATS for shell, go test for Go)
./scripts/test.sh

# Build Go binaries (analyze + status + watch + dupes)
make build              # local architecture
make release-arm64      # release for arm64
make release-amd64      # release for amd64

# Run Go components directly
go run ./cmd/analyze
go run ./cmd/status
go run ./cmd/watch
go run ./cmd/dupes
```

## Architecture

### Dual-Language Design

| Layer | Language | Purpose |
|-------|----------|---------|
| CLI entry + orchestration | Bash | `burrow` script routes subcommands, interactive menu |
| Feature modules | Bash | `lib/clean/`, `lib/optimize/`, `lib/uninstall/`, `lib/manage/` |
| Core libraries | Bash | `lib/core/` (file ops, UI, logging, sudo, path validation) |
| Command wrappers | Bash | `bin/*.sh` (thin wrappers that source libs and run features) |
| Disk analyzer TUI | Go | `cmd/analyze/` (bubbletea + lipgloss) |
| System monitor TUI | Go | `cmd/status/` (bubbletea + lipgloss, gopsutil for metrics) |
| Duplicate finder | Go | `cmd/dupes/` (xxhash content hashing, conserve/restore) |
| Threshold alerts | Go | `cmd/watch/` (background polling, macOS notifications) |

### Directory Layout

```
burrow                  # Main CLI entry point — routes subcommands
bw                      # Lightweight alias -> burrow
bin/                    # Command wrappers (*.sh) + compiled Go binaries
lib/
  core/                 # Shared libraries: common.sh, base.sh, log.sh, file_ops.sh, ui.sh, sudo.sh, etc.
  clean/                # Cleanup modules: user.sh, dev.sh, project.sh, system.sh, apps.sh, caches.sh
  optimize/             # Optimization: tasks.sh, maintenance.sh
  uninstall/            # App removal: batch.sh, brew.sh
  manage/               # Config management: whitelist.sh, schedule.sh, hook.sh
  size/                 # Dev cache size audit: main.sh
  doctor/               # Developer environment health checks: checks.sh
  log/                  # Operations log viewer: viewer.sh
  report/               # Machine health JSON report: main.sh
cmd/
  analyze/              # Go disk analyzer (bubbletea MVC)
  status/               # Go system monitor (bubbletea, gopsutil metrics)
  dupes/                # Go duplicate file finder (xxhash, conserve/restore, Finder trash)
  watch/                # Go threshold alerts (rules, notifications, predictive disk)
tests/                  # BATS test suites
scripts/                # Dev/CI scripts (check.sh, test.sh)
install.sh              # Standalone installer
```

### Commands

| Command | Type | Description |
|---------|------|-------------|
| `bw clean` | Shell | Deep cleanup with safety validation, dry-run, whitelist |
| `bw uninstall` | Shell | Remove apps + launch agents, preferences, remnants |
| `bw optimize` | Shell | Refresh caches & services |
| `bw analyze` | Go TUI | Visual disk explorer with Finder trash integration |
| `bw status` | Go TUI | Live system health dashboard (CPU, GPU, memory, disk, network, battery) |
| `bw purge` | Shell | Clean project build artifacts (node_modules, target, etc.) |
| `bw installer` | Shell | Find and remove installer files |
| `bw size` | Shell | Developer cache size audit (table + `--json`) |
| `bw doctor` | Shell | Developer environment health checks (`--json`) |
| `bw log` | Shell | Operations log viewer (`--since`, `--grep`, `--tail`) |
| `bw report` | Shell | Machine health snapshot as JSON (`--out`) |
| `bw dupes` | Go | Find and manage duplicate files (`--conserve`, `--restore`, `--delete`, `--json`) |
| `bw watch` | Go | Background threshold alerts with macOS notifications |
| `bw schedule` | Shell | LaunchAgent maintenance (install/remove/status) |
| `bw hook` | Shell | Shell cd-hook integration (bash/zsh/fish) |
| `bw touchid` | Shell | Configure Touch ID for sudo |
| `bw completion` | Shell | Shell tab completion setup |

### Command Routing

`mole` dispatches via case statement: `clean` -> `bin/clean.sh`, `status` -> `bin/status.sh`, etc.
Without arguments, shows interactive menu with arrow/number/vim key navigation.

### Shell Library Loading

`lib/core/common.sh` is the orchestrator, sourcing all core modules in order:
`base.sh` -> `log.sh` -> `timeout.sh` -> `file_ops.sh` -> `help.sh` -> `ui.sh` -> `app_protection.sh` -> `sudo.sh`

Each module guards against double-sourcing: `if [[ -n "${MOLE_COMMON_LOADED:-}" ]]; then return 0; fi`

### Go Components

- `cmd/analyze/`: Charmbracelet bubbletea MVC. Concurrent filesystem scanning, heap-based top-N tracking, singleflight dedup, Finder trash integration
- `cmd/status/`: Charmbracelet bubbletea MVC. Real-time metrics every 1s, RingBuffer history, composite health score (0-100), SMART health, Time Machine backup, network connections, per-process RSS, battery health in score
- `cmd/dupes/`: CLI (no TUI). Multi-phase pipeline: walk → size group → inode dedup → partial xxhash (4KB) → full xxhash → sort by reclaimable. Three modes: report (default), delete (Finder Trash), conserve (relocate with manifest + restore). Cross-volume copy+verify for conserve
- `cmd/watch/`: CLI (no TUI). Rule-based threshold monitoring, configurable via `~/.config/burrow/watch_rules`, macOS notifications via osascript, 15-minute cooldown per rule, predictive disk space projection

## Code Conventions

### Bash (all shell scripts)

- **Bash 3.2+ compatible** (macOS default) - no associative arrays, no `${var,,}`, no `readarray`
- **BSD commands only** - `stat -f%z` not `stat --format`, `sed -i ''` not `sed -i`
- `set -euo pipefail` in all scripts
- 4-space indent, `snake_case` functions, `local` for function vars, `readonly` for constants
- Quote all variables: `"$variable"` - no unquoted expansions
- Use `[[ ]]` not `[ ]` for tests
- Handle pipefail: `cmd || true`, check `${#array[@]} -gt 0` before iterating, `((count++)) || true`
- **Never use `rm -rf` directly** - always use safe wrappers: `safe_remove()`, `safe_find_delete()`, `safe_sudo_remove()`
- Path validation is mandatory before any deletion (see `lib/core/file_ops.sh`)
- Logging via `log_info`, `log_success`, `log_warning`, `log_error` - never raw `echo` for user output
- Debug mode: check `BW_DEBUG` variable, format as `[MODULE_NAME] message` to stderr
- Use `command cp -f` in install scripts to bypass shell aliases (`cp -i`)

### Go (cmd/analyze, cmd/status, cmd/dupes, cmd/watch)

- Files focused on single responsibility, <500 lines each
- Extract constants to `constants.go` - no magic numbers
- Use context for timeout control on external commands
- Explicit error returns, no panic in production code
- Table-driven tests, mock data for unavailable metrics
- Format with `goimports` then `gofmt`
- Lint with `golangci-lint` (govet, staticcheck, errcheck, ineffassign, unused, modernize)
- Module path: `github.com/jimmy-jain/burrow`

### Linter Configuration

- **shellcheck**: disabled SC2155, SC2034, SC2059, SC1091, SC2038 (see `.shellcheckrc`)
- **golangci-lint**: govet (all except shadow/fieldalignment), errcheck (excludes Close/Run/Start), staticcheck (all except QF1003/SA9003)
- **shfmt**: follows `.editorconfig` (4-space indent for shell)

## Safety Rules

These are critical - Burrow performs destructive operations on user systems:

1. **Protected system paths** - Never delete: `/`, `/System/*`, `/bin/*`, `/sbin/*`, `/usr/bin`, `/usr/lib`, `/etc/*`, `/private/etc/*`, `/Library/Extensions`
2. **Protected apps** - Safari, Finder, Mail, Messages, Notes, Calendar, Reminders (see `lib/core/app_protection.sh`)
3. **Symlink resolution** - Always resolve symlink targets and validate resolved paths against protection lists
4. **Path traversal prevention** - Reject paths containing `..`
5. **Whitelist support** - User-protected paths in `~/.config/burrow/whitelist` must be respected
6. **Dry-run first** - Destructive commands should support `--dry-run` preview
7. **Operation logging** - All deletions logged to `operations.log` (5MB rotation)
8. **Trash over delete** - `bw analyze` and `bw dupes --delete` move to Trash via Finder (recoverable), not permanent deletion
9. **Pre-clean snapshot** - APFS local snapshot created before `bw clean` runs (skip with `BW_SKIP_SNAPSHOT=1`)
10. **First-run safety** - First `bw clean` forces dry-run with confirmation before real cleanup
11. **Risk categorization** - Dry-run output shows `[LOW]`/`[MEDIUM]`/`[HIGH]` risk labels

## Development & Release Strategy

### Branching Model

| Branch | Purpose | CI Workflows |
|--------|---------|--------------|
| `main` | Stable releases only | check, test, codeql, release (on `V*` tag) |
| `dev` | Integration branch, nightly builds | check, test, codeql, nightly |
| `feat/*`, `fix/*` | Feature/fix branches | PR into `dev` |

### Release Channels

| Channel | Source | Install method |
|---------|--------|----------------|
| `stable` | Tagged `V*` on `main` | `brew install jimmy-jain/burrow/burrow` |
| `nightly` | Every push to `dev` | `curl -fsSL .../install.sh \| BURROW_VERSION=main bash` |
| `head` | Latest `main` commit | `brew install --HEAD jimmy-jain/burrow/burrow` |

### Workflow

1. Create `feat/*` or `fix/*` branch from `dev`
2. PR into `dev` — runs check, test, codeql, security
3. Merge to `dev` — triggers nightly build (test → build → publish pre-release)
4. When ready for stable release: PR `dev` → `main`, tag `V*`
5. Tag push triggers release workflow (build → attest → GitHub release → Homebrew formula update)

### CI Pipelines

| Workflow | Trigger | What it does |
|----------|---------|--------------|
| `check.yml` | Push/PR to main, dev | Format (shfmt, goimports, gofmt) + lint (shellcheck, golangci-lint) |
| `test.yml` | Push/PR to main, dev | BATS shell tests, Go tests, macOS compatibility matrix, security checks |
| `codeql.yml` | Push/PR to main, dev + weekly | Static analysis (Go + Actions) |
| `nightly.yml` | Push to dev | Full test suite → build both archs → publish rolling pre-release |
| `release.yml` | `V*` tag push | Go tests → build → verify binaries → checksums → attestation → GitHub release → Homebrew |

## Testing

- **Shell tests**: BATS framework, 30 test suites in `tests/` - run via `./scripts/test.sh`
- **Go tests**: Standard `go test` in `cmd/analyze/`, `cmd/status/`, `cmd/dupes/`, `cmd/watch/`
- **CI tests on**: macOS 14 (Sonoma) and macOS 15 (Sequoia)
- **Security checks in CI**: unsafe `rm -rf` detection, app protection validation, secret scanning, high-risk path regression
- **TDD workflow**: Write tests first, then implement until tests pass

## User Configuration

| Path | Purpose |
|------|---------|
| `~/.config/burrow/whitelist` | Protected cache paths (one per line, `#` comments) |
| `~/.config/burrow/purge_paths` | Custom project scan directories |
| `~/.config/burrow/status_prefs` | Status panel preferences |
| `~/.config/burrow/watch_rules` | Threshold alert rules (e.g., `disk_free_gb < 10`) |
| `~/.config/burrow/install_channel` | Install metadata (channel, commit hash) |
| `~/.config/burrow/first_run_done` | Sentinel file for first-run dry-run |
| `~/.config/burrow/size_history.json` | Disk size history for predictive projections |
| `~/.cache/burrow/` | Update notification cache, version check timestamps |
| `~/Library/LaunchAgents/dev.burrow.maintenance.plist` | Scheduled maintenance agent |

## Key Dependencies

- **Go 1.25.0**, bubbletea v1.3.10, lipgloss v1.1.0, gopsutil v4.26.2, xxhash/v2 v2.3.0
- **Dev tools**: shfmt, shellcheck, bats-core, golangci-lint, goimports
