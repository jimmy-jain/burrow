# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.3.0] - 2026-03-29

### Added

- **Claude Code plugin** (`.claude-plugin/`) — exposes `bw` commands as Claude slash commands and a skill:
  - `/clean`, `/analyze`, `/status`, `/dupes`, `/doctor`, `/size`, `/maintain` slash commands
  - `burrow` skill for AI-assisted maintenance workflows
- **MCP executor test isolation** — `BURROW_FALLBACK_PATH` env var allows overriding the hardcoded `/usr/local/bin/burrow` fallback in tests, fixing `TestResolveBinaryMissing` on machines with burrow installed

### Changed

- **Time Machine cleanup is now scan-only** — `bw clean` no longer deletes incomplete `.inProgress` backup directories. Incomplete backups are reported with the exact `sudo tmutil delete` command for the user to run manually, preventing accidental backup data loss
- **Free space reporting uses `diskutil info`** — `get_free_space()` now reads "Container Free Space" from `diskutil info /System/Volumes/Data` instead of `df -h Available`. This reports actual non-purgeable free space, so the before/after numbers correctly reflect cache cleanup (APFS marks caches as purgeable, which `df` already counts as available)
