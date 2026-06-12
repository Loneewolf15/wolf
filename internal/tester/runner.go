package tester

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wolflang/wolf/internal/compiler"
	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
)

func Run(projectRoot string) error {
	var testFiles []string
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && (info.Name() == ".git" || info.Name() == ".wolf_modules" || info.Name() == "wolf_out") {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".wolf") && !strings.HasSuffix(info.Name(), "_test_runner.wolf") {
			testFiles = append(testFiles, path)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan for tests: %w", err)
	}

	if len(testFiles) == 0 {
		fmt.Println("wolf test: no test files found")
		return nil
	}

	type testFunc struct {
		file string
		name string
	}
	var testFuncs []testFunc

	for _, file := range testFiles {
		source, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		l := lexer.New(string(source), filepath.Base(file))
		tokens, errs := l.Tokenize()
		if len(errs) > 0 {
			return fmt.Errorf("lex error in %s", file)
		}
		p := parser.New(tokens, filepath.Base(file))
		ast, errs := p.Parse()
		if len(errs) > 0 {
			return fmt.Errorf("parse error in %s", file)
		}

		if strings.HasSuffix(file, "_test.wolf") {
			for _, stmt := range ast.Statements {
				if fd, ok := stmt.(*parser.FuncDecl); ok {
					if strings.HasPrefix(fd.Name, "test_") || strings.Contains(fd.Name, "::test_") || strings.Contains(fd.Name, "_test_") {
						relPath, _ := filepath.Rel(projectRoot, file)
						testFuncs = append(testFuncs, testFunc{
							file: relPath,
							name: fd.Name,
						})
					}
				}
			}
		}
	}

	if len(testFuncs) == 0 {
		fmt.Println("wolf test: no test functions (starting with test_) found")
		return nil
	}

	// Generate runner file
	runnerContent := `
var $passed: int = 0
var $failed: int = 0

func assert($cond: bool, $msg: string) {
    if !$cond {
        error($msg)
    }
}
`
	// Append all test files contents directly
	for _, tf := range testFiles {
		source, err := os.ReadFile(tf)
		if err == nil {
			runnerContent += "\n// === FROM " + filepath.Base(tf) + " ===\n"
			runnerContent += string(source)
			runnerContent += "\n"
		}
	}

	runnerContent += "\n"

	for _, tf := range testFuncs {
		runnerContent += fmt.Sprintf(`try {
    %s()
    print("✅ %s")
    $passed = $passed + 1
} catch ($e) {
    print("❌ %s - FAILED: " .. $e)
    $failed = $failed + 1
}
`, tf.name, tf.name, tf.name)
	}

	runnerContent += `
print("\nTests: " .. $passed .. " passed, " .. $failed .. " failed")
if $failed > 0 {
    wolf_system_exit(1)
}
`

	runnerPath := filepath.Join(projectRoot, "_wolf_test_runner.wolf")
	if err := os.WriteFile(runnerPath, []byte(runnerContent), 0644); err != nil {
		return fmt.Errorf("failed to write test runner: %w", err)
	}
	defer os.Remove(runnerPath)

	fmt.Printf("wolf test: Compiling %d tests...\n", len(testFuncs))
	startBuild := time.Now()

	c := compiler.New()
	c.ProjectRoot = projectRoot
	c.Verbose = false
	// Avoid caching issues
	c.OutDir = filepath.Join(projectRoot, "wolf_out_test_runner")
	// defer os.RemoveAll(c.OutDir)

	result, err := c.Build(runnerContent, runnerPath)
	if err != nil {
		return fmt.Errorf("test compilation failed:\n%v\n%s", err, strings.Join(result.Errors, "\n"))
	}

	fmt.Printf("wolf test: Running tests (compiled in %v)...\n\n", time.Since(startBuild))
	
	startRun := time.Now()
	cmd := exec.Command(result.OutputPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	elapsed := time.Since(startRun)

	fmt.Print(stdout.String())
	if stderr.Len() > 0 {
		fmt.Fprintln(os.Stderr, "STDERR:", stderr.String())
	}

	fmt.Printf("\nTotal Time: %v\n", elapsed)

	if runErr != nil {
		return fmt.Errorf("tests failed")
	}

	return nil
}
