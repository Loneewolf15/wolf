# 🐺 Wolf Language — Project Vault

> **The persistent brain for the Wolf compiler project.**
> Every session reads from this vault. Every session writes back to it.
> Built to last until Wolf compiles Wolf.

## Vision

Wolf is a compiled programming language with PHP-like syntax that compiles to **native machine code via LLVM IR**.

**Target:** 100X faster than PHP, with the safety of Rust and simplicity of Go.
**Timeline:**
- **Phase 1 (March 2026):** Production Hardening ✅ (Static linking, Graceful shutdown, DB pooling, CLI mode)
- **Phase 2 (Current):** Stdlib Expansion (Regex, Arrays, HTTP Client)
- **Year 1:** Wolf compiler self-hosted

## Vault Structure

```
.wolf-vault/
├── README.md                  ← You are here
├── RnD/                       ← Architecture decisions, benchmarks, research
├── Product/                   ← Feature specs, roadmap, API contracts
├── Marketing/                 ← Content strategy, messaging, positioning
├── Legal/                     ← Licenses, compliance, IP
├── Execution/                 ← Sprint plans, dependency graphs, handoffs
└── Sessions/                  ← Per-session handoff notes
```

## Session Protocol

| Command | When | What it does |
|---------|------|-------------|
| `/resume` | Start of every session | Reads execution plan + last handoff notes |
| `/wrap-up` | End of every session | Updates all departments + creates handoff for next session |

**Workflow files:** `.agents/workflows/resume.md` and `.agents/workflows/wrap-up.md`
