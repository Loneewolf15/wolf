import re

with open("runtime/wolf_http_engine.c", "r") as f:
    content = f.read()

# Modify WolfConnCtx
ctx_mod = """    WolfArena* arena;

    /* io_uring async state */
    char* read_buf;
    ssize_t bytes_in;
    struct WolfCore* core;
"""
content = re.sub(r'    WolfArena\* arena;\n', ctx_mod, content)

# Modify WolfCore
core_mod = """    volatile int    ready;           /* set to 1 when thread enters its main loop */
    void*           args;            /* original WolfCoreArgs */
"""
content = re.sub(r'    volatile int    ready;           /\* set to 1 when thread enters its main loop \*/\n', core_mod, content)

with open("runtime/wolf_http_engine.c", "w") as f:
    f.write(content)
