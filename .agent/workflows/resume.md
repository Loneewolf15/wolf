---
description: Start a Wolf development session — reads vault context and picks up exactly where we left off
---

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

---

## 🐺 Auto Wolf Pack Triage (runs after resume report)

// turbo
5. Run the Bloodhound — scan for regressions:
```
cd /home/askme/Pictures/wolf-lang
grep -n "FAIL\|panic\|SIGSEGV\|error:" test.log 2>/dev/null | head -30 || echo "(no test.log — clean)"
go test ./... 2>&1 | grep -E "^(FAIL|ok)" | head -20
```

6. Act as the **Bloodhound** (`.wolf-vault/Agents/bloodhound.md`):
   - List any failures ranked by P0–P3 blast radius.
   - If any P0 or P1 present: draft a Minimum Reproducible Script and block proceeding.
   - If all green: state "✅ Bloodhound: No regressions found."

// turbo
7. Run the Compass — evaluate current sprint priorities:
```
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Execution/plan.md
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Product/roadmap.md 2>/dev/null || echo "(no roadmap)"
```

8. Act as the **Compass** (`.wolf-vault/Agents/compass.md`):
   - Score top 3 candidate tasks using the Decision Matrix.
   - Output the recommended next task with a GO / DEFER / BLOCK verdict.

// turbo
9. Run the Sentinel — validate the runtime/emitter safety:
```
grep -n "malloc\|free\|static.*=\|nanosleep\|while.*1" /home/askme/Pictures/wolf-lang/runtime/wolf_runtime.c | head -30
ls -lh /home/askme/Pictures/wolf-lang/wolf 2>/dev/null || echo "(binary not yet built)"
```

10. Act as the **Sentinel** (`.wolf-vault/Agents/sentinel.md`):
    - Run the Scalability Checklist on the C runtime state.
    - Flag any O(n²) patterns or hot-path `malloc` calls.
    - Output a one-line scaling health verdict.

---

11. Final summary for the user:
    - Bloodhound status (regressions / clean)
    - Compass pick (recommended next task)
    - Sentinel verdict (scaling health)
    - Ready to begin work 🚀
