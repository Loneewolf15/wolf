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
func (c *Compiler) AutoDiscover(projectRoot string) ([]*parser.Program, error) {
	var asts []*parser.Program

	dirsToScan := []string{"libraries", "models", "controllers", "services", "helpers"}

	for _, dir := range dirsToScan {
		fullPath := filepath.Join(projectRoot, dir)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue // Skip if directory doesn't exist
		}

		err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".wolf") {
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
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return asts, nil
}
