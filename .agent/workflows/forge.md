---
description: Invoke the Forge — evaluate bare-metal/hardware compatibility of a runtime component
---

// turbo
1. Read the agent brain:
```
cat /home/askme/Pictures/wolf-lang/.wolf-vault/Agents/forge.md
```

// turbo
2. Audit current runtime for OS dependencies:
```
grep -n "#include\|syscall\|pthread\|malloc\|socket\|unistd\|mysql_\|redis_" \
  /home/askme/Pictures/wolf-lang/runtime/wolf_runtime.c | head -60
grep -n "#include" /home/askme/Pictures/wolf-lang/runtime/wolf_runtime.h | head -30
```

// turbo
3. Check current LLVM target support in the emitter:
```
grep -n "target\|triple\|freestanding\|march\|mcpu" \
  /home/askme/Pictures/wolf-lang/internal/emitter/llvm_emitter.go | head -20
grep -n "freestanding\|WOLF_BARE" \
  /home/askme/Pictures/wolf-lang/internal/compiler/compiler.go 2>/dev/null | head -10
```

4. Act as the Forge:
   - Run the **4 Questions** from `forge.md` on every function in the component being reviewed.
   - Identify exactly which `#include`s, syscalls, or deps must be wrapped in `#ifdef WOLF_FREESTANDING` guards.
   - Estimate the stripped binary's flash footprint (target ≤ 256KB for embedded).
   - Produce a **Forge Review annotation** (format from `forge.md`) with SHIP / CONDITIONAL / SERVER-ONLY verdict.
   - If CONDITIONAL: provide the exact `#ifdef` strip plan needed to unlock embedded use.
