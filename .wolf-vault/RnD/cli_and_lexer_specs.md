# Technical Spec: CLI & Lexer Enhancements

## 1. Multi-Style Comments (Lexer)
Wolf now supports 3 styles of comments, increasing compatibility with PHP, JS, and C developers.

### Syntax:
*   **Hash**: `# This is a comment` (Legacy/PHP-style)
*   **Double-Slash**: `// This is a comment` (C++/JS-style)
*   **Block**: `/* This is a multi-line comment */` (C-style)

### Implementation (internal/lexer/lexer.go):
- **Single-line**: `scanComment()` consumes characters until `\n` or `EOF`.
- **Block**: `scanMultiLineComment()` consumes until `*/` or `EOF` (with line/column tracking for error reporting).

---

## 2. Native CLI Argument Support
Wolf programs compiled to native binaries can now access command-line arguments directly.

### Syntax (Wolf):
```wolf
$count = argc()
$program_name = argv(0)
$first_arg = argv(1)
```

### LLVM Emitter (internal/emitter/llvm_emitter.go):
The entry point is generated for top-level code:
```llvm
define i32 @main(i32 %argc, ptr %argv) {
entry:
  call void @wolf_init_args(i32 %argc, ptr %argv)
  ; ... top level code calls @wolf_main ...
}
```

**Function Prefixing Architecture**:
To avoid naming collisions with C standard libraries and the entry point itself, ALL Wolf-defined global functions and methods are prefixed with `wolf_` in the generated LLVM IR:
- `func main()` becomes `define void @wolf_main()`
- `func my_handler()` becomes `define void @wolf_my_handler()`
- `class User { func save() }` becomes `define void @wolf_User_save()`

These are resolved during the LLVM emission phase in `emitter/llvm_emitter.go`.

---

## 2. Native CLI Support
Implemented `argc()` and `argv(index)` built-ins, allowing Wolf programs to run as terminal tools or background services.
- `wolf_argc()`: Returns `int64_t`.
- `wolf_argv(int64_t index)`: Returns `const char*` with bounds checking.

---

## 4. Smart Scaffolding (`wolf new`)
The `wolf new` command now supports mode-based project generation.

### Syntax:
```bash
# Create a minimal script
wolf new my_tool --type=script

# Create a full MVC API
wolf new my_api --type=api
```

### Features:
- **Script Mode**: Minimal directory structure, `src/main.wolf`.
- **API Mode**: Comprehensive MVC structure (`controllers/`, `models/`, `libraries/`), `docker-compose.yml`, and base `Controller` class.
- **Config**: Automatic generation of `wolf.config.json` with appropriate defaults.

---

## 3. Regular Expression Completion
Fully implemented the `preg_*` family using PCRE2 for high performance.
- Supports `preg_match`, `preg_match_all`, `preg_replace`, `preg_split`.
- Uses thread-local results for efficiency.

**Status**: Verified in URL Shortener POC. 🐺
