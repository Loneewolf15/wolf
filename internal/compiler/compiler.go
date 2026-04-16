// Package compiler orchestrates the full Wolf compilation pipeline.
// Wolf compiles to native machine code via LLVM IR.
package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wolflang/wolf/internal/config"
	"github.com/wolflang/wolf/internal/emitter"
	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
	"github.com/wolflang/wolf/internal/resolver"
	"github.com/wolflang/wolf/internal/typechecker"
)

// Compiler orchestrates the full Wolf compilation pipeline.
type Compiler struct {
	StrictMode bool
	OutDir     string // default: wolf_out/
	Verbose    bool
	Config     *config.WolfConfig // loaded from wolf.config + env vars
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
		OutDir:     cfg.Build.OutDir,
		StrictMode: cfg.Build.StrictMode,
		Config:     cfg,
	}, nil
}

// CompileResult holds the output of a compilation.
type CompileResult struct {
	LLVMSource string   // generated LLVM IR
	OutputPath string   // path to compiled binary
	Errors     []string // compilation errors (human-readable)
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
	projectRoot := filepath.Dir(filename)
	discoveredASTs, err := c.AutoDiscover(projectRoot)
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

	fmt.Printf(">> Phase 6: Emit LLVM\n")
	// Phase 6: Emit LLVM IR (WIR → .ll)
	llvmEmit := emitter.NewLLVMEmitter()
	llvmEmit.TargetTriple = detectTargetTriple()
	llvmSource := llvmEmit.Emit(irProgram)
	result.LLVMSource = llvmSource

	fmt.Printf(">> Done returning\n")
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

	// Find wolf runtime
	wolfRoot, err := findWolfRoot()
	if err != nil {
		return result, err
	}
	runtimeC := filepath.Join(wolfRoot, "runtime", "wolf_runtime.c")

	// Compile LLVM IR to object file
	objFile := filepath.Join(outDir, baseName+".o")
	compiled := false

	// Strategy 1: Use LLC if available
	var compileErrors []string

	fmt.Printf(">> Testing hasLLC\n")
	if hasLLC() {
		fmt.Printf(">> Running llc\n")
		llcCmd := exec.Command("llc", "-filetype=obj", "-relocation-model=pic", "-o", objFile, llFile)
		if out, err := llcCmd.CombinedOutput(); err != nil {
			compileErrors = append(compileErrors, fmt.Sprintf("llc error: %s\n%s", err, string(out)))
			if c.Verbose {
				fmt.Printf("wolf: llc failed: %s\n%s\n", err, string(out))
			}
		} else {
			compiled = true
			if c.Verbose {
				fmt.Printf("wolf: compiled .ll → .o via llc\n")
			}
		}
		fmt.Printf(">> llc finished\n")
	}

	fmt.Printf(">> Checking compiled\n")
	// Strategy 2: Use clang to compile .ll directly
	if !compiled && hasClang() {
		fmt.Printf(">> Running clang (ll -> o)\n")
		clangCmd := exec.Command("clang", "-c", "-O2", "-o", objFile, llFile)
		if out, err := clangCmd.CombinedOutput(); err != nil {
			compileErrors = append(compileErrors, fmt.Sprintf("clang error: %s\n%s", err, string(out)))
			if c.Verbose {
				fmt.Printf("wolf: clang .ll compilation failed: %s\n%s\n", err, string(out))
			}
		} else {
			compiled = true
			if c.Verbose {
				fmt.Printf("wolf: compiled .ll → .o via clang\n")
			}
		}
	}

	// Strategy 3: Use llvm-as + llc pipeline
	if !compiled {
		fmt.Printf(">> Running llvm-as\n")
		bcFile := filepath.Join(outDir, baseName+".bc")
		llvmAsCmd := exec.Command("llvm-as", "-o", bcFile, llFile)
		if asOut, err := llvmAsCmd.CombinedOutput(); err == nil {
			llcCmd := exec.Command("llc", "-filetype=obj", "-relocation-model=pic", "-o", objFile, bcFile)
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

	// Detect DB client flags
	dbCflags := ""
	dbLibs := ""

	driver := "mysql"
	if c.Config != nil && c.Config.DB.Driver != "" {
		driver = c.Config.DB.Driver
	}

	if driver == "mysql" {
		dbLibs = "-lmysqlclient"
		for _, mysqlConfig := range []string{"/opt/lampp/bin/mysql_config", "/usr/local/mysql/bin/mysql_config", "mysql_config"} {
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

	// Always link hiredis for Redis support
	redisLibs := "-lhiredis"

	// Production-grade library discovery via bundled static libs or pkg-config
	wolfRoot, _ = findWolfRoot()
	staticDir := filepath.Join(wolfRoot, "third_party", "lib")

	// Platform-specific static paths
	var bundledPath string
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		bundledPath = filepath.Join(staticDir, "linux_x64")
	} else if runtime.GOOS == "darwin" {
		bundledPath = filepath.Join(staticDir, "macos")
	}

	useStatic := false
	if bundledPath != "" {
		if _, err := os.Stat(filepath.Join(bundledPath, "libcrypto.a")); err == nil {
			useStatic = true
		}
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
	rtArgs := []string{"-c", optFlag}

	// DB include flags
	if dbCflags != "" {
		rtArgs = append(rtArgs, strings.Fields(dbCflags)...)
	}
	if cryptoCflags != "" {
		rtArgs = append(rtArgs, strings.Fields(cryptoCflags)...)
	}

	// Enable real Redis implementation
	rtArgs = append(rtArgs, "-DWOLF_REDIS_ENABLED")

	// Bake wolf.config values into the runtime as -D constants.
	// This is how pool size, timeouts, credentials, and server limits
	// reach wolf_runtime.c without needing a config file at runtime.
	rtArgs = append(rtArgs, c.configCFlags()...)

	if os.Getenv("WOLF_DEBUG") != "" {
		rtArgs = append(rtArgs, "-DWOLF_DEBUG")
	}

	rtArgs = append(rtArgs, "-o", runtimeObj, runtimeC)
	rtCmd := exec.Command(cc, rtArgs...)
	if out, err := rtCmd.CombinedOutput(); err != nil {
		return result, fmt.Errorf("failed to compile wolf runtime: %s\n%s", err, string(out))
	}

	if c.Verbose {
		fmt.Printf("wolf: compiled wolf_runtime.c → %s\n", runtimeObj)
	}

	// Link everything into final binary
	binaryPath := filepath.Join(outDir, baseName)
	linkArgs := []string{"-o", binaryPath, objFile, runtimeObj, "-lpthread"}

	// Prioritize system libraries to avoid XAMPP version conflicts
	if prioritizedPath != "" {
		linkArgs = append(linkArgs, prioritizedPath)
	}

	linkArgs = append(linkArgs, strings.Fields(dbLibs)...)
	linkArgs = append(linkArgs, strings.Fields(redisLibs)...)
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
		linkArgs = append(linkArgs, "-lm", "-lcurl")
	}

	linkCmd := exec.Command(cc, linkArgs...)
	if out, err := linkCmd.CombinedOutput(); err != nil {
		return result, fmt.Errorf("linking failed: %s\n%s", err, string(out))
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

	return []string{
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

func findWolfRoot() (string, error) {
	exePath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exePath)
		runtimePath := filepath.Join(dir, "runtime", "wolf_runtime.c")
		if _, err := os.Stat(runtimePath); err == nil {
			return dir, nil
		}
		parentDir := filepath.Dir(dir)
		runtimePath = filepath.Join(parentDir, "runtime", "wolf_runtime.c")
		if _, err := os.Stat(runtimePath); err == nil {
			return parentDir, nil
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		runtimePath := filepath.Join(cwd, "runtime", "wolf_runtime.c")
		if _, err := os.Stat(runtimePath); err == nil {
			return cwd, nil
		}
		parentDir := filepath.Dir(cwd)
		runtimePath = filepath.Join(parentDir, "runtime", "wolf_runtime.c")
		if _, err := os.Stat(runtimePath); err == nil {
			return parentDir, nil
		}
	}

	return "", fmt.Errorf("could not find wolf runtime directory (looked for runtime/wolf_runtime.c) in %s", cwd)
}

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
