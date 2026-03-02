package lexer

import "fmt"

// WolfError represents a standardized Wolf compiler error.
// Format: wolf: error at line X, col Y: <message>
// This format is machine-parseable and reusable across all compiler phases.
type WolfError struct {
	Pos     Position
	Message string
	Phase   string // "lexer", "parser", "resolver", etc.
}

// Error implements the error interface.
func (e *WolfError) Error() string {
	if e.Pos.File != "" {
		return fmt.Sprintf("wolf: %s error at %s:%d, col %d: %s",
			e.Phase, e.Pos.File, e.Pos.Line, e.Pos.Col, e.Message)
	}
	return fmt.Sprintf("wolf: %s error at line %d, col %d: %s",
		e.Phase, e.Pos.Line, e.Pos.Col, e.Message)
}

// NewLexerError creates a new error for the lexer phase.
func NewLexerError(pos Position, message string) *WolfError {
	return &WolfError{
		Pos:     pos,
		Message: message,
		Phase:   "lexer",
	}
}

// NewLexerErrorf creates a new formatted error for the lexer phase.
func NewLexerErrorf(pos Position, format string, args ...interface{}) *WolfError {
	return &WolfError{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
		Phase:   "lexer",
	}
}
