# Burrow - CLAUDE.md

Burrow is a macOS-only system maintenance CLI that combines cleanup, uninstall, disk analysis, duplicate detection, live monitoring, threshold alerts, reporting, and AI-tool integration in one project.

This file reflects the repo as it exists today, based on the checked-in code, build files, hidden config, and command entrypoints in this workspace.

## Project Snapshot

- Primary CLI entrypoint: `burrow`
- User-facing alias: `bw`
- Main implementation split:
  - Bash for orchestration, destructive flows, configuration, and menus
  - Go for performance-sensitive analyzers, dashboards, duplicate scanning, alerts, and MCP server integration
- Go module path: `github.com/jimmy-jain/burrow`
- Current Go toolchain in repo: `go 1.25.0`

## Verified Entrypoints

### User CLI

- `burrow` is the real router
- `bw` is the lightweight alias users run most often
- `burrow` sources:
  - `lib/core/common.sh`
  - `lib/core/commands.sh`

### Go-backed wrappers

These shell wrappers exec bundled binaries from `bin/` when present:

- `bin/analyze.sh` -> `bin/analyze-go`
- `bin/status.sh` -> `bin/status-go`
- `bin/dupes.sh` -> `bin/dupes-go`
- `bin/watch.sh` -> `bin/watch-go`

### Separate MCP binary

- `cmd/mcp/` builds to `bin/burrow-mcp`
- This is not a normal `bw mcp` subcommand
- It exposes Burrow capabilities over MCP stdio for assistant/tool integrations

## Quick Reference

```bash
# Dev setup
brew install shfmt shellcheck bats-core golangci-lint
go install golang.org/x/tools/cmd/goimports@latest
git config core.hooksPath .githooks

# Quality checks
./scripts/check.sh

# Tests
./scripts/test.sh

# Local build
make build

# Release builds
make release-arm64
make release-amd64

# Run Go programs directly
go run ./cmd/analyze
go run ./cmd/status
go run ./cmd/dupes
go run ./cmd/watch
go run ./cmd/mcp
```

## Actual Command Surface

The shared command list in `lib/core/commands.sh` and router in `burrow` currently expose:

| Command | Type | Notes |
|---------|------|-------|
| `bw clean` | Bash | Deep cleanup with dry-run and whitelist support |
| `bw uninstall` | Bash | App removal plus remnants and launch agents |
| `bw optimize` | Bash | Maintenance/optimization flows |
| `bw analyze` | Go TUI / JSON | Disk explorer; supports `--json` |
| `bw status` | Go TUI / JSON | Live system dashboard; auto-JSON when piped |
| `bw purge` | Bash | Project artifact cleanup |
| `bw installer` | Bash | Installer file cleanup |
| `bw touchid` | Bash | Touch ID sudo configuration |
| `bw size` | Bash | Developer cache size audit |
| `bw doctor` | Bash | Developer environment checks |
| `bw log` | Bash | Operations log viewer |
| `bw report` | Bash | Combined machine health report |
| `bw dupes` | Go CLI | Duplicate detection, delete, conserve, restore |
| `bw watch` | Go CLI | Rule-based monitoring with notifications |
| `bw schedule` | Bash | LaunchAgent maintenance scheduling |
| `bw hook` | Bash | Shell hook integration |
| `bw launchers` | Bash | Raycast/Alfred quick launcher setup |
| `bw completion` | Bash | Shell completion setup |
| `bw update` | Bash | Update flow for script or Homebrew installs |
| `bw remove` | Bash | Remove Burrow from the system |
| `bw help` | Bash | Help output |
| `bw version` | Bash | Version/system info |

## Architecture

### High-level layout

| Layer | Language | Purpose |
|-------|----------|---------|
| CLI routing | Bash | `burrow`, `bw`, command dispatch, version/update/remove |
| Core shell libraries | Bash | Logging, UI, safe file ops, help text, sudo, app protection |
| Feature modules | Bash | Cleanup, uninstall, optimize, report, doctor, size, manage, log |
| TUI / scanning tools | Go | Analyze disk usage, status dashboard |
| CLI utilities | Go | Duplicate detection, threshold watch, MCP server |
| Plugin integration | JSON/Markdown | Claude plugin metadata and skill routing |
| CI / release | YAML | Checks, tests, tagging, release publishing |

### Shell loading order

`lib/core/common.sh` is the orchestrator and is guarded by:

```bash
if [[ -n "${BURROW_COMMON_LOADED:-}" ]]; then
    return 0
fi
```

It sources core modules in this order:

1. `base.sh`
2. `log.sh`
3. `timeout.sh`
4. `file_ops.sh`
5. `help.sh`
6. `ui.sh`
7. `app_protection.sh`
8. `sudo.sh` when present

### Shell feature areas

- `lib/clean/`: user, system, developer, project, browser, app cache cleanup
- `lib/uninstall/`: app uninstall flows and brew-related removal
- `lib/optimize/`: maintenance and update prompting
- `lib/manage/`: whitelist, schedule, hook, purge path config, update helpers, autofix
- `lib/doctor/`: machine/developer health checks
- `lib/size/`: cache size reporting
- `lib/report/`: combined JSON/system reports
- `lib/log/`: operations log viewer
- `lib/check/`: health/report helper shell code
- `lib/ui/`: menu/app selector UI helpers

### Go command packages

- `cmd/analyze/`
  - Bubble Tea disk analyzer
  - TUI plus `--json`
  - Overview mode when no path is supplied
  - Caching and background prefetch support
- `cmd/status/`
  - Bubble Tea live dashboard
  - JSON output support
  - Process alerting and rolling metrics
- `cmd/dupes/`
  - Report, delete, conserve, restore modes
  - Uses xxhash-based duplicate pipeline
- `cmd/watch/`
  - Rule parser and evaluator
  - macOS notification integration
  - Configurable interval and one-shot mode
- `cmd/mcp/`
  - MCP stdio server
  - Registers Burrow tools for status, analyze, dupes, doctor, size, report, clean preview/execute, duplicate conserve/restore

## Repository Layout

```text
burrow                  Main CLI router
bw                      Lightweight alias
bin/                    Command wrappers, helper scripts, compiled Go binaries
cmd/                    Go command packages
  analyze/              Disk analyzer
  status/               Live system monitor
  dupes/                Duplicate file finder
  watch/                Threshold monitor
  mcp/                  MCP server
lib/                    Bash implementation
  core/                 Shared infrastructure
  clean/                Cleanup modules
  uninstall/            App removal flows
  optimize/             Maintenance/update helpers
  manage/               Scheduling, hooks, config helpers
  doctor/               Environment checks
  size/                 Cache size audit
  report/               Machine health report
  log/                  Operations log viewer
  check/                Report/check support code
  ui/                   Menu helpers
scripts/                Dev scripts, formatting, tests, launcher setup
tests/                  BATS and shell integration tests
.github/workflows/      CI and release workflows
.githooks/              Local git hooks
.claude-plugin/         Claude plugin metadata, commands, and Burrow skill
Formula/                Homebrew formula
assets/                 Branding/screenshots
deprecate/              Old assets/workflows retained outside active paths
```

## Hidden Files And What They Mean

These hidden files matter to day-to-day work in this repo:

- `.editorconfig`
  - Shell uses 4 spaces
  - YAML uses 2 spaces
  - Makefile uses tabs
- `.shellcheckrc`
  - Disables `SC2155`, `SC2034`, `SC2059`, `SC1091`, `SC2038`
- `.golangci.yml`
  - Enables `govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`
  - Uses `modules-download-mode: readonly`
  - Config uses `version: "2"` schema — requires golangci-lint v2
- `.githooks/pre-commit`
  - Repo-defined hooks path is intended via `git config core.hooksPath .githooks`
- `.github/workflows/`
  - Active checked-in workflows are `check.yml`, `test.yml`, `release.yml`, `auto-tag.yml`
- `.claude-plugin/`
  - Committed integration assets for Claude plugin/skill use
- `.claude/`
  - Present locally in this workspace but ignored by git
  - Contains local settings, permissions, and upstream-sync notes
  - Useful context for local development, but not part of the shipped product

## Code Conventions

### Bash

- Target Bash `3.2+` on macOS
- Do not use associative arrays, `${var,,}`, or `readarray`
- Prefer BSD/macOS-compatible flags and behavior
- Use `set -euo pipefail`
- Use 4-space indentation
- Prefer `snake_case` function names
- Quote variable expansions
- Prefer `[[ ... ]]` over `[ ... ]`
- Favor `local` for function-scoped variables
- Prefer Burrow safe deletion/path-validation helpers in normal feature code
- Logging should go through `log_info`, `log_success`, `log_warning`, `log_error`

### Important Bash nuance

The repo standard is to avoid direct destructive deletion in feature code, but standalone scripts such as `install.sh` and `uninstall.sh` contain self-contained removal helpers because they must work without sourcing the full library stack. Preserve that distinction when editing.

### Go

- Keep files focused and reasonably small
- Prefer explicit errors over panics
- Use table-driven tests
- Use `goimports` then `gofmt`
- Lint with `golangci-lint`
- Most production Go code lives under `cmd/`
- Several commands are Darwin-only via build tags

## Safety Rules

Burrow performs system maintenance and can delete user data, so these constraints are central:

1. Never delete protected system paths like `/`, `/System/*`, `/bin/*`, `/sbin/*`, `/usr/bin`, `/usr/lib`, `/etc/*`, `/private/etc/*`, or `/Library/Extensions`.
2. Respect protected apps such as Safari, Finder, Mail, Messages, Notes, Calendar, and Reminders.
3. Resolve symlinks before destructive actions and validate resolved targets.
4. Reject path traversal patterns like `..`.
5. Respect user whitelist entries in `~/.config/burrow/whitelist`.
6. Prefer `--dry-run` support for destructive commands.
7. Log operations to Burrow's operations log.
8. Prefer Trash/recoverable flows where the command is designed for it.
9. `bw clean` supports snapshot-oriented safety behavior and first-run caution.
10. Surface risk honestly in previews and prompts.

## Testing And Quality

### Checked-in tooling

- `scripts/check.sh`
  - Formats shell and Go code
  - Runs `golangci-lint` or `go vet`
  - Runs ShellCheck
  - Runs syntax checks
- `scripts/test.sh`
  - Lints test scripts
  - Runs BATS suites
  - Runs `go build`, `go vet`, and `go test ./cmd/...`
  - Checks module loading and lightweight integration flows

### Current test footprint in this repo

- `41` BATS files under `tests/`
- `25` Go `_test.go` files under `cmd/`

## Build And Release Reality

### Make targets

- `make build`
  - Builds local `analyze`, `status`, `watch`, `dupes`, and `burrow-mcp`
- `make release-arm64`
- `make release-amd64`

### Current GitHub workflows

- `check.yml`
  - Push/PR on `main` and `dev`
  - Runs shell checks and Go linting
- `test.yml`
  - Push/PR on `main` and `dev`
  - Runs on `macos-14` (arm64); amd64 covered by release build matrix
- `release.yml`
  - Runs on `V*` tag push
  - Tests, builds artifacts, publishes GitHub release
- `auto-tag.yml`
  - Triggered by `workflow_run` when both `check` and `test` succeed on `main`
  - Bumps `VERSION` in `burrow`, commits, and creates a `V*` tag from conventional commits
  - `feat:` → minor bump, everything else → patch bump

### Important note

Older docs in the repo mention additional workflows such as `nightly` or `codeql`, but those are not present in the active `.github/workflows/` directory right now. Use the checked-in workflow files as the source of truth.

## User Config And Runtime Files

- `~/.config/burrow/whitelist`
- `~/.config/burrow/purge_paths`
- `~/.config/burrow/status_prefs`
- `~/.config/burrow/watch_rules`
- `~/.config/burrow/install_channel`
- `~/.config/burrow/first_run_done`
- `~/.config/burrow/launcher_version`
- `~/.config/burrow/size_history.json`
- `~/.cache/burrow/`
- `~/Library/LaunchAgents/dev.burrow.maintenance.plist`

## Agent Notes

- Treat `burrow`, `README.md`, `Makefile`, `.github/workflows/`, and the hidden config files above as the primary truth sources for project behavior.
- If `AGENTS.md` or older guidance disagree with the code, trust the code and update the docs.
- The worktree may contain local ignored Claude metadata under `.claude/`; do not assume that content ships with the project.
