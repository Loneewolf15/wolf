# Sprint Brief — 2026-04-11 (Compass)

## 🧭 Compass Verdict
**GO** on the next Phase 4 task: **Closures / First-class functions**.

## 1. Selected Task: Closures & First-class functions
**Phase:** 4 (Language Completeness)
**Priority:** High (Foundation for Generics and Async mapping)

### Why this task?
The `TryCatch` error unwinding structure introduced robust local LLVM block branching and basic block state management. With Enums and Errors complete, the next core language primitive blocking the broader ecosystem (including advanced array functions like `array_map` and the `channel` pub/sub layer) is functional closures.

### Implementation Guidelines
1. **Parser/Lexer:** The lexer already contains preliminary support for closures (`Lexer` tokenization pass). We need the parser to generate an `ir.FuncLit` node dynamically and emit a capture struct.
2. **Emitter (`llvm_emitter.go`):** Update the `emitFuncLit` function (currently used internally by `@supervise`) to support true environment capturing. When a closure escapes, bounded variables must be boxed.
3. **C Runtime (`wolf_runtime.c`):** Implement the `wolf_closure_t` struct wrapper which encapsulates the function pointer + an environment array pointer `void** env`. 

## 2. Alternatives Considered
- **Pattern Matching (`match`):** High value, but closures are a more fundamental building block and prerequisite for higher-order standard library functions.
- **Interfaces / traits:** Requires substantially more architectural planning regarding vtables vs duck-typing. First-class functions must be handled first.

## 3. Sentinel & Forge Pre-Clearance
- **Sentinel:** Ensure the `wolf_closure_t` allocation uses the thread-local request arena (`wolf_req_alloc`), strictly forbidding global heap `malloc()` to preserve execution speed and request-bounded teardown.
- **Forge:** No OS-level constraints. Fully compatible with `WOLF_FREESTANDING`.
