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
}

// New creates a new Compiler with defaults.
func New() *Compiler {
	return &Compiler{
		OutDir: "wolf_out",
	}
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

	// Phase 1: Lex
	l := lexer.New(source, filename)
	tokens, lexErrors := l.Tokenize()
	if len(lexErrors) > 0 {
		for _, e := range lexErrors {
			result.Errors = append(result.Errors, e.Error())
		}
		return result, fmt.Errorf("lexer errors: %d errors found", len(lexErrors))
	}

	if c.Verbose {
		fmt.Printf("wolf: lexed %d tokens\n", len(tokens))
	}

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

	if c.Verbose {
		fmt.Printf("wolf: parsed %d top-level statements from main file\n", len(program.Statements))
	}

	// Phase 2.5: Auto-Discover TraversyMVC Libraries and Controllers
	projectRoot := filepath.Dir(filename)
	discoveredASTs, err := c.AutoDiscover(projectRoot)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, fmt.Errorf("autodiscovery failed: %w", err)
	}

	for _, ast := range discoveredASTs {
		program.Statements = append(program.Statements, ast.Statements...)
	}
	
	// Generate the __compiler_dispatch_controller method based on all discovered classes
	dispatchFunc := generateDispatcherAST(program)
	if dispatchFunc != nil {
		program.Statements = append(program.Statements, dispatchFunc)
	}

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

	// Phase 4: Type Check
	tc := typechecker.New(res, filename)
	tc.SetStrictMode(c.StrictMode)
	typeErrors := tc.Check(program)
	if len(typeErrors) > 0 {
		for _, e := range typeErrors {
			result.Errors = append(result.Errors, e.Error())
		}
		return result, fmt.Errorf("type errors: %d errors found", len(typeErrors))
	}

	// Phase 5: Emit WIR (AST → Wolf IR)
	irEmit := emitter.New(res)
	irProgram := irEmit.Emit(program)

	// Phase 6: Emit LLVM IR (WIR → .ll)
	llvmEmit := emitter.NewLLVMEmitter()
	llvmSource := llvmEmit.Emit(irProgram)
	result.LLVMSource = llvmSource

	if c.Verbose {
		fmt.Println("wolf: LLVM IR generated successfully")
	}

	return result, nil
}

// Build compiles a Wolf source file to a native binary via LLVM.
func (c *Compiler) Build(source, filename string) (*CompileResult, error) {
	result, err := c.Compile(source, filename)
	if err != nil {
		return result, err
	}

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
	}

	// Write LLVM IR
	llFile := filepath.Join(outDir, baseName+".ll")
	if err := os.WriteFile(llFile, []byte(result.LLVMSource), 0644); err != nil {
		return result, fmt.Errorf("failed to write LLVM IR: %w", err)
	}

	if c.Verbose {
		fmt.Printf("wolf: wrote LLVM IR → %s\n", llFile)
	}

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
	if hasLLC() {
		llcCmd := exec.Command("llc", "-filetype=obj", "-relocation-model=pic", "-o", objFile, llFile)
		if out, err := llcCmd.CombinedOutput(); err != nil {
			if c.Verbose {
				fmt.Printf("wolf: llc failed: %s\n%s\n", err, string(out))
			}
		} else {
			compiled = true
			if c.Verbose {
				fmt.Printf("wolf: compiled .ll → .o via llc\n")
			}
		}
	}

	// Strategy 2: Use clang to compile .ll directly
	if !compiled && hasClang() {
		clangCmd := exec.Command("clang", "-c", "-O2", "-o", objFile, llFile)
		if out, err := clangCmd.CombinedOutput(); err != nil {
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
		bcFile := filepath.Join(outDir, baseName+".bc")
		llvmAsCmd := exec.Command("llvm-as", "-o", bcFile, llFile)
		if _, err := llvmAsCmd.CombinedOutput(); err == nil {
			llcCmd := exec.Command("llc", "-filetype=obj", "-relocation-model=pic", "-o", objFile, bcFile)
			if _, err := llcCmd.CombinedOutput(); err == nil {
				compiled = true
			}
		}
	}

	if !compiled {
		return result, fmt.Errorf("LLVM compilation failed: neither llc nor clang found.\n  Install LLVM: sudo apt-get install clang llvm")
	}

	// Compile wolf runtime
	runtimeObj := filepath.Join(outDir, "wolf_runtime.o")
	rtCmd := exec.Command(cc, "-c", "-O2", "-o", runtimeObj, runtimeC)
	if out, err := rtCmd.CombinedOutput(); err != nil {
		return result, fmt.Errorf("failed to compile wolf runtime: %s\n%s", err, string(out))
	}

	if c.Verbose {
		fmt.Printf("wolf: compiled wolf_runtime.c → %s\n", runtimeObj)
	}

	// Link everything into final binary
	binaryPath := filepath.Join(outDir, baseName)
	linkArgs := []string{"-o", binaryPath, objFile, runtimeObj, "-lpthread"}

	if runtime.GOOS == "linux" {
		linkArgs = append(linkArgs, "-lm")
	}

	linkCmd := exec.Command(cc, linkArgs...)
	if out, err := linkCmd.CombinedOutput(); err != nil {
		return result, fmt.Errorf("linking failed: %s\n%s", err, string(out))
	}

	result.OutputPath = binaryPath

	if c.Verbose {
		fmt.Printf("wolf: linked → %s\n", binaryPath)
	}

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

// ========== Helpers ==========

// findWolfRoot locates the wolf compiler's root directory to find the runtime.
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
		// When running tests from a subdirectory like `e2e/` or `internal/`
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
