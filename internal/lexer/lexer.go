package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lexer tokenizes Wolf source code into a stream of tokens.
type Lexer struct {
	source   string   // the raw source code
	file     string   // source file name
	tokens   []Token  // accumulated tokens
	errors   []*WolfError // accumulated errors

	start   int  // start position of the current token
	current int  // current read position
	line    int  // current line number (1-based)
	col     int  // current column number (1-based)
	startLine int // line at start of current token
	startCol  int // col at start of current token
}

// New creates a new Lexer for the given source file.
func New(source, file string) *Lexer {
	return &Lexer{
		source: source,
		file:   file,
		line:   1,
		col:    1,
	}
}

// Tokenize scans the entire source and returns the token slice and any errors.
func (l *Lexer) Tokenize() ([]Token, []*WolfError) {
	for !l.isAtEnd() {
		l.start = l.current
		l.startLine = l.line
		l.startCol = l.col
		l.scanToken()
	}

	l.tokens = append(l.tokens, Token{
		Type:    TOKEN_EOF,
		Literal: "",
		Pos:     Position{File: l.file, Line: l.line, Col: l.col, Offset: l.current},
	})

	return l.tokens, l.errors
}

// Errors returns accumulated lexer errors.
func (l *Lexer) Errors() []*WolfError {
	return l.errors
}

// --- Core scanning ---

func (l *Lexer) scanToken() {
	ch := l.advance()

	switch ch {
	// Single-character tokens
	case '(':
		l.addToken(TOKEN_LPAREN)
	case ')':
		l.addToken(TOKEN_RPAREN)
	case '{':
		l.addToken(TOKEN_LBRACE)
	case '}':
		l.addToken(TOKEN_RBRACE)
	case '[':
		l.addToken(TOKEN_LBRACKET)
	case ']':
		l.addToken(TOKEN_RBRACKET)
	case ',':
		l.addToken(TOKEN_COMMA)
	case ';':
		l.addToken(TOKEN_SEMICOLON)
	case '?':
		l.addToken(TOKEN_QUESTION)
	case '%':
		l.addToken(TOKEN_MODULO)

	// One or two character tokens
	case '+':
		if l.match('+') {
			l.addToken(TOKEN_INCREMENT)
		} else if l.match('=') {
			l.addToken(TOKEN_PLUS_ASSIGN)
		} else {
			l.addToken(TOKEN_PLUS)
		}
	case '-':
		if l.match('-') {
			l.addToken(TOKEN_DECREMENT)
		} else if l.match('=') {
			l.addToken(TOKEN_MINUS_ASSIGN)
		} else if l.match('>') {
			l.addToken(TOKEN_ARROW)
		} else {
			l.addToken(TOKEN_MINUS)
		}
	case '*':
		if l.match('=') {
			l.addToken(TOKEN_STAR_ASSIGN)
		} else {
			l.addToken(TOKEN_STAR)
		}
	case '/':
		if l.match('=') {
			l.addToken(TOKEN_SLASH_ASSIGN)
		} else {
			l.addToken(TOKEN_SLASH)
		}
	case '=':
		if l.match('=') {
			l.addToken(TOKEN_EQUAL)
		} else if l.match('>') {
			l.addToken(TOKEN_FAT_ARROW)
		} else {
			l.addToken(TOKEN_ASSIGN)
		}
	case '!':
		if l.match('=') {
			l.addToken(TOKEN_NOT_EQUAL)
		} else {
			l.addToken(TOKEN_NOT)
		}
	case '<':
		if l.match('=') {
			l.addToken(TOKEN_LESS_EQUAL)
		} else {
			l.addToken(TOKEN_LESS)
		}
	case '>':
		if l.match('=') {
			l.addToken(TOKEN_GREATER_EQUAL)
		} else {
			l.addToken(TOKEN_GREATER)
		}
	case '&':
		if l.match('&') {
			l.addToken(TOKEN_AND)
		} else {
			l.addError("unexpected character '&', did you mean '&&'?")
		}
	case '|':
		if l.match('|') {
			l.addToken(TOKEN_OR)
		} else {
			l.addError("unexpected character '|', did you mean '||'?")
		}
	case ':':
		if l.match(':') {
			l.addToken(TOKEN_DOUBLE_COLON)
		} else {
			l.addToken(TOKEN_COLON)
		}
	case '.':
		if l.match('.') {
			l.addToken(TOKEN_DOT_DOT)
		} else {
			l.addToken(TOKEN_DOT)
		}

	// Comment — # to end of line
	case '#':
		l.scanComment()

	// String literal
	case '"':
		l.scanString()

	// Dollar variable
	case '$':
		l.scanDollarIdent()

	// At sign — @ml or standalone @
	case '@':
		l.scanAtKeyword()

	// Whitespace — skip
	case ' ', '\t', '\r':
		// ignore

	// Newline
	case '\n':
		// line/col already advanced by advance()

	default:
		if isDigit(ch) {
			l.scanNumber()
		} else if isAlpha(ch) {
			l.scanIdentifier()
		} else {
			l.addError("unexpected character '%c'", ch)
		}
	}
}

// --- Scanning helpers ---

// scanComment consumes a # comment to the end of the line.
func (l *Lexer) scanComment() {
	for !l.isAtEnd() && l.peek() != '\n' {
		l.advance()
	}
	// Comments are discarded — no token emitted
}

// scanString handles double-quoted strings with interpolation support.
// Simple strings (no interpolation) emit a single TOKEN_STRING.
// Strings with {$var} or ${expr} emit interpolation segment tokens.
func (l *Lexer) scanString() {
	var hasInterpolation bool

	// First pass: check if interpolation exists
	saved := l.current
	savedLine := l.line
	savedCol := l.col
	for !l.isAtEnd() {
		ch := l.peek()
		if ch == '"' {
			break
		}
		if ch == '\\' {
			l.advance()
			if !l.isAtEnd() {
				l.advance()
			}
			continue
		}
		if ch == '{' && !l.isAtEnd() {
			next := l.peekNext()
			if next == '$' {
				hasInterpolation = true
				break
			}
		}
		if ch == '$' && !l.isAtEnd() {
			next := l.peekNext()
			if next == '{' {
				hasInterpolation = true
				break
			}
		}
		l.advance()
	}

	// Restore position
	l.current = saved
	l.line = savedLine
	l.col = savedCol

	if hasInterpolation {
		l.scanInterpolatedString()
	} else {
		l.scanSimpleString()
	}
}

// scanSimpleString handles a plain double-quoted string (no interpolation).
func (l *Lexer) scanSimpleString() {
	var sb strings.Builder

	for !l.isAtEnd() {
		ch := l.peek()
		if ch == '"' {
			break
		}
		if ch == '\n' {
			l.line++
			l.col = 1
		}
		if ch == '\\' {
			l.advance()
			if l.isAtEnd() {
				l.addError("unterminated string — unexpected end of file after backslash")
				return
			}
			escaped := l.advance()
			switch escaped {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '{':
				sb.WriteByte('{')
			case '$':
				sb.WriteByte('$')
			default:
				sb.WriteByte('\\')
				sb.WriteRune(escaped)
			}
			continue
		}
		sb.WriteRune(l.advance())
	}

	if l.isAtEnd() {
		l.addError("unterminated string — expected closing '\"'")
		return
	}

	l.advance() // consume closing "

	l.tokens = append(l.tokens, Token{
		Type:    TOKEN_STRING,
		Literal: sb.String(),
		Pos:     Position{File: l.file, Line: l.startLine, Col: l.startCol, Offset: l.start},
	})
}

// scanInterpolatedString handles strings containing {$var} or ${expr} segments.
// Emits: STRING_INTERP_START, then alternating STRING_LITERAL_PART and variable/expr tokens,
// then STRING_INTERP_END.
func (l *Lexer) scanInterpolatedString() {
	l.tokens = append(l.tokens, Token{
		Type:    TOKEN_STRING_INTERP_START,
		Literal: "\"",
		Pos:     Position{File: l.file, Line: l.startLine, Col: l.startCol, Offset: l.start},
	})

	var sb strings.Builder

	for !l.isAtEnd() {
		ch := l.peek()

		if ch == '"' {
			// End of string — emit any remaining literal part
			if sb.Len() > 0 {
				l.emitStringLitPart(sb.String())
				sb.Reset()
			}
			l.advance() // consume closing "
			l.tokens = append(l.tokens, Token{
				Type:    TOKEN_STRING_INTERP_END,
				Literal: "\"",
				Pos:     Position{File: l.file, Line: l.line, Col: l.col - 1, Offset: l.current - 1},
			})
			return
		}

		if ch == '\\' {
			l.advance()
			if l.isAtEnd() {
				l.addError("unterminated string — unexpected end of file after backslash")
				return
			}
			escaped := l.advance()
			switch escaped {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '{':
				sb.WriteByte('{')
			case '$':
				sb.WriteByte('$')
			default:
				sb.WriteByte('\\')
				sb.WriteRune(escaped)
			}
			continue
		}

		// {$var} interpolation
		if ch == '{' && l.peekNext() == '$' {
			if sb.Len() > 0 {
				l.emitStringLitPart(sb.String())
				sb.Reset()
			}
			interpStartPos := Position{File: l.file, Line: l.line, Col: l.col, Offset: l.current}
			l.advance() // consume {
			l.advance() // consume $
			l.tokens = append(l.tokens, Token{
				Type:    TOKEN_INTERP_LBRACE,
				Literal: "{$",
				Pos:     interpStartPos,
			})
			// Scan the variable name or expression after $
			l.scanInterpContent()
			continue
		}

		// ${expr} interpolation
		if ch == '$' && l.peekNext() == '{' {
			if sb.Len() > 0 {
				l.emitStringLitPart(sb.String())
				sb.Reset()
			}
			interpStartPos := Position{File: l.file, Line: l.line, Col: l.col, Offset: l.current}
			l.advance() // consume $
			l.advance() // consume {
			l.tokens = append(l.tokens, Token{
				Type:    TOKEN_DOLLAR_LBRACE,
				Literal: "${",
				Pos:     interpStartPos,
			})
			// Scan expression tokens until }
			l.scanInterpExpression()
			continue
		}

		if ch == '\n' {
			l.line++
			l.col = 1
		}
		sb.WriteRune(l.advance())
	}

	l.addError("unterminated interpolated string — expected closing '\"'")
}

// scanInterpContent scans a variable name after {$ inside string interpolation.
// After the variable name, it expects a closing }.
func (l *Lexer) scanInterpContent() {
	// We're right after {$ — scan the identifier part
	varStart := l.current
	varLine := l.line
	varCol := l.col

	for !l.isAtEnd() && (isAlphaNumeric(l.peek()) || l.peek() == '_') {
		l.advance()
	}

	varName := l.source[varStart:l.current]
	if len(varName) == 0 {
		l.addError("expected variable name after '{$' in string interpolation")
		return
	}

	l.tokens = append(l.tokens, Token{
		Type:    TOKEN_DOLLAR_IDENT,
		Literal: "$" + varName,
		Pos:     Position{File: l.file, Line: varLine, Col: varCol, Offset: varStart},
	})

	// Handle optional ->property access chains inside interpolation
	for !l.isAtEnd() && l.peek() == '-' && l.peekNext() == '>' {
		arrowPos := Position{File: l.file, Line: l.line, Col: l.col, Offset: l.current}
		l.advance() // consume -
		l.advance() // consume >
		l.tokens = append(l.tokens, Token{
			Type:    TOKEN_ARROW,
			Literal: "->",
			Pos:     arrowPos,
		})

		propStart := l.current
		propLine := l.line
		propCol := l.col
		for !l.isAtEnd() && (isAlphaNumeric(l.peek()) || l.peek() == '_') {
			l.advance()
		}
		propName := l.source[propStart:l.current]
		if len(propName) > 0 {
			l.tokens = append(l.tokens, Token{
				Type:    TOKEN_IDENT,
				Literal: propName,
				Pos:     Position{File: l.file, Line: propLine, Col: propCol, Offset: propStart},
			})
		}
	}

	// Expect closing }
	if !l.isAtEnd() && l.peek() == '}' {
		l.advance()
		l.tokens = append(l.tokens, Token{
			Type:    TOKEN_RBRACE,
			Literal: "}",
			Pos:     Position{File: l.file, Line: l.line, Col: l.col - 1, Offset: l.current - 1},
		})
	} else {
		l.addError("expected '}' to close string interpolation")
	}
}

// scanInterpExpression scans expression tokens inside ${...} until matching }.
func (l *Lexer) scanInterpExpression() {
	depth := 1
	for !l.isAtEnd() && depth > 0 {
		l.start = l.current
		l.startLine = l.line
		l.startCol = l.col

		ch := l.peek()
		if ch == '}' {
			depth--
			if depth == 0 {
				l.advance()
				l.tokens = append(l.tokens, Token{
					Type:    TOKEN_RBRACE,
					Literal: "}",
					Pos:     Position{File: l.file, Line: l.line, Col: l.col - 1, Offset: l.current - 1},
				})
				return
			}
		}
		if ch == '{' {
			depth++
		}
		// Re-use the main scanToken to tokenize the expression
		l.scanToken()
	}

	if depth > 0 {
		l.addError("unterminated expression interpolation — expected '}'")
	}
}

// emitStringLitPart emits a STRING_LITERAL_PART token.
func (l *Lexer) emitStringLitPart(text string) {
	l.tokens = append(l.tokens, Token{
		Type:    TOKEN_STRING_LITERAL_PART,
		Literal: text,
		Pos:     Position{File: l.file, Line: l.line, Col: l.col, Offset: l.current},
	})
}

// scanDollarIdent handles $ followed by an identifier.
func (l *Lexer) scanDollarIdent() {
	// Check for ${
	if !l.isAtEnd() && l.peek() == '{' {
		l.advance() // consume {
		l.addToken(TOKEN_DOLLAR_LBRACE)
		return
	}

	if l.isAtEnd() || (!isAlpha(l.peek()) && l.peek() != '_') {
		l.addError("expected identifier after '$'")
		return
	}

	for !l.isAtEnd() && (isAlphaNumeric(l.peek()) || l.peek() == '_') {
		l.advance()
	}

	literal := l.source[l.start:l.current]
	l.tokens = append(l.tokens, Token{
		Type:    TOKEN_DOLLAR_IDENT,
		Literal: literal,
		Pos:     Position{File: l.file, Line: l.startLine, Col: l.startCol, Offset: l.start},
	})
}

// scanAtKeyword handles @ followed by "ml" or other identifiers.
func (l *Lexer) scanAtKeyword() {
	if l.isAtEnd() || !isAlpha(l.peek()) {
		l.addToken(TOKEN_AT)
		return
	}

	wordStart := l.current
	for !l.isAtEnd() && isAlphaNumeric(l.peek()) {
		l.advance()
	}

	word := l.source[wordStart:l.current]
	if word == "ml" {
		l.tokens = append(l.tokens, Token{
			Type:    TOKEN_AT_ML,
			Literal: "@ml",
			Pos:     Position{File: l.file, Line: l.startLine, Col: l.startCol, Offset: l.start},
		})
	} else {
		// Unknown @ keyword — emit as AT + IDENT
		l.tokens = append(l.tokens, Token{
			Type:    TOKEN_AT,
			Literal: "@",
			Pos:     Position{File: l.file, Line: l.startLine, Col: l.startCol, Offset: l.start},
		})
		l.tokens = append(l.tokens, Token{
			Type:    TOKEN_IDENT,
			Literal: word,
			Pos:     Position{File: l.file, Line: l.startLine, Col: l.startCol + 1, Offset: wordStart},
		})
	}
}

// scanNumber handles integer and float literals.
func (l *Lexer) scanNumber() {
	for !l.isAtEnd() && isDigit(l.peek()) {
		l.advance()
	}

	// Check for float
	if !l.isAtEnd() && l.peek() == '.' && isDigit(l.peekNext()) {
		l.advance() // consume .
		for !l.isAtEnd() && isDigit(l.peek()) {
			l.advance()
		}
		l.addToken(TOKEN_FLOAT)
		return
	}

	l.addToken(TOKEN_INT)
}

// scanIdentifier handles bare identifiers (keywords, type names, function names).
func (l *Lexer) scanIdentifier() {
	for !l.isAtEnd() && (isAlphaNumeric(l.peek()) || l.peek() == '_') {
		l.advance()
	}

	text := l.source[l.start:l.current]
	tokType := LookupIdent(text)

	l.tokens = append(l.tokens, Token{
		Type:    tokType,
		Literal: text,
		Pos:     Position{File: l.file, Line: l.startLine, Col: l.startCol, Offset: l.start},
	})
}

// --- Character reading ---

// advance consumes and returns the current rune, advancing the position.
func (l *Lexer) advance() rune {
	if l.isAtEnd() {
		return 0
	}
	r, size := utf8.DecodeRuneInString(l.source[l.current:])
	l.current += size
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}

// peek returns the current rune without consuming it.
func (l *Lexer) peek() rune {
	if l.isAtEnd() {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.source[l.current:])
	return r
}

// peekNext returns the rune after the current one without consuming.
func (l *Lexer) peekNext() rune {
	if l.isAtEnd() {
		return 0
	}
	_, size := utf8.DecodeRuneInString(l.source[l.current:])
	if l.current+size >= len(l.source) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.source[l.current+size:])
	return r
}

// match checks if the current character matches expected and consumes it if so.
func (l *Lexer) match(expected rune) bool {
	if l.isAtEnd() {
		return false
	}
	r, _ := utf8.DecodeRuneInString(l.source[l.current:])
	if r != expected {
		return false
	}
	l.advance()
	return true
}

// isAtEnd returns true if we have consumed all source characters.
func (l *Lexer) isAtEnd() bool {
	return l.current >= len(l.source)
}

// --- Token emission ---

// addToken emits a token of the given type with the current token text.
func (l *Lexer) addToken(tokenType TokenType) {
	text := l.source[l.start:l.current]
	l.tokens = append(l.tokens, Token{
		Type:    tokenType,
		Literal: text,
		Pos:     Position{File: l.file, Line: l.startLine, Col: l.startCol, Offset: l.start},
	})
}

// addError records a lexer error at the current token start position.
func (l *Lexer) addError(format string, args ...interface{}) {
	pos := Position{File: l.file, Line: l.startLine, Col: l.startCol, Offset: l.start}
	l.errors = append(l.errors, NewLexerErrorf(pos, format, args...))
	// Emit an ILLEGAL token so the parser can see something went wrong
	l.tokens = append(l.tokens, Token{
		Type:    TOKEN_ILLEGAL,
		Literal: l.source[l.start:l.current],
		Pos:     pos,
	})
}

// --- Character classification ---

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isAlpha(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || unicode.IsLetter(ch)
}

func isAlphaNumeric(ch rune) bool {
	return isAlpha(ch) || isDigit(ch)
}
