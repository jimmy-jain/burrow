---
description: System health check — CPU, memory, disk, battery, network
allowed-tools: Bash(burrow status:*), Bash(bw status:*)
---

Run a system health check and summarize the results.

```
!burrow status --json
```

Analyze the JSON output and present a concise health summary:
- Overall health score and status
- Flag any metrics that are warning or critical (high CPU, low memory, low disk, battery degraded)
- If disk is below 20% free, suggest running `/clean`
- If health score is below 70, recommend specific actions
- Keep it short — table format for metrics, prose only for actionable items
