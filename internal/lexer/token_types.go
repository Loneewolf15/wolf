// Package lexer implements the Wolf language lexer (tokenizer).
// It reads UTF-8 source and emits a slice of typed Tokens with
// accurate line and column tracking.
package lexer

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Special tokens
	TOKEN_ILLEGAL TokenType = iota
	TOKEN_EOF

	// Literals
	TOKEN_DOLLAR_IDENT // $myVar
	TOKEN_IDENT        // bareIdentifier (for types, function names, class names)
	TOKEN_STRING       // "hello"
	TOKEN_INT          // 42
	TOKEN_FLOAT        // 3.14
	TOKEN_BOOL_TRUE    // true
	TOKEN_BOOL_FALSE   // false
	TOKEN_NIL          // nil

	// String interpolation segments
	TOKEN_STRING_INTERP_START // opening "... before {$
	TOKEN_STRING_INTERP_END   // closing ...after }..."
	TOKEN_STRING_LITERAL_PART // literal text segment inside interpolated string

	// Keywords
	TOKEN_FUNC     // func
	TOKEN_CLASS    // class
	TOKEN_NEW      // new
	TOKEN_IF       // if
	TOKEN_ELSE     // else
	TOKEN_ELSE_IF  // else if (parsed as two tokens, combined by parser)
	TOKEN_FOR      // for
	TOKEN_FOREACH  // foreach
	TOKEN_WHILE    // while
	TOKEN_MATCH    // match
	TOKEN_RETURN   // return
	TOKEN_ASYNC    // async
	TOKEN_AWAIT    // await
	TOKEN_PARALLEL // parallel
	TOKEN_CHANNEL  // channel
	TOKEN_SEND     // send
	TOKEN_RECEIVE  // receive
	TOKEN_TRY      // try
	TOKEN_CATCH    // catch
	TOKEN_IMPORT   // import
	TOKEN_PRINT    // print
	TOKEN_VAR      // var
	TOKEN_PRIVATE  // private
	TOKEN_PUBLIC   // public
	TOKEN_STATIC   // static
	TOKEN_AS       // as
	TOKEN_IN       // in
	TOKEN_ERROR    // error

	// ML block
	TOKEN_AT_ML    // @ml
	TOKEN_AT_MODEL // @ml model (parsed as @ml + optional model(...))

	// Operators — Arithmetic
	TOKEN_PLUS      // +
	TOKEN_MINUS     // -
	TOKEN_STAR      // *
	TOKEN_SLASH     // /
	TOKEN_MODULO    // %
	TOKEN_INCREMENT // ++
	TOKEN_DECREMENT // --

	// Operators — Comparison
	TOKEN_EQUAL         // ==
	TOKEN_NOT_EQUAL     // !=
	TOKEN_LESS          // <
	TOKEN_LESS_EQUAL    // <=
	TOKEN_GREATER       // >
	TOKEN_GREATER_EQUAL // >=

	// Operators — Logical
	TOKEN_AND // &&
	TOKEN_OR  // ||
	TOKEN_NOT // !

	// Operators — Assignment
	TOKEN_ASSIGN       // =
	TOKEN_PLUS_ASSIGN  // +=
	TOKEN_MINUS_ASSIGN // -=
	TOKEN_STAR_ASSIGN  // *=
	TOKEN_SLASH_ASSIGN // /=

	// Operators — Arrow
	TOKEN_ARROW     // ->
	TOKEN_FAT_ARROW // =>

	// Operators — Misc
	TOKEN_DOT          // .
	TOKEN_DOT_DOT      // .. (string concat, PHP style)
	TOKEN_QUESTION     // ?
	TOKEN_COLON        // :
	TOKEN_DOUBLE_COLON // ::

	// Delimiters
	TOKEN_COMMA     // ,
	TOKEN_SEMICOLON // ;
	TOKEN_HASH      // # (comment marker — typically consumed by lexer)
	TOKEN_AT        // @ (standalone)

	// Brackets
	TOKEN_LPAREN   // (
	TOKEN_RPAREN   // )
	TOKEN_LBRACE   // {
	TOKEN_RBRACE   // }
	TOKEN_LBRACKET // [
	TOKEN_RBRACKET // ]

	// Dollar brace for string interpolation expressions
	TOKEN_DOLLAR_LBRACE // ${
	TOKEN_INTERP_LBRACE // {$ inside string interpolation
)

// tokenTypeNames provides human-readable names for token types.
var tokenTypeNames = map[TokenType]string{
	TOKEN_ILLEGAL:             "ILLEGAL",
	TOKEN_EOF:                 "EOF",
	TOKEN_DOLLAR_IDENT:        "DOLLAR_IDENT",
	TOKEN_IDENT:               "IDENT",
	TOKEN_STRING:              "STRING",
	TOKEN_INT:                 "INT",
	TOKEN_FLOAT:               "FLOAT",
	TOKEN_BOOL_TRUE:           "TRUE",
	TOKEN_BOOL_FALSE:          "FALSE",
	TOKEN_NIL:                 "NIL",
	TOKEN_STRING_INTERP_START: "STRING_INTERP_START",
	TOKEN_STRING_INTERP_END:   "STRING_INTERP_END",
	TOKEN_STRING_LITERAL_PART: "STRING_LITERAL_PART",
	TOKEN_FUNC:                "FUNC",
	TOKEN_CLASS:               "CLASS",
	TOKEN_NEW:                 "NEW",
	TOKEN_IF:                  "IF",
	TOKEN_ELSE:                "ELSE",
	TOKEN_ELSE_IF:             "ELSE_IF",
	TOKEN_FOR:                 "FOR",
	TOKEN_FOREACH:             "FOREACH",
	TOKEN_WHILE:               "WHILE",
	TOKEN_MATCH:               "MATCH",
	TOKEN_RETURN:              "RETURN",
	TOKEN_ASYNC:               "ASYNC",
	TOKEN_AWAIT:               "AWAIT",
	TOKEN_PARALLEL:            "PARALLEL",
	TOKEN_CHANNEL:             "CHANNEL",
	TOKEN_SEND:                "SEND",
	TOKEN_RECEIVE:             "RECEIVE",
	TOKEN_TRY:                 "TRY",
	TOKEN_CATCH:               "CATCH",
	TOKEN_IMPORT:              "IMPORT",
	TOKEN_PRINT:               "PRINT",
	TOKEN_VAR:                 "VAR",
	TOKEN_PRIVATE:             "PRIVATE",
	TOKEN_PUBLIC:              "PUBLIC",
	TOKEN_STATIC:              "STATIC",
	TOKEN_AS:                  "AS",
	TOKEN_IN:                  "IN",
	TOKEN_ERROR:               "ERROR",
	TOKEN_AT_ML:               "AT_ML",
	TOKEN_AT_MODEL:            "AT_MODEL",
	TOKEN_PLUS:                "PLUS",
	TOKEN_MINUS:               "MINUS",
	TOKEN_STAR:                "STAR",
	TOKEN_SLASH:               "SLASH",
	TOKEN_MODULO:              "MODULO",
	TOKEN_INCREMENT:           "INCREMENT",
	TOKEN_DECREMENT:           "DECREMENT",
	TOKEN_EQUAL:               "EQUAL",
	TOKEN_NOT_EQUAL:           "NOT_EQUAL",
	TOKEN_LESS:                "LESS",
	TOKEN_LESS_EQUAL:          "LESS_EQUAL",
	TOKEN_GREATER:             "GREATER",
	TOKEN_GREATER_EQUAL:       "GREATER_EQUAL",
	TOKEN_AND:                 "AND",
	TOKEN_OR:                  "OR",
	TOKEN_NOT:                 "NOT",
	TOKEN_ASSIGN:              "ASSIGN",
	TOKEN_PLUS_ASSIGN:         "PLUS_ASSIGN",
	TOKEN_MINUS_ASSIGN:        "MINUS_ASSIGN",
	TOKEN_STAR_ASSIGN:         "STAR_ASSIGN",
	TOKEN_SLASH_ASSIGN:        "SLASH_ASSIGN",
	TOKEN_ARROW:               "ARROW",
	TOKEN_FAT_ARROW:           "FAT_ARROW",
	TOKEN_DOT:                 "DOT",
	TOKEN_DOT_DOT:             "DOT_DOT",
	TOKEN_QUESTION:            "QUESTION",
	TOKEN_COLON:               "COLON",
	TOKEN_DOUBLE_COLON:        "DOUBLE_COLON",
	TOKEN_COMMA:               "COMMA",
	TOKEN_SEMICOLON:           "SEMICOLON",
	TOKEN_HASH:                "HASH",
	TOKEN_AT:                  "AT",
	TOKEN_LPAREN:              "LPAREN",
	TOKEN_RPAREN:              "RPAREN",
	TOKEN_LBRACE:              "LBRACE",
	TOKEN_RBRACE:              "RBRACE",
	TOKEN_LBRACKET:            "LBRACKET",
	TOKEN_RBRACKET:            "RBRACKET",
	TOKEN_DOLLAR_LBRACE:       "DOLLAR_LBRACE",
	TOKEN_INTERP_LBRACE:       "INTERP_LBRACE",
}

// String returns the human-readable name of the token type.
func (t TokenType) String() string {
	if name, ok := tokenTypeNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

// keywords maps reserved words to their token types.
var keywords = map[string]TokenType{
	"func":     TOKEN_FUNC,
	"class":    TOKEN_CLASS,
	"new":      TOKEN_NEW,
	"if":       TOKEN_IF,
	"else":     TOKEN_ELSE,
	"for":      TOKEN_FOR,
	"foreach":  TOKEN_FOREACH,
	"while":    TOKEN_WHILE,
	"match":    TOKEN_MATCH,
	"return":   TOKEN_RETURN,
	"async":    TOKEN_ASYNC,
	"await":    TOKEN_AWAIT,
	"parallel": TOKEN_PARALLEL,
	"channel":  TOKEN_CHANNEL,
	"send":     TOKEN_SEND,
	"receive":  TOKEN_RECEIVE,
	"try":      TOKEN_TRY,
	"catch":    TOKEN_CATCH,
	"import":   TOKEN_IMPORT,
	"print":    TOKEN_PRINT,
	"var":      TOKEN_VAR,
	"private":  TOKEN_PRIVATE,
	"public":   TOKEN_PUBLIC,
	"static":   TOKEN_STATIC,
	"true":     TOKEN_BOOL_TRUE,
	"false":    TOKEN_BOOL_FALSE,
	"nil":      TOKEN_NIL,
	"as":       TOKEN_AS,
	"in":       TOKEN_IN,
	"error":    TOKEN_ERROR,
}

// LookupIdent checks whether an identifier is a keyword.
// Returns the keyword token type if found, TOKEN_IDENT otherwise.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TOKEN_IDENT
}
