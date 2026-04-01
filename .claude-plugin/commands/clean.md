---
description: Preview and execute system cleanup (caches, logs, temp files)
allowed-tools: Bash(burrow clean:*), Bash(bw clean:*)
argument-hint: "[--execute]"
---

## Context

User wants to clean up their Mac. This is a two-phase process: preview first, then execute if requested.

Arguments: !`echo "${ARGUMENTS}"`

## Instructions

### Phase 1: Preview (default)

If no `--execute` flag was passed, run a dry-run preview:

```
!burrow clean --dry-run
```

Present the results:
- Total space reclaimable
- Breakdown by category (caches, logs, temp files, etc.)
- Risk levels for each category ([LOW], [MEDIUM], [HIGH])
- Ask the user if they want to proceed with the actual cleanup

### Phase 2: Execute

If the user passed `--execute` or confirmed after preview:

```
!burrow clean
```

Report what was cleaned and how much space was freed.

**Safety**: Always preview before executing. Never skip the dry-run unless the user explicitly passes `--execute`.
