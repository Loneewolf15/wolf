# Handoff — 2026-05-02

## Where We Left Off

We fixed **BUG-050 (P0 SIGSEGV)** in the Package System — the first full session on autodiscovery/namespace support.

### What was done
The root cause of the SIGSEGV was a **double namespace mangling** in `parser.go`:
- `parseFuncDecl()` applies `p.namespace + "_"` prefix to ALL identifiers inside the file, including methods inside a class body.
- This produced method names like `Dummy_get` instead of `get`.
- Then `llvm_emitter.go` built `cls.Name + "_" + method.Name` = `Dummy_Api_Dummy_get` — wrong.
- The `funcSigs` lookup at dispatch time looked for `Dummy_Api_get` → miss → fell through to `methodDispatch["get"]` = `wolf_qb_get` → SIGSEGV (query builder got a class object pointer).

**Fix (1 file, 2 hunks):** `internal/parser/parser.go` `parseClassDecl()`:
- Save `p.namespace` before entering class body
- Clear `p.namespace = ""` during method parsing
- Restore `p.namespace = savedNamespace` after `}`

### Commits This Session
```
51cfccf (HEAD -> main) fix(parser): suppress namespace prefix for class methods to prevent double-mangling
8c4fcce fix(emitter): isolate varClass per function for method dispatch
c62e69e feat(lexer,parser): add 'protected' visibility keyword and roadmap update
```

### Test Status
- `./wolf run e2e/testdata/44_package_system.wolf` → prints `Dummy Data` ✅
- `./wolf build e2e/testdata/43_visibility.wolf` → prints `Generic\n5` ✅
- `go test ./internal/...` → all green ✅ (compiler, emitter, lexer, parser, resolver, typechecker)

## Next Immediate Task

**BUG-049 (P1) — Fix inherited method dispatch for child classes**

`$d->bark()` (where `Dog extends Animal` and `bark()` calls `$this->makeSound()`) is silent — the inherited `makeSound()` is not being dispatched.

The issue: when emitting `Dog`'s `bark()` method body, `$this->makeSound()` looks up `varClass["this"] = "Dog"` → builds `directName = "Dog_makeSound"` → not in `funcSigs` (it's `Animal_makeSound`) → falls through to suffix search or no-op.

**Key files:**
- `internal/emitter/llvm_emitter.go` — `emitMethodCall` (around line 3476)
- `internal/emitter/ir_emitter.go` — `emitClass` (line 586, fn.Receiver = c.Name)

**Fix approach:** When `directName` is not found in `funcSigs` and `fn.Receiver` is set, walk the class inheritance chain (stored in `cls.Extends`) to find the method in a parent class's `funcSigs`.

**After BUG-049:** `wolf_dns_lookup` 2s timeout (P0 roadmap).

## Open Issues / Watch Out For

1. **`wolf___compiler_create_model`** — the dispatch function in the generated IR uses a string-compare chain (`strcmp`) keyed by `"Dummy_Api"`. As more packages are added, this needs to be a dynamic hash dispatch, not hardcoded compares. Not urgent for now.

2. **E2E full suite** — `go test ./e2e/...` still hangs/times out on local due to HTTP server test port conflicts. Use `./wolf run <file>` or `go test ./internal/...` for iteration.

3. **`p.namespace` restoration** — the fix restores namespace after class body, but nested class declarations (classes inside classes) are not a Wolf feature yet. If they ever are, this logic must be recursive.

## Relevant Files Modified This Session
- `internal/parser/parser.go` — BUG-050 fix (parseClassDecl namespace suppression)
- `e2e/testdata/44_package_system.out` — expected output file added
- `.wolf-vault/Execution/plan.md` — Package System marked Done, next tasks updated
- `.wolf-vault/RnD/bugs_fixed.md` — BUG-050 entry + ledger updated
- `.wolf-vault/Sessions/latest_handoff.md` — this file