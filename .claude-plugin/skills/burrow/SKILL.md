---
name: burrow
description: |
  This skill should be used when the user asks about "system maintenance", "clean my Mac",
  "disk space", "what's using space", "find duplicates", "system health", "cache cleanup",
  "developer environment check", "Mac optimization", "free up space", "disk analysis",
  "how much space", "storage", "battery health", "memory usage", "CPU usage",
  or references burrow/bw commands.
version: 0.2.4
---

# Burrow — macOS System Maintenance

Burrow is a macOS system maintenance CLI. When users ask about system health, disk space, cleanup, or Mac optimization, use the appropriate burrow command.

## Available Commands

| Command | What it does |
|---------|-------------|
| `/status` | System health — CPU, memory, disk, battery, network |
| `/clean` | Preview and execute cache/log/temp cleanup |
| `/analyze <path>` | Disk usage analysis for a directory |
| `/dupes <path>` | Find duplicate files by content hash |
| `/doctor` | Developer environment health checks |
| `/size` | Developer cache size audit |
| `/maintain` | Full maintenance routine (all of the above) |

## Quick Routing

- "How's my system doing?" → `/status`
- "Clean up my Mac" / "free up space" → `/clean`
- "What's using disk space?" → `/analyze`
- "Find duplicate files" → `/dupes`
- "Check my dev environment" → `/doctor`
- "How big are my caches?" → `/size`
- "Full maintenance" → `/maintain`

## CLI Usage

All commands are available via `burrow <command>` or `bw <command>`. Most support `--json` for structured output.
