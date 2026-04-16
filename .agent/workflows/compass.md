---
description: Invoke the Compass — evaluate the next sprint task against the roadmap and dependency graph
---

// turbo
1. Read the agent brain:
```
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Agents/compass.md
```

// turbo
2. Capture current project state:
```
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Execution/plan.md
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Product/manifesto.md 2>/dev/null || echo "(no manifesto.md)"
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Product/roadmap.md 2>/dev/null || echo "(no roadmap.md)"
cd /home/askme/Pictures/wolf-lang && git log --oneline -10
```

// turbo
3. Check for any open bugs that must be resolved first:
```
grep "Open:" /home/askme/Pictures/wolf-lang/.wolf-vault/RnD/bugs_fixed.md
```

4. Act as the Compass:
   - Apply the **Decision Matrix** from `compass.md` to the current "Next Unblocked Tasks".
   - Check dependency ordering — verify no layers are being skipped.
   - Produce a **Sprint Brief** (format from `compass.md`) for the top 3 candidate tasks.
   - Save the brief to `.wolf-vault/Sessions/sprint_brief_YYYY-MM-DD.md`.
   - Report the **Verdict** (GO / DEFER / BLOCK) for each candidate.
