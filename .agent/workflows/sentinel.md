---
description: Invoke the Sentinel — review any proposed runtime or emitter change for scaling safety
---

// turbo
1. Read the agent brain:
```
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Agents/sentinel.md
```

// turbo
2. Read the current C runtime and architecture:
```
cat /home/askme/Pictures/wolf-lang/.wolf-vault/RnD/architecture.md
wc -l /home/askme/Pictures/wolf-lang/runtime/wolf_runtime.c
grep -n "malloc\|free\|static.*=\|nanosleep\|while.*1" /home/askme/Pictures/wolf-lang/runtime/wolf_runtime.c | head -40
```

// turbo
3. Measure current binary size across platforms:
```
ls -lh /home/askme/Pictures/wolf-lang/wolf 2>/dev/null || echo "(binary not built)"
cd /home/askme/Pictures/wolf-lang && go build -o ./wolf ./cmd/wolf 2>&1 && ls -lh ./wolf
```

4. Act as the Sentinel:
   - Run the **Scalability Checklist** from `sentinel.md` on the change being reviewed.
   - Apply the **5 Questions** to every function or data structure involved.
   - Write a **Sentinel Review annotation** (format from `sentinel.md`) for each flagged area.
   - Deliver a final **Verdict**: APPROVED, APPROVED WITH NOTES, or REJECTED.
   - If REJECTED, provide a concrete alternative that passes all checks.
