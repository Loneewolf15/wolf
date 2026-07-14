import re

with open("runtime/wolf_http_engine.c", "r") as f:
    content = f.read()

# Add read_buf and bytes_in and core to WolfConnCtx
ctx_struct_match = re.search(r'typedef struct \{[\s\S]*?\} WolfConnCtx;', content)
if ctx_struct_match:
    struct_str = ctx_struct_match.group(0)
    struct_mod = struct_str.replace("WolfArena* arena;", "WolfArena* arena;\n    char* read_buf;\n    ssize_t bytes_in;\n    struct WolfCore* core;")
    content = content.replace(struct_str, struct_mod)

# Add args to WolfCore
core_struct_match = re.search(r'typedef struct WolfCore \{[\s\S]*?\} WolfCore;', content)
if core_struct_match:
    struct_str = core_struct_match.group(0)
    struct_mod = struct_str.replace("volatile int    ready;", "volatile int    ready;\n    void*           args;")
    content = content.replace(struct_str, struct_mod)

with open("runtime/wolf_http_engine.c", "w") as f:
    f.write(content)
