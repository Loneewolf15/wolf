---
description: End a Wolf development session — updates the vault and creates handoff notes for the next session
---

# /wrap-up — End Session

## What this command does
Updates all vault department files and creates a detailed handoff note so the next session
(or a parallel agent) can pick up instantly with full context.

// turbo
1. Run the full test suite to confirm green state:
```
cd /home/askme/Pictures/wolf-lang && go test ./internal/... ./e2e/... 2>&1 | tail -15
```

// turbo
2. Get the commit summary for this session:
```
cd /home/askme/Pictures/wolf-lang && git log --oneline $(git log --oneline | tail -1 | cut -d' ' -f1)..HEAD 2>/dev/null || git log --oneline -5
```

3. Update `.wolf-vault/Execution/plan.md`:
   - Mark completed tasks as ✅
   - Update "Active Tasks" table
   - Add this session to "Session History" block
   - Update "Next Unblocked Tasks" list

4. Update `.wolf-vault/RnD/bugs_fixed.md`:
   - Add any new bugs fixed this session to the cumulative log
   - Update the Status Ledger at the bottom

5. Update `.wolf-vault/RnD/architecture.md`:
   - Add any new ADRs (architecture decision records) made this session
   - Update the Runtime Subsystems table with new status

6. Write `.wolf-vault/Sessions/latest_handoff.md` with:
   ```markdown
   # Handoff — [DATE]

   ## Where We Left Off
   [What was the last thing completed / in progress]

   ## Commits This Session
   [git log --oneline output]

   ## Tests Status
   [pass/fail counts]

   ## Next Immediate Task
   [Specific task, specific file, specific function to touch next]

   ## Open Issues / Watch Out For
   [Anything the next agent should know to avoid stepping on landmines]

   ## Relevant Files Modified This Session
   [List of files]
   ```

7. Report summary to user — what was accomplished, what's next.
