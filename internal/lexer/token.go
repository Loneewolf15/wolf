package lexer

import "fmt"

// Position tracks a location in source code.
type Position struct {
	File   string // source file name
	Line   int    // 1-based line number
	Col    int    // 1-based column number
	Offset int    // 0-based byte offset
}

// String returns a human-readable position string.
func (p Position) String() string {
	if p.File != "" {
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Col)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

// Token represents a single lexical token produced by the lexer.
type Token struct {
	Type    TokenType // the type of this token
	Literal string    // the literal text of this token
	Pos     Position  // source position of this token
}

// String returns a human-readable token representation for debugging.
func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, %s)", t.Type, t.Literal, t.Pos)
}
