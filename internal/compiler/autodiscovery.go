package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
)

// AutoDiscover scans the project for classes in models/, controllers/, and libraries/.
// It parses them into ASTs to be injected into the main program before compilation.
func (c *Compiler) AutoDiscover(projectRoot string, filename string) ([]*parser.Program, error) {
	var asts []*parser.Program

	// Option A: Clean separation. If compiling compiler internals, don't auto-discover application models/controllers.
	isCompilerInternal := strings.Contains(filepath.ToSlash(filename), "src/compiler")
	var dirsToScan []string
	if !isCompilerInternal {
		dirsToScan = []string{"packages", "config", "libraries", "models", "controllers", "services", "helpers", ".wolf_modules"}
	} else {
		// Scan src/compiler when building the compiler
		dirsToScan = []string{"src/compiler"}
	}

	// Determine WOLF_ROOT for standard library discovery
	wolfRoot := os.Getenv("WOLF_ROOT")
	var scanPaths []string

	if wolfRoot != "" {
		stdPath := filepath.Join(wolfRoot, "std")
		if _, err := os.Stat(stdPath); err == nil {
			scanPaths = append(scanPaths, stdPath)
		}
	}

	for _, dir := range dirsToScan {
		fullPath := filepath.Join(projectRoot, dir)
		scanPaths = append(scanPaths, fullPath)
	}

	for _, fullPath := range scanPaths {
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue // Skip if directory doesn't exist
		}

		err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				if strings.HasSuffix(info.Name(), ".wolf") {
					absPath, _ := filepath.Abs(path)
					absFilename, _ := filepath.Abs(filename)
					if absPath == absFilename {
						return nil // Skip the file being compiled to avoid redefinition
					}
					if strings.HasSuffix(info.Name(), "_test.wolf") {
						return nil // Skip test files
					}

					if c.Verbose {
						fmt.Printf("wolf: auto-discovered %s\n", path)
					}

					source, readErr := os.ReadFile(path)
					if readErr != nil {
						return fmt.Errorf("failed to read %s: %w", path, readErr)
					}

					// Lex
					l := lexer.New(string(source), info.Name())
					tokens, lexErrs := l.Tokenize()
					if len(lexErrs) > 0 {
						return fmt.Errorf("lex error in %s: %v", path, lexErrs)
					}

					// Parse
					p := parser.New(tokens, info.Name())
					fileAST, parseErrs := p.Parse()
					if len(parseErrs) > 0 {
						return fmt.Errorf("parse error in %s: %v", path, parseErrs)
					}

					asts = append(asts, fileAST)
				} else if strings.HasSuffix(info.Name(), ".go") {
					if c.Verbose {
						fmt.Printf("wolf: auto-discovered Go plugin %s\n", path)
					}
					c.GoPlugins = append(c.GoPlugins, path)
				}
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return asts, nil
}
