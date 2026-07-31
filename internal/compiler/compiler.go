// Package compiler orchestrates the full Wolf compilation pipeline.
// Wolf compiles to native machine code via LLVM IR.
package compiler

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"wolf/internal/config"
	"wolf/internal/emitter"
	"wolf/internal/ir"
	"wolf/internal/lexer"
	"wolf/internal/parser"
	"wolf/internal/resolver"
	"wolf/internal/typechecker"
)

// goPluginMu serializes concurrent calls to buildGoPlugins.
// The Go plugin build uses a shared cache directory (/tmp/.../cache/go_src)
// and a shared archive (/tmp/.../cache/goplugin.a); concurrent access causes
// go mod init races and 'text file busy' link errors.
var goPluginMu sync.Mutex

// Compiler orchestrates the full Wolf compilation pipeline.
type Compiler struct {
	StrictMode  bool
	OutDir      string // default: wolf_out/
	Verbose     bool
	Config      *config.WolfConfig // loaded from wolf.config + env vars
	ProjectRoot string             // Root directory of the project for autodiscovery
	GoPlugins   []string           // Discovered .go files for Zero-Friction C/Go interop
	VFS         map[string]string  // Virtual File System for LSP unsaved file content
}

// New creates a Compiler with defaults and no config file.
// Prefer NewWithConfig when a project root is available.
func New() *Compiler {
	return &Compiler{
		OutDir: "wolf_out",
	}
}

// NewWithConfig creates a Compiler that loads wolf.config from projectRoot,
// walking up the directory tree until it finds one. Environment variables
// always override file values. If no wolf.config exists the defaults are used.
func NewWithConfig(projectRoot string) (*Compiler, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}
	return &Compiler{
		OutDir:      cfg.Build.OutDir,
		StrictMode:  cfg.Build.StrictMode,
		Config:      cfg,
		ProjectRoot: projectRoot,
	}, nil
}

// CompileResult holds the output of a compilation.
type CompileResult struct {
	LLVMSource    string             // generated LLVM IR
	OutputPath    string             // path to compiled binary
	Errors        []string           // compilation errors (human-readable)
	Diagnostics   []*lexer.WolfError // structured errors for LSP
	Program       *parser.Program    // for LSP AST walking
	Resolver      *resolver.Resolver // for LSP scope checking
	RequiresCurl  bool               // AST auto-linking flag for HTTP features
	RequiresRedis bool               // AST auto-linking flag for Redis features
}

// Compile runs the full pipeline: source → tokens → AST → resolve → typecheck → WIR → LLVM IR.
func (c *Compiler) Compile(source, filename string) (*CompileResult, error) {
	result := &CompileResult{}
	fmt.Printf(">> Phase 1: Lexing %s\n", filename)

	// Phase 1: Lex
	l := lexer.New(source, filename)
	tokens, lexErrors := l.Tokenize()
	if len(lexErrors) > 0 {
		for _, e := range lexErrors {
			result.Errors = append(result.Errors, e.Error())
		}
		return result, fmt.Errorf("lexer errors: %d errors found", len(lexErrors))
	}

	fmt.Printf(">> Phase 2: Parsing\n")
	// Phase 2: Parse
	p := parser.New(tokens, filename)
	var program *parser.Program
	var parseErrors []*lexer.WolfError
	program, parseErrors = p.Parse()
	if len(parseErrors) > 0 {
		for _, e := range parseErrors {
			result.Errors = append(result.Errors, e.Error())
		}
		return result, fmt.Errorf("parser errors: %d errors found", len(parseErrors))
	}

	fmt.Printf(">> Phase 2.5: AutoDiscover\n")
	// Phase 2.5: Auto-Discover Libraries and Controllers
	projectRoot := c.ProjectRoot
	if projectRoot == "" {
		projectRoot = filepath.Dir(filename)
	}
	discoveredASTs, err := c.AutoDiscover(projectRoot, filename, program)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, fmt.Errorf("autodiscovery failed: %w", err)
	}

	var allDiscovered []parser.Statement
	for _, ast := range discoveredASTs {
		allDiscovered = append(allDiscovered, ast.Statements...)
	}
	// Prepend all discovered (including config defines) BEFORE main statements
	// so defines run before serve()
	program.Statements = append(allDiscovered, program.Statements...)

	fmt.Printf(">> Phase 2.8: Dispatchers\n")
	// Generate the __compiler_dispatch_controller method based on all discovered classes
	dispatchFunc := generateDispatcherAST(program)
	if dispatchFunc != nil {
		program.Statements = append(program.Statements, dispatchFunc)
	}

	// Generate the __compiler_create_model method based on all discovered models
	factoryFunc := generateModelFactoryAST(program)
	if factoryFunc != nil {
		program.Statements = append(program.Statements, factoryFunc)
	}

	fmt.Printf(">> Phase 3: Resolve\n")
	// Phase 3: Resolve
	res := resolver.New(filename)
	res.SetStrictMode(c.StrictMode)
	resolveErrors := res.Resolve(program)
	if len(resolveErrors) > 0 {
		for _, e := range resolveErrors {
			result.Errors = append(result.Errors, e.Error())
		}
		return result, fmt.Errorf("resolver errors: %d errors found", len(resolveErrors))
	}

	fmt.Printf(">> Phase 4: Typecheck\n")
	// Phase 4: Type Check
	tc := typechecker.New(res, filename)
	tc.SetStrictMode(c.StrictMode)
	typeErrors := tc.Check(program)
	var hardTypeErrors []*lexer.WolfError
	for _, e := range typeErrors {
		if e.IsWarning {
			fmt.Fprintf(os.Stderr, "%s\n", e.Error())
		} else {
			result.Errors = append(result.Errors, e.Error())
			hardTypeErrors = append(hardTypeErrors, e)
		}
	}
	if len(hardTypeErrors) > 0 {
		return result, fmt.Errorf("type errors: %d errors found", len(hardTypeErrors))
	}

	fmt.Printf(">> Phase 5: Emit WIR\n")
	// Phase 5: Emit WIR (AST → Wolf IR)
	irEmit := emitter.New(res)
	irProgram := irEmit.Emit(program)

	fmt.Printf(">> Phase 5.5: Build Go Plugins (c-archive)\n")
	cgoFuncs, err := c.buildGoPlugins(irProgram)
	if err != nil {
		return result, fmt.Errorf("go plugin build error: %w", err)
	}

	fmt.Printf(">> Phase 6: Emit LLVM\n")
	// Phase 6: Emit LLVM IR (WIR → .ll)
	llvmEmit := emitter.NewLLVMEmitter()
	llvmEmit.CgoFuncs = cgoFuncs
	if c.Config != nil {
		llvmEmit.CompilerMode = c.Config.Target.Mode
		llvmEmit.Shared = c.Config.Target.Shared
	}
	llvmEmit.TargetTriple = detectTargetTriple()
	llvmSource := llvmEmit.Emit(irProgram)
	if len(llvmEmit.Errors) > 0 {
		for _, errStr := range llvmEmit.Errors {
			result.Errors = append(result.Errors, errStr)
		}
		return result, fmt.Errorf("LLVM emission errors: %d errors found", len(llvmEmit.Errors))
	}
	result.LLVMSource = llvmSource
	result.RequiresCurl = llvmEmit.RequiresCurl
	result.RequiresRedis = llvmEmit.RequiresRedis

	fmt.Printf(">> Done returning\n")
	return result, nil
}

// Check runs the compiler pipeline up to Phase 4 (Type Checking) and returns errors if any.
func (c *Compiler) Check(source, filename string) (*CompileResult, error) {
	result := &CompileResult{
		OutputPath:  "",
		Errors:      []string{},
		Diagnostics: []*lexer.WolfError{},
	}

	fmt.Printf(">> Phase 1: Lexing %s\n", filename)
	l := lexer.New(source, filename)
	tokens, lexErrs := l.Tokenize()
	if len(lexErrs) > 0 {
		for _, e := range lexErrs {
			result.Errors = append(result.Errors, e.Error())
			result.Diagnostics = append(result.Diagnostics, e)
		}
		return result, fmt.Errorf("lex errors: %d errors found", len(lexErrs))
	}

	fmt.Printf(">> Phase 2: Parsing\n")
	p := parser.New(tokens, filename)
	program, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		for _, e := range parseErrs {
			result.Errors = append(result.Errors, e.Error())
			result.Diagnostics = append(result.Diagnostics, e)
		}
		return result, fmt.Errorf("parse errors: %d errors found", len(parseErrs))
	}

	fmt.Printf(">> Phase 2.5: AutoDiscover\n")
	projectRoot := c.ProjectRoot
	if projectRoot == "" {
		projectRoot = filepath.Dir(filename)
	}
	discoveredASTs, err := c.AutoDiscover(projectRoot, filename, program)
	if err != nil {
		fmt.Printf("wolf: autodiscovery warning: %v\n", err)
	} else {
		var allDiscovered []parser.Statement
		for _, ast := range discoveredASTs {
			allDiscovered = append(allDiscovered, ast.Statements...)
		}
		program.Statements = append(allDiscovered, program.Statements...)
	}

	fmt.Printf(">> Phase 2.8: Dispatchers\n")
	factoryFunc := generateModelFactoryAST(program)
	if factoryFunc != nil {
		program.Statements = append(program.Statements, factoryFunc)
	}

	fmt.Printf(">> Phase 3: Resolve\n")
	res := resolver.New(filename)
	res.SetStrictMode(c.StrictMode)
	resolveErrors := res.Resolve(program)
	if len(resolveErrors) > 0 {
		for _, e := range resolveErrors {
			result.Errors = append(result.Errors, e.Error())
			result.Diagnostics = append(result.Diagnostics, e)
		}
		return result, fmt.Errorf("resolver errors: %d errors found", len(resolveErrors))
	}

	fmt.Printf(">> Phase 4: Typecheck\n")
	tc := typechecker.New(res, filename)
	tc.SetStrictMode(c.StrictMode)
	typeErrors := tc.Check(program)
	var hardTypeErrors []*lexer.WolfError
	for _, e := range typeErrors {
		result.Diagnostics = append(result.Diagnostics, e)
		if e.IsWarning {
			fmt.Fprintf(os.Stderr, "%s\n", e.Error())
		} else {
			result.Errors = append(result.Errors, e.Error())
			hardTypeErrors = append(hardTypeErrors, e)
		}
	}
	if len(hardTypeErrors) > 0 {
		return result, fmt.Errorf("type errors: %d errors found", len(hardTypeErrors))
	}

	result.Program = program
	result.Resolver = res

	return result, nil
}

// Build compiles a Wolf source file to a native binary via LLVM.
func (c *Compiler) Build(source, filename string) (*CompileResult, error) {
	fmt.Printf(">> Build started: %s\n", filename)
	result, err := c.Compile(source, filename)
	if err != nil {
		return result, err
	}

	fmt.Printf(">> Compile finished\n")
	outDir, err := filepath.Abs(c.OutDir)
	if err != nil {
		return result, fmt.Errorf("failed to resolve output directory: %w", err)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return result, fmt.Errorf("failed to create output directory: %w", err)
	}

	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	cc := findCC()
	var zigArgs []string
	if c.Config != nil && c.Config.Target.Static {
		if path, err := exec.LookPath("zig"); err == nil {
			cc = path
			zigArgs = []string{"cc", "-target", "x86_64-linux-musl"}
		} else {
			return result, fmt.Errorf("zig compiler is required for static builds but was not found in PATH")
		}
	}
	if c.Verbose {
		fmt.Printf("wolf: using C compiler: %s\n", cc)
		if c.Config != nil {
			fmt.Printf("wolf: config env=%s pool_size=%d port=%d\n",
				c.Config.App.Env, c.Config.DB.PoolSize, c.Config.Server.Port)
		}
	}

	// Write LLVM IR
	llFile := filepath.Join(outDir, baseName+".ll")
	if err := os.WriteFile(llFile, []byte(result.LLVMSource), 0644); err != nil {
		return result, fmt.Errorf("failed to write LLVM IR: %w", err)
	}

	if c.Verbose {
		fmt.Printf("wolf: wrote LLVM IR → %s\n", llFile)
	}

	// Clean up .ll file unless keep_ll is set
	// LLVM IR file is kept deliberately for debug

	// Extract embedded assets to a cache directory
	assetsDir, err := ensureAssetsExtracted()
	if err != nil {
		return result, fmt.Errorf("failed to extract compiler assets: %w", err)
	}
	runtimeC := filepath.Join(assetsDir, "runtime", "wolf_runtime.c")

	// Determine if we need to link the Go Plugin Archive
	goPluginArchive := filepath.Join(assetsDir, "..", "cache", "goplugin.a")
	hasGoPlugin := false
	if _, err := os.Stat(goPluginArchive); err == nil {
		hasGoPlugin = true
	}

	// ── P0: LLVM Optimizer Pass ─────────────────────────────────────────────
	// When opt is available and optimisation is enabled, run the full LLVM
	// middle-end pipeline (O3) on the .ll before handing it to llc.
	// This enables: TCO, loop unrolling, constant folding, function inlining,
	// SROA, GVN, and all standard LLVM optimization passes.
	// Gracefully falls back to the raw .ll if opt is not installed.
	optimise := c.Config == nil || c.Config.Build.Optimise // default: true
	llOrBcFile := llFile                                   // llc will read this; may be upgraded to .bc after opt
	if optimise && hasOpt() {
		optLevel := "-O3"
		bcFile := filepath.Join(outDir, baseName+".bc")
		fmt.Printf(">> Running opt %s (.ll → .bc)\n", optLevel)
		optCmd := exec.Command("opt", optLevel, "-o", bcFile, llFile)
		if out, err := optCmd.CombinedOutput(); err != nil {
			// opt failed — warn and proceed with unoptimized IR
			if c.Verbose {
				fmt.Printf("wolf: opt failed (falling back to unoptimized IR): %s\n%s\n", err, string(out))
			}
		} else {
			llOrBcFile = bcFile // hand the optimized bitcode to llc
			if c.Verbose {
				fmt.Printf("wolf: opt %s applied → %s\n", optLevel, bcFile)
			}
		}
		fmt.Printf(">> opt finished\n")
	}

	// Compile LLVM IR (or optimized bitcode) to object file
	objFile := filepath.Join(outDir, baseName+".o")
	compiled := false

	// Strategy 1: Use LLC if available
	var compileErrors []string

	fmt.Printf(">> Testing hasLLC\n")
	if hasLLC() {
		fmt.Printf(">> Running llc\n")
		llcCmd := exec.Command("llc", "-filetype=obj", "-relocation-model=pic", "-o", objFile, llOrBcFile)
		if out, err := llcCmd.CombinedOutput(); err != nil {
			compileErrors = append(compileErrors, fmt.Sprintf("llc error: %s\n%s", err, string(out)))
			if c.Verbose {
				fmt.Printf("wolf: llc failed: %s\n%s\n", err, string(out))
			}
		} else {
			compiled = true
			if c.Verbose {
				fmt.Printf("wolf: compiled → .o via llc\n")
			}
		}
		fmt.Printf(">> llc finished\n")
	}

	fmt.Printf(">> Checking compiled\n")
	// Strategy 2: Use clang to compile .ll/.bc directly
	if !compiled && hasClang() {
		fmt.Printf(">> Running clang (→ o)\n")
		clangOptFlag := "-O0"
		if optimise {
			clangOptFlag = "-O3"
		}
		clangCmd := exec.Command("clang", "-c", clangOptFlag, "-o", objFile, llOrBcFile)
		if out, err := clangCmd.CombinedOutput(); err != nil {
			compileErrors = append(compileErrors, fmt.Sprintf("clang error: %s\n%s", err, string(out)))
			if c.Verbose {
				fmt.Printf("wolf: clang compilation failed: %s\n%s\n", err, string(out))
			}
		} else {
			compiled = true
			if c.Verbose {
				fmt.Printf("wolf: compiled → .o via clang\n")
			}
		}
	}

	// Strategy 3: Use llvm-as + llc pipeline (raw .ll only — opt already ran above)
	if !compiled {
		fmt.Printf(">> Running llvm-as\n")
		bcFallback := filepath.Join(outDir, baseName+"_as.bc")
		llvmAsCmd := exec.Command("llvm-as", "-o", bcFallback, llFile)
		if asOut, err := llvmAsCmd.CombinedOutput(); err == nil {
			llcCmd := exec.Command("llc", "-filetype=obj", "-relocation-model=pic", "-o", objFile, bcFallback)
			if llcOut, err := llcCmd.CombinedOutput(); err == nil {
				compiled = true
			} else {
				compileErrors = append(compileErrors, fmt.Sprintf("llc (bc) error: %s\n%s", err, string(llcOut)))
			}
		} else {
			compileErrors = append(compileErrors, fmt.Sprintf("llvm-as error: %s\n%s", err, string(asOut)))
		}
	}

	if !compiled {
		return result, fmt.Errorf("LLVM compilation failed:\n%s\nIf tools are missing, install them: sudo apt-get install clang llvm", strings.Join(compileErrors, "\n"))
	}

	fmt.Printf(">> Setting up wolf runtime args\n")
	// Compile wolf runtime
	runtimeObj := filepath.Join(outDir, "wolf_runtime.o")

	// Production-grade library discovery via bundled static libs or pkg-config
	staticDir := filepath.Join(assetsDir, "third_party", "lib")

	// Platform-specific static paths
	var bundledPath string
	if c.Config != nil && c.Config.Target.Static {
		bundledPath = filepath.Join(staticDir, "linux_x64_musl")
	} else if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		bundledPath = filepath.Join(staticDir, "linux_x64")
	} else if runtime.GOOS == "darwin" {
		bundledPath = filepath.Join(staticDir, "macos")
	}

	useStatic := false
	if bundledPath != "" {
		if _, err := os.Stat(filepath.Join(bundledPath, "libsodium.a")); err == nil {
			useStatic = true
		}
	}

	// Detect DB client flags
	dbCflags := ""
	dbLibs := ""

	driver := "mysql"
	if c.Config != nil && c.Config.DB.Driver != "" {
		driver = c.Config.DB.Driver
	}

	if driver == "mysql" {
		if c.Config != nil && c.Config.Target.Static {
			dbLibs = filepath.Join(bundledPath, "libmariadb.a")
			dbCflags = "-I" + filepath.Join(bundledPath, "../../include")
		} else {
			dbLibs = "-lmysqlclient"
			for _, mysqlConfig := range []string{"/opt/lampp/bin/mysql_config", "/usr/local/mysql/bin/mysql_config", "mariadb_config", "mysql_config"} {
				if path, err := exec.LookPath(mysqlConfig); err == nil {
					if out, err := exec.Command(path, "--cflags").Output(); err == nil {
						dbCflags = strings.TrimSpace(string(out))
					}
					if out, err := exec.Command(path, "--libs").Output(); err == nil {
						dbLibs = strings.TrimSpace(string(out))
					}
					break
				}
			}
		}
	} else if driver == "postgres" {
		dbLibs = "-lpq"
		if path, err := exec.LookPath("pg_config"); err == nil {
			if out, err := exec.Command(path, "--includedir").Output(); err == nil {
				dbCflags = "-I" + strings.TrimSpace(string(out))
			}
			if out, err := exec.Command(path, "--libdir").Output(); err == nil {
				dbLibs = "-L" + strings.TrimSpace(string(out)) + " -lpq"
			}
		}
	} else if driver == "mssql" {
		// Mock MSSQL - no library needed yet
		dbCflags = ""
		dbLibs = ""
	}

	redisLibs := ""
	if result.RequiresRedis {
		redisLibs = "-lhiredis"
	}



	sslLibs, sslCflags := getPkgConfig("openssl")
	sodiumLibs, sodiumCflags := getPkgConfig("libsodium")

	// Fallback to defaults if pkg-config fails
	if sslLibs == "" {
		sslLibs = "-lssl -lcrypto"
	}
	if sodiumLibs == "" {
		sodiumLibs = "-lsodium"
	}

	cryptoLibs := sodiumLibs + " " + sslLibs
	cryptoCflags := sodiumCflags + " " + sslCflags

	if useStatic {
		// Use absolute paths to .a files to force static linking
		cryptoLibs = fmt.Sprintf("%s %s %s",
			filepath.Join(bundledPath, "libsodium.a"),
			filepath.Join(bundledPath, "libssl.a"),
			filepath.Join(bundledPath, "libcrypto.a"))
		cryptoCflags = "-I" + filepath.Join(bundledPath, "../../include")
	}

	// Special case: If user has XAMPP installed, its mysql_config might add a -L path
	// that contains an outdated libcrypto.so.1.1. We MUST prioritize the system path
	// found by pkg-config (if any) to ensure OpenSSL 3.x symbols like EVP_DigestSignUpdate work.
	// NOTE: If using static libs, we don't need to prioritize path as much, but we keep it for other libs.
	var prioritizedPath string
	if !useStatic {
		for _, lib := range []string{"openssl", "libsodium"} {
			if path, _ := getPkgConfigVariable(lib, "libdir"); path != "" {
				prioritizedPath = "-L" + path
				break
			}
		}
	}

	// Build runtime compile args — optimisation level from config
	optFlag := "-O2"
	if c.Config != nil && !c.Config.Build.Optimise {
		optFlag = "-O0"
	}
	rtArgs := append(append([]string{}, zigArgs...), "-c", optFlag, "-pthread", "-g")
	if runtime.GOOS == "linux" {
		if c.Config != nil && c.Config.Target.Static {
			rtArgs = append(rtArgs, "-I"+filepath.Join(bundledPath, "../../include"))
		} else {
			rtArgs = append(rtArgs, "-I/tmp/liburing/src/include")
		}
	}

	// DB include flags
	if dbCflags != "" {
		rtArgs = append(rtArgs, strings.Fields(dbCflags)...)
	}
	if cryptoCflags != "" {
		rtArgs = append(rtArgs, strings.Fields(cryptoCflags)...)
	}

	// Enable real Redis implementation if AST detected it
	if result.RequiresRedis {
		rtArgs = append(rtArgs, "-DWOLF_REDIS_ENABLED")
	}

	httpClientEnabled := false
	if result.RequiresCurl {
		httpClientEnabled = true
	}
	if httpClientEnabled {
		rtArgs = append(rtArgs, "-DWOLF_HTTP_CLIENT_ENABLED")
	}

	// Bake wolf.config values into the runtime as -D constants.
	// This is how pool size, timeouts, credentials, and server limits
	// reach wolf_runtime.c without needing a config file at runtime.
	rtArgs = append(rtArgs, c.configCFlags()...)

	if os.Getenv("WOLF_DEBUG") != "" {
		rtArgs = append(rtArgs, "-DWOLF_DEBUG")
	}

	if os.Getenv("WOLF_ASAN") != "" {
		rtArgs = append(rtArgs, "-fsanitize=address", "-fno-omit-frame-pointer")
	}
	if os.Getenv("WOLF_TSAN") != "" {
		rtArgs = append(rtArgs, "-fsanitize=thread", "-fno-omit-frame-pointer")
	}
        rtArgs = append(rtArgs, "-DWOLF_BUILD_TARGET_API")

	if c.Config != nil && c.Config.Target.Shared {
		rtArgs = append(rtArgs, "-DWOLF_HOST_SHELL")
	}

	// Get cache key BEFORE adding dynamic paths
	var rtCacheHit bool
	cacheKey, err := runtimeCacheKey(runtimeC, rtArgs)

	rtArgs = append(rtArgs, "-o", runtimeObj, runtimeC)

	// ---- Runtime object cache (hash-based) ----
	// Cache key = SHA-256(runtime C source + all compile flags).
	// On a hit we copy the cached .o rather than recompiling (~37s → ~1ms).
	if err == nil {
		cacheDir := filepath.Join(os.TempDir(), "wolf_rt_cache")
		_ = os.MkdirAll(cacheDir, 0755)
		cachedObj := filepath.Join(cacheDir, cacheKey+".o")
		if _, err := os.Stat(cachedObj); err == nil {
			// Cache hit — copy cached object to target location
			if copyFile(cachedObj, runtimeObj) == nil {
				rtCacheHit = true
			}
		}
		if !rtCacheHit {
			fmt.Printf("wolf: runtime compile args: %v\n", rtArgs)
			rtCmd := exec.Command(cc, rtArgs...)
			if out, err := rtCmd.CombinedOutput(); err != nil {
				return result, fmt.Errorf("failed to compile wolf runtime: %s\n%s", err, string(out))
			}
			// Store result in cache for future runs
			_ = copyFile(runtimeObj, cachedObj)
		}
	} else {
		// Fallback: compile without cache
		rtCmd := exec.Command(cc, rtArgs...)
		if out, err := rtCmd.CombinedOutput(); err != nil {
			return result, fmt.Errorf("failed to compile wolf runtime: %s\n%s", err, string(out))
		}
	}

	if c.Verbose {
		fmt.Printf("wolf: compiled wolf_runtime.c → %s\n", runtimeObj)
	}

	// Link everything into final binary
	binaryPath := filepath.Join(outDir, baseName)
	if c.Config != nil && c.Config.Target.Shared {
		binaryPath += ".so"
	}

	var linkArgs []string
	if c.Config != nil && c.Config.Target.Shared {
		linkArgs = append(append([]string{}, zigArgs...), "-shared", "-fPIC", "-o", binaryPath, objFile, "-g")
		if hasGoPlugin {
			linkArgs = append(linkArgs, goPluginArchive)
		}
		if runtime.GOOS == "darwin" {
			linkArgs = append(linkArgs, "-undefined", "dynamic_lookup")
		}
	} else {
		linkArgs = append(append([]string{}, zigArgs...), "-o", binaryPath, objFile, runtimeObj, "-pthread", "-g")
		if c.Config != nil && c.Config.Target.Static {
			linkArgs = append(linkArgs, "-static")
		}
		if hasGoPlugin {
			linkArgs = append(linkArgs, goPluginArchive)
		}
	}

	if os.Getenv("WOLF_ASAN") != "" {
		linkArgs = append(linkArgs, "-fsanitize=address")
	}
	if os.Getenv("WOLF_TSAN") != "" {
		linkArgs = append(linkArgs, "-fsanitize=thread")
	}

	// Prioritize system libraries to avoid XAMPP version conflicts
	if prioritizedPath != "" {
		linkArgs = append(linkArgs, prioritizedPath)
	}

	linkArgs = append(linkArgs, strings.Fields(dbLibs)...)
	if redisLibs != "" {
		linkArgs = append(linkArgs, strings.Fields(redisLibs)...)
	}
	linkArgs = append(linkArgs, strings.Fields(cryptoLibs)...)
	// Auto-extract rpath from -L flags in dbLibs so binary finds the DB library at runtime
	for _, field := range strings.Fields(dbLibs) {
		if strings.HasPrefix(field, "-L") {
			libPath := strings.TrimPrefix(field, "-L")
			libPath = strings.TrimSuffix(libPath, "/")
			linkArgs = append(linkArgs, "-Wl,-rpath,"+libPath)
		}
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		linkArgs = append(linkArgs, "-lm")
		if runtime.GOOS == "linux" {
			if c.Config != nil && c.Config.Target.Static {
				linkArgs = append(linkArgs, filepath.Join(bundledPath, "liburing.a"))
			} else {
				linkArgs = append(linkArgs, "-luring")
			}
		}
		if httpClientEnabled {
			if c.Config != nil && c.Config.Target.Static {
				linkArgs = append(linkArgs, filepath.Join(bundledPath, "libcurl.a"))
			} else {
				linkArgs = append(linkArgs, "-lcurl")
			}
		}
		if c.Config != nil && c.Config.Target.Static {
			linkArgs = append(linkArgs, filepath.Join(bundledPath, "libz.a"), filepath.Join(bundledPath, "libucontext.a"))
		}
	}

	linkCmd := exec.Command(cc, linkArgs...)
	fmt.Printf("wolf: linker command: %v\n", linkCmd.Args)
	if out, err := linkCmd.CombinedOutput(); err != nil {
		return result, fmt.Errorf("linking failed: %s\n%s", err, string(out))
	}

	if c.Config != nil && c.Config.Target.Shared {
		// Also link wolf_host executable using just runtimeObj
		hostPath := filepath.Join(outDir, "wolf_host")

		hostLinkArgs := []string{"-o", hostPath, runtimeObj, "-pthread", "-g", "-rdynamic", "-luring"}
		if runtime.GOOS == "darwin" {
			// -rdynamic on mac is sometimes -Wl,-export_dynamic
			// actually clang supports -rdynamic directly on macOS usually.
		}
		if os.Getenv("WOLF_ASAN") != "" {
			hostLinkArgs = append(hostLinkArgs, "-fsanitize=address")
		}
		if os.Getenv("WOLF_TSAN") != "" {
			hostLinkArgs = append(hostLinkArgs, "-fsanitize=thread")
		}
		if prioritizedPath != "" {
			hostLinkArgs = append(hostLinkArgs, prioritizedPath)
		}
		hostLinkArgs = append(hostLinkArgs, strings.Fields(dbLibs)...)
		if redisLibs != "" {
			hostLinkArgs = append(hostLinkArgs, strings.Fields(redisLibs)...)
		}
		hostLinkArgs = append(hostLinkArgs, strings.Fields(cryptoLibs)...)
		for _, field := range strings.Fields(dbLibs) {
			if strings.HasPrefix(field, "-L") {
				libPath := strings.TrimPrefix(field, "-L")
				libPath = strings.TrimSuffix(libPath, "/")
				hostLinkArgs = append(hostLinkArgs, "-Wl,-rpath,"+libPath)
			}
		}
		if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
			hostLinkArgs = append(hostLinkArgs, "-lm", "-ldl")
			if httpClientEnabled {
				hostLinkArgs = append(hostLinkArgs, "-lcurl")
			}
		}
		hostCmd := exec.Command(cc, hostLinkArgs...)
		if out, err := hostCmd.CombinedOutput(); err != nil {
			return result, fmt.Errorf("wolf_host linking failed: %s\n%s", err, string(out))
		}
	}

	result.OutputPath = binaryPath

	if c.Verbose {
		fmt.Printf("wolf: linked → %s\n", binaryPath)
	}

	// Write build config snapshot (excludes credentials)
	_ = c.writeConfigSnapshot(outDir)

	return result, nil
}

// Run compiles and immediately executes a Wolf source file.
func (c *Compiler) Run(source, filename string) error {
	result, err := c.Build(source, filename)
	if err != nil {
		return err
	}

	cmd := exec.Command(result.OutputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// ========== Config helpers ==========

// configCFlags returns -D flags that bake wolf.config values into wolf_runtime.c
// at compile time. Called nil-safe — returns nil when no config is loaded.
func (c *Compiler) configCFlags() []string {
	if c.Config == nil {
		return nil
	}
	cfg := c.Config
	var driverFlag string
	switch cfg.DB.Driver {
	case "postgres":
		driverFlag = "-DWOLF_DB_POSTGRES=1"
	case "mssql":
		driverFlag = "-DWOLF_DB_MSSQL=1"
	default:
		driverFlag = "-DWOLF_DB_MYSQL=1"
	}

	// Build target flag — gates entire subsystems at compile time:
	//   API    → full HTTP engine, Thread-Per-Core, arena pool, DB pool
	//   Script → lightweight build, single arena, no HTTP engine spun up
	//   MCU    → no OS primitives: no pthreads, no epoll, static heap only
	targetFlag := "-DWOLF_BUILD_TARGET_SCRIPT" // safe default
	switch cfg.Target.Mode {
	case "api":
		targetFlag = "-DWOLF_BUILD_TARGET_API"
	case "mcu":
		targetFlag = "-DWOLF_BUILD_TARGET_MCU"
	}

	return []string{
		targetFlag,
		driverFlag,
		// DB pool
		fmt.Sprintf("-DWOLF_DB_POOL_SIZE=%d", cfg.DB.PoolSize),
		fmt.Sprintf("-DWOLF_DB_POOL_MIN_IDLE=%d", cfg.DB.PoolMinIdle),
		fmt.Sprintf("-DWOLF_DB_POOL_TIMEOUT=%d", cfg.DB.PoolTimeout),
		fmt.Sprintf("-DWOLF_DB_MAX_RETRIES=%d", cfg.DB.MaxRetries),
		// Server limits
		fmt.Sprintf("-DWOLF_MAX_CONCURRENT_REQUESTS=%d", cfg.Server.MaxConcurrent),
		fmt.Sprintf("-DWOLF_MAX_REQUEST_SIZE=%d", cfg.Server.MaxRequestSize),
		fmt.Sprintf("-DWOLF_MAX_UPLOADS=%d", cfg.Server.MaxUploads),
		// Worker threads — 0 means auto-detect in C runtime (wolf_engine_create)
		fmt.Sprintf("-DWOLF_WORKER_THREADS=%d", cfg.Server.Workers),
		// DB credentials — baked as string literals so wolf source just calls db_connect()
		fmt.Sprintf("-DWOLF_DB_HOST=\"%s\"", escapeCStr(cfg.DB.Host)),
		fmt.Sprintf("-DWOLF_DB_PORT=%d", cfg.DB.Port),
		fmt.Sprintf("-DWOLF_DB_NAME=\"%s\"", escapeCStr(cfg.DB.Name)),
		fmt.Sprintf("-DWOLF_DB_USER=\"%s\"", escapeCStr(cfg.DB.User)),
		fmt.Sprintf("-DWOLF_DB_PASS=\"%s\"", escapeCStr(cfg.DB.Password)),
		// App environment
		fmt.Sprintf("-DWOLF_APP_ENV=\"%s\"", escapeCStr(cfg.App.Env)),
		fmt.Sprintf("-DWOLF_APP_DEBUG=%d", boolToInt(cfg.App.Debug)),
	}
}

// writeConfigSnapshot writes a .wolf_build_config file to outDir so deployment
// tooling can inspect compiled-in settings. Never contains credentials.
func (c *Compiler) writeConfigSnapshot(outDir string) error {
	if c.Config == nil {
		return nil
	}
	cfg := c.Config
	lines := []string{
		"# Wolf build config snapshot — generated at compile time",
		"# Does not contain credentials.",
		"",
		fmt.Sprintf("app_name        = %s", cfg.App.Name),
		fmt.Sprintf("app_env         = %s", cfg.App.Env),
		fmt.Sprintf("app_version     = %s", cfg.App.Version),
		"",
		fmt.Sprintf("server_host     = %s", cfg.Server.Host),
		fmt.Sprintf("server_port     = %d", cfg.Server.Port),
		fmt.Sprintf("server_workers  = %d", cfg.Server.Workers),
		fmt.Sprintf("max_concurrent  = %d", cfg.Server.MaxConcurrent),
		"",
		fmt.Sprintf("db_host         = %s", cfg.DB.Host),
		fmt.Sprintf("db_port         = %d", cfg.DB.Port),
		fmt.Sprintf("db_name         = %s", cfg.DB.Name),
		fmt.Sprintf("db_pool_size    = %d", cfg.DB.PoolSize),
		fmt.Sprintf("db_pool_min_idle= %d", cfg.DB.PoolMinIdle),
		fmt.Sprintf("db_pool_timeout = %d", cfg.DB.PoolTimeout),
		"",
		fmt.Sprintf("redis_host      = %s", cfg.Redis.Host),
		fmt.Sprintf("redis_port      = %d", cfg.Redis.Port),
	}
	path := filepath.Join(outDir, ".wolf_build_config")
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func escapeCStr(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ========== Build helpers ==========

func findCC() string {
	for _, cc := range []string{"clang", "gcc", "cc"} {
		if path, err := exec.LookPath(cc); err == nil {
			return path
		}
	}
	return "cc"
}

func hasLLC() bool {
	_, err := exec.LookPath("llc")
	return err == nil
}

func hasClang() bool {
	_, err := exec.LookPath("clang")
	return err == nil
}

// hasOpt returns true if the LLVM `opt` optimizer is available on PATH.
// Tries the unversioned name first, then common versioned suffixes.
func hasOpt() bool {
	for _, name := range []string{"opt", "opt-15", "opt-14", "opt-16", "opt-17", "opt-18"} {
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
	}
	return false
}

func detectTargetTriple() string {
	for _, tool := range []string{"llvm-config", "llvm-config-15", "llvm-config-14"} {
		if path, err := exec.LookPath(tool); err == nil {
			if out, err := exec.Command(path, "--host-target").Output(); err == nil {
				if triple := strings.TrimSpace(string(out)); triple != "" {
					return triple
				}
			}
		}
	}
	if path, err := exec.LookPath("clang"); err == nil {
		if out, err := exec.Command(path, "-dumpmachine").Output(); err == nil {
			if triple := strings.TrimSpace(string(out)); triple != "" {
				return triple
			}
		}
	}
	if path, err := exec.LookPath("gcc"); err == nil {
		if out, err := exec.Command(path, "-dumpmachine").Output(); err == nil {
			if triple := strings.TrimSpace(string(out)); triple != "" {
				return triple
			}
		}
	}
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "arm64-apple-macosx11.0.0"
		}
		return "x86_64-apple-macosx10.15.0"
	case "windows":
		return "x86_64-pc-windows-msvc"
	default:
		if runtime.GOARCH == "arm64" {
			return "aarch64-unknown-linux-gnu"
		}
		return "x86_64-pc-linux-gnu"
	}
}

func getPkgConfig(lib string) (libs string, cflags string) {
	if path, err := exec.LookPath("pkg-config"); err == nil {
		if out, err := exec.Command(path, "--libs", lib).Output(); err == nil {
			libs = strings.TrimSpace(string(out))
		}
		if out, err := exec.Command(path, "--cflags", lib).Output(); err == nil {
			cflags = strings.TrimSpace(string(out))
		}
	}
	return
}

func getPkgConfigVariable(lib, variable string) (string, error) {
	if path, err := exec.LookPath("pkg-config"); err == nil {
		out, err := exec.Command(path, "--variable="+variable, lib).Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
		return "", err
	}
	return "", fmt.Errorf("pkg-config not found")
}

// runtimeCacheKey returns a hex SHA-256 hash of wolf_runtime.c + wolf_http_engine.c
// content + compile flags.
// Used as the filename for the cached .o in /tmp/wolf_rt_cache/.
// wolf_http_engine.c is #include'd by wolf_runtime.c, so changes to either file
// must invalidate the cache.
func runtimeCacheKey(runtimeC string, flags []string) (string, error) {
	h := sha256.New()
	
	// Hash all .c and .h files in the runtime directory
	runtimeDir := filepath.Dir(runtimeC)
	err := filepath.Walk(runtimeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".c") || strings.HasSuffix(path, ".h")) {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			_, _ = io.Copy(h, f)
			f.Close()
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	// Mix in the flags (exclude the -o <path> and the runtimeC path at the end,
	// which vary per test but don't affect the compiled object content).
	for _, flag := range flags {
		if flag == "-o" || flag == runtimeC {
			continue
		}
		h.Write([]byte(flag))
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16], nil
}

// loadConfig reads the config file safely
func (c *Compiler) loadConfig(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	c.Config = cfg
	return nil
}

// buildGoPlugins handles the "Zero-Friction C/Go Interop" compilation.
// It creates a synthetic main.go that imports all discovered .go plugins and mlbridge,
// compiles them into a c-archive, and parses the resulting header.
// The shared cache directory is protected by goPluginMu to prevent parallel races.
func (c *Compiler) buildGoPlugins(irProg *ir.Program) ([]emitter.CGOFunction, error) {
	if len(c.GoPlugins) == 0 && !irProg.RequiresMLBridge {
		return nil, nil
	}

	goPluginMu.Lock()
	defer goPluginMu.Unlock()

	assetsDir, err := ensureAssetsExtracted()
	if err != nil {
		return nil, err
	}
	cacheDir := filepath.Join(assetsDir, "..", "cache", "go_src")
	os.RemoveAll(cacheDir)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	// Copy internal/mlbridge/bridge.go to cacheDir as main_mlbridge.go
	bridgeSrcPath := filepath.Join(c.ProjectRoot, "internal", "mlbridge", "bridge.go")
	if bSrc, err := os.ReadFile(bridgeSrcPath); err == nil {
		bSrcStr := strings.Replace(string(bSrc), "package mlbridge", "package main", 1)
		os.WriteFile(filepath.Join(cacheDir, "main_mlbridge.go"), []byte(bSrcStr), 0644)
	} else {
		// Fallback for tests or local execution
		bSrc, err := os.ReadFile(filepath.Join("..", "internal", "mlbridge", "bridge.go"))
		if err == nil {
			bSrcStr := strings.Replace(string(bSrc), "package mlbridge", "package main", 1)
			os.WriteFile(filepath.Join(cacheDir, "main_mlbridge.go"), []byte(bSrcStr), 0644)
		} else {
			fmt.Printf("wolf warning: could not locate internal/mlbridge/bridge.go for ML compilation\n")
		}
	}

	// Create a synthetic main.go
	mainGoPath := filepath.Join(cacheDir, "main.go")
	mainContent := "package main\n\nimport \"C\"\n\n"

	for i, pluginPath := range c.GoPlugins {
		// Read the plugin, replace `package [whatever]` with `package main`
		srcBytes, err := os.ReadFile(pluginPath)
		if err != nil {
			return nil, err
		}

		lines := strings.Split(string(srcBytes), "\n")
		for j, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "package ") {
				lines[j] = "package main"
				break
			}
		}

		pluginDest := filepath.Join(cacheDir, fmt.Sprintf("plugin_%d.go", i))
		if err := os.WriteFile(pluginDest, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return nil, err
		}
	}

	mainContent += "\nfunc main() {}\n"
	if err := os.WriteFile(mainGoPath, []byte(mainContent), 0644); err != nil {
		return nil, err
	}

	// Initialize go mod to avoid go.mod not found errors
	if _, err := os.Stat(filepath.Join(cacheDir, "go.mod")); os.IsNotExist(err) {
		cmdMod := exec.Command("go", "mod", "init", "goplugin")
		cmdMod.Dir = cacheDir
		if out, err := cmdMod.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to init go mod: %s\n%s", err, string(out))
		}
	}

	archivePath := filepath.Join(assetsDir, "..", "cache", "goplugin.a")

	// Build the archive
	cmd := exec.Command("go", "build", "-buildmode=c-archive", "-o", archivePath, ".")
	cmd.Dir = cacheDir
	cmdEnv := append(os.Environ(), "CGO_ENABLED=1")
	if c.Config != nil && c.Config.Target.Static {
		if _, err := exec.LookPath("musl-gcc"); err == nil {
			cmdEnv = append(cmdEnv, "CC=musl-gcc", "CGO_LDFLAGS=-static")
		} else if _, err := exec.LookPath("zig"); err == nil {
			cmdEnv = append(cmdEnv, "CC=zig cc -target x86_64-linux-musl", "CGO_LDFLAGS=-static")
		} else {
			return nil, fmt.Errorf("static compilation requested but neither musl-gcc nor zig were found in PATH")
		}
	}
	cmd.Env = cmdEnv

	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to compile go plugins: %s\n%s", err, string(out))
	}

	// Parse the header
	headerPath := filepath.Join(assetsDir, "..", "cache", "goplugin.h")
	funcs, err := emitter.ParseCGOHeader(headerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cgo header: %w", err)
	}

	if c.Verbose {
		fmt.Printf("wolf: generated goplugin.a with %d exported functions\n", len(funcs))
		for _, f := range funcs {
			fmt.Printf("  - %s %s(%v)\n", f.ReturnType, f.Name, f.Params)
		}
	}

	return funcs, nil
}

// copyFile copies src to dst, creating dst if needed.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
