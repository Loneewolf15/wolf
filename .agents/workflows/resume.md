---
description: Start a Wolf development session — reads vault context and picks up exactly where we left off
---

# /resume — Start Session

## What this command does
Reads the Wolf Vault to give full context on the project state before any work begins.

// turbo
1. Read the execution plan and last handoff:
```
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Execution/plan.md
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Sessions/latest_handoff.md 2>/dev/null || echo "(no handoff yet — fresh session)"
```

// turbo
2. Read the architecture reference:
```
cat /home/askme/Pictures/wolf-lang/.wolf-vault/RnD/architecture.md
```

// turbo
3. Check current git status and recent commits:
```
cd /home/askme/Pictures/wolf-lang && git log --oneline -10 && echo "---" && git status --short
```

4. Report to user:
   - Where we left off (from handoff notes)
   - Current active sprint task
   - Next unblocked tasks from execution plan
   - Any open bugs from RnD/bugs_fixed.md marked as "In progress"
   - Current test status: `go test ./internal/... 2>&1 | tail -5`
