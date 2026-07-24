package explain

// pattern defines a single error recognition rule with a priority-ordered
// keyword list and a structured explanation template.
type pattern struct {
	// code is the Wolf-specific error code (e.g., "W-E001").
	code string
	// keywords is a list of substrings to match against the raw error (case-insensitive).
	// The first keyword match wins — list them most-specific first.
	keywords []string
	// summary is one sentence describing the problem.
	summary string
	// detail explains why the error occurs.
	detail string
	// fix is an actionable suggestion.
	fix string
	// example shows corrected code (optional).
	example string
}

// allPatterns returns the complete built-in Wolf error pattern database,
// ordered from most-specific to most-generic within each phase group.
// The Explainer iterates patterns in order; the first keyword match wins.
func allPatterns() []pattern {
	return []pattern{

		// ── Lexer / Parse Phase ──────────────────────────────────────────────

		{
			code:     "W-E001",
			keywords: []string{"unexpected token", "unexpected character"},
			summary:  "The lexer encountered a character or token it does not recognise.",
			detail:   "Wolf's lexer found a character sequence that does not belong to the Wolf grammar. Common causes: stray punctuation, an unsupported operator, or copy-pasted code from another language (e.g. Python's ':' block syntax or JS's '=>' arrow).",
			fix:      "Check the file and line number above. Remove or replace the unrecognised character. Use 'wolf check <file.wolf>' for quick diagnostics.",
			example:  "// ❌ Stray Python-style colon\nif $x > 0:\n    echo $x\n\n// ✅ Wolf syntax\nif ($x > 0) {\n    echo $x\n}",
		},
		{
			code:     "W-E002",
			keywords: []string{"unterminated string", "unclosed string"},
			summary:  "A string literal was opened but never closed.",
			detail:   "The lexer reached end-of-file (or end-of-line) while still inside a string literal. Every '\"' must have a matching closing '\"' on the same line.",
			fix:      "Add the missing closing '\"' to the string literal. If you need a multi-line string, use string concatenation with '..'.",
			example:  "// ❌ Missing closing quote\n$name = \"Alice\n\n// ✅\n$name = \"Alice\"",
		},
		{
			code:     "W-E003",
			keywords: []string{"expected '}'", "expected }", "missing closing brace"},
			summary:  "A block was opened with '{' but never closed with '}'.",
			detail:   "Wolf requires every opening brace '{' to have a matching closing brace '}'. This is often a missing brace at the end of a function, class, or if-block.",
			fix:      "Add the missing '}' to close the block. Use an editor with brace-matching highlighting to find the mismatch quickly.",
		},
		{
			code:     "W-E004",
			keywords: []string{"expected '('", "expected (", "missing opening paren"},
			summary:  "A function call or control-flow statement is missing its opening '('.",
			detail:   "Wolf requires parentheses around condition expressions and function call argument lists. This differs from some languages that allow bare expressions.",
			fix:      "Add '(' and ')' around the expression.",
			example:  "// ❌\nif $x > 0 {\n\n// ✅\nif ($x > 0) {",
		},
		{
			code:     "W-E005",
			keywords: []string{"expected ';'", "expected semicolon", "missing semicolon"},
			summary:  "A statement is missing its terminating semicolon.",
			detail:   "Wolf requires semicolons at the end of each statement. Automatic semicolon insertion is not performed.",
			fix:      "Add ';' at the end of the statement.",
			example:  "// ❌\n$x = 1\n\n// ✅\n$x = 1;",
		},
		{
			code:     "W-E006",
			keywords: []string{"parse error", "syntax error"},
			summary:  "The parser encountered a syntax it cannot recognise.",
			detail:   "Wolf's parser found a construct that does not match any production rule in the Wolf grammar. This is often a typo, missing keyword, or an unsupported language feature.",
			fix:      "Review the line and column indicated. Consult the Wolf language reference at https://wolflang.dev/docs/syntax.",
		},
		{
			code:     "W-E007",
			keywords: []string{"expected class name", "invalid class name"},
			summary:  "A class declaration has an invalid or missing name.",
			detail:   "Class names in Wolf must start with an uppercase letter and contain only alphanumeric characters. Reserved words cannot be used as class names.",
			fix:      "Rename the class to start with an uppercase letter (e.g., 'UserController', not 'userController').",
			example:  "// ❌\nclass userController {}\n\n// ✅\nclass UserController {}",
		},
		{
			code:     "W-E008",
			keywords: []string{"enum", "expected enum value", "invalid enum"},
			summary:  "An enum declaration or enum access has an invalid syntax.",
			detail:   "Wolf enums must be declared with 'enum Name { VARIANT1, VARIANT2 }' and accessed with 'Name::VARIANT'. Check that enum names are uppercase and variants are comma-separated.",
			fix:      "Ensure the enum uses the correct declaration syntax and that all variants are comma-separated.",
			example:  "// ✅\nenum Status { ACTIVE, INACTIVE, PENDING }\n$s = Status::ACTIVE;",
		},

		// ── Resolver Phase ───────────────────────────────────────────────────

		{
			code:     "W-E010",
			keywords: []string{"undefined variable", "undeclared variable", "unknown variable"},
			summary:  "A variable was used before it was declared.",
			detail:   "Wolf requires all variables to be declared with 'var $name = ...' before use. Variables in Wolf are prefixed with '$'. Accessing a variable that has not been declared in the current scope produces this error.",
			fix:      "Declare the variable with 'var $name = <value>;' before using it. Check for typos in the variable name (Wolf variable names are case-sensitive).",
			example:  "// ❌\necho $message;\n\n// ✅\nvar $message = \"Hello\";\necho $message;",
		},
		{
			code:     "W-E011",
			keywords: []string{"undefined function", "unknown function", "undeclared function"},
			summary:  "A function was called that has not been declared.",
			detail:   "Wolf's resolver could not find a function with this name in the current scope, any parent scope, or the Wolf stdlib. Check for typos. Wolf stdlib functions do not require an import — if you are calling a stdlib function, verify the exact name at https://wolflang.dev/docs/stdlib.",
			fix:      "Declare the function with 'func name($arg) { ... }' before calling it, or fix the typo in the function name.",
		},
		{
			code:     "W-E012",
			keywords: []string{"undefined class", "unknown class", "undeclared class"},
			summary:  "A class was instantiated or referenced that has not been declared.",
			detail:   "Wolf could not find a class with this name. Either the class file was not included in the project (use '--project <dir>' to auto-discover), or the class name is misspelled.",
			fix:      "Ensure the class is declared in a file reachable from your project root. Run 'wolf build --project <dir>' for multi-file projects.",
		},
		{
			code:     "W-E013",
			keywords: []string{"undefined method", "unknown method", "no method"},
			summary:  "A method was called on an object whose class does not define it.",
			detail:   "The object's class exists but the method name was not found in its definition. This may be a typo or the method may be defined in a parent class that is not yet resolved.",
			fix:      "Check the method name for typos. Verify the class definition includes the method. For inherited methods, ensure the parent class is available.",
		},
		{
			code:     "W-E014",
			keywords: []string{"undefined property", "unknown property", "no property"},
			summary:  "An object property was accessed that has not been declared.",
			detail:   "Wolf classes require all properties to be declared at the class level with 'public $prop = default;'. Dynamically-created properties are not supported.",
			fix:      "Add 'public $propertyName = <default>;' to the class declaration.",
			example:  "// ✅\nclass User {\n    public $name = \"\"\n    public $age  = 0\n}",
		},

		// ── Type Check Phase ─────────────────────────────────────────────────

		{
			code:     "W-E020",
			keywords: []string{"type mismatch", "type error", "cannot assign"},
			summary:  "A value of the wrong type was assigned or passed to a function.",
			detail:   "Wolf's type checker detected that a value's inferred type does not match the expected type at this site. Wolf uses structural type inference — if a function expects a string and receives an integer, this error fires.",
			fix:      "Cast the value explicitly using 'strval()', 'intval()', or 'floatval()', or adjust the function signature to accept the correct type.",
		},
		{
			code:     "W-E021",
			keywords: []string{"cannot compare", "comparison between"},
			summary:  "Two incompatible types are being compared.",
			detail:   "Wolf does not allow direct comparison between certain types (e.g., a map and a string). Comparing a map or class instance with '==' often indicates a logic error.",
			fix:      "Compare specific properties or keys instead of the entire object. For strings, use 'Strings::equals($a, $b)'.",
		},
		{
			code:     "W-E022",
			keywords: []string{"null reference", "nil dereference", "nil pointer"},
			summary:  "A nil/null value was dereferenced.",
			detail:   "An operation was performed on a value that is null/nil. Wolf does not use pointer semantics but function return values can be empty (e.g., map lookups for missing keys return empty string).",
			fix:      "Add a nil/empty check before dereferencing. Use 'if ($val != \"\") { ... }' for string results or 'if ($map[\"key\"] != \"\") { ... }' for map lookups.",
		},

		// ── LLVM Emission Phase ───────────────────────────────────────────────

		{
			code:     "W-E030",
			keywords: []string{"llvm error", "llvm ir error", "invalid llvm"},
			summary:  "The LLVM backend encountered an invalid IR construct.",
			detail:   "The Wolf LLVM emitter produced IR that LLVM's verifier rejected. This is almost always a Wolf compiler bug rather than a user code error.",
			fix:      "1. Run 'wolf build <file.wolf> --dump-wir' to inspect the WolfIR before LLVM lowering.\n2. Check the error message for the specific LLVM instruction that failed.\n3. File a bug at https://wolf/issues with a minimal reproduction.",
		},
		{
			code:     "W-E031",
			keywords: []string{"undefined reference", "undefined symbol", "linker error", "ld error"},
			summary:  "The linker could not find a referenced symbol.",
			detail:   "The compiled Wolf object references a symbol (function or variable) that was not found in the linked libraries or object files. Common causes: missing C runtime, missing libcurl, or missing libsodium.",
			fix:      "Ensure the Wolf C runtime is compiled and linked. Check that all required system libraries (libcurl, libsodium, libpthread) are installed.\n  Linux: 'sudo apt install libcurl4-openssl-dev libsodium-dev'\n  macOS: 'brew install curl libsodium'",
		},
		{
			code:     "W-E032",
			keywords: []string{"segmentation fault", "segfault", "signal 11"},
			summary:  "The compiled binary crashed with a segmentation fault at runtime.",
			detail:   "A null pointer dereference or memory access violation occurred at runtime. In Wolf, this almost always means a C runtime function received an unexpected argument type from the LLVM emitter.",
			fix:      "1. Run 'wolf build <file.wolf> --verbose' to see LLVM IR emission details.\n2. Check if the crashing function receives a map where it expects a string (common pattern: wolf_http_req_file receiving a ptr instead of const char*).\n3. Add explicit 'strval()' casts around arguments passed to low-level functions.",
		},
		{
			code:     "W-E033",
			keywords: []string{"double-mangling", "double_mangling", "double mangling", "namespace prefix"},
			summary:  "A method name was mangled twice, producing an invalid symbol.",
			detail:   "Wolf mangles class method names as 'ClassName_methodName'. If the namespace prefix is applied twice, the result is 'Namespace_ClassName_methodName' causing LLVM undefined reference errors.",
			fix:      "This is a known compiler bug. Ensure class declarations do not re-apply the namespace prefix inside the class body. Check parser.go parseClassDecl() for namespace suppression.",
		},

		// ── Linker / Binary Phase ─────────────────────────────────────────────

		{
			code:     "W-E040",
			keywords: []string{"cannot find llc", "llc not found", "llc: command not found"},
			summary:  "The LLVM static compiler 'llc' was not found on your PATH.",
			detail:   "Wolf requires 'llc' (the LLVM static compiler) and 'clang' to lower LLVM IR to a native binary. These are part of the LLVM toolchain.",
			fix:      "Install LLVM 14+:\n  Linux: 'sudo apt install llvm clang'\n  macOS: 'brew install llvm && export PATH=\"$(brew --prefix llvm)/bin:$PATH\"'\n  Windows: Download from https://releases.llvm.org/",
		},
		{
			code:     "W-E041",
			keywords: []string{"cannot find clang", "clang not found", "clang: command not found"},
			summary:  "The C compiler 'clang' was not found on your PATH.",
			detail:   "Wolf requires 'clang' to compile the C runtime and link the final binary. It is part of the LLVM toolchain.",
			fix:      "Install clang:\n  Linux: 'sudo apt install clang'\n  macOS: 'brew install llvm && export PATH=\"$(brew --prefix llvm)/bin:$PATH\"'\n  Windows: Download from https://releases.llvm.org/",
		},
		{
			code:     "W-E042",
			keywords: []string{"permission denied", "cannot write", "access denied"},
			summary:  "The compiler could not write output files due to a permission error.",
			detail:   "The wolf build process attempted to write to a directory or file that the current user does not have write access to.",
			fix:      "Check permissions on the project directory and the 'wolf_out/' directory. Run 'chmod -R u+w .' in your project root.",
		},

		// ── Runtime / HTTP Phase ──────────────────────────────────────────────

		{
			code:     "W-E050",
			keywords: []string{"port already in use", "address already in use", "bind failed"},
			summary:  "The HTTP server could not bind to the requested port.",
			detail:   "Another process is already listening on the port Wolf is trying to use (default: 8080). This is common if a previous Wolf dev server did not shut down cleanly.",
			fix:      "1. Kill the process holding the port: 'lsof -ti:8080 | xargs kill -9'\n2. Or configure a different port in wolf.config: '[target] port = 9090'",
		},
		{
			code:     "W-E051",
			keywords: []string{"database connection", "db connect", "cannot connect to", "mysql error", "postgres error"},
			summary:  "Wolf could not connect to the database.",
			detail:   "The database driver failed to establish a connection. The database server may be down, credentials may be wrong, or the host/port is unreachable.",
			fix:      "1. Verify your wolf.config [db] section: host, port, name, user, pass.\n2. Confirm the database server is running: 'mysql -u <user> -p' or 'psql -U <user>'.\n3. Check firewall rules if connecting to a remote host.",
		},
		{
			code:     "W-E052",
			keywords: []string{"redis", "cannot connect to redis", "redis connection"},
			summary:  "Wolf could not connect to Redis.",
			detail:   "The Redis client (hiredis) failed to establish a connection. Redis may not be running or the configured host/port is wrong.",
			fix:      "1. Start Redis: 'redis-server'\n2. Check wolf.config [cache] section for correct host and port.\n3. Test with: 'redis-cli ping'",
		},

		// ── Wolf Self-Hosted Compiler Phase ───────────────────────────────────

		{
			code:     "W-E060",
			keywords: []string{"hit parse error on", "wolf-self parse error"},
			summary:  "The self-hosted Wolf compiler hit a parse error while compiling Wolf source.",
			detail:   "The native Wolf parser (Parser.wolf) encountered a construct it does not yet support. The Go bootstrap supports more grammar than the current self-hosted parser.",
			fix:      "1. Check which construct triggered the error (the error message names the token).\n2. If it is 'enum', add parseEnum() support to Parser.wolf.\n3. Run 'wolf build src/compiler/main.wolf' to rebuild the self-hosted binary after fixing.",
		},
		{
			code:     "W-E061",
			keywords: []string{"autodiscover", "auto-discover", "auto discover", "file not found in project"},
			summary:  "AutoDiscovery could not locate required Wolf files in the project directory.",
			detail:   "Wolf's multi-file project mode scans the project directory recursively for .wolf files. If a class or function is declared in a file that is not under the project root, it will not be discovered.",
			fix:      "Run 'wolf build --project <dir>' where <dir> is the root containing all your .wolf files. Ensure no files are in directories excluded by .wolfignore.",
		},

		// ── Config / Setup Phase ──────────────────────────────────────────────

		{
			code:     "W-E070",
			keywords: []string{"wolf.config", "config not found", "missing config"},
			summary:  "A wolf.config file was expected but not found.",
			detail:   "Wolf looks for wolf.config in the project root. Without it, HTTP mode defaults may not be applied correctly.",
			fix:      "Create a wolf.config file:\n  'wolf new <name>' generates one automatically.\n  Or create manually — see https://wolflang.dev/docs/config",
		},
		{
			code:     "W-E071",
			keywords: []string{"wolf.mod", "module not found", "package not found"},
			summary:  "A required package declared in wolf.mod was not found.",
			detail:   "The wolf.mod file lists package dependencies, but 'wolf install' has not been run or the package registry is unreachable.",
			fix:      "Run 'wolf install' in the project root to install all declared dependencies.",
		},

		// ── Upload / File Phase ───────────────────────────────────────────────

		{
			code:     "W-E080",
			keywords: []string{"400 bad request", "multipart", "wolf_http_req_file"},
			summary:  "A file upload request was rejected with HTTP 400.",
			detail:   "The multipart/form-data parser in the Wolf HTTP engine rejected the upload. This is often caused by a Content-Type mismatch or the LLVM emitter passing the wrong argument type to wolf_http_req_file (ptr instead of const char*).",
			fix:      "1. Ensure your HTTP client sends 'Content-Type: multipart/form-data; boundary=...'.\n2. In Wolf code, the field name argument to req_file() must be a plain string literal, not a computed value.\n3. See BUG-088 in .wolf-vault/RnD/bugs_fixed.md for the context mismatch details.",
		},
	}
}
