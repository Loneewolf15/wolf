# Model Routing Reference

## Table of Contents
1. [Routing Decision Framework](#routing-decision-framework)
2. [PRD Signals — Gemini](#prd-signals--gemini)
3. [PRD Signals — Claude](#prd-signals--claude)
4. [Phase Routing Table](#phase-routing-table)
5. [Conflict Resolution](#conflict-resolution)
6. [Routing Plan Template](#routing-plan-template)

---

## Routing Decision Framework

Read the PRD and look for the signals below. Each phase of the lang-builder workflow has a **default model** but signals from the PRD can override it.

**Core principle:** Gemini leads when breadth, exploration, or multi-modal context is needed. Claude leads when depth, precision, and code correctness are paramount.

---

## PRD Signals — Gemini

Route to **Gemini** when the PRD mentions or implies:

| Signal | Examples |
|--------|---------|
| Research / survey | "inspired by Rust, Python, Lua", "compare type systems", "based on prior art" |
| Visual / diagram output | architecture diagrams, syntax railway diagrams, flowcharts |
| Open-ended exploration | "not sure about paradigm yet", "explore options", "brainstorm features" |
| Multi-modal input | design mockups attached, hand-drawn grammar sketches |
| Long context needs | large existing codebase to analyze for DSL embedding |
| Ecosystem design | package manager, toolchain integration, IDE support strategy |
| Target audience research | "designed for non-programmers", "domain experts", "students" |
| Performance benchmarking | comparing VM strategies, GC approaches across languages |

---

## PRD Signals — Claude

Route to **Claude** when the PRD mentions or implies:

| Signal | Examples |
|--------|---------|
| Formal grammar | EBNF, ANTLR, PEG, BNF, CFG |
| Precise code generation | "write a lexer in Python", "implement parser in Rust" |
| Type system precision | "Hindley-Milner", "structural typing", "gradual typing" |
| Compiler phases | "emit LLVM IR", "generate bytecode", "translate to C" |
| Test-driven correctness | comprehensive edge case coverage, property-based tests |
| Step-by-step spec | evaluation rules, reduction semantics, operational semantics |
| Small/deterministic output | single-file interpreter, embedded DSL in <500 LOC |
| Security/safety requirements | sandboxing, memory safety, capability model |

---

## Phase Routing Table

| Phase | Default Model | Override to Gemini if… | Override to Claude if… |
|-------|--------------|------------------------|------------------------|
| 0 – PRD Intake | Either | PRD is missing / vague | PRD is detailed and technical |
| 1 – Language Design | **Gemini** | Always Gemini unless user disagrees | User has exact design locked in |
| 2 – Formal Spec | **Claude** | Grammar is intentionally informal | — |
| 3a – Lexer/Parser | **Claude** | — | — (always Claude) |
| 3b – AST/Evaluator | **Claude** | — | — (always Claude) |
| 3c – Compiler backend | **Claude** | — | — (always Claude) |
| 4 – Tooling/Ecosystem | **Gemini** | — | Tooling code required now |
| 5 – Docs/Packaging | **Claude** | Narrative/marketing docs | Technical reference docs |

---

## Conflict Resolution

When PRD signals are mixed (e.g., both research and precise grammar):

1. **Split the phase** — run Gemini for the exploratory sub-task, hand off output to Claude for the precise sub-task.
2. **Prefer Claude** for any sub-task where incorrect output causes downstream bugs (grammar errors compound through all later phases).
3. **Prefer Gemini** when the user explicitly says "just explore" or "I'm not sure yet."
4. Document the split in the Routing Plan so the user can override.

---

## Routing Plan Template

Output this table at the end of Phase 0 before beginning work:

```
## Routing Plan

| Phase | Model | Justification |
|-------|-------|---------------|
| 1 – Language Design | Gemini | PRD is exploratory; paradigm not yet decided |
| 2 – Formal Spec | Claude | Grammar must be precise for parser generation |
| 3 – Implementation | Claude | Correctness-critical code generation |
| 4 – Tooling | Gemini | Ecosystem survey and REPL design brainstorm |
| 5 – Docs | Claude | Technical reference docs required |

> ⚠️ Override any phase by telling me which model you prefer.
```
