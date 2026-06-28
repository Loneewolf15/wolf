## AXIOM Session — 2026-06-27
**Domain probed:** Memory Management
**Question asked:** `wolf_arena_acquire()` returns an overflow arena when the pool is exhausted, and the overflow arena is `malloc`'d and tagged `is_overflow=1`. Since `wolf_ctx_free_stack` uses an O(1) index-based free list targeting fixed indices `0` to `WOLF_CORE_CTX_MAX`, how does `wolf_core_free_ctx` safely recycle or free an arbitrary overflow context without corrupting the fixed-size `wolf_ctx_free_stack` array index boundary?
**Answer given:** (Waiting for user response)
**AXIOM verdict:** TBD
**Risk logged:** Yes — Potential out-of-bounds array write or memory leak on overflow context recycling.
**Next probe:** TBD
