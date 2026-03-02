package resolver

import (
	"testing"

	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
)

// --- Test Helpers ---

func resolveSource(t *testing.T, source string) (*Resolver, *parser.Program) {
	t.Helper()
	l := lexer.New(source, "test.wolf")
	tokens, _ := l.Tokenize()
	p := parser.New(tokens, "test.wolf")
	program, _ := p.Parse()
	r := New("test.wolf")
	r.Resolve(program)
	return r, program
}

func resolveStrict(t *testing.T, source string) (*Resolver, []*lexer.WolfError) {
	t.Helper()
	l := lexer.New(source, "test.wolf")
	tokens, _ := l.Tokenize()
	p := parser.New(tokens, "test.wolf")
	program, _ := p.Parse()
	r := New("test.wolf")
	r.SetStrictMode(true)
	errs := r.Resolve(program)
	return r, errs
}

// --- Scope Tests ---

func TestGlobalScope(t *testing.T) {
	r, _ := resolveSource(t, `$name = "Wolf"`)
	names := r.ResolvedNames()
	if goName, ok := names["$name"]; !ok {
		t.Error("Expected $name to be resolved")
	} else if goName != "name" {
		t.Errorf("Expected Go name 'name', got '%s'", goName)
	}
}

func TestDollarStripping(t *testing.T) {
	r, _ := resolveSource(t, `$firstName = "John"
$_private = true
$count123 = 0`)
	names := r.ResolvedNames()

	cases := map[string]string{
		"$firstName": "firstName",
		"$_private":  "_private",
		"$count123":  "count123",
	}

	for wolfName, expectedGo := range cases {
		if goName, ok := names[wolfName]; !ok {
			t.Errorf("Expected %s to be resolved", wolfName)
		} else if goName != expectedGo {
			t.Errorf("Expected Go name '%s' for %s, got '%s'", expectedGo, wolfName, goName)
		}
	}
}

func TestFunctionScope(t *testing.T) {
	r, _ := resolveSource(t, `func greet($name) {
		$msg = "Hello"
	}`)
	names := r.ResolvedNames()
	if _, ok := names["greet"]; !ok {
		t.Error("Expected 'greet' to be resolved")
	}
	if _, ok := names["$name"]; !ok {
		t.Error("Expected $name to be resolved")
	}
	if _, ok := names["$msg"]; !ok {
		t.Error("Expected $msg to be resolved")
	}
}

func TestClassScope(t *testing.T) {
	r, _ := resolveSource(t, `class Driver {
		$name: string
		func __construct($name) {
			$this->name = $name
		}
	}`)
	names := r.ResolvedNames()
	if _, ok := names["Driver"]; !ok {
		t.Error("Expected 'Driver' to be resolved")
	}
	if _, ok := names["$this"]; !ok {
		t.Error("Expected $this to be resolved (auto-declared in class)")
	}
}

func TestForeachScope(t *testing.T) {
	r, _ := resolveSource(t, `foreach $items as $key => $value {
		print($key)
	}`)
	names := r.ResolvedNames()
	if _, ok := names["$key"]; !ok {
		t.Error("Expected $key to be resolved")
	}
	if _, ok := names["$value"]; !ok {
		t.Error("Expected $value to be resolved")
	}
}

func TestTryCatchScope(t *testing.T) {
	r, _ := resolveSource(t, `try {
		$x = 1
	} catch ($e) {
		print($e)
	}`)
	names := r.ResolvedNames()
	if _, ok := names["$e"]; !ok {
		t.Error("Expected $e to be resolved in catch scope")
	}
}

func TestDestructureScope(t *testing.T) {
	r, _ := resolveSource(t, `[$data, $err] = parseInput($raw)`)
	names := r.ResolvedNames()
	if _, ok := names["$data"]; !ok {
		t.Error("Expected $data to be resolved")
	}
	if _, ok := names["$err"]; !ok {
		t.Error("Expected $err to be resolved")
	}
}

func TestClosureScope(t *testing.T) {
	r, _ := resolveSource(t, `$multiply = func($a, $b) { return $a * $b }`)
	names := r.ResolvedNames()
	if _, ok := names["$a"]; !ok {
		t.Error("Expected $a to be resolved")
	}
	if _, ok := names["$b"]; !ok {
		t.Error("Expected $b to be resolved")
	}
}

func TestBuiltins(t *testing.T) {
	r, _ := resolveSource(t, `print($this)`)
	names := r.ResolvedNames()
	if goName, ok := names["$this"]; !ok {
		t.Error("Expected $this to be a builtin")
	} else if goName != "this" {
		t.Errorf("Expected Go name 'this', got '%s'", goName)
	}
}

// --- Strict Mode Tests ---

func TestStrictModeUndeclaredVar(t *testing.T) {
	_, errs := resolveStrict(t, `print($undeclared)`)
	if len(errs) == 0 {
		t.Error("Expected error for undeclared variable in strict mode")
	}
}

func TestStrictModeDeclaredVarNoError(t *testing.T) {
	_, errs := resolveStrict(t, `var $x: int = 42
print($x)`)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("Unexpected error: %s", e.Error())
		}
	}
}

func TestDynamicModeAutoDeclaration(t *testing.T) {
	r, _ := resolveSource(t, `$x = 42
print($x)`)
	if len(r.Errors()) > 0 {
		t.Error("Dynamic mode should not error on auto-declared variables")
	}
}

// --- ML Block Tests ---

func TestMLBlockOutVars(t *testing.T) {
	r, _ := resolveSource(t, `@ml {
		result = compute()
	}`)
	// In dynamic mode, @ml doesn't produce errors
	if len(r.Errors()) > 0 {
		t.Errorf("Expected no errors, got %d", len(r.Errors()))
	}
}

// --- StripDollar ---

func TestStripDollar(t *testing.T) {
	cases := map[string]string{
		"$name":  "name",
		"$_priv": "_priv",
		"$x":     "x",
		"myFunc": "myFunc",
		"":       "",
	}
	for input, expected := range cases {
		result := stripDollar(input)
		if result != expected {
			t.Errorf("stripDollar(%q) = %q, expected %q", input, result, expected)
		}
	}
}

// --- Nested Scope ---

func TestNestedScope(t *testing.T) {
	r, _ := resolveSource(t, `
		$outer = 1
		if $outer > 0 {
			$inner = 2
			if $inner > 1 {
				$deepest = 3
			}
		}
	`)
	names := r.ResolvedNames()
	for _, name := range []string{"$outer", "$inner", "$deepest"} {
		if _, ok := names[name]; !ok {
			t.Errorf("Expected %s to be resolved", name)
		}
	}
}

func TestForLoopScope(t *testing.T) {
	r, _ := resolveSource(t, `for $i = 0; $i < 10; $i++ { print($i) }`)
	names := r.ResolvedNames()
	if _, ok := names["$i"]; !ok {
		t.Error("Expected $i to be resolved in for loop")
	}
}

func TestMultipleFunctions(t *testing.T) {
	r, _ := resolveSource(t, `
func foo($a) { return $a }
func bar($b) { return $b }
`)
	names := r.ResolvedNames()
	if _, ok := names["foo"]; !ok {
		t.Error("Expected 'foo' to be resolved")
	}
	if _, ok := names["bar"]; !ok {
		t.Error("Expected 'bar' to be resolved")
	}
}
