---
name: lang-builder
description: >-
  A programming language design and implementation helper. Use this skill when
  a user wants to design, spec, prototype, or build a custom programming
  language, DSL (domain-specific language), or interpreter/compiler. It
  analyzes the language PRD (Product Requirements Document) to determine which
  AI model should lead each phase - Gemini handles research-heavy, multi-modal,
  and architecture-exploration phases; Claude handles precise code generation,
  grammar specification, and compiler/interpreter implementation. Triggers on
  requests like "help me build a language", "design a DSL for X",
  "write a lexer/parser", "create a programming language",
  or "help me make an interpreter/compiler."
---

# Lang Builder Skill

A specialized workflow for designing and building programming languages (general-purpose or DSL). It ingests a PRD and routes tasks intelligently between **Gemini** and **Claude** models.

## Model Routing

Read [`references/model-routing.md`](references/model-routing.md) to understand how to classify PRD requirements and select the right model for each task. Do this **before** beginning any phase.

**Quick summary:**
- **Gemini** → exploration, research, architecture brainstorming, multi-modal diagrams, ecosystem survey
- **Claude** → grammar writing (EBNF/ANTLR/PEG), precise code (lexer, parser, AST, interpreter, compiler), type system, test suites

## Workflow

### Phase 0 — PRD Intake & Model Routing

1. Ask the user for their PRD (or help draft one if missing). See [`references/lang-concepts.md`](references/lang-concepts.md) for PRD sections to prompt for.
2. Analyze the PRD using the routing rules in `references/model-routing.md`.
3. Output a **Routing Plan**: a table mapping each phase to a model with a one-line justification.
4. Confirm the plan with the user before proceeding.

### Phase 1 — Language Design (Gemini-led)

- Define language goals, target audience, paradigm (functional, OOP, procedural, etc.)
- Survey similar languages; identify what to borrow and what to avoid
- Sketch high-level syntax and semantic goals
- Produce a **Language Design Doc** (features, non-goals, example programs)

### Phase 2 — Formal Specification (Claude-led)

- Write the formal grammar (EBNF or ANTLR4 `.g4` format preferred)
- Define the type system and scoping rules
- Specify standard library primitives
- Produce a `grammar.g4` (or equivalent) and a `spec.md`

### Phase 3 — Implementation (Claude-led)

Follow the order: **Lexer → Parser → AST → Evaluator/Interpreter or Compiler backend**

Refer to [`references/lang-concepts.md`](references/lang-concepts.md) for implementation patterns per paradigm.

For each component:
1. Generate the code in the user's chosen implementation language (Python, Rust, Go, etc.)
2. Write unit tests alongside the component
3. Validate with the user's example programs from Phase 1

### Phase 4 — Tooling & Ecosystem (Gemini-led)

- REPL design
- Error message strategy (human-friendly)
- LSP (Language Server Protocol) feasibility
- Package manager / module system sketch

### Phase 5 — Documentation & Packaging

- Write a `README.md` for the language repo
- Write a "Getting Started" tutorial using example programs
- Output a final deliverables checklist

## Key Rules

- **Always route first.** Never skip Phase 0 model routing step.
- **One phase at a time.** Confirm the output of each phase before starting the next.
- **Keep grammar files separate.** Never embed grammar inline in prose docs; always write to a dedicated `.g4` or `grammar.ebnf` file.
- **Tests are not optional.** Every generated code component must have accompanying tests.
- If a PRD is missing key sections, prompt the user before proceeding.
