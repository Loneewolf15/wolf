import re
with open("wolf_http_engine.c", "r") as f:
    text = f.read()

replacement = r"""
    int fd = open("/tmp/crash.log", O_WRONLY | O_CREAT | O_APPEND, 0666);
    if (fd >= 0) {
        dprintf(fd, "[WOLF-CRASH] Caught SIGSEGV (Segmentation Fault)\n");
        dprintf(fd, "  -> Request ID:   %d\n", ctx->request_id);
        dprintf(fd, "  -> Endpoint:     %s\n", ctx->path ? ctx->path : "unknown");
        dprintf(fd, "  -> Arena Usage:  %zu bytes (cap: %zu)\n", ctx->arena_used, ctx->arena_cap);
        dprintf(fd, "  -> OOM Triggered: %s\n", ctx->oom_triggered ? "true" : "false");
        
        void *buffer[100];
        int nptrs = backtrace(buffer, 100);
        backtrace_symbols_fd(buffer, nptrs, fd);
        close(fd);
    }
    
    // reset default handler and raise
"""
text = re.sub(r'printf\("\[WOLF-CRASH\] Caught SIGSEGV.*?\n"\);.*?backtrace_symbols_fd.*?\}', replacement, text, flags=re.DOTALL)
with open("wolf_http_engine.c", "w") as f:
    f.write(text)
