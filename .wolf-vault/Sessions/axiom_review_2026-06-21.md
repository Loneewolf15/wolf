## AXIOM Session — 2026-06-21
**Domain probed:** Memory Management
**Question asked:** `wolf_arena_alloc` falls back to `calloc` for oversized allocations and tracks them in an array of size 64. If a request requires a 65th oversized allocation, the allocator immediately frees the pointer and returns `NULL`, which will silently trigger segfaults in the LLVM-compiled code if it assumes non-nullable strings. Why is the overflow limit hardcoded to 64 without a reallocating dynamic array or a graceful HTTP 500 rejection mechanism?
**Answer given:** Intrusive linked list + per-context byte quota, or alternatively geometric block-growth strategy (`std::pmr` style) to keep O(1) bump allocation even for large requests.
**AXIOM verdict:** SOUND. Geometric block-growth paired with a hard per-request ceiling is the architecturally correct choice for Wolf. It bounds the `malloc` calls to `O(log N)` while preserving the bump-pointer fast path.
**Next probe:** [Session Complete]
