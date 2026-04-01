---
description: Developer environment health checks
allowed-tools: Bash(burrow doctor:*), Bash(bw doctor:*)
---

Check the health of the developer environment.

```
!burrow doctor --json
```

Present the results:
- Group checks by status (pass/warn/fail)
- For any failures, explain what's wrong and how to fix it
- For warnings, note the issue and whether it's urgent
- Keep passing checks brief — just a summary count
