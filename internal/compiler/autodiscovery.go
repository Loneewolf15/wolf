package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"wolf/internal/lexer"
	"wolf/internal/parser"
)

// AutoDiscover scans the project for dependencies.
// If mainAST has imports, it uses Strict Import Crawling.
// If mainAST has no imports (or is nil), it falls back to Legacy Directory Globbing.
func (c *Compiler) AutoDiscover(projectRoot string, filename string, mainAST *parser.Program) ([]*parser.Program, error) {
	hasImports := false
	if mainAST != nil {
		for _, stmt := range mainAST.Statements {
			if _, ok := stmt.(*parser.ImportStmt); ok {
				hasImports = true
				break
			}
		}
	}

	if hasImports {
		if c.Verbose {
			fmt.Printf("wolf: strict import mode enabled for %s\n", filename)
		}
		return c.crawlImports(projectRoot, filename, mainAST)
	}

	return c.legacyAutoDiscover(projectRoot, filename)
}

func (c *Compiler) crawlImports(projectRoot string, mainFilename string, mainAST *parser.Program) ([]*parser.Program, error) {
	var asts []*parser.Program
	visited := make(map[string]bool)
	importStack := make(map[string]bool)

	absMain, err := filepath.Abs(mainFilename)
	if err != nil {
		return nil, err
	}
	visited[absMain] = true
	importStack[absMain] = true

	var crawl func(program *parser.Program, currentFile string) error
	crawl = func(program *parser.Program, currentFile string) error {
		currentDir := filepath.Dir(currentFile)

		for _, stmt := range program.Statements {
			imp, ok := stmt.(*parser.ImportStmt)
			if !ok {
				continue
			}

			// Resolve path
			resolvedPath := c.resolveImportPath(projectRoot, currentDir, imp.Path)
			absResolved, err := filepath.Abs(resolvedPath)
			if err != nil {
				return err
			}

			// Cycle detection
			if importStack[absResolved] {
				return fmt.Errorf("circular import cycle detected involving: %s", absResolved)
			}
			if visited[absResolved] {
				continue // deduplicate
			}

			// Mark as visiting
			importStack[absResolved] = true

			var source []byte
			if vfsSrc, ok := c.VFS[absResolved]; ok {
				source = []byte(vfsSrc)
			} else {
				var readErr error
				source, readErr = os.ReadFile(absResolved)
				if readErr != nil {
					return fmt.Errorf("failed to read imported file %s: %w", absResolved, readErr)
				}
			}

			if c.Verbose {
				fmt.Printf("wolf: imported %s\n", absResolved)
			}

			l := lexer.New(string(source), filepath.Base(absResolved))
			tokens, lexErrs := l.Tokenize()
			if len(lexErrs) > 0 {
				return fmt.Errorf("lex error in %s: %v", absResolved, lexErrs)
			}

			p := parser.New(tokens, filepath.Base(absResolved))
			fileAST, parseErrs := p.Parse()
			if len(parseErrs) > 0 {
				return fmt.Errorf("parse error in %s: %v", absResolved, parseErrs)
			}

			visited[absResolved] = true
			asts = append(asts, fileAST)

			// Recursively crawl
			if err := crawl(fileAST, absResolved); err != nil {
				return err
			}

			// Unmark visiting
			delete(importStack, absResolved)
		}
		return nil
	}

	if err := crawl(mainAST, absMain); err != nil {
		return nil, err
	}

	return asts, nil
}

func (c *Compiler) resolveImportPath(projectRoot, currentDir, importPath string) string {
	wolfRoot := os.Getenv("WOLF_ROOT")

	if strings.HasPrefix(importPath, "wolf/std/") {
		if wolfRoot != "" {
			stdPath := strings.TrimPrefix(importPath, "wolf/std/")
			return c.ensureWolfExt(filepath.Join(wolfRoot, "std", stdPath))
		}
	} else if strings.HasPrefix(importPath, "github.com/") {
		return c.ensureWolfExt(filepath.Join(projectRoot, ".wolf_modules", importPath))
	} else if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		return c.ensureWolfExt(filepath.Join(currentDir, importPath))
	}

	// Fallback to project root
	return c.ensureWolfExt(filepath.Join(projectRoot, importPath))
}

func (c *Compiler) ensureWolfExt(path string) string {
	// If it's a directory, assume directory/main.wolf or directory/directory.wolf
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		mainWolf := filepath.Join(path, "main.wolf")
		if _, err := os.Stat(mainWolf); err == nil {
			return mainWolf
		}
		dirWolf := filepath.Join(path, filepath.Base(path)+".wolf")
		if _, err := os.Stat(dirWolf); err == nil {
			return dirWolf
		}
	}

	if !strings.HasSuffix(path, ".wolf") {
		return path + ".wolf"
	}
	return path
}

func (c *Compiler) legacyAutoDiscover(projectRoot string, filename string) ([]*parser.Program, error) {
	var asts []*parser.Program

	isCompilerInternal := strings.Contains(filepath.ToSlash(filename), "src/compiler")
	var dirsToScan []string
	if !isCompilerInternal {
		dirsToScan = []string{"packages", "config", "libraries", "models", "controllers", "services", "helpers", ".wolf_modules"}
	} else {
		// Scan projectRoot (.) when building the compiler (e.g. from e2e/ test runner)
		dirsToScan = []string{"."}
	}

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
			continue
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
						return nil
					}
					if strings.HasSuffix(info.Name(), "_test.wolf") {
						return nil
					}
					if info.Name() == "main.wolf" && absPath != absFilename {
						return nil
					}

					if c.Verbose || true {
						fmt.Printf("wolf: auto-discovered %s\n", path)
					}

					var source []byte
					if vfsSrc, ok := c.VFS[absPath]; ok {
						source = []byte(vfsSrc)
					} else {
						var readErr error
						source, readErr = os.ReadFile(absPath)
						if readErr != nil {
							return fmt.Errorf("failed to read %s: %w", absPath, readErr)
						}
					}

					l := lexer.New(string(source), info.Name())
					tokens, lexErrs := l.Tokenize()
					if len(lexErrs) > 0 {
						return fmt.Errorf("lex error in %s: %v", path, lexErrs)
					}

					p := parser.New(tokens, info.Name())
					fileAST, parseErrs := p.Parse()
					if len(parseErrs) > 0 {
						return fmt.Errorf("parse error in %s: %v", path, parseErrs)
					}

					asts = append(asts, fileAST)
				} else if strings.HasSuffix(info.Name(), ".go") {
					if strings.HasSuffix(info.Name(), "_test.go") {
						return nil
					}
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
