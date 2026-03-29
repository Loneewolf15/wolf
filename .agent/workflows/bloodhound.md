---
description: Invoke the Bloodhound — scan logs for bugs, rank by blast radius, generate MRS
---

// turbo
1. Read the agent brain:
```
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Agents/bloodhound.md
```

// turbo
2. Pull the latest test logs and build errors:
```
cat /tmp/build_err.log 2>/dev/null || echo "(no build_err.log)"
grep -n "FAIL\|panic\|SIGSEGV\|error:" /home/askme/Pictures/wolf-lang/test.log 2>/dev/null | head -60 || echo "(no test.log)"
cd /home/askme/Pictures/wolf-lang && go test ./... 2>&1 | grep -E "FAIL|panic|--- FAIL" | head -30
```

// turbo
3. Check the current bug ledger for any open items:
```
grep "Open\|In progress\|WIP" /home/askme/Pictures/wolf-lang/.wolf-vault/RnD/bugs_fixed.md
```

4. Act as the Bloodhound:
   - Rank every failure found by the P0–P3 blast radius scale from `bloodhound.md`.
   - For each P0 or P1 issue: draft a **Minimum Reproducible Script (MRS)** in `e2e/testdata/_bug_XXX.wolf`.
   - Propose a fix for the highest-priority item only.
   - Never fix more than one bug per workflow run.
   - Append the new bug and fix to `.wolf-vault/RnD/bugs_fixed.md`.
