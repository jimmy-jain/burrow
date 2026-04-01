---
description: Disk usage analysis for a directory
allowed-tools: Bash(burrow analyze:*), Bash(bw analyze:*)
argument-hint: "<path>"
---

## Context

Analyze disk usage for a given path.

Arguments: !`echo "${ARGUMENTS}"`

## Instructions

If no path argument was provided, default to the home directory `~`.

Run the analysis:

```
!burrow analyze --json ${ARGUMENTS:-~}
```

Present the results:
- Top space consumers in a table (name, size, % of total)
- Highlight anything unexpectedly large
- Suggest cleanup actions for obvious targets (node_modules, .cache dirs, build artifacts)
- Keep output concise — focus on actionable findings
