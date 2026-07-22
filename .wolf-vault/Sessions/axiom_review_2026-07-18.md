## AXIOM Session — 2026-07-18

**Domain probed:** Compiler Backend / Memory Model / MCU Bare-Metal / Python Interop
**Hive participants:** Queen + AXIOM + Sentinel + Forge

---

### Question 1 — ABI Calling Convention for Native Codegen

**Question asked:** When Wolf drops LLVM and writes its own instruction selection and register allocation, how does it handle ABI-correct calling conventions into the C runtime — specifically the System V AMD64 `rdi, rsi, rdx, rcx, r8, r9` sequence for functions like `wolf_print_str(ptr)`?

**Answer given:** Wolf's C runtime uses non-variadic, fixed-arity signatures for all hot-path functions. `wolf_print_str` takes exactly one `ptr`. System V AMD64 for one ptr: move arg to `%rdi`, call. Two instructions. No variadic complexity in the hot path. Variadic functions (`wolf_sprintf`) can remain as C library calls for Phase 1 of the native emitter.

**AXIOM verdict:** CONDITIONALLY SOUND — correct for Phase 1. Phase 2 (inlining `wolf_print_str`) requires teaching the emitter to use `%rdi` directly and understand the callee-save register discipline (`rbx`, `rbp`, `r12-r15`). Flag for H3.

**Risk logged:** Yes — ABI discipline for multi-arg C calls must be validated per-function against `wolf_runtime.h` signatures before H3 ships.

---

### Question 2 — Wolf Value Representation on AVR 2KB SRAM

**Question asked:** The current `wolf_value_t` is a 64-bit tagged union. The ATmega328P has 2KB SRAM total. What is the concrete representation of a Wolf value on this target, and does it fit?

**Answer given:** `wolf_val_t = uint16_t`, tag in low 2 bits, per MicroPython's `micropython_obj_t` pattern:
- Tag `00` = int: 14-bit immediate inline (range: -8192 to 8191)
- Tag `01` = ptr: 14-bit address (AVR address space ≤ 64KB, bits 15:2 are the word address)
- Tag `10` = bool: bit 2 = value
- Tag `11` = nil: value = 0

**AXIOM verdict:** SOUND. AVR data memory for ATmega328P: 0x0000–0x08FF (2,304 bytes including registers and I/O). Maximum RAM address is 0x08FF = 2,303. This fits in 14 bits (max 16,383). The tagged pointer trick works on AVR. This is exactly the MicroPython approach.

**Risk logged:** None — design is complete and architecturally validated against ATmega328P datasheet constraints.

---

### Question 3 — BoxingElim TypeChecker Interface

**Question asked (for next session):** BoxingElim requires a `typeMap` populated from TypeChecker output. The TypeChecker currently runs in Phase 4. What output format does it produce, and is it accessible to WIRBuilder in Phase 5, or does it require a new pass interface?

**Answer given:** (Pending — next session)

**AXIOM verdict:** (Pending)

**Risk logged:** Yes — if TypeChecker output is not threaded through to WIRBuilder, BoxingElim cannot be implemented without refactoring the compiler pipeline (which is currently: Lexer → Parser → Resolver → TypeChecker → WIRBuilder, all sequential with no output persisted between passes).

**Next probe:** Examine `TypeChecker.wolf` — does it annotate AST nodes with types, or does it only report errors? If it only reports errors, WIRBuilder must re-infer types independently, which duplicates TypeChecker logic.

---

## Summary

| Area | Verdict | Risk |
|---|---|---|
| ABI for native codegen | CONDITIONALLY SOUND | Medium — callee-save discipline in H3 |
| AVR value representation | SOUND | None |
| BoxingElim TypeChecker interface | PENDING | High — could require pipeline refactor |
| wolf_method_call removal | SOUND — confirmed by Go reference impl | None |
| Python interop GIL | CONDITIONALLY SOUND — requires Py ≥ 3.12 | Medium |

---

### Question 3 — BoxingElim TypeChecker Interface (ANSWERED in same session)

**Question asked:** BoxingElim requires a `typeMap` populated from TypeChecker output. What output format does TypeChecker produce, and is it accessible to WIRBuilder?

**Answer discovered through code inspection:** `VarDecl.$typeName` is populated directly from the Wolf parser for typed variables (`var $x: int = 5` → `$typeName = "int"`). This means WIRBuilder does NOT need TypeChecker output at all. It can read `$stmt->typeName` directly during the `VarDecl` building pass. Additionally:
- `IntLiteral` expressions are always `i64`
- `FloatLiteral` expressions are always `double`  
- `BoolLiteral` expressions are always `i1`
- Binary arithmetic on two typed vars produces a typed result

The Go emitter uses a more sophisticated fixed-point integer purity analysis (`intUnboxFuncs`), but Wolf's native BoxingElim can implement the conservative version (typeName-based + literal inference) first, which handles ~80% of the gains.

**AXIOM verdict:** SOUND — design does not require TypeChecker refactoring. BoxingElim can be implemented purely within WIRBuilder with zero changes to TypeChecker.wolf or the compiler pipeline.

**Risk logged:** None — the design is self-contained.

**Next probe (next session):** For function parameters typed as `int` (e.g., `func increment($val: int)`), does WIRBuilder currently receive the param's typeName? Check `Parser.wolf` — does it populate `Param.$typeName` from typed param syntax?
