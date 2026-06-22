cat << 'INNER_EOF' >> runtime/wolf_http_engine.c

/* Helper to execute longjmp for OOM panics */
void wolf_engine_longjmp_oom(void) {
    WolfConnCtx* ctx = __atomic_load_n(&wolf_active_ctx, __ATOMIC_ACQUIRE);
    if (ctx) {
        longjmp(ctx->oom_jump, 1);
    }
}
INNER_EOF
