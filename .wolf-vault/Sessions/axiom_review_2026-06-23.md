## AXIOM Session — 2026-06-23

**Domain probed:** Compiler / Type System
**Question asked:** When AutoDiscover returns multiple ASTs from different files, and you call Resolver->Resolve() on each sequentially, the Resolver's hoistTopLevel runs per-file — so when resolving Lexer.wolf, it doesn't know Token exists yet. How does your naive multi-file resolution avoid "undefined class" errors?
**Answer given:** Correctly identified: naive per-file resolution WILL break. The fix is to merge ALL discovered AST statements into a single flattened Program before calling Resolve() once. This mirrors the Go bootstrap exactly (compiler.go:113).
**AXIOM verdict:** CONDITIONALLY SOUND — Resolved correctly by implementation. The merged-program pattern is the right approach.
**Risk logged:** No outstanding risks from this session.
**Next probe:** Does wolf_file_list_dir return entries in sorted or OS-readdir order? Non-deterministic order doesn't break correctness (all files discovered before resolution) but affects error-message reproducibility across builds.
=== AXIOM End-to-End Run ===
Native binary executed. Result: Hit PARSE ERROR on 'enum' in Token.wolf.
Conclusion: AutoDiscover works natively, but Parser.wolf is missing parseEnum support, which exists in Go bootstrap.
