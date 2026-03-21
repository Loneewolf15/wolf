package emitter

import (
	"strings"
	"testing"

	"github.com/wolflang/wolf/internal/ir"
	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
	"github.com/wolflang/wolf/internal/resolver"
)

// --- Test Helpers ---

func parseAndEmitIR(t *testing.T, source string) *ir.Program {
	t.Helper()
	l := lexer.New(source, "test.wolf")
	tokens, errs := l.Tokenize()
	if len(errs) > 0 {
		t.Fatalf("Lex errors: %v", errs)
	}
	p := parser.New(tokens, "test.wolf")
	program, perrs := p.Parse()
	if len(perrs) > 0 {
		t.Fatalf("Parse errors: %v", perrs)
	}
	res := resolver.New("test.wolf")
	res.Resolve(program)
	e := New(res)
	return e.Emit(program)
}

func parseAndEmitLLVM(t *testing.T, source string) string {
	t.Helper()
	irProg := parseAndEmitIR(t, source)
	le := NewLLVMEmitter()
	return le.Emit(irProg)
}

// ========== IR Emitter Tests ==========

func TestIREmitPackage(t *testing.T) {
	irProg := parseAndEmitIR(t, `print("hello")`)
	if irProg.Package != "main" {
		t.Errorf("Expected package 'main', got '%s'", irProg.Package)
	}
}

func TestIREmitImports(t *testing.T) {
	irProg := parseAndEmitIR(t, `print("hello")`)
	found := false
	for _, imp := range irProg.Imports {
		if imp == "fmt" {
			found = true
		}
	}
	if !found {
		t.Error("Expected 'fmt' import for print")
	}
}

func TestIREmitPrintAsFmtPrintln(t *testing.T) {
	irProg := parseAndEmitIR(t, `print("hello")`)
	if len(irProg.InitStmts) == 0 {
		t.Fatal("Expected init stmts for top-level print")
	}
	exprStmt, ok := irProg.InitStmts[0].(*ir.ExprStmt)
	if !ok {
		t.Fatalf("Expected ExprStmt, got %T", irProg.InitStmts[0])
	}
	call, ok := exprStmt.Expr.(*ir.CallExpr)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", exprStmt.Expr)
	}
	ident, ok := call.Callee.(*ir.Ident)
	if !ok {
		t.Fatalf("Expected Ident callee, got %T", call.Callee)
	}
	if ident.Name != "fmt.Println" {
		t.Errorf("Expected 'fmt.Println', got '%s'", ident.Name)
	}
}

func TestIREmitFunctionCreation(t *testing.T) {
	irProg := parseAndEmitIR(t, `func greet($name) {
		print($name)
	}`)
	if len(irProg.Functions) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(irProg.Functions))
	}
	fn := irProg.Functions[0]
	if fn.Name != "greet" {
		t.Errorf("Expected 'greet', got '%s'", fn.Name)
	}
	if len(fn.Params) != 1 {
		t.Fatalf("Expected 1 param, got %d", len(fn.Params))
	}
	if fn.Params[0].Name != "name" {
		t.Errorf("Expected param 'name', got '%s'", fn.Params[0].Name)
	}
}

func TestIREmitClassToStruct(t *testing.T) {
	irProg := parseAndEmitIR(t, `class Dog {
		$name: string
		func __construct($name) {
			$this->name = $name
		}
	}`)
	if len(irProg.Classes) != 1 {
		t.Fatalf("Expected 1 class, got %d", len(irProg.Classes))
	}
	cls := irProg.Classes[0]
	if cls.Name != "Dog" {
		t.Errorf("Expected 'Dog', got '%s'", cls.Name)
	}
	if len(cls.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(cls.Fields))
	}
	if cls.Constructor == nil {
		t.Error("Expected constructor")
	}
	if cls.Constructor.Name != "NewDog" {
		t.Errorf("Expected 'NewDog', got '%s'", cls.Constructor.Name)
	}
}

func TestIREmitMatchToSwitch(t *testing.T) {
	irProg := parseAndEmitIR(t, `$x = "a"
match $x {
	"a" => { print("A") }
	_ => { print("default") }
}`)
	if len(irProg.InitStmts) < 2 {
		t.Fatal("Expected at least 2 init stmts")
	}
	sw, ok := irProg.InitStmts[1].(*ir.SwitchStmt)
	if !ok {
		t.Fatalf("Expected SwitchStmt, got %T", irProg.InitStmts[1])
	}
	if len(sw.Cases) != 1 {
		t.Errorf("Expected 1 case, got %d", len(sw.Cases))
	}
	if sw.Default == nil {
		t.Error("Expected default case from wildcard")
	}
}

func TestIREmitForeachToRange(t *testing.T) {
	irProg := parseAndEmitIR(t, `$items = [1, 2, 3]
foreach $items as $item {
	print($item)
}`)
	if len(irProg.InitStmts) < 2 {
		t.Fatal("Expected at least 2 stmts")
	}
	_, ok := irProg.InitStmts[1].(*ir.RangeStmt)
	if !ok {
		t.Fatalf("Expected RangeStmt, got %T", irProg.InitStmts[1])
	}
}

func TestIREmitErrorExpr(t *testing.T) {
	irProg := parseAndEmitIR(t, `$e = error("something bad")`)
	found := false
	for _, imp := range irProg.Imports {
		if imp == "errors" {
			found = true
		}
	}
	if !found {
		t.Error("Expected 'errors' import for error()")
	}
}

func TestIREmitInterpolationToSprintf(t *testing.T) {
	irProg := parseAndEmitIR(t, `$name = "Wolf"
$msg = "Hello {$name}"`)
	found := false
	for _, imp := range irProg.Imports {
		if imp == "fmt" {
			found = true
		}
	}
	if !found {
		t.Error("Expected 'fmt' import for interpolation")
	}
}

func TestIREmitDollarStripping(t *testing.T) {
	irProg := parseAndEmitIR(t, `$myVar = 42`)
	if len(irProg.InitStmts) == 0 {
		t.Fatal("Expected init stmt")
	}
	assign, ok := irProg.InitStmts[0].(*ir.AssignStmt)
	if !ok {
		t.Fatalf("Expected AssignStmt, got %T", irProg.InitStmts[0])
	}
	ident, ok := assign.Target.(*ir.Ident)
	if !ok {
		t.Fatalf("Expected Ident, got %T", assign.Target)
	}
	if ident.Name != "myVar" {
		t.Errorf("Expected 'myVar' (stripped), got '%s'", ident.Name)
	}
}

func TestIREmitNewExprToConstructor(t *testing.T) {
	irProg := parseAndEmitIR(t, `$x = new Foo(1, 2)`)
	if len(irProg.InitStmts) == 0 {
		t.Fatal("Expected init stmts")
	}
	assign, ok := irProg.InitStmts[0].(*ir.AssignStmt)
	if !ok {
		t.Fatalf("Expected AssignStmt, got %T", irProg.InitStmts[0])
	}
	call, ok := assign.Value.(*ir.CallExpr)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", assign.Value)
	}
	ident, ok := call.Callee.(*ir.Ident)
	if !ok {
		t.Fatalf("Expected Ident callee, got %T", call.Callee)
	}
	if ident.Name != "NewFoo" {
		t.Errorf("Expected 'NewFoo', got '%s'", ident.Name)
	}
}

// ========== LLVM Emitter Tests ==========

func TestLLVMEmitHeader(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `print("hello")`)
	if !strings.Contains(llSrc, "target triple") {
		t.Error("LLVM IR should contain target triple")
	}
	if !strings.Contains(llSrc, "declare void @wolf_print_str(ptr)") {
		t.Error("LLVM IR should declare wolf_print_str")
	}
}

func TestLLVMEmitMainFunction(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `$x = 42`)
	if !strings.Contains(llSrc, "define i32 @main(") {
		t.Error("Expected main function definition")
	}
	if !strings.Contains(llSrc, "ret i32 0") {
		t.Error("Expected return 0 in main")
	}
}

func TestLLVMEmitStringConstant(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `print("hello")`)
	if !strings.Contains(llSrc, `c"hello\00"`) {
		t.Errorf("Expected string constant, got:\n%s", llSrc)
	}
}

func TestLLVMEmitPrintStr(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `print("hello")`)
	if !strings.Contains(llSrc, "call void @wolf_print_str(ptr") {
		t.Errorf("Expected wolf_print_str call, got:\n%s", llSrc)
	}
	if !strings.Contains(llSrc, "call void @wolf_println()") {
		t.Errorf("Expected wolf_println call, got:\n%s", llSrc)
	}
}

func TestLLVMEmitIntVariable(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `$x = 42`)
	if !strings.Contains(llSrc, "%x = alloca i64") {
		t.Errorf("Expected alloca i64 for int, got:\n%s", llSrc)
	}
	if !strings.Contains(llSrc, "store i64 42, ptr %x") {
		t.Errorf("Expected store i64, got:\n%s", llSrc)
	}
}

func TestLLVMEmitFloatVariable(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `$pi = 3.14`)
	if !strings.Contains(llSrc, "%pi = alloca double") {
		t.Errorf("Expected alloca double for float, got:\n%s", llSrc)
	}
	if !strings.Contains(llSrc, "store double 3.14, ptr %pi") {
		t.Errorf("Expected store double, got:\n%s", llSrc)
	}
}

func TestLLVMEmitBoolVariable(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `$flag = true`)
	if !strings.Contains(llSrc, "%flag = alloca i1") {
		t.Errorf("Expected alloca i1 for bool, got:\n%s", llSrc)
	}
	if !strings.Contains(llSrc, "store i1 1, ptr %flag") {
		t.Errorf("Expected store i1, got:\n%s", llSrc)
	}
}

func TestLLVMEmitStringVariable(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `$name = "Wolf"`)
	if !strings.Contains(llSrc, "%name = alloca ptr") {
		t.Errorf("Expected alloca ptr for string, got:\n%s", llSrc)
	}
	if !strings.Contains(llSrc, `c"Wolf\00"`) {
		t.Errorf("Expected string constant, got:\n%s", llSrc)
	}
}

func TestLLVMEmitArithmetic(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `$a = 10
$b = 3
$c = $a + $b`)
	if !strings.Contains(llSrc, "add i64") {
		t.Errorf("Expected add i64, got:\n%s", llSrc)
	}
}

func TestLLVMEmitComparison(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `$x = 5
$y = 10
$result = $x < $y`)
	if !strings.Contains(llSrc, "icmp slt i64") {
		t.Errorf("Expected icmp slt, got:\n%s", llSrc)
	}
}

func TestLLVMEmitFunction(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `func greet($name) {
		print($name)
	}`)
	if !strings.Contains(llSrc, "define void @wolf_greet(") {
		t.Errorf("Expected function definition, got:\n%s", llSrc)
	}
}

func TestLLVMEmitIfElse(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `$x = 5
if $x > 3 {
	print("yes")
} else {
	print("no")
}`)
	if !strings.Contains(llSrc, "br i1") {
		t.Errorf("Expected conditional branch, got:\n%s", llSrc)
	}
	if !strings.Contains(llSrc, "if.then") {
		t.Error("Expected if.then label")
	}
	if !strings.Contains(llSrc, "if.else") {
		t.Error("Expected if.else label")
	}
}

func TestLLVMEmitForLoop(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `for $i = 0; $i < 5; $i++ {
		print($i)
	}`)
	if !strings.Contains(llSrc, "for.cond") {
		t.Errorf("Expected for.cond label, got:\n%s", llSrc)
	}
	if !strings.Contains(llSrc, "for.body") {
		t.Error("Expected for.body label")
	}
	if !strings.Contains(llSrc, "for.end") {
		t.Error("Expected for.end label")
	}
}

func TestLLVMEmitPrintInt(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `print(42)`)
	if !strings.Contains(llSrc, "call void @wolf_print_int(i64 42)") {
		t.Errorf("Expected wolf_print_int(i64 42), got:\n%s", llSrc)
	}
}

func TestLLVMEmitPrintBool(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `print(true)`)
	if !strings.Contains(llSrc, "call void @wolf_print_bool(i1 1)") {
		t.Errorf("Expected wolf_print_bool(i1 1), got:\n%s", llSrc)
	}
}

// ========== Edge Cases ==========

func TestLLVMEmitEmptyProgram(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, ``)
	if !strings.Contains(llSrc, "target triple") {
		t.Error("Empty program should still have target triple")
	}
}

func TestLLVMEmitMultipleFunctions(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `
func a() { print("a") }
func b() { print("b") }
func c() { print("c") }
`)
	if strings.Count(llSrc, "define") < 3 {
		t.Errorf("Expected 3+ function definitions, got:\n%s", llSrc)
	}
}

func TestLLVMEmitNestedIf(t *testing.T) {
	llSrc := parseAndEmitLLVM(t, `$x = 1
if $x > 0 {
	if $x < 10 {
		print("ok")
	}
}`)
	brCount := strings.Count(llSrc, "br i1")
	if brCount < 2 {
		t.Errorf("Expected 2+ conditional branches, got %d in:\n%s", brCount, llSrc)
	}
}
