package compiler

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// ========== STRESS TESTS ==========

// --- Concurrent Compilation ---

func TestConcurrentCompilations(t *testing.T) {
	sources := []string{
		`print("hello")`,
		`$x = 42`,
		`func add($a, $b) -> float { return $a + $b }`,
		`$items = [1, 2, 3]`,
		`if true { print("yes") }`,
		`for $i = 0; $i < 10; $i++ { print($i) }`,
		`$name = "Wolf"
print("Hello {$name}")`,
		`func greet($who) { print("hi") }
greet("world")`,
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(sources)*10)

	for round := 0; round < 10; round++ {
		for _, src := range sources {
			wg.Add(1)
			go func(source string) {
				defer wg.Done()
				c := New()
				result, err := c.Compile(source, "stress.wolf")
				if err != nil {
					errors <- fmt.Errorf("concurrency error: %v — %v", err, result.Errors)
				}
			}(src)
		}
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent compile failed: %v", err)
	}
}

// --- Large Program ---

func TestCompileLargeProgram(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("# Large Wolf program\n")

	// 100 variables
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("$var_%d = %d\n", i, i))
	}

	// 50 functions
	for i := 0; i < 50; i++ {
		sb.WriteString(fmt.Sprintf("func fn_%d($x) -> float {\n  return $x * %d.0\n}\n", i, i+1))
	}

	// 20 print statements with interpolation
	for i := 0; i < 20; i++ {
		sb.WriteString(fmt.Sprintf("print(\"Value %d: {$var_%d}\")\n", i, i))
	}

	src := compileSource(t, sb.String())
	if !strings.Contains(src, "define i32 @main()") {
		t.Error("Large program should compile with main")
	}

	// Should have many function definitions
	fnCount := strings.Count(src, "define")
	if fnCount < 50 {
		t.Errorf("Expected 50+ definitions, got %d", fnCount)
	}
}

// --- Deeply Nested Control Flow ---

func TestDeeplyNestedIf(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("$x = 1\n")
	depth := 10
	for i := 0; i < depth; i++ {
		sb.WriteString("if $x > 0 {\n")
	}
	sb.WriteString("print(\"deep\")\n")
	for i := 0; i < depth; i++ {
		sb.WriteString("}\n")
	}

	src := compileSource(t, sb.String())
	brCount := strings.Count(src, "br i1")
	if brCount < depth {
		t.Errorf("Expected %d conditional branches, got %d", depth, brCount)
	}
}

func TestDeeplyNestedLoops(t *testing.T) {
	src := compileSource(t, `
for $i = 0; $i < 3; $i++ {
	for $j = 0; $j < 3; $j++ {
		for $k = 0; $k < 3; $k++ {
			print($i)
		}
	}
}`)
	condCount := strings.Count(src, "for.cond")
	if condCount < 3 {
		t.Errorf("Expected 3 nested loop conditions, got %d", condCount)
	}
}

// --- String Edge Cases ---

func TestEmptyStringLiteral(t *testing.T) {
	src := compileSource(t, `$s = ""`)
	if !strings.Contains(src, `%s = alloca ptr`) {
		t.Errorf("Expected alloca ptr for string, got:\n%s", src)
	}
}

func TestLongStringInterpolation(t *testing.T) {
	src := compileSource(t, `$a = "one"
$b = "two"
$c = "three"
$msg = "{$a} and {$b} and {$c}"`)
	if !strings.Contains(src, "wolf_string_concat") {
		t.Error("Expected string_concat for interpolation")
	}
}

// --- Numeric Edge Cases ---

func TestLargeInteger(t *testing.T) {
	src := compileSource(t, `$big = 999999999999`)
	if !strings.Contains(src, "999999999999") {
		t.Error("Expected large integer")
	}
}

func TestNegativeFloat(t *testing.T) {
	src := compileSource(t, `$neg = -3.14`)
	// LLVM represents unary negation as subtraction from 0
	if !strings.Contains(src, "3.14") {
		t.Error("Expected 3.14 in output")
	}
}

func TestZeroValues(t *testing.T) {
	src := compileSource(t, `$z1 = 0
$z2 = 0.0
$z3 = ""
$z4 = false
$z5 = nil`)
	if !strings.Contains(src, "store i64 0, ptr %z1") {
		t.Error("Expected zero int")
	}
	if !strings.Contains(src, "%z4 = alloca i1") {
		t.Error("Expected bool alloca")
	}
}

// --- Function Edge Cases ---

func TestFunctionNoParams(t *testing.T) {
	src := compileSource(t, `func hello() {
	print("hello")
}`)
	if !strings.Contains(src, "define void @hello()") {
		t.Errorf("Expected no-param function, got:\n%s", src)
	}
}

func TestFunctionManyParams(t *testing.T) {
	src := compileSource(t, `func many($a, $b, $c, $d, $e) -> float {
	return $a
}`)
	if !strings.Contains(src, "define double @many(") {
		t.Errorf("Expected function with many params, got:\n%s", src)
	}
}

func TestRecursiveFunction(t *testing.T) {
	src := compileSource(t, `func countdown($n) -> float {
	if $n > 0.0 {
		print($n)
		return countdown($n - 1.0)
	}
	return 0.0
}`)
	if !strings.Contains(src, "define double @countdown(") {
		t.Errorf("Expected recursive function, got:\n%s", src)
	}
}

// --- Variable Reassignment ---

func TestVariableReassignment(t *testing.T) {
	src := compileSource(t, `$x = 1
$x = 2
$x = 3`)
	// First should have alloca, rest should just store
	allocaCount := strings.Count(src, "%x = alloca")
	if allocaCount != 1 {
		t.Errorf("Expected 1 alloca for x, got %d", allocaCount)
	}
	storeCount := strings.Count(src, "store i64")
	if storeCount < 3 {
		t.Errorf("Expected 3+ stores, got %d", storeCount)
	}
}

// --- Compound Expressions ---

func TestComparisonOperators(t *testing.T) {
	src := compileSource(t, `$a = 1
$b = 2
$r1 = $a == $b
$r2 = $a != $b
$r3 = $a < $b
$r4 = $a > $b`)
	for _, op := range []string{"icmp eq", "icmp ne", "icmp slt", "icmp sgt"} {
		if !strings.Contains(src, op) {
			t.Errorf("Expected %s", op)
		}
	}
}

// --- Error Recovery ---

func TestInvalidSyntaxRecovery(t *testing.T) {
	c := New()
	result, err := c.Compile("$x = ", "bad.wolf")
	if err == nil {
		t.Error("Expected error for incomplete expression")
	}
	_ = result
}

func TestUnterminatedString(t *testing.T) {
	c := New()
	result, err := c.Compile(`$x = "unterminated`, "bad.wolf")
	if err == nil {
		t.Error("Expected error for unterminated string")
	}
	_ = result
}

func TestMultipleErrors(t *testing.T) {
	c := New()
	c.StrictMode = true
	result, err := c.Compile(`var $x: int = "wrong"
var $y: float = "bad"`, "bad.wolf")
	if err == nil {
		t.Error("Expected errors in strict mode")
	}
	if len(result.Errors) < 2 {
		t.Errorf("Expected 2+ errors, got %d", len(result.Errors))
	}
}

// --- Pipeline Correctness ---

func TestPipelineProducesValidLLVMIR(t *testing.T) {
	programs := []struct {
		name string
		src  string
	}{
		{"empty", ""},
		{"print", `print("test")`},
		{"var", `$x = 42`},
		{"func", `func foo() { print("x") }`},
		{"if", `$x = 1
if $x > 0 { print("yes") }`},
		{"for", `for $i = 0; $i < 5; $i++ { print($i) }`},
		{"while", `$run = true
while $run { $run = false }`},
		{"class", `class Foo { $x: int }`},
		{"array", `$a = [1, 2, 3]`},
		{"map", `$m = {"k": "v"}`},
		{"interp", `$n = "w"
print("{$n}")`},
	}

	for _, prog := range programs {
		t.Run(prog.name, func(t *testing.T) {
			c := New()
			result, err := c.Compile(prog.src, "test.wolf")
			if err != nil {
				t.Fatalf("Compile failed: %v — %v", err, result.Errors)
			}
			if !strings.Contains(result.LLVMSource, "target triple") {
				t.Error("Missing target triple in LLVM IR")
			}
		})
	}
}
