---
description: Developer cache size audit (Homebrew, npm, pip, Docker, etc.)
allowed-tools: Bash(burrow size:*), Bash(bw size:*)
---

Audit developer cache sizes.

```
!burrow size --json
```

Present the results:
- Table of caches sorted by size (largest first)
- Total footprint
- Flag caches over 1GB as cleanup candidates
- Suggest specific cleanup commands for the largest caches (e.g., `brew cleanup`, `docker system prune`, `npm cache clean`)
