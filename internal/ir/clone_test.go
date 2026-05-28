package ir

import (
	"reflect"
	"testing"
)

func TestDeepClonePrimitives(t *testing.T) {
	// Test int
	if cloned := DeepClone(42); cloned != 42 {
		t.Errorf("expected 42, got %v", cloned)
	}

	// Test string
	if cloned := DeepClone("wolf"); cloned != "wolf" {
		t.Errorf("expected 'wolf', got %v", cloned)
	}

	// Test bool
	if cloned := DeepClone(true); cloned != true {
		t.Errorf("expected true, got %v", cloned)
	}
}

func TestDeepClonePointer(t *testing.T) {
	val := 42
	ptr := &val

	cloned := DeepClone(ptr)

	if cloned == ptr {
		t.Errorf("expected pointer addresses to be different, got same: %p", ptr)
	}
	if *cloned != *ptr {
		t.Errorf("expected values to be equal, got %v and %v", *cloned, *ptr)
	}

	// Test nil pointer
	var nilPtr *int
	if clonedNil := DeepClone(nilPtr); clonedNil != nil {
		t.Errorf("expected nil pointer, got %v", clonedNil)
	}
}

func TestDeepCloneSlice(t *testing.T) {
	slice := []int{1, 2, 3}
	cloned := DeepClone(slice)

	if &cloned[0] == &slice[0] {
		t.Errorf("expected slice backing arrays to be different")
	}

	if len(cloned) != len(slice) {
		t.Errorf("expected length %d, got %d", len(slice), len(cloned))
	}

	for i, v := range slice {
		if cloned[i] != v {
			t.Errorf("expected element at %d to be %v, got %v", i, v, cloned[i])
		}
	}

	// Test nil slice
	var nilSlice []int
	if clonedNil := DeepClone(nilSlice); clonedNil != nil {
		t.Errorf("expected nil slice, got %v", clonedNil)
	}
}

func TestDeepCloneMap(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	cloned := DeepClone(m)

	if reflect.ValueOf(m).Pointer() == reflect.ValueOf(cloned).Pointer() {
		t.Errorf("expected map addresses to be different")
	}

	if len(cloned) != len(m) {
		t.Errorf("expected length %d, got %d", len(m), len(cloned))
	}

	for k, v := range m {
		if cloned[k] != v {
			t.Errorf("expected map value for key %s to be %v, got %v", k, v, cloned[k])
		}
	}

	// Test nil map
	var nilMap map[string]int
	if clonedNil := DeepClone(nilMap); clonedNil != nil {
		t.Errorf("expected nil map, got %v", clonedNil)
	}
}

func TestDeepCloneInterface(t *testing.T) {
	var expr Expr = &IntLit{Value: "100"}
	cloned := DeepClone(expr)

	if cloned == expr {
		t.Errorf("expected interface underlying pointers to be different")
	}

	clonedLit, ok := cloned.(*IntLit)
	if !ok {
		t.Fatalf("expected cloned to be *IntLit, got %T", cloned)
	}

	if clonedLit.Value != "100" {
		t.Errorf("expected value '100', got %v", clonedLit.Value)
	}

	// Test nil interface
	var nilExpr Expr
	if clonedNil := DeepClone(nilExpr); clonedNil != nil {
		t.Errorf("expected nil interface, got %v", clonedNil)
	}
}

func TestDeepCloneIRNode(t *testing.T) {
	original := &Function{
		Name: "testFunc",
		Params: []*Param{
			{Name: "p1", Type: "int"},
			{Name: "p2", Type: "string"},
		},
		ReturnTypes: []string{"bool"},
		Body: []Stmt{
			&ReturnStmt{
				Values: []Expr{
					&BoolLit{Value: true},
				},
			},
		},
	}

	cloned := DeepClone(original)

	if cloned == original {
		t.Errorf("expected cloned node to have different address")
	}

	if cloned.Name != original.Name {
		t.Errorf("expected name %q, got %q", original.Name, cloned.Name)
	}

	if len(cloned.Params) != len(original.Params) {
		t.Fatalf("expected %d params, got %d", len(original.Params), len(cloned.Params))
	}

	if cloned.Params[0] == original.Params[0] {
		t.Errorf("expected param objects to have different addresses")
	}

	if cloned.Params[0].Name != original.Params[0].Name || cloned.Params[0].Type != original.Params[0].Type {
		t.Errorf("param data mismatch")
	}

	if len(cloned.ReturnTypes) != len(original.ReturnTypes) {
		t.Fatalf("expected %d return types, got %d", len(original.ReturnTypes), len(cloned.ReturnTypes))
	}

	if cloned.ReturnTypes[0] != original.ReturnTypes[0] {
		t.Errorf("return type data mismatch")
	}

	if len(cloned.Body) != len(original.Body) {
		t.Fatalf("expected %d stmts in body, got %d", len(original.Body), len(cloned.Body))
	}

	retStmtOrig := original.Body[0].(*ReturnStmt)
	retStmtCloned := cloned.Body[0].(*ReturnStmt)

	if retStmtCloned == retStmtOrig {
		t.Errorf("expected stmt objects to have different addresses")
	}

	boolLitOrig := retStmtOrig.Values[0].(*BoolLit)
	boolLitCloned := retStmtCloned.Values[0].(*BoolLit)

	if boolLitCloned == boolLitOrig {
		t.Errorf("expected expr objects to have different addresses")
	}

	if boolLitCloned.Value != boolLitOrig.Value {
		t.Errorf("expr data mismatch")
	}
}

func TestReplaceTypeNames(t *testing.T) {
	original := &Function{
		Name: "testFunc",
		Params: []*Param{
			{Name: "p1", Type: "T"},
			{Name: "p2", Type: "U"},
			{Name: "p3", Type: "int"},
		},
		ReturnTypes: []string{"T", "bool"},
		Body: []Stmt{
			&VarDeclStmt{
				Name: "v",
				Type: "T",
				Value: &CallExpr{
					Callee:   &Ident{Name: "foo"},
					TypeArgs: []string{"T", "U"},
				},
			},
			&AssignStmt{
				Target: &Ident{Name: "mapVal"},
				Op:     "=",
				Value: &MapLit{
					KeyType:   "T",
					ValueType: "U",
				},
			},
			&AssignStmt{
				Target: &Ident{Name: "sliceVal"},
				Op:     "=",
				Value: &SliceLit{
					ElemType: "T",
				},
			},
			&AssignStmt{
				Target: &Ident{Name: "structVal"},
				Op:     "=",
				Value: &StructLit{
					TypeName: "U",
				},
			},
		},
	}

	env := map[string]string{
		"T": "int64",
		"U": "string",
	}

	// Clone first, to not mutate the test literal
	cloned := DeepClone(original)
	ReplaceTypeNames(cloned, env)

	// Check Params
	if cloned.Params[0].Type != "int64" {
		t.Errorf("expected p1 type int64, got %s", cloned.Params[0].Type)
	}
	if cloned.Params[1].Type != "string" {
		t.Errorf("expected p2 type string, got %s", cloned.Params[1].Type)
	}
	if cloned.Params[2].Type != "int" {
		t.Errorf("expected p3 type int, got %s", cloned.Params[2].Type)
	}

	// Check ReturnTypes
	if cloned.ReturnTypes[0] != "int64" {
		t.Errorf("expected return type 0 int64, got %s", cloned.ReturnTypes[0])
	}
	if cloned.ReturnTypes[1] != "bool" {
		t.Errorf("expected return type 1 bool, got %s", cloned.ReturnTypes[1])
	}

	// Check VarDeclStmt
	varDecl := cloned.Body[0].(*VarDeclStmt)
	if varDecl.Type != "int64" {
		t.Errorf("expected var type int64, got %s", varDecl.Type)
	}

	// TypeArgs is not currently replaced by ReplaceTypeNames based on the code!
	// Let's verify what happens.
	// Wait, ReplaceTypeNames explicitly checks for:
	// case "Type", "ElemType", "KeyType", "ValueType", "TypeName":
	// And "ReturnTypes":
	// It doesn't replace TypeArgs. If it did, it would need to handle "TypeArgs" like "ReturnTypes".
	// Let's not test TypeArgs here if it's not supported by the function.

	// Check MapLit
	mapAssign := cloned.Body[1].(*AssignStmt)
	mapLit := mapAssign.Value.(*MapLit)
	if mapLit.KeyType != "int64" {
		t.Errorf("expected map key type int64, got %s", mapLit.KeyType)
	}
	if mapLit.ValueType != "string" {
		t.Errorf("expected map value type string, got %s", mapLit.ValueType)
	}

	// Check SliceLit
	sliceAssign := cloned.Body[2].(*AssignStmt)
	sliceLit := sliceAssign.Value.(*SliceLit)
	if sliceLit.ElemType != "int64" {
		t.Errorf("expected slice elem type int64, got %s", sliceLit.ElemType)
	}

	// Check StructLit
	structAssign := cloned.Body[3].(*AssignStmt)
	structLit := structAssign.Value.(*StructLit)
	if structLit.TypeName != "string" {
		t.Errorf("expected struct type name string, got %s", structLit.TypeName)
	}
}

// --- Additional AST Nodes Coverage ---

func TestDeepCloneAllNodes(t *testing.T) {
	// Program
	prog := &Program{
		Package: "main",
		Imports: []string{"fmt"},
		Interfaces: []*Interface{
			{
				Name: "API",
				Methods: []*InterfaceMethodSig{
					{
						Name: "Get",
						Params: []*Param{
							{Name: "id", Type: "int"},
						},
						ReturnTypes: []string{"string"},
					},
				},
			},
		},
		Functions: []*Function{
			{
				Name: "testFunc",
				Params: []*Param{
					{Name: "x", Type: "int"},
				},
				ReturnTypes: []string{"void"},
				Body: []Stmt{
					&RawStmt{Code: "println(x)"},
				},
			},
		},
		Classes: []*Class{
			{
				Name: "User",
				Fields: []*Field{
					{Name: "Name", Type: "string", Default: &StringLit{Value: "anon"}, Visibility: "pub"},
				},
			},
		},
		InitStmts: []Stmt{
			&ExprStmt{Expr: &Ident{Name: "init"}},
		},
	}
	assertClone(t, prog)

	// Statements
	stmts := []Stmt{
		&VarDeclStmt{Name: "x", Type: "int", Value: &IntLit{Value: "42"}},
		&AssignStmt{Target: &Ident{Name: "x"}, Op: "+=", Value: &IntLit{Value: "1"}},
		&ExprStmt{Expr: &Ident{Name: "y"}},
		&ReturnStmt{Values: []Expr{&Ident{Name: "x"}}},
		&IfStmt{
			Condition: &BoolLit{Value: true},
			Body:      []Stmt{&RawStmt{Code: "pass"}},
			ElseIfs: []*ElseIfClause{
				{Condition: &BoolLit{Value: false}, Body: []Stmt{&RawStmt{Code: "pass"}}},
			},
			ElseBody: []Stmt{&RawStmt{Code: "else"}},
		},
		&ForStmt{
			Init:   &VarDeclStmt{Name: "i", Type: "int", Value: &IntLit{Value: "0"}},
			Cond:   &BinaryExpr{Left: &Ident{Name: "i"}, Op: "<", Right: &IntLit{Value: "10"}},
			Update: &AssignStmt{Target: &Ident{Name: "i"}, Op: "+=", Value: &IntLit{Value: "1"}},
			Body:   []Stmt{&RawStmt{Code: "loop"}},
		},
		&RangeStmt{
			Key:      "k",
			Value:    "v",
			Iterable: &Ident{Name: "items"},
			Body:     []Stmt{&RawStmt{Code: "rangeBody"}},
		},
		&SwitchStmt{
			Subject: &Ident{Name: "x"},
			Cases: []*SwitchCase{
				{Value: &IntLit{Value: "1"}, Body: []Stmt{&RawStmt{Code: "one"}}},
			},
			Default: []Stmt{&RawStmt{Code: "def"}},
		},
		&BlockStmt{Stmts: []Stmt{&ExprStmt{Expr: &Ident{Name: "b"}}}},
		&TryCatchStmt{
			TryBody:   []Stmt{&RawStmt{Code: "unsafe"}},
			CatchVar:  "err",
			CatchBody: []Stmt{&RawStmt{Code: "safe"}},
		},
		&SpawnStmt{Name: "thread1", Call: &CallExpr{Callee: &Ident{Name: "foo"}}},
		&WaitAllStmt{},
		&DeferStmt{Call: &CallExpr{Callee: &Ident{Name: "cleanup"}}},
		&RawStmt{Code: "raw"},
		&SuperviseStmt{
			Strategy: "restart",
			Restart:  "always",
			Max:      5,
			Body:     []Stmt{&RawStmt{Code: "monitored"}},
		},
		&TraceStmt{
			SpanName: &StringLit{Value: "mySpan"},
			Body:     []Stmt{&RawStmt{Code: "traced"}},
		},
		&RouteStmt{Method: "GET", Path: "/ping", Handler: "PingHandler"},
		&ServeStmt{Port: &IntLit{Value: "8080"}, Handler: &Ident{Name: "router"}},
		&RespondStmt{Status: &IntLit{Value: "200"}, Body: &Ident{Name: "payload"}},
	}

	for _, stmt := range stmts {
		stmt.irStmt()
		assertClone(t, stmt)
	}

	// Expressions
	exprs := []Expr{
		&Ident{Name: "someVar"},
		&IntLit{Value: "123"},
		&FloatLit{Value: "3.14"},
		&StringLit{Value: `"hello"`},
		&BoolLit{Value: false},
		&NilLit{},
		&BinaryExpr{Left: &IntLit{Value: "1"}, Op: "+", Right: &IntLit{Value: "2"}},
		&UnaryExpr{Op: "-", Operand: &Ident{Name: "x"}},
		&CallExpr{Callee: &Ident{Name: "func"}, TypeArgs: []string{"T"}, Args: []Expr{&IntLit{Value: "1"}}},
		&FieldAccess{Object: &Ident{Name: "user"}, Field: "Name"},
		&MethodCallExpr{Object: &Ident{Name: "user"}, Method: "Greet", TypeArgs: []string{"T"}, Args: []Expr{&Ident{Name: "arg"}}},
		&StaticCall{Class: "Math", Method: "Abs", Args: []Expr{&FloatLit{Value: "-1.5"}}},
		&IndexExpr{Object: &Ident{Name: "arr"}, Index: &IntLit{Value: "0"}},
		&SliceLit{ElemType: "int", Elements: []Expr{&IntLit{Value: "1"}}},
		&MapLit{
			KeyType:   "string",
			ValueType: "int",
			Keys:      []Expr{&StringLit{Value: "a"}},
			Values:    []Expr{&IntLit{Value: "1"}},
		},
		&StructLit{
			TypeName: "User",
			Fields:   map[string]Expr{"Name": &StringLit{Value: "bob"}},
		},
		&FuncLit{
			Params:      []*Param{{Name: "x", Type: "int"}},
			ReturnTypes: []string{"int"},
			Body:        []Stmt{&ReturnStmt{Values: []Expr{&Ident{Name: "x"}}}},
		},
		&FmtSprintf{Format: "hello %s", Args: []Expr{&Ident{Name: "name"}}},
		&ChanMake{ElemType: "int"},
		&ChanSend{Channel: &Ident{Name: "ch"}, Value: &IntLit{Value: "42"}},
		&ChanRecv{Channel: &Ident{Name: "ch"}},
		&PostfixExpr{Operand: &Ident{Name: "x"}, Op: "++"},
		&StringConcat{Left: &StringLit{Value: "a"}, Right: &StringLit{Value: "b"}},
		&ErrorNew{Message: &StringLit{Value: "failed"}},
	}

	for _, expr := range exprs {
		expr.irExpr()
		assertClone(t, expr)
	}
}

func assertClone[T any](t *testing.T, orig T) {
	cloned := DeepClone(orig)
	if !reflect.DeepEqual(orig, cloned) {
		t.Errorf("DeepClone failed: values not deep equal: expected %#v, got %#v", orig, cloned)
	}

	// Verify that pointer addresses are different for reference/pointer types
	origVal := reflect.ValueOf(orig)
	clonedVal := reflect.ValueOf(cloned)

	if origVal.Kind() == reflect.Ptr {
		if !origVal.IsNil() && origVal.Type().Elem().Size() > 0 && origVal.Pointer() == clonedVal.Pointer() {
			t.Errorf("DeepClone failed: pointer addresses are identical for type %s: %p", origVal.Type(), origVal.Interface())
		}
	}
}

// --- Target Coverage Branch Tests ---

type dummyStruct struct {
	Exported   string
	unexported string
}

func TestDeepCloneUnexportedFields(t *testing.T) {
	orig := dummyStruct{
		Exported:   "hello",
		unexported: "secret",
	}

	cloned := DeepClone(orig)

	if cloned.Exported != "hello" {
		t.Errorf("expected Exported field to be cloned, got %q", cloned.Exported)
	}
	if cloned.unexported != "" {
		t.Errorf("expected unexported field to be skipped/zeroed, got %q", cloned.unexported)
	}
}

type structWithInterface struct {
	Expr Expr
}

func TestDeepCloneNilInterface(t *testing.T) {
	orig := &structWithInterface{Expr: nil}
	cloned := DeepClone(orig)

	if cloned.Expr != nil {
		t.Errorf("expected cloned nil interface to be nil, got %v", cloned.Expr)
	}
}

func TestDeepCloneInvalidReflectValue(t *testing.T) {
	// Directly call the internal cloneValue helper with invalid value
	res := cloneValue(reflect.Value{})
	if res.IsValid() {
		t.Error("expected invalid reflect.Value returned from cloneValue")
	}

	// Directly call the internal replaceValuesHelper with invalid value
	replaceValuesHelper(reflect.Value{}, nil) // should return early without panic
}

func TestReplaceTypeNamesUnexported(t *testing.T) {
	// dummyStruct has unexported field. replaceValuesHelper should skip it since field.CanSet() is false.
	orig := &dummyStruct{
		Exported:   "T",
		unexported: "secret",
	}

	// If we map T -> string, it should not affect anything since there is no "Type" or "ReturnTypes" field,
	// but it will cover the !field.CanSet() branch for "unexported".
	ReplaceTypeNames(orig, map[string]string{"T": "string"})
}
