package parser

import (
	"testing"

	"github.com/wolflang/wolf/internal/lexer"
)

// --- Test Helpers ---

func parseSource(t *testing.T, source string) *Program {
	t.Helper()
	l := lexer.New(source, "test.wolf")
	tokens, lexErrs := l.Tokenize()
	if len(lexErrs) > 0 {
		for _, e := range lexErrs {
			t.Logf("Lexer error: %s", e.Error())
		}
	}
	p := New(tokens, "test.wolf")
	program, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		for _, e := range parseErrs {
			t.Errorf("Parse error: %s", e.Error())
		}
	}
	return program
}

func parseSourceExpectErrors(t *testing.T, source string) (*Program, []*lexer.WolfError) {
	t.Helper()
	l := lexer.New(source, "test.wolf")
	tokens, _ := l.Tokenize()
	p := New(tokens, "test.wolf")
	return p.Parse()
}

// --- Program & Expression Statements ---

func TestEmptyProgram(t *testing.T) {
	prog := parseSource(t, "")
	if len(prog.Statements) != 0 {
		t.Errorf("Expected 0 statements, got %d", len(prog.Statements))
	}
}

// --- Variable Assignment ---

func TestSimpleAssignment(t *testing.T) {
	prog := parseSource(t, `$name = "Wolf"`)
	if len(prog.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(prog.Statements))
	}
	stmt, ok := prog.Statements[0].(*ExpressionStmt)
	if !ok {
		t.Fatalf("Expected ExpressionStmt, got %T", prog.Statements[0])
	}
	binExpr, ok := stmt.Expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected BinaryExpr, got %T", stmt.Expr)
	}
	if binExpr.Op != "=" {
		t.Errorf("Expected op '=', got '%s'", binExpr.Op)
	}
	dollarIdent, ok := binExpr.Left.(*DollarIdent)
	if !ok {
		t.Fatalf("Expected DollarIdent, got %T", binExpr.Left)
	}
	if dollarIdent.Name != "$name" {
		t.Errorf("Expected '$name', got '%s'", dollarIdent.Name)
	}
}

func TestIntAssignment(t *testing.T) {
	prog := parseSource(t, "$count = 42")
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	intLit, ok := binExpr.Right.(*IntLiteral)
	if !ok {
		t.Fatalf("Expected IntLiteral, got %T", binExpr.Right)
	}
	if intLit.Value != "42" {
		t.Errorf("Expected '42', got '%s'", intLit.Value)
	}
}

func TestFloatAssignment(t *testing.T) {
	prog := parseSource(t, "$price = 9.99")
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	floatLit, ok := binExpr.Right.(*FloatLiteral)
	if !ok {
		t.Fatalf("Expected FloatLiteral, got %T", binExpr.Right)
	}
	if floatLit.Value != "9.99" {
		t.Errorf("Expected '9.99', got '%s'", floatLit.Value)
	}
}

func TestBoolAssignment(t *testing.T) {
	prog := parseSource(t, "$active = true")
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	boolLit, ok := binExpr.Right.(*BoolLiteral)
	if !ok {
		t.Fatalf("Expected BoolLiteral, got %T", binExpr.Right)
	}
	if boolLit.Value != true {
		t.Errorf("Expected true, got false")
	}
}

func TestArrayLiteral(t *testing.T) {
	prog := parseSource(t, "$items = [1, 2, 3]")
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	arr, ok := binExpr.Right.(*ArrayLiteral)
	if !ok {
		t.Fatalf("Expected ArrayLiteral, got %T", binExpr.Right)
	}
	if len(arr.Elements) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestMapLiteral(t *testing.T) {
	prog := parseSource(t, `$user = {"id": 1, "name": "Wolf"}`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	ml, ok := binExpr.Right.(*MapLiteral)
	if !ok {
		t.Fatalf("Expected MapLiteral, got %T", binExpr.Right)
	}
	if len(ml.Keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(ml.Keys))
	}
}

// --- Typed Variable Declaration ---

func TestVarDecl(t *testing.T) {
	prog := parseSource(t, "var $userId: int = 42")
	stmt, ok := prog.Statements[0].(*VarDecl)
	if !ok {
		t.Fatalf("Expected VarDecl, got %T", prog.Statements[0])
	}
	if stmt.Name != "$userId" {
		t.Errorf("Expected '$userId', got '%s'", stmt.Name)
	}
	if stmt.TypeName != "int" {
		t.Errorf("Expected type 'int', got '%s'", stmt.TypeName)
	}
	intLit, ok := stmt.Value.(*IntLiteral)
	if !ok {
		t.Fatalf("Expected IntLiteral, got %T", stmt.Value)
	}
	if intLit.Value != "42" {
		t.Errorf("Expected '42', got '%s'", intLit.Value)
	}
}

func TestVarDeclStringType(t *testing.T) {
	prog := parseSource(t, `var $label: string = "hello"`)
	stmt := prog.Statements[0].(*VarDecl)
	if stmt.TypeName != "string" {
		t.Errorf("Expected type 'string', got '%s'", stmt.TypeName)
	}
}

// --- Function Declaration ---

func TestFuncDecl(t *testing.T) {
	prog := parseSource(t, `func greet($name) {
		return "Hello"
	}`)
	stmt, ok := prog.Statements[0].(*FuncDecl)
	if !ok {
		t.Fatalf("Expected FuncDecl, got %T", prog.Statements[0])
	}
	if stmt.Name != "greet" {
		t.Errorf("Expected 'greet', got '%s'", stmt.Name)
	}
	if len(stmt.Params) != 1 {
		t.Fatalf("Expected 1 param, got %d", len(stmt.Params))
	}
	if stmt.Params[0].Name != "$name" {
		t.Errorf("Expected '$name', got '%s'", stmt.Params[0].Name)
	}
}

func TestFuncDeclMultipleReturns(t *testing.T) {
	prog := parseSource(t, `func divide($a, $b) -> (float, error) {
		return 0, nil
	}`)
	stmt := prog.Statements[0].(*FuncDecl)
	if stmt.Name != "divide" {
		t.Errorf("Expected 'divide', got '%s'", stmt.Name)
	}
	if stmt.ReturnType == nil {
		t.Fatal("Expected return type spec")
	}
	if len(stmt.ReturnType.Types) != 2 {
		t.Errorf("Expected 2 return types, got %d", len(stmt.ReturnType.Types))
	}
	if stmt.ReturnType.Types[0] != "float" {
		t.Errorf("Expected 'float', got '%s'", stmt.ReturnType.Types[0])
	}
	if stmt.ReturnType.Types[1] != "error" {
		t.Errorf("Expected 'error', got '%s'", stmt.ReturnType.Types[1])
	}
}

func TestArrowFunction(t *testing.T) {
	prog := parseSource(t, "func double($n) => $n * 2")
	stmt := prog.Statements[0].(*FuncDecl)
	if stmt.Name != "double" {
		t.Errorf("Expected 'double', got '%s'", stmt.Name)
	}
	if stmt.ArrowExpr == nil {
		t.Fatal("Expected arrow expression")
	}
	binExpr, ok := stmt.ArrowExpr.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected BinaryExpr, got %T", stmt.ArrowExpr)
	}
	if binExpr.Op != "*" {
		t.Errorf("Expected op '*', got '%s'", binExpr.Op)
	}
}

func TestFuncDefaultParam(t *testing.T) {
	prog := parseSource(t, `func createUser($name, $role = "user") {
		return nil
	}`)
	stmt := prog.Statements[0].(*FuncDecl)
	if len(stmt.Params) != 2 {
		t.Fatalf("Expected 2 params, got %d", len(stmt.Params))
	}
	if stmt.Params[1].Default == nil {
		t.Fatal("Expected default value for second param")
	}
}

// --- Closure Expression ---

func TestClosureExpr(t *testing.T) {
	prog := parseSource(t, `$multiply = func($a, $b) { return $a * $b }`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	closure, ok := binExpr.Right.(*ClosureExpr)
	if !ok {
		t.Fatalf("Expected ClosureExpr, got %T", binExpr.Right)
	}
	if len(closure.Params) != 2 {
		t.Errorf("Expected 2 params, got %d", len(closure.Params))
	}
}

// --- If/Else ---

func TestIfStmt(t *testing.T) {
	prog := parseSource(t, `if $score > 90 { $grade = "A" }`)
	stmt, ok := prog.Statements[0].(*IfStmt)
	if !ok {
		t.Fatalf("Expected IfStmt, got %T", prog.Statements[0])
	}
	binExpr := stmt.Condition.(*BinaryExpr)
	if binExpr.Op != ">" {
		t.Errorf("Expected op '>', got '%s'", binExpr.Op)
	}
}

func TestIfElseStmt(t *testing.T) {
	prog := parseSource(t, `if $x > 0 { $y = 1 } else { $y = 0 }`)
	stmt := prog.Statements[0].(*IfStmt)
	if stmt.ElseBody == nil {
		t.Fatal("Expected else body")
	}
}

func TestIfElseIfElseStmt(t *testing.T) {
	prog := parseSource(t, `if $score > 90 { $grade = "A" } else if $score > 75 { $grade = "B" } else { $grade = "C" }`)
	stmt := prog.Statements[0].(*IfStmt)
	if len(stmt.ElseIfs) != 1 {
		t.Errorf("Expected 1 else-if, got %d", len(stmt.ElseIfs))
	}
	if stmt.ElseBody == nil {
		t.Fatal("Expected else body")
	}
}

// --- For Loop ---

func TestForStmt(t *testing.T) {
	prog := parseSource(t, `for $i = 0; $i < 10; $i++ { print($i) }`)
	stmt, ok := prog.Statements[0].(*ForStmt)
	if !ok {
		t.Fatalf("Expected ForStmt, got %T", prog.Statements[0])
	}
	if stmt.Init == nil {
		t.Fatal("Expected init statement")
	}
	if stmt.Condition == nil {
		t.Fatal("Expected condition")
	}
	if stmt.Update == nil {
		t.Fatal("Expected update statement")
	}
}

// --- Foreach ---

func TestForeachStmt(t *testing.T) {
	prog := parseSource(t, `foreach $items as $item { print($item) }`)
	stmt, ok := prog.Statements[0].(*ForeachStmt)
	if !ok {
		t.Fatalf("Expected ForeachStmt, got %T", prog.Statements[0])
	}
	if stmt.ValueVar != "$item" {
		t.Errorf("Expected '$item', got '%s'", stmt.ValueVar)
	}
	if stmt.KeyVar != "" {
		t.Errorf("Expected empty key var, got '%s'", stmt.KeyVar)
	}
}

func TestForeachKeyValueStmt(t *testing.T) {
	prog := parseSource(t, `foreach $users as $id => $user { print($id) }`)
	stmt := prog.Statements[0].(*ForeachStmt)
	if stmt.KeyVar != "$id" {
		t.Errorf("Expected key '$id', got '%s'", stmt.KeyVar)
	}
	if stmt.ValueVar != "$user" {
		t.Errorf("Expected value '$user', got '%s'", stmt.ValueVar)
	}
}

// --- While ---

func TestWhileStmt(t *testing.T) {
	prog := parseSource(t, `while $running { $running = false }`)
	_, ok := prog.Statements[0].(*WhileStmt)
	if !ok {
		t.Fatalf("Expected WhileStmt, got %T", prog.Statements[0])
	}
}

// --- Match ---

func TestMatchStmt(t *testing.T) {
	prog := parseSource(t, `match $status {
		"active" => handleActive()
		"pending" => handlePending()
		_ => handleDefault()
	}`)
	stmt, ok := prog.Statements[0].(*MatchStmt)
	if !ok {
		t.Fatalf("Expected MatchStmt, got %T", prog.Statements[0])
	}
	if len(stmt.Arms) != 3 {
		t.Errorf("Expected 3 arms, got %d", len(stmt.Arms))
	}
	// Check wildcard
	_, isWild := stmt.Arms[2].Pattern.(*Wildcard)
	if !isWild {
		t.Errorf("Expected Wildcard pattern for last arm, got %T", stmt.Arms[2].Pattern)
	}
}

// --- Return ---

func TestReturnStmt(t *testing.T) {
	prog := parseSource(t, `func foo() { return 42 }`)
	funcDecl := prog.Statements[0].(*FuncDecl)
	retStmt := funcDecl.Body.Statements[0].(*ReturnStmt)
	if len(retStmt.Values) != 1 {
		t.Fatalf("Expected 1 return value, got %d", len(retStmt.Values))
	}
	intLit, ok := retStmt.Values[0].(*IntLiteral)
	if !ok {
		t.Fatalf("Expected IntLiteral, got %T", retStmt.Values[0])
	}
	if intLit.Value != "42" {
		t.Errorf("Expected '42', got '%s'", intLit.Value)
	}
}

// --- Class Declaration ---

func TestClassDecl(t *testing.T) {
	prog := parseSource(t, `class Driver {
		$name: string
		$rating: float
		$online: bool = false
		func __construct($name, $rating) {
			$this->name = $name
		}
		func goOnline() { $this->online = true }
	}`)
	cls, ok := prog.Statements[0].(*ClassDecl)
	if !ok {
		t.Fatalf("Expected ClassDecl, got %T", prog.Statements[0])
	}
	if cls.Name != "Driver" {
		t.Errorf("Expected 'Driver', got '%s'", cls.Name)
	}
	if len(cls.Properties) != 3 {
		t.Errorf("Expected 3 properties, got %d", len(cls.Properties))
	}
	if len(cls.Methods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(cls.Methods))
	}
}

func TestClassPropertyWithDefault(t *testing.T) {
	prog := parseSource(t, `class Foo { $active: bool = false }`)
	cls := prog.Statements[0].(*ClassDecl)
	if cls.Properties[0].Default == nil {
		t.Fatal("Expected default value")
	}
}

// --- Try/Catch ---

func TestTryCatch(t *testing.T) {
	prog := parseSource(t, `try {
		$user = fetchUser($id)
	} catch ($e) {
		print($e)
	}`)
	tc, ok := prog.Statements[0].(*TryCatchStmt)
	if !ok {
		t.Fatalf("Expected TryCatchStmt, got %T", prog.Statements[0])
	}
	if tc.CatchVar != "$e" {
		t.Errorf("Expected '$e', got '%s'", tc.CatchVar)
	}
}

// --- Async/Await ---

func TestAsyncExpr(t *testing.T) {
	prog := parseSource(t, `$task = async { return fetchDrivers($location) }`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	asyncExpr, ok := binExpr.Right.(*AsyncExpr)
	if !ok {
		t.Fatalf("Expected AsyncExpr, got %T", binExpr.Right)
	}
	if len(asyncExpr.Body.Statements) != 1 {
		t.Errorf("Expected 1 statement in async body, got %d", len(asyncExpr.Body.Statements))
	}
}

func TestAwaitExpr(t *testing.T) {
	prog := parseSource(t, `$drivers = await $task`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	awaitExpr, ok := binExpr.Right.(*AwaitExpr)
	if !ok {
		t.Fatalf("Expected AwaitExpr, got %T", binExpr.Right)
	}
	dollarIdent := awaitExpr.Expr.(*DollarIdent)
	if dollarIdent.Name != "$task" {
		t.Errorf("Expected '$task', got '%s'", dollarIdent.Name)
	}
}

// --- Parallel ---

func TestParallelBlock(t *testing.T) {
	prog := parseSource(t, `parallel {
		$drivers = fetchDrivers($location)
		$surge = calculateSurge($zone)
	}`)
	_, ok := prog.Statements[0].(*ParallelStmt)
	if !ok {
		t.Fatalf("Expected ParallelStmt, got %T", prog.Statements[0])
	}
}

// --- Channel Operations ---

func TestChannelExpr(t *testing.T) {
	prog := parseSource(t, `$ch = channel(int)`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	chExpr, ok := binExpr.Right.(*ChannelExpr)
	if !ok {
		t.Fatalf("Expected ChannelExpr, got %T", binExpr.Right)
	}
	if chExpr.ElemType != "int" {
		t.Errorf("Expected 'int', got '%s'", chExpr.ElemType)
	}
}

func TestSendExpr(t *testing.T) {
	prog := parseSource(t, `send($ch, 42)`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	sendExpr, ok := stmt.Expr.(*SendExpr)
	if !ok {
		t.Fatalf("Expected SendExpr, got %T", stmt.Expr)
	}
	if sendExpr.Channel == nil || sendExpr.Value == nil {
		t.Fatal("Expected channel and value")
	}
}

func TestReceiveExpr(t *testing.T) {
	prog := parseSource(t, `$val = receive($ch)`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	_, ok := binExpr.Right.(*ReceiveExpr)
	if !ok {
		t.Fatalf("Expected ReceiveExpr, got %T", binExpr.Right)
	}
}

// --- Destructuring ---

func TestDestructureAssign(t *testing.T) {
	prog := parseSource(t, `[$data, $err] = parseInput($raw)`)
	stmt, ok := prog.Statements[0].(*DestructureAssign)
	if !ok {
		t.Fatalf("Expected DestructureAssign, got %T", prog.Statements[0])
	}
	if len(stmt.Names) != 2 {
		t.Errorf("Expected 2 names, got %d", len(stmt.Names))
	}
	if stmt.Names[0] != "$data" || stmt.Names[1] != "$err" {
		t.Errorf("Expected [$data, $err], got %v", stmt.Names)
	}
}

// --- New Expression ---

func TestNewExpr(t *testing.T) {
	prog := parseSource(t, `$driver = new Driver("Ade", 4.8)`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	newExpr, ok := binExpr.Right.(*NewExpr)
	if !ok {
		t.Fatalf("Expected NewExpr, got %T", binExpr.Right)
	}
	if newExpr.ClassName != "Driver" {
		t.Errorf("Expected 'Driver', got '%s'", newExpr.ClassName)
	}
	if len(newExpr.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(newExpr.Args))
	}
}

// --- Property Access & Method Call ---

func TestPropertyAccess(t *testing.T) {
	prog := parseSource(t, "$driver->name")
	stmt := prog.Statements[0].(*ExpressionStmt)
	prop, ok := stmt.Expr.(*PropertyAccess)
	if !ok {
		t.Fatalf("Expected PropertyAccess, got %T", stmt.Expr)
	}
	if prop.Property != "name" {
		t.Errorf("Expected 'name', got '%s'", prop.Property)
	}
}

func TestMethodCall(t *testing.T) {
	prog := parseSource(t, "$driver->goOnline()")
	stmt := prog.Statements[0].(*ExpressionStmt)
	mc, ok := stmt.Expr.(*MethodCall)
	if !ok {
		t.Fatalf("Expected MethodCall, got %T", stmt.Expr)
	}
	if mc.Method != "goOnline" {
		t.Errorf("Expected 'goOnline', got '%s'", mc.Method)
	}
}

func TestChainedPropertyAccess(t *testing.T) {
	prog := parseSource(t, `$this->db->query("SELECT 1")`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	mc, ok := stmt.Expr.(*MethodCall)
	if !ok {
		t.Fatalf("Expected MethodCall, got %T", stmt.Expr)
	}
	if mc.Method != "query" {
		t.Errorf("Expected 'query', got '%s'", mc.Method)
	}
	// mc.Object should be a PropertyAccess ($this->db)
	inner, ok := mc.Object.(*PropertyAccess)
	if !ok {
		t.Fatalf("Expected inner PropertyAccess, got %T", mc.Object)
	}
	if inner.Property != "db" {
		t.Errorf("Expected 'db', got '%s'", inner.Property)
	}
}

// --- Named Parameters ---

func TestNamedArguments(t *testing.T) {
	prog := parseSource(t, `createUser(name: "Ada", email: "ada@wolf.dev")`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	call, ok := stmt.Expr.(*CallExpr)
	if !ok {
		t.Fatalf("Expected CallExpr, got %T", stmt.Expr)
	}
	if len(call.NamedArgs) != 2 {
		t.Errorf("Expected 2 named args, got %d", len(call.NamedArgs))
	}
	if call.NamedArgs[0].Name != "name" {
		t.Errorf("Expected 'name', got '%s'", call.NamedArgs[0].Name)
	}
}

// --- Operator Precedence ---

func TestMulBeforeAdd(t *testing.T) {
	// 2 + 3 * 4 should parse as 2 + (3 * 4)
	prog := parseSource(t, "$x = 2 + 3 * 4")
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr) // =
	add, ok := binExpr.Right.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected BinaryExpr for +, got %T", binExpr.Right)
	}
	if add.Op != "+" {
		t.Errorf("Expected '+', got '%s'", add.Op)
	}
	mul, ok := add.Right.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected BinaryExpr for *, got %T", add.Right)
	}
	if mul.Op != "*" {
		t.Errorf("Expected '*', got '%s'", mul.Op)
	}
}

func TestComparisonPrecedence(t *testing.T) {
	// $a > $b && $c < $d → ($a > $b) && ($c < $d)
	prog := parseSource(t, "$a > $b && $c < $d")
	stmt := prog.Statements[0].(*ExpressionStmt)
	andExpr, ok := stmt.Expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected BinaryExpr for &&, got %T", stmt.Expr)
	}
	if andExpr.Op != "&&" {
		t.Errorf("Expected '&&', got '%s'", andExpr.Op)
	}
}

// --- Unary ---

func TestUnaryNot(t *testing.T) {
	prog := parseSource(t, "!$active")
	stmt := prog.Statements[0].(*ExpressionStmt)
	unary, ok := stmt.Expr.(*UnaryExpr)
	if !ok {
		t.Fatalf("Expected UnaryExpr, got %T", stmt.Expr)
	}
	if unary.Op != "!" {
		t.Errorf("Expected '!', got '%s'", unary.Op)
	}
}

func TestUnaryNegation(t *testing.T) {
	prog := parseSource(t, "-$x")
	stmt := prog.Statements[0].(*ExpressionStmt)
	unary, ok := stmt.Expr.(*UnaryExpr)
	if !ok {
		t.Fatalf("Expected UnaryExpr, got %T", stmt.Expr)
	}
	if unary.Op != "-" {
		t.Errorf("Expected '-', got '%s'", unary.Op)
	}
}

// --- Postfix ---

func TestPostfixIncrement(t *testing.T) {
	prog := parseSource(t, "$i++")
	stmt := prog.Statements[0].(*ExpressionStmt)
	post, ok := stmt.Expr.(*PostfixExpr)
	if !ok {
		t.Fatalf("Expected PostfixExpr, got %T", stmt.Expr)
	}
	if post.Op != "++" {
		t.Errorf("Expected '++', got '%s'", post.Op)
	}
}

// --- Index Expression ---

func TestIndexExpr(t *testing.T) {
	prog := parseSource(t, `$items[0]`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	idx, ok := stmt.Expr.(*IndexExpr)
	if !ok {
		t.Fatalf("Expected IndexExpr, got %T", stmt.Expr)
	}
	if idx.Index == nil {
		t.Fatal("Expected index expression")
	}
}

// --- Print ---

func TestPrintExpr(t *testing.T) {
	prog := parseSource(t, `print("Hello from Wolf!")`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	printExpr, ok := stmt.Expr.(*PrintExpr)
	if !ok {
		t.Fatalf("Expected PrintExpr, got %T", stmt.Expr)
	}
	strLit, ok := printExpr.Arg.(*StringLiteral)
	if !ok {
		t.Fatalf("Expected StringLiteral, got %T", printExpr.Arg)
	}
	if strLit.Value != "Hello from Wolf!" {
		t.Errorf("Expected 'Hello from Wolf!', got '%s'", strLit.Value)
	}
}

// --- Error Expression ---

func TestErrorExpr(t *testing.T) {
	prog := parseSource(t, `error("division by zero")`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	errExpr, ok := stmt.Expr.(*ErrorExpr)
	if !ok {
		t.Fatalf("Expected ErrorExpr, got %T", stmt.Expr)
	}
	if errExpr.Message == nil {
		t.Fatal("Expected message")
	}
}

// --- Interpolated String ---

func TestInterpolatedString(t *testing.T) {
	prog := parseSource(t, `$msg = "Hello {$name}"`)
	stmt := prog.Statements[0].(*ExpressionStmt)
	binExpr := stmt.Expr.(*BinaryExpr)
	interp, ok := binExpr.Right.(*InterpolatedString)
	if !ok {
		t.Fatalf("Expected InterpolatedString, got %T", binExpr.Right)
	}
	if len(interp.Parts) != 2 {
		t.Errorf("Expected 2 parts, got %d", len(interp.Parts))
	}
}

// --- @ml Block ---

func TestMLBlock(t *testing.T) {
	prog := parseSource(t, `@ml {
		import numpy as np
		result = np.array(items).mean()
	}`)
	_, ok := prog.Statements[0].(*MLBlockStmt)
	if !ok {
		t.Fatalf("Expected MLBlockStmt, got %T", prog.Statements[0])
	}
}

// --- Import ---

func TestImportStmt(t *testing.T) {
	prog := parseSource(t, `import "math"`)
	imp, ok := prog.Statements[0].(*ImportStmt)
	if !ok {
		t.Fatalf("Expected ImportStmt, got %T", prog.Statements[0])
	}
	if imp.Path != "math" {
		t.Errorf("Expected 'math', got '%s'", imp.Path)
	}
}

// --- Integration: Full Wolf Program ---

func TestFullWolfProgram(t *testing.T) {
	src := `
$name = "Wolf"
$count = 0
$price = 9.99

func greet($name) {
	return "Hello {$name}"
}

class Driver {
	$name: string
	$rating: float
	func __construct($name, $rating) {
		$this->name = $name
		$this->rating = $rating
	}
	func summary() {
		return "Driver"
	}
}

$driver = new Driver("Ade", 4.8)
$driver->goOnline()
print($driver->summary())

if $count > 0 {
	print("has items")
} else {
	print("empty")
}

for $i = 0; $i < 10; $i++ {
	print($i)
}

foreach $items as $item {
	print($item)
}

match $status {
	"active" => handleActive()
	_ => handleDefault()
}

try {
	$user = fetchUser($id)
} catch ($e) {
	print($e)
}

$task = async { return fetchDrivers($location) }
$drivers = await $task

parallel {
	$a = compute()
	$b = compute2()
}

[$data, $err] = parseInput($raw)
`
	prog := parseSource(t, src)
	// Just verify it parses without errors and produces a reasonable number of statements
	if len(prog.Statements) < 10 {
		t.Errorf("Expected at least 10 statements for full program, got %d", len(prog.Statements))
	}
}

// --- Error Recovery ---

func TestMissingClosingBrace(t *testing.T) {
	_, errs := parseSourceExpectErrors(t, `if $x { $y = 1`)
	if len(errs) == 0 {
		t.Error("Expected parse errors for missing closing brace")
	}
}

func TestMissingParenInFunc(t *testing.T) {
	_, errs := parseSourceExpectErrors(t, `func foo $x { return 1 }`)
	if len(errs) == 0 {
		t.Error("Expected parse errors for missing parentheses")
	}
}
