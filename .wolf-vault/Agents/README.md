# 🐺 Wolf Pack Agents — Team Overview

> "A language that builds itself."

## The Pack

| Agent | Role | When to invoke |
| :--- | :--- | :--- |
| 🐕 **Bloodhound** | Bug Fixer & Ranker | Test failures, regressions, compiler panics |
| 🧭 **Compass** | Roadmap & Prioritization | Before any new sprint task or feature |
| 🛡️ **Sentinel** | 10× Speed Enforcer & Scaling Gatekeeper | Before merging runtime or emitter changes |
| ⚡ **Forge** | Hardware & Bare-Metal Systems Agent | Before any `--freestanding` or embedded target work |

---

## The Full Pipeline (runs automatically after `/resume`)

```
/resume reads vault
       │
       ▼
 ┌─────────────┐
 │ 🐕 BLOODHOUND│ ← "Any regressions since last session?"
 └──────┬──────┘
        │ Clean ✅
        ▼
 ┌─────────────┐
 │ 🧭 COMPASS  │ ← "What should we build next?"
 └──────┬──────┘
        │ GO ✅
        ▼
[Work is done this session]
        │
        ▼
 ┌──────────────────┐
 │  🛡️ SENTINEL     │ ← "Will this scale? Will this be 10× fast?"
 └──────┬───────────┘
        │ APPROVED ✅
        ▼
 ┌──────────────┐
 │  ⚡ FORGE    │ ← "Can this run bare-metal?" (if touching runtime)
 └──────┬───────┘
        │ SHIP ✅
        ▼
    /wrap-up  →  vault updated
```

---

## Sentinel — The 10x Speed Mandate
The Sentinel now enforces two axes:
- **Scaling** — concurrency, lock hygiene, arena memory, binary size
- **Speed** — Zero-cost abstractions, SoA data locality, SIMD vectorization, branchless ops

> "If the CPU is waiting for memory, Wolf is failing. If a branch is unpredictable, Wolf is failing. We don't just run code; we orchestrate the silicon."

**Wolf vs the world:**

| Feature | C | Rust | Go | Wolf 🐺 |
| :--- | :---: | :---: | :---: | :---: |
| Syntax | Hard | Very Hard | Easy | Easiest |
| Speed | 1.0× | 1.0× | ~1.5× slower | **Goal: 0.1× (10× faster)** |
| Hardware | Native | Native | Requires runtime | Native (LLVM) |
| Safety | Manual | Compile-time | Runtime | AI-Guardrails |

---

## Forge — The Hardware Mandate
Wolf targets every piece of silicon LLVM supports:

| Target Class | Examples | Flag |
| :--- | :--- | :--- |
| Cloud/Server | Linux/amd64, macOS/arm64 | (default) |
| Embedded Linux | Raspberry Pi | `--target=linux-arm` |
| Bare Metal | ESP32, STM32, Arduino | `--freestanding` |
| Custom Silicon | RISC-V | `--target=<triple>` |

---

## Slash Commands

| Command | Agent invoked |
| :--- | :--- |
| `/resume` | Reads vault → auto-runs Bloodhound → Compass → Sentinel |
| `/bloodhound` | Full standalone bug scan |
| `/compass` | Full standalone sprint planning |
| `/sentinel` | Full standalone scaling + speed audit |
| `/forge` | Full standalone hardware compatibility audit |
| `/wrap-up` | Updates vault → creates handoff → closes session |

## Agent Files
- [bloodhound.md](file:///home/askme/Pictures/wolf-lang/.wolf-vault/Agents/bloodhound.md)
- [compass.md](file:///home/askme/Pictures/wolf-lang/.wolf-vault/Agents/compass.md)
- [sentinel.md](file:///home/askme/Pictures/wolf-lang/.wolf-vault/Agents/sentinel.md)
- [forge.md](file:///home/askme/Pictures/wolf-lang/.wolf-vault/Agents/forge.md)
