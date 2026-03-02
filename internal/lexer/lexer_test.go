package lexer

import (
	"testing"
)

// --- Test Helpers ---

func tokenize(t *testing.T, source string) []Token {
	t.Helper()
	l := New(source, "test.wolf")
	tokens, errs := l.Tokenize()
	if len(errs) > 0 {
		for _, e := range errs {
			t.Logf("Lexer error: %s", e.Error())
		}
	}
	return tokens
}

func tokenizeExpectNoErrors(t *testing.T, source string) []Token {
	t.Helper()
	l := New(source, "test.wolf")
	tokens, errs := l.Tokenize()
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("Unexpected lexer error: %s", e.Error())
		}
	}
	return tokens
}

func tokenizeExpectErrors(t *testing.T, source string) ([]Token, []*WolfError) {
	t.Helper()
	l := New(source, "test.wolf")
	return l.Tokenize()
}

func assertToken(t *testing.T, tok Token, expectedType TokenType, expectedLiteral string) {
	t.Helper()
	if tok.Type != expectedType {
		t.Errorf("Expected token type %s, got %s (literal=%q)", expectedType, tok.Type, tok.Literal)
	}
	if tok.Literal != expectedLiteral {
		t.Errorf("Expected literal %q, got %q (type=%s)", expectedLiteral, tok.Literal, tok.Type)
	}
}

func assertTokenType(t *testing.T, tok Token, expectedType TokenType) {
	t.Helper()
	if tok.Type != expectedType {
		t.Errorf("Expected token type %s, got %s (literal=%q)", expectedType, tok.Type, tok.Literal)
	}
}

// --- Empty & EOF ---

func TestEmptyFile(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "")
	if len(tokens) != 1 {
		t.Fatalf("Expected 1 token (EOF), got %d", len(tokens))
	}
	assertToken(t, tokens[0], TOKEN_EOF, "")
}

func TestWhitespaceOnly(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "   \t\t\n\n  \r\n  ")
	if len(tokens) != 1 {
		t.Fatalf("Expected 1 token (EOF), got %d", len(tokens))
	}
	assertToken(t, tokens[0], TOKEN_EOF, "")
}

// --- Dollar Variables ---

func TestDollarIdent(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "$name")
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$name")
}

func TestDollarIdentUnderscore(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "$_private")
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$_private")
}

func TestDollarIdentWithNumbers(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "$var123")
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$var123")
}

func TestMultipleDollarIdents(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "$a = $b + $c")
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$a")
	assertToken(t, tokens[1], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[2], TOKEN_DOLLAR_IDENT, "$b")
	assertToken(t, tokens[3], TOKEN_PLUS, "+")
	assertToken(t, tokens[4], TOKEN_DOLLAR_IDENT, "$c")
}

// --- Simple Strings ---

func TestSimpleString(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, `"hello world"`)
	assertToken(t, tokens[0], TOKEN_STRING, "hello world")
}

func TestEmptyString(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, `""`)
	assertToken(t, tokens[0], TOKEN_STRING, "")
}

func TestStringEscapeSequences(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, `"hello\nworld\t!"`)
	assertToken(t, tokens[0], TOKEN_STRING, "hello\nworld\t!")
}

func TestStringEscapedQuote(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, `"say \"hi\""`)
	assertToken(t, tokens[0], TOKEN_STRING, `say "hi"`)
}

func TestStringEscapedBackslash(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, `"path\\to\\file"`)
	assertToken(t, tokens[0], TOKEN_STRING, `path\to\file`)
}

// --- String Interpolation ---

func TestStringInterpolationSimple(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, `"Hello {$name}"`)
	// Expected: INTERP_START, "Hello ", INTERP_LBRACE, $name, RBRACE, INTERP_END
	assertTokenType(t, tokens[0], TOKEN_STRING_INTERP_START)
	assertToken(t, tokens[1], TOKEN_STRING_LITERAL_PART, "Hello ")
	assertTokenType(t, tokens[2], TOKEN_INTERP_LBRACE)
	assertToken(t, tokens[3], TOKEN_DOLLAR_IDENT, "$name")
	assertTokenType(t, tokens[4], TOKEN_RBRACE)
	assertTokenType(t, tokens[5], TOKEN_STRING_INTERP_END)
}

func TestStringInterpolationExpression(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, `"Total: ${$price * $qty}"`)
	assertTokenType(t, tokens[0], TOKEN_STRING_INTERP_START)
	assertToken(t, tokens[1], TOKEN_STRING_LITERAL_PART, "Total: ")
	assertTokenType(t, tokens[2], TOKEN_DOLLAR_LBRACE)
	assertToken(t, tokens[3], TOKEN_DOLLAR_IDENT, "$price")
	assertTokenType(t, tokens[4], TOKEN_STAR)
	assertToken(t, tokens[5], TOKEN_DOLLAR_IDENT, "$qty")
	assertTokenType(t, tokens[6], TOKEN_RBRACE)
	assertTokenType(t, tokens[7], TOKEN_STRING_INTERP_END)
}

func TestStringInterpolationPropertyAccess(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, `"Driver {$driver->name}"`)
	assertTokenType(t, tokens[0], TOKEN_STRING_INTERP_START)
	assertToken(t, tokens[1], TOKEN_STRING_LITERAL_PART, "Driver ")
	assertTokenType(t, tokens[2], TOKEN_INTERP_LBRACE)
	assertToken(t, tokens[3], TOKEN_DOLLAR_IDENT, "$driver")
	assertTokenType(t, tokens[4], TOKEN_ARROW)
	assertToken(t, tokens[5], TOKEN_IDENT, "name")
	assertTokenType(t, tokens[6], TOKEN_RBRACE)
	assertTokenType(t, tokens[7], TOKEN_STRING_INTERP_END)
}

// --- Integers ---

func TestInteger(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "42")
	assertToken(t, tokens[0], TOKEN_INT, "42")
}

func TestIntegerZero(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "0")
	assertToken(t, tokens[0], TOKEN_INT, "0")
}

func TestIntegerLarge(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "1234567890")
	assertToken(t, tokens[0], TOKEN_INT, "1234567890")
}

// --- Floats ---

func TestFloat(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "3.14")
	assertToken(t, tokens[0], TOKEN_FLOAT, "3.14")
}

func TestFloatLeadingZero(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "0.5")
	assertToken(t, tokens[0], TOKEN_FLOAT, "0.5")
}

// --- Booleans & Nil ---

func TestBoolTrue(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "true")
	assertToken(t, tokens[0], TOKEN_BOOL_TRUE, "true")
}

func TestBoolFalse(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "false")
	assertToken(t, tokens[0], TOKEN_BOOL_FALSE, "false")
}

func TestNil(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "nil")
	assertToken(t, tokens[0], TOKEN_NIL, "nil")
}

// --- Keywords ---

func TestAllKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"func", TOKEN_FUNC},
		{"class", TOKEN_CLASS},
		{"new", TOKEN_NEW},
		{"if", TOKEN_IF},
		{"else", TOKEN_ELSE},
		{"for", TOKEN_FOR},
		{"foreach", TOKEN_FOREACH},
		{"while", TOKEN_WHILE},
		{"match", TOKEN_MATCH},
		{"return", TOKEN_RETURN},
		{"async", TOKEN_ASYNC},
		{"await", TOKEN_AWAIT},
		{"parallel", TOKEN_PARALLEL},
		{"channel", TOKEN_CHANNEL},
		{"send", TOKEN_SEND},
		{"receive", TOKEN_RECEIVE},
		{"try", TOKEN_TRY},
		{"catch", TOKEN_CATCH},
		{"import", TOKEN_IMPORT},
		{"print", TOKEN_PRINT},
		{"var", TOKEN_VAR},
		{"private", TOKEN_PRIVATE},
		{"public", TOKEN_PUBLIC},
		{"static", TOKEN_STATIC},
		{"as", TOKEN_AS},
		{"in", TOKEN_IN},
		{"error", TOKEN_ERROR},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := tokenizeExpectNoErrors(t, tt.input)
			assertToken(t, tokens[0], tt.expected, tt.input)
		})
	}
}

// --- Operators ---

func TestArithmeticOperators(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "+ - * / %")
	assertToken(t, tokens[0], TOKEN_PLUS, "+")
	assertToken(t, tokens[1], TOKEN_MINUS, "-")
	assertToken(t, tokens[2], TOKEN_STAR, "*")
	assertToken(t, tokens[3], TOKEN_SLASH, "/")
	assertToken(t, tokens[4], TOKEN_MODULO, "%")
}

func TestIncrementDecrement(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "++ --")
	assertToken(t, tokens[0], TOKEN_INCREMENT, "++")
	assertToken(t, tokens[1], TOKEN_DECREMENT, "--")
}

func TestComparisonOperators(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "== != < <= > >=")
	assertToken(t, tokens[0], TOKEN_EQUAL, "==")
	assertToken(t, tokens[1], TOKEN_NOT_EQUAL, "!=")
	assertToken(t, tokens[2], TOKEN_LESS, "<")
	assertToken(t, tokens[3], TOKEN_LESS_EQUAL, "<=")
	assertToken(t, tokens[4], TOKEN_GREATER, ">")
	assertToken(t, tokens[5], TOKEN_GREATER_EQUAL, ">=")
}

func TestLogicalOperators(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "&& || !")
	assertToken(t, tokens[0], TOKEN_AND, "&&")
	assertToken(t, tokens[1], TOKEN_OR, "||")
	assertToken(t, tokens[2], TOKEN_NOT, "!")
}

func TestAssignmentOperators(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "= += -= *= /=")
	assertToken(t, tokens[0], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[1], TOKEN_PLUS_ASSIGN, "+=")
	assertToken(t, tokens[2], TOKEN_MINUS_ASSIGN, "-=")
	assertToken(t, tokens[3], TOKEN_STAR_ASSIGN, "*=")
	assertToken(t, tokens[4], TOKEN_SLASH_ASSIGN, "/=")
}

func TestArrowOperators(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "-> =>")
	assertToken(t, tokens[0], TOKEN_ARROW, "->")
	assertToken(t, tokens[1], TOKEN_FAT_ARROW, "=>")
}

func TestDotOperators(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, ". ..")
	assertToken(t, tokens[0], TOKEN_DOT, ".")
	assertToken(t, tokens[1], TOKEN_DOT_DOT, "..")
}

func TestColonOperators(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, ": ::")
	assertToken(t, tokens[0], TOKEN_COLON, ":")
	assertToken(t, tokens[1], TOKEN_DOUBLE_COLON, "::")
}

// --- Delimiters & Brackets ---

func TestBrackets(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "( ) { } [ ]")
	assertToken(t, tokens[0], TOKEN_LPAREN, "(")
	assertToken(t, tokens[1], TOKEN_RPAREN, ")")
	assertToken(t, tokens[2], TOKEN_LBRACE, "{")
	assertToken(t, tokens[3], TOKEN_RBRACE, "}")
	assertToken(t, tokens[4], TOKEN_LBRACKET, "[")
	assertToken(t, tokens[5], TOKEN_RBRACKET, "]")
}

func TestDelimiters(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, ", ; ?")
	assertToken(t, tokens[0], TOKEN_COMMA, ",")
	assertToken(t, tokens[1], TOKEN_SEMICOLON, ";")
	assertToken(t, tokens[2], TOKEN_QUESTION, "?")
}

// --- @ml ---

func TestAtMl(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "@ml")
	assertToken(t, tokens[0], TOKEN_AT_ML, "@ml")
}

func TestAtUnknown(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "@something")
	assertToken(t, tokens[0], TOKEN_AT, "@")
	assertToken(t, tokens[1], TOKEN_IDENT, "something")
}

// --- Comments ---

func TestComment(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "# this is a comment\n$x")
	// Comment is discarded, only $x and EOF remain
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$x")
	assertToken(t, tokens[1], TOKEN_EOF, "")
}

func TestCommentAtEndOfLine(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "$a = 5 # assign five")
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$a")
	assertToken(t, tokens[1], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[2], TOKEN_INT, "5")
	assertToken(t, tokens[3], TOKEN_EOF, "")
}

// --- Identifiers (bare) ---

func TestBareIdentifier(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "myFunction")
	assertToken(t, tokens[0], TOKEN_IDENT, "myFunction")
}

func TestIdentifierWithUnderscore(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "my_function")
	assertToken(t, tokens[0], TOKEN_IDENT, "my_function")
}

// --- Line & Column tracking ---

func TestLineTracking(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "$a\n$b\n$c")
	if tokens[0].Pos.Line != 1 {
		t.Errorf("Expected $a on line 1, got line %d", tokens[0].Pos.Line)
	}
	if tokens[1].Pos.Line != 2 {
		t.Errorf("Expected $b on line 2, got line %d", tokens[1].Pos.Line)
	}
	if tokens[2].Pos.Line != 3 {
		t.Errorf("Expected $c on line 3, got line %d", tokens[2].Pos.Line)
	}
}

func TestColumnTracking(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "$a = $b")
	if tokens[0].Pos.Col != 1 {
		t.Errorf("Expected $a at col 1, got col %d", tokens[0].Pos.Col)
	}
	// $a takes 2 chars, space is col 3, = is col 4
	if tokens[1].Pos.Col != 4 {
		t.Errorf("Expected = at col 4, got col %d", tokens[1].Pos.Col)
	}
}

// --- Complex Programs ---

func TestHelloWorld(t *testing.T) {
	src := `print("Hello from Wolf!")`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_PRINT, "print")
	assertToken(t, tokens[1], TOKEN_LPAREN, "(")
	assertToken(t, tokens[2], TOKEN_STRING, "Hello from Wolf!")
	assertToken(t, tokens[3], TOKEN_RPAREN, ")")
	assertToken(t, tokens[4], TOKEN_EOF, "")
}

func TestVariableAssignment(t *testing.T) {
	src := `$name = "Wolf"
$count = 0
$price = 9.99
$active = true
$items = [1, 2, 3]`
	tokens := tokenizeExpectNoErrors(t, src)

	// $name = "Wolf"
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$name")
	assertToken(t, tokens[1], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[2], TOKEN_STRING, "Wolf")

	// $count = 0
	assertToken(t, tokens[3], TOKEN_DOLLAR_IDENT, "$count")
	assertToken(t, tokens[4], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[5], TOKEN_INT, "0")

	// $price = 9.99
	assertToken(t, tokens[6], TOKEN_DOLLAR_IDENT, "$price")
	assertToken(t, tokens[7], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[8], TOKEN_FLOAT, "9.99")

	// $active = true
	assertToken(t, tokens[9], TOKEN_DOLLAR_IDENT, "$active")
	assertToken(t, tokens[10], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[11], TOKEN_BOOL_TRUE, "true")

	// $items = [1, 2, 3]
	assertToken(t, tokens[12], TOKEN_DOLLAR_IDENT, "$items")
	assertToken(t, tokens[13], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[14], TOKEN_LBRACKET, "[")
	assertToken(t, tokens[15], TOKEN_INT, "1")
	assertToken(t, tokens[16], TOKEN_COMMA, ",")
	assertToken(t, tokens[17], TOKEN_INT, "2")
	assertToken(t, tokens[18], TOKEN_COMMA, ",")
	assertToken(t, tokens[19], TOKEN_INT, "3")
	assertToken(t, tokens[20], TOKEN_RBRACKET, "]")
}

func TestFunctionDeclaration(t *testing.T) {
	src := `func greet($name) {
	return "Hello {$name}"
}`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_FUNC, "func")
	assertToken(t, tokens[1], TOKEN_IDENT, "greet")
	assertToken(t, tokens[2], TOKEN_LPAREN, "(")
	assertToken(t, tokens[3], TOKEN_DOLLAR_IDENT, "$name")
	assertToken(t, tokens[4], TOKEN_RPAREN, ")")
	assertToken(t, tokens[5], TOKEN_LBRACE, "{")
	assertToken(t, tokens[6], TOKEN_RETURN, "return")
	// The string with interpolation
	assertTokenType(t, tokens[7], TOKEN_STRING_INTERP_START)
	assertToken(t, tokens[8], TOKEN_STRING_LITERAL_PART, "Hello ")
	assertTokenType(t, tokens[9], TOKEN_INTERP_LBRACE)
	assertToken(t, tokens[10], TOKEN_DOLLAR_IDENT, "$name")
	assertTokenType(t, tokens[11], TOKEN_RBRACE)
	assertTokenType(t, tokens[12], TOKEN_STRING_INTERP_END)
	assertToken(t, tokens[13], TOKEN_RBRACE, "}")
}

func TestClassDeclaration(t *testing.T) {
	src := `class Driver {
	$name: string
	$rating: float
	func __construct($name, $rating) {
		$this->name = $name
	}
}`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_CLASS, "class")
	assertToken(t, tokens[1], TOKEN_IDENT, "Driver")
	assertToken(t, tokens[2], TOKEN_LBRACE, "{")
	assertToken(t, tokens[3], TOKEN_DOLLAR_IDENT, "$name")
	assertToken(t, tokens[4], TOKEN_COLON, ":")
	assertToken(t, tokens[5], TOKEN_IDENT, "string")
}

func TestForLoop(t *testing.T) {
	src := `for $i = 0; $i < 10; $i++ { print($i) }`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_FOR, "for")
	assertToken(t, tokens[1], TOKEN_DOLLAR_IDENT, "$i")
	assertToken(t, tokens[2], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[3], TOKEN_INT, "0")
	assertToken(t, tokens[4], TOKEN_SEMICOLON, ";")
	assertToken(t, tokens[5], TOKEN_DOLLAR_IDENT, "$i")
	assertToken(t, tokens[6], TOKEN_LESS, "<")
	assertToken(t, tokens[7], TOKEN_INT, "10")
	assertToken(t, tokens[8], TOKEN_SEMICOLON, ";")
	assertToken(t, tokens[9], TOKEN_DOLLAR_IDENT, "$i")
	assertToken(t, tokens[10], TOKEN_INCREMENT, "++")
}

func TestForeachWithKeyValue(t *testing.T) {
	src := `foreach $users as $id => $user { print($id) }`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_FOREACH, "foreach")
	assertToken(t, tokens[1], TOKEN_DOLLAR_IDENT, "$users")
	assertToken(t, tokens[2], TOKEN_AS, "as")
	assertToken(t, tokens[3], TOKEN_DOLLAR_IDENT, "$id")
	assertToken(t, tokens[4], TOKEN_FAT_ARROW, "=>")
	assertToken(t, tokens[5], TOKEN_DOLLAR_IDENT, "$user")
}

func TestMatchStatement(t *testing.T) {
	src := `match $status {
	"active" => handleActive()
	_ => handleDefault()
}`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_MATCH, "match")
	assertToken(t, tokens[1], TOKEN_DOLLAR_IDENT, "$status")
	assertToken(t, tokens[2], TOKEN_LBRACE, "{")
	assertToken(t, tokens[3], TOKEN_STRING, "active")
	assertToken(t, tokens[4], TOKEN_FAT_ARROW, "=>")
	assertToken(t, tokens[5], TOKEN_IDENT, "handleActive")
}

func TestAtMlBlock(t *testing.T) {
	src := `@ml {
	import numpy as np
	result = np.array(items).mean()
}`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_AT_ML, "@ml")
	assertToken(t, tokens[1], TOKEN_LBRACE, "{")
}

func TestAsyncAwait(t *testing.T) {
	src := `$task = async { return fetchDrivers($location) }
$drivers = await $task`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$task")
	assertToken(t, tokens[1], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[2], TOKEN_ASYNC, "async")
	assertToken(t, tokens[3], TOKEN_LBRACE, "{")
}

func TestTryCatch(t *testing.T) {
	src := `try {
	$user = fetchUser($id)
} catch ($e) {
	print($e)
}`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_TRY, "try")
	assertToken(t, tokens[1], TOKEN_LBRACE, "{")
}

func TestStrictTyping(t *testing.T) {
	src := `var $userId: int = 42`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_VAR, "var")
	assertToken(t, tokens[1], TOKEN_DOLLAR_IDENT, "$userId")
	assertToken(t, tokens[2], TOKEN_COLON, ":")
	assertToken(t, tokens[3], TOKEN_IDENT, "int")
	assertToken(t, tokens[4], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[5], TOKEN_INT, "42")
}

func TestArrowFunction(t *testing.T) {
	src := `func double($n) => $n * 2`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_FUNC, "func")
	assertToken(t, tokens[1], TOKEN_IDENT, "double")
	assertToken(t, tokens[2], TOKEN_LPAREN, "(")
	assertToken(t, tokens[3], TOKEN_DOLLAR_IDENT, "$n")
	assertToken(t, tokens[4], TOKEN_RPAREN, ")")
	assertToken(t, tokens[5], TOKEN_FAT_ARROW, "=>")
	assertToken(t, tokens[6], TOKEN_DOLLAR_IDENT, "$n")
	assertToken(t, tokens[7], TOKEN_STAR, "*")
	assertToken(t, tokens[8], TOKEN_INT, "2")
}

func TestMultipleReturnValues(t *testing.T) {
	src := `func divide($a, $b) -> (float, error) {}`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_FUNC, "func")
	assertToken(t, tokens[1], TOKEN_IDENT, "divide")
	assertToken(t, tokens[2], TOKEN_LPAREN, "(")
	assertToken(t, tokens[3], TOKEN_DOLLAR_IDENT, "$a")
	assertToken(t, tokens[4], TOKEN_COMMA, ",")
	assertToken(t, tokens[5], TOKEN_DOLLAR_IDENT, "$b")
	assertToken(t, tokens[6], TOKEN_RPAREN, ")")
	assertToken(t, tokens[7], TOKEN_ARROW, "->")
	assertToken(t, tokens[8], TOKEN_LPAREN, "(")
}

func TestNegativeNumber(t *testing.T) {
	// -7 is tokenized as MINUS + INT, which is semantically negative
	tokens := tokenizeExpectNoErrors(t, "-7")
	assertToken(t, tokens[0], TOKEN_MINUS, "-")
	assertToken(t, tokens[1], TOKEN_INT, "7")
}

// --- Error Cases ---

func TestUnterminatedString(t *testing.T) {
	_, errs := tokenizeExpectErrors(t, `"hello`)
	if len(errs) == 0 {
		t.Error("Expected an error for unterminated string")
	}
}

func TestDollarAlone(t *testing.T) {
	_, errs := tokenizeExpectErrors(t, `$ + 1`)
	if len(errs) == 0 {
		t.Error("Expected an error for standalone $")
	}
}

func TestUnexpectedCharacter(t *testing.T) {
	_, errs := tokenizeExpectErrors(t, "~")
	if len(errs) == 0 {
		t.Error("Expected an error for unexpected character ~")
	}
}

func TestSingleAmpersand(t *testing.T) {
	_, errs := tokenizeExpectErrors(t, "&")
	if len(errs) == 0 {
		t.Error("Expected an error for single &")
	}
}

func TestSinglePipe(t *testing.T) {
	_, errs := tokenizeExpectErrors(t, "|")
	if len(errs) == 0 {
		t.Error("Expected an error for single |")
	}
}

// --- Unicode ---

func TestUnicodeInString(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, `"こんにちは世界"`)
	assertToken(t, tokens[0], TOKEN_STRING, "こんにちは世界")
}

func TestUnicodeIdentifier(t *testing.T) {
	tokens := tokenizeExpectNoErrors(t, "résumé")
	assertToken(t, tokens[0], TOKEN_IDENT, "résumé")
}

// --- Error Message Format ---

func TestErrorFormat(t *testing.T) {
	pos := Position{File: "main.wolf", Line: 4, Col: 12}
	err := NewLexerError(pos, "unexpected character '^'")
	expected := "wolf: lexer error at main.wolf:4, col 12: unexpected character '^'"
	if err.Error() != expected {
		t.Errorf("Error format mismatch.\nExpected: %s\nGot:      %s", expected, err.Error())
	}
}

func TestErrorFormatNoFile(t *testing.T) {
	pos := Position{Line: 7, Col: 3}
	err := NewLexerError(pos, "unterminated string")
	expected := "wolf: lexer error at line 7, col 3: unterminated string"
	if err.Error() != expected {
		t.Errorf("Error format mismatch.\nExpected: %s\nGot:      %s", expected, err.Error())
	}
}

// --- File Position ---

func TestFilePositionInToken(t *testing.T) {
	l := New("$x", "myfile.wolf")
	tokens, _ := l.Tokenize()
	if tokens[0].Pos.File != "myfile.wolf" {
		t.Errorf("Expected file 'myfile.wolf', got '%s'", tokens[0].Pos.File)
	}
}

// --- Database Pattern ---

func TestDatabaseThisPattern(t *testing.T) {
	src := `$this->db->query("SELECT * FROM users")`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$this")
	assertToken(t, tokens[1], TOKEN_ARROW, "->")
	assertToken(t, tokens[2], TOKEN_IDENT, "db")
	assertToken(t, tokens[3], TOKEN_ARROW, "->")
	assertToken(t, tokens[4], TOKEN_IDENT, "query")
	assertToken(t, tokens[5], TOKEN_LPAREN, "(")
	assertToken(t, tokens[6], TOKEN_STRING, "SELECT * FROM users")
	assertToken(t, tokens[7], TOKEN_RPAREN, ")")
}

func TestNewDatabase(t *testing.T) {
	src := `$this->db = new Database`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$this")
	assertToken(t, tokens[1], TOKEN_ARROW, "->")
	assertToken(t, tokens[2], TOKEN_IDENT, "db")
	assertToken(t, tokens[3], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[4], TOKEN_NEW, "new")
	assertToken(t, tokens[5], TOKEN_IDENT, "Database")
}

// --- Channel Operations ---

func TestChannelOperations(t *testing.T) {
	src := `$ch = channel(int)
async { send($ch, computeResult()) }
$value = receive($ch)`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$ch")
	assertToken(t, tokens[1], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[2], TOKEN_CHANNEL, "channel")
}

// --- Parallel Block ---

func TestParallelBlock(t *testing.T) {
	src := `parallel {
	$drivers = fetchDrivers($location)
	$surge = calculateSurge($zone)
}`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_PARALLEL, "parallel")
	assertToken(t, tokens[1], TOKEN_LBRACE, "{")
}

// --- Named Parameters ---

func TestNamedParameters(t *testing.T) {
	src := `createUser(name: "Ada", email: "ada@wolf.dev")`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_IDENT, "createUser")
	assertToken(t, tokens[1], TOKEN_LPAREN, "(")
	assertToken(t, tokens[2], TOKEN_IDENT, "name")
	assertToken(t, tokens[3], TOKEN_COLON, ":")
	assertToken(t, tokens[4], TOKEN_STRING, "Ada")
}

// --- Map / Object Literal ---

func TestMapLiteral(t *testing.T) {
	src := `$user = {"id": 1, "name": "Wolf"}`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_DOLLAR_IDENT, "$user")
	assertToken(t, tokens[1], TOKEN_ASSIGN, "=")
	assertToken(t, tokens[2], TOKEN_LBRACE, "{")
	assertToken(t, tokens[3], TOKEN_STRING, "id")
	assertToken(t, tokens[4], TOKEN_COLON, ":")
	assertToken(t, tokens[5], TOKEN_INT, "1")
}

// --- Error Handling Pattern ---

func TestGoStyleMultipleReturn(t *testing.T) {
	src := `[$data, $err] = parseInput($raw)`
	tokens := tokenizeExpectNoErrors(t, src)
	assertToken(t, tokens[0], TOKEN_LBRACKET, "[")
	assertToken(t, tokens[1], TOKEN_DOLLAR_IDENT, "$data")
	assertToken(t, tokens[2], TOKEN_COMMA, ",")
	assertToken(t, tokens[3], TOKEN_DOLLAR_IDENT, "$err")
	assertToken(t, tokens[4], TOKEN_RBRACKET, "]")
	assertToken(t, tokens[5], TOKEN_ASSIGN, "=")
}
