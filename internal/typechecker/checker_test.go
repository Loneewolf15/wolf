package typechecker

import (
	"testing"

	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
	"github.com/wolflang/wolf/internal/resolver"
)

// --- Test Helpers ---

func checkSource(t *testing.T, source string) []*lexer.WolfError {
	t.Helper()
	l := lexer.New(source, "test.wolf")
	tokens, _ := l.Tokenize()
	p := parser.New(tokens, "test.wolf")
	program, _ := p.Parse()
	r := resolver.New("test.wolf")
	r.Resolve(program)
	c := New(r, "test.wolf")
	return c.Check(program)
}

func checkStrict(t *testing.T, source string) []*lexer.WolfError {
	t.Helper()
	l := lexer.New(source, "test.wolf")
	tokens, _ := l.Tokenize()
	p := parser.New(tokens, "test.wolf")
	program, _ := p.Parse()
	r := resolver.New("test.wolf")
	r.SetStrictMode(true)
	r.Resolve(program)
	c := New(r, "test.wolf")
	c.SetStrictMode(true)
	return c.Check(program)
}

// --- Dynamic Mode (default) ---

func TestDynamicNoErrors(t *testing.T) {
	errs := checkSource(t, `$x = 42
$name = "Wolf"
$active = true
$items = [1, 2, 3]`)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("Unexpected error: %s", e.Error())
		}
	}
}

func TestDynamicFunctionNoErrors(t *testing.T) {
	errs := checkSource(t, `func greet($name) {
		return "Hello"
	}`)
	if len(errs) > 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}
}

func TestDynamicClassNoErrors(t *testing.T) {
	errs := checkSource(t, `class Driver {
		$name: string
		$rating: float
		func __construct($name, $rating) {
			$this->name = $name
		}
	}`)
	if len(errs) > 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}
}

// --- Strict Mode ---

func TestStrictTypeMismatch(t *testing.T) {
	errs := checkStrict(t, `var $x: int = "hello"`)
	if len(errs) == 0 {
		t.Error("Expected type mismatch error in strict mode")
	}
}

func TestStrictTypeMatch(t *testing.T) {
	errs := checkStrict(t, `var $x: int = 42`)
	// Filter only typechecker errors (not resolver errors)
	var tcErrs []*lexer.WolfError
	for _, e := range errs {
		if e.Phase == "typechecker" {
			tcErrs = append(tcErrs, e)
		}
	}
	if len(tcErrs) > 0 {
		for _, e := range tcErrs {
			t.Errorf("Unexpected type error: %s", e.Error())
		}
	}
}

func TestStrictFloatType(t *testing.T) {
	errs := checkStrict(t, `var $price: float = 9.99`)
	var tcErrs []*lexer.WolfError
	for _, e := range errs {
		if e.Phase == "typechecker" {
			tcErrs = append(tcErrs, e)
		}
	}
	if len(tcErrs) > 0 {
		t.Errorf("Expected no type errors, got %d", len(tcErrs))
	}
}

func TestStrictStringType(t *testing.T) {
	errs := checkStrict(t, `var $name: string = "Wolf"`)
	var tcErrs []*lexer.WolfError
	for _, e := range errs {
		if e.Phase == "typechecker" {
			tcErrs = append(tcErrs, e)
		}
	}
	if len(tcErrs) > 0 {
		t.Errorf("Expected no type errors, got %d", len(tcErrs))
	}
}

func TestStrictBoolType(t *testing.T) {
	errs := checkStrict(t, `var $active: bool = true`)
	var tcErrs []*lexer.WolfError
	for _, e := range errs {
		if e.Phase == "typechecker" {
			tcErrs = append(tcErrs, e)
		}
	}
	if len(tcErrs) > 0 {
		t.Errorf("Expected no type errors, got %d", len(tcErrs))
	}
}

// --- Type Inference ---

func TestInferInt(t *testing.T) {
	c := New(nil, "test.wolf")
	typ := c.inferType(&parser.IntLiteral{Value: "42"})
	if typ != TypeInt {
		t.Errorf("Expected TypeInt, got %s", typ)
	}
}

func TestInferFloat(t *testing.T) {
	c := New(nil, "test.wolf")
	typ := c.inferType(&parser.FloatLiteral{Value: "3.14"})
	if typ != TypeFloat {
		t.Errorf("Expected TypeFloat, got %s", typ)
	}
}

func TestInferString(t *testing.T) {
	c := New(nil, "test.wolf")
	typ := c.inferType(&parser.StringLiteral{Value: "hello"})
	if typ != TypeString {
		t.Errorf("Expected TypeString, got %s", typ)
	}
}

func TestInferBool(t *testing.T) {
	c := New(nil, "test.wolf")
	typ := c.inferType(&parser.BoolLiteral{Value: true})
	if typ != TypeBool {
		t.Errorf("Expected TypeBool, got %s", typ)
	}
}

func TestInferNil(t *testing.T) {
	c := New(nil, "test.wolf")
	typ := c.inferType(&parser.NilLiteral{})
	if typ != TypeNil {
		t.Errorf("Expected TypeNil, got %s", typ)
	}
}

func TestInferArray(t *testing.T) {
	c := New(nil, "test.wolf")
	typ := c.inferType(&parser.ArrayLiteral{})
	if typ != TypeArray {
		t.Errorf("Expected TypeArray, got %s", typ)
	}
}

func TestInferMap(t *testing.T) {
	c := New(nil, "test.wolf")
	typ := c.inferType(&parser.MapLiteral{})
	if typ != TypeMap {
		t.Errorf("Expected TypeMap, got %s", typ)
	}
}

func TestInferDynamic(t *testing.T) {
	c := New(nil, "test.wolf")
	typ := c.inferType(&parser.DollarIdent{Name: "$x"})
	if typ != TypeAny {
		t.Errorf("Expected TypeAny for dynamic, got %s", typ)
	}
}

// --- Type Names ---

func TestTypeString(t *testing.T) {
	cases := map[WolfType]string{
		TypeInt:     "int",
		TypeFloat:   "float",
		TypeString:  "string",
		TypeBool:    "bool",
		TypeNil:     "nil",
		TypeAny:     "any",
		TypeArray:   "array",
		TypeMap:     "map",
		TypeFunc:    "func",
		TypeChannel: "channel",
		TypeError:   "error",
	}
	for typ, expected := range cases {
		if typ.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, typ.String())
		}
	}
}

// --- Complex Programs ---

func TestFullProgramNoErrors(t *testing.T) {
	errs := checkSource(t, `
$name = "Wolf"
$count = 0

func greet($name) {
	return "Hello"
}

class Driver {
	$name: string
	$rating: float
	func __construct($name, $rating) {
		$this->name = $name
		$this->rating = $rating
	}
}

$driver = new Driver("Ade", 4.8)

if $count > 0 {
	print("items")
} else {
	print("empty")
}

for $i = 0; $i < 10; $i++ {
	print($i)
}

foreach $items as $item {
	print($item)
}

try {
	$user = fetchUser($id)
} catch ($e) {
	print($e)
}

[$data, $err] = parseInput($raw)
`)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("Unexpected error: %s", e.Error())
		}
	}
}
