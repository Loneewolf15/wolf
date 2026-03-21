package compiler

import (
	"strings"
	"testing"
)

// compileSource is a test helper that runs the full pipeline and returns generated LLVM IR.
func compileSource(t *testing.T, wolfSource string) string {
	t.Helper()
	c := New()
	result, err := c.Compile(wolfSource, "test.wolf")
	if err != nil {
		t.Fatalf("Compilation failed: %v\nErrors: %v", err, result.Errors)
	}
	return result.LLVMSource
}

// compileAndExpectError runs the pipeline and expects failure.
func compileAndExpectError(t *testing.T, wolfSource string) []string {
	t.Helper()
	c := New()
	c.StrictMode = true
	result, err := c.Compile(wolfSource, "test.wolf")
	if err == nil {
		t.Fatal("Expected compilation error but got none")
	}
	return result.Errors
}

// ========== Hello World ==========

func TestCompileHelloWorld(t *testing.T) {
	src := compileSource(t, `print("Hello from Wolf!")`)
	if !strings.Contains(src, `wolf_print_str`) {
		t.Error("Expected wolf_print_str in output")
	}
	if !strings.Contains(src, `define i32 @main(`) {
		t.Error("Expected main function")
	}
	if !strings.Contains(src, `Hello from Wolf!`) {
		t.Error("Expected string constant")
	}
}

// ========== Variables ==========

func TestCompileVariableAssignment(t *testing.T) {
	src := compileSource(t, `$name = "Wolf"`)
	if !strings.Contains(src, `%name = alloca ptr`) {
		t.Errorf("Expected alloca ptr for string, got:\n%s", src)
	}
}

func TestCompileMultipleVars(t *testing.T) {
	src := compileSource(t, `$x = 10
$y = 20
$z = $x + $y`)
	if !strings.Contains(src, `%x = alloca i64`) {
		t.Errorf("Expected x alloca i64, got:\n%s", src)
	}
	if !strings.Contains(src, `%y = alloca i64`) {
		t.Errorf("Expected y alloca i64, got:\n%s", src)
	}
	if !strings.Contains(src, `add i64`) {
		t.Error("Expected add i64")
	}
}

func TestCompileTypedVarDecl(t *testing.T) {
	src := compileSource(t, `var $count: int = 0`)
	if !strings.Contains(src, `%count = alloca i64`) {
		t.Errorf("Expected typed var decl, got:\n%s", src)
	}
}

// ========== Functions ==========

func TestCompileFunction(t *testing.T) {
	src := compileSource(t, `func greet($name) {
		print("hello")
	}`)
	if !strings.Contains(src, `define void @wolf_greet(`) {
		t.Errorf("Expected function definition, got:\n%s", src)
	}
}

func TestCompileFunctionWithReturn(t *testing.T) {
	src := compileSource(t, `func add($a, $b) -> int {
		return $a + $b
	}`)
	if !strings.Contains(src, `define i64 @wolf_add(`) {
		t.Errorf("Expected int return function, got:\n%s", src)
	}
}

func TestCompileArrowFunction(t *testing.T) {
	src := compileSource(t, `func double($x) => $x * 2`)
	if !strings.Contains(src, `define`) && !strings.Contains(src, `@wolf_double`) {
		t.Errorf("Expected arrow function, got:\n%s", src)
	}
}

// ========== Control Flow ==========

func TestCompileIfElse(t *testing.T) {
	src := compileSource(t, `
$x = 10
if $x > 5 {
	print("big")
} else {
	print("small")
}`)
	if !strings.Contains(src, `br i1`) {
		t.Errorf("Expected conditional branch, got:\n%s", src)
	}
	if !strings.Contains(src, `if.then`) {
		t.Error("Expected if.then label")
	}
	if !strings.Contains(src, `if.else`) {
		t.Error("Expected if.else label")
	}
}

func TestCompileForLoop(t *testing.T) {
	src := compileSource(t, `for $i = 0; $i < 10; $i++ {
	print($i)
}`)
	if !strings.Contains(src, `for.cond`) {
		t.Errorf("Expected for.cond label, got:\n%s", src)
	}
	if !strings.Contains(src, `for.body`) {
		t.Error("Expected for.body label")
	}
}

func TestCompileWhileLoop(t *testing.T) {
	src := compileSource(t, `$running = true
while $running {
	print("loop")
}`)
	if !strings.Contains(src, `for.cond`) {
		t.Errorf("Expected while→for conversion, got:\n%s", src)
	}
}

func TestCompileForeach(t *testing.T) {
	src := compileSource(t, `$items = [1, 2, 3]
foreach $items as $item {
	print($item)
}`)
	// Should at least compile without error
	if !strings.Contains(src, `define i32 @main(`) {
		t.Error("Expected main function")
	}
}

func TestCompileMatch(t *testing.T) {
	src := compileSource(t, `$status = "ok"
match $status {
	"ok" => { print("success") }
	"error" => { print("fail") }
	_ => { print("unknown") }
}`)
	if !strings.Contains(src, `define i32 @main(`) {
		t.Error("Expected main function")
	}
}

// ========== Expressions ==========

func TestCompileNewExpr(t *testing.T) {
	src := compileSource(t, `$user = new User("Ada", "ada@wolf.dev")`)
	if !strings.Contains(src, `define i32 @main(`) {
		t.Error("Expected main function")
	}
}

func TestCompileArrayLiteral(t *testing.T) {
	src := compileSource(t, `$items = [1, 2, 3]`)
	if !strings.Contains(src, `define i32 @main(`) {
		t.Error("Expected main function")
	}
}

func TestCompileMapLiteral(t *testing.T) {
	src := compileSource(t, `$config = {"host": "localhost", "port": 8080}`)
	if !strings.Contains(src, `define i32 @main(`) {
		t.Error("Expected main function")
	}
}

// ========== Concurrency ==========

func TestCompileParallel(t *testing.T) {
	src := compileSource(t, `parallel {
	print("task1")
	print("task2")
}`)
	if !strings.Contains(src, `define i32 @main(`) {
		t.Error("Expected main function")
	}
}

func TestCompileChannel(t *testing.T) {
	src := compileSource(t, `$ch = channel(int)`)
	if !strings.Contains(src, `define i32 @main(`) {
		t.Error("Expected main function")
	}
}

// ========== String Interpolation ==========

func TestCompileInterpolation(t *testing.T) {
	src := compileSource(t, `$name = "Wolf"
$greeting = "Hello, {$name}!"`)
	if !strings.Contains(src, `wolf_string_concat`) {
		t.Errorf("Expected string_concat for interpolation, got:\n%s", src)
	}
}

// ========== Error Handling ==========

func TestCompileTryCatch(t *testing.T) {
	src := compileSource(t, `try {
	print("risky")
} catch ($e) {
	print("caught")
}`)
	if !strings.Contains(src, `wolf_print_str`) {
		t.Error("Expected print in try body")
	}
}

// ========== Full Program ==========

func TestCompileFullProgram(t *testing.T) {
	src := compileSource(t, `
$greeting = "Hello"
$count = 3

func sayHello($name) {
	print("Hello")
}

for $i = 0; $i < $count; $i++ {
	sayHello("World")
}

if $count > 0 {
	print("done")
}
`)
	if !strings.Contains(src, `define void @wolf_sayHello(`) {
		t.Error("Missing function definition")
	}
	if !strings.Contains(src, `define i32 @main(`) {
		t.Error("Missing main function")
	}
}

// ========== Strict Mode ==========

func TestStrictModeTypeMismatch(t *testing.T) {
	c := New()
	c.StrictMode = true
	result, err := c.Compile(`var $x: int = "hello"`, "test.wolf")
	if err == nil {
		t.Error("Expected type mismatch error in strict mode")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected error messages")
	}
}

// ========== Edge Cases ==========

func TestCompileEmptyProgram(t *testing.T) {
	src := compileSource(t, ``)
	if !strings.Contains(src, `target triple`) {
		t.Error("Empty program should still have target triple")
	}
}

func TestCompileBoolLiterals(t *testing.T) {
	src := compileSource(t, `$active = true
$deleted = false`)
	if !strings.Contains(src, `%active = alloca i1`) {
		t.Error("Expected alloca i1 for bool")
	}
}

func TestCompileNilLiteral(t *testing.T) {
	src := compileSource(t, `$data = nil`)
	if !strings.Contains(src, `%data = alloca ptr`) {
		t.Errorf("Expected alloca ptr for nil, got:\n%s", src)
	}
}

func TestCompileUnaryNot(t *testing.T) {
	src := compileSource(t, `$x = !true`)
	if !strings.Contains(src, `xor i1`) {
		t.Errorf("Expected xor i1 for !, got:\n%s", src)
	}
}

func TestCompileUnaryNegation(t *testing.T) {
	src := compileSource(t, `$x = -42`)
	if !strings.Contains(src, `sub i64 0, 42`) {
		t.Errorf("Expected sub i64 0, 42 for negation, got:\n%s", src)
	}
}
