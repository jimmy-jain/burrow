---
description: Find duplicate files and reclaim disk space
allowed-tools: Bash(burrow dupes:*), Bash(bw dupes:*)
argument-hint: "<path> [--min-size SIZE]"
---

## Context

Find duplicate files by content hash.

Arguments: !`echo "${ARGUMENTS}"`

## Instructions

If no path argument was provided, default to the home directory `~`.

Scan for duplicates:

```
!burrow dupes --json ${ARGUMENTS:-~}
```

Present the results:
- Total reclaimable space
- Group duplicates by content, showing file paths and sizes
- Rank by reclaimable space (largest first)
- Suggest which copies to keep (prefer shorter paths, standard locations)
- If significant space can be reclaimed, mention `burrow dupes --conserve <dir> <path>` as a safe option (moves to conservation dir with restore manifest)
