---
description: Full maintenance routine — health check, cleanup preview, dev environment audit
allowed-tools: Bash(burrow status:*), Bash(burrow clean:*), Bash(burrow doctor:*), Bash(burrow size:*), Bash(bw status:*), Bash(bw clean:*), Bash(bw doctor:*), Bash(bw size:*)
---

Run a full system maintenance check. This combines multiple burrow commands into one report.

## Step 1: System Health

```
!burrow status --json
```

## Step 2: Developer Environment

```
!burrow doctor --json
```

## Step 3: Cache Sizes

```
!burrow size --json
```

## Step 4: Cleanup Preview

```
!burrow clean --dry-run
```

## Output

Combine all results into a single maintenance report:

1. **Health Summary** — overall score, any warnings/critical metrics
2. **Dev Environment** — any failed checks or outdated tools
3. **Cache Sizes** — largest caches, total dev cache footprint
4. **Cleanup Available** — reclaimable space, breakdown by risk level

End with prioritized recommendations (most impactful first). Ask if the user wants to proceed with any suggested actions.
