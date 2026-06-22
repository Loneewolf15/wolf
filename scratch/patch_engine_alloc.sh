sed -i 's/if (wolf_active_ctx) { longjmp(wolf_active_ctx->oom_jump, 1); }/wolf_panic_oom();/' runtime/wolf_http_engine.c
sed -i 's/if (wolf_active_ctx) longjmp(wolf_active_ctx->oom_jump, 1);/wolf_panic_oom();/' runtime/wolf_http_engine.c
