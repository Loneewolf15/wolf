// Package emitter transforms the Wolf AST into WIR (Wolf IR).
package emitter

import (
	"strings"

	"github.com/wolflang/wolf/internal/ir"
	"github.com/wolflang/wolf/internal/parser"
	"github.com/wolflang/wolf/internal/resolver"
)

// IREmitter transforms a Wolf AST into WIR.
type IREmitter struct {
	resolver     *resolver.Resolver
	imports      map[string]bool // track needed imports
	declared     map[string]bool // track declared variables (for := vs =)
	httpHandlers map[string]bool // track functions used as HTTP handlers
}

// New creates a new IREmitter.
func New(res *resolver.Resolver) *IREmitter {
	return &IREmitter{
		resolver:     res,
		imports:      make(map[string]bool),
		declared:     make(map[string]bool),
		httpHandlers: make(map[string]bool),
	}
}

// Emit transforms the AST program into a WIR program.
func (e *IREmitter) Emit(program *parser.Program) *ir.Program {
	irProg := &ir.Program{
		Package: "main",
	}

	// Pre-scan: find route() calls to mark handler functions
	e.scanForHTTPHandlers(program)

	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *parser.FuncDecl:
			fn := e.emitFunction(s)
			// If this function is used as an HTTP handler, inject w/r params
			if e.httpHandlers[s.Name] {
				httpParams := []*ir.Param{
					{Name: "w", Type: "http.ResponseWriter"},
					{Name: "r", Type: "*http.Request"},
				}
				fn.Params = append(httpParams, fn.Params...)
			}
			irProg.Functions = append(irProg.Functions, fn)
		case *parser.ClassDecl:
			cls := e.emitClass(s)
			irProg.Classes = append(irProg.Classes, cls)
		default:
			irStmt := e.emitStmt(stmt)
			if irStmt != nil {
				irProg.InitStmts = append(irProg.InitStmts, irStmt)
			}
		}
	}

	// Collect imports
	for imp := range e.imports {
		irProg.Imports = append(irProg.Imports, imp)
	}

	return irProg
}

// scanForHTTPHandlers finds route() calls and marks the handler function names.
func (e *IREmitter) scanForHTTPHandlers(program *parser.Program) {
	for _, stmt := range program.Statements {
		if exprStmt, ok := stmt.(*parser.ExpressionStmt); ok {
			if call, ok := exprStmt.Expr.(*parser.CallExpr); ok {
				if ident, ok := call.Callee.(*parser.Identifier); ok {
					if ident.Name == "route" && len(call.Args) >= 3 {
						if handlerIdent, ok := call.Args[2].(*parser.Identifier); ok {
							name := handlerIdent.Name
							if len(name) > 0 && name[0] == '$' {
								name = name[1:]
							}
							e.httpHandlers[name] = true
						}
					}
				}
			}
		}
	}
}

// ========== Statement Emission ==========

func (e *IREmitter) emitStmt(stmt parser.Statement) ir.Stmt {
	switch s := stmt.(type) {
	case *parser.ExpressionStmt:
		return e.emitExprStmt(s)
	case *parser.VarDecl:
		return e.emitVarDecl(s)
	case *parser.AssignStmt:
		return e.emitAssignStmt(s)
	case *parser.ReturnStmt:
		return e.emitReturnStmt(s)
	case *parser.IfStmt:
		return e.emitIfStmt(s)
	case *parser.ForStmt:
		return e.emitForStmt(s)
	case *parser.ForeachStmt:
		return e.emitForeachStmt(s)
	case *parser.WhileStmt:
		return e.emitWhileStmt(s)
	case *parser.MatchStmt:
		return e.emitMatchStmt(s)
	case *parser.TryCatchStmt:
		return e.emitTryCatch(s)
	case *parser.ParallelStmt:
		return e.emitParallel(s)
	case *parser.DestructureAssign:
		return e.emitDestructure(s)
	case *parser.BlockStmt:
		return e.emitBlock(s)
	default:
		return nil
	}
}

func (e *IREmitter) emitExprStmt(s *parser.ExpressionStmt) ir.Stmt {
	// Check if this is an assignment: BinaryExpr with = op
	if bin, ok := s.Expr.(*parser.BinaryExpr); ok {
		if bin.Op == "=" || bin.Op == "+=" || bin.Op == "-=" || bin.Op == "*=" || bin.Op == "/=" {
			target := e.emitExpr(bin.Left)
			op := bin.Op

			// For simple = assignments, check if this is the first assignment
			// to this variable — if so, use := (Go short variable declaration)
			if op == "=" {
				if ident, ok := target.(*ir.Ident); ok {
					if !e.declared[ident.Name] {
						op = ":="
						e.declared[ident.Name] = true
					}
				}
			}

			return &ir.AssignStmt{
				Target: target,
				Op:     op,
				Value:  e.emitExpr(bin.Right),
			}
		}
	}

	// Check for HTTP built-in calls: route(), serve(), respond()
	if call, ok := s.Expr.(*parser.CallExpr); ok {
		if ident, ok := call.Callee.(*parser.Identifier); ok {
			switch ident.Name {
			case "route":
				return e.emitRouteCall(call)
			case "serve":
				return e.emitServeCall(call)
			case "respond":
				return e.emitRespondCall(call)
			}
		}
	}

	return &ir.ExprStmt{Expr: e.emitExpr(s.Expr)}
}

// emitRouteCall: route("GET", "/path", handler_func)
func (e *IREmitter) emitRouteCall(call *parser.CallExpr) ir.Stmt {
	e.imports["net/http"] = true
	e.imports["encoding/json"] = true
	e.imports["log"] = true

	method := ""
	path := ""
	handler := ""

	if len(call.Args) >= 2 {
		if lit, ok := call.Args[0].(*parser.StringLiteral); ok {
			method = lit.Value
		}
		if lit, ok := call.Args[1].(*parser.StringLiteral); ok {
			path = lit.Value
		}
	}
	if len(call.Args) >= 3 {
		if ident, ok := call.Args[2].(*parser.Identifier); ok {
			handler = e.resolveGoName(ident.Name)
		}
	}

	return &ir.RouteStmt{
		Method:  method,
		Path:    path,
		Handler: handler,
	}
}

// emitServeCall: serve(8080)
func (e *IREmitter) emitServeCall(call *parser.CallExpr) ir.Stmt {
	e.imports["net/http"] = true
	e.imports["fmt"] = true
	e.imports["log"] = true

	var port ir.Expr
	var handler ir.Expr

	if len(call.Args) >= 1 {
		port = e.emitExpr(call.Args[0])
	} else {
		port = &ir.IntLit{Value: "8080"}
	}

	if len(call.Args) >= 2 {
		handler = e.emitExpr(call.Args[1])
	}

	return &ir.ServeStmt{Port: port, Handler: handler}
}

// emitRespondCall: respond(200, {"key": "value"})
func (e *IREmitter) emitRespondCall(call *parser.CallExpr) ir.Stmt {
	e.imports["net/http"] = true
	e.imports["encoding/json"] = true

	var status ir.Expr
	var body ir.Expr

	if len(call.Args) >= 1 {
		status = e.emitExpr(call.Args[0])
	}
	if len(call.Args) >= 2 {
		body = e.emitExpr(call.Args[1])
	}

	return &ir.RespondStmt{
		Status: status,
		Body:   body,
	}
}

func (e *IREmitter) emitVarDecl(s *parser.VarDecl) ir.Stmt {
	goName := e.resolveGoName(s.Name)
	goType := e.wolfTypeToGo(s.TypeName)

	var value ir.Expr
	if s.Value != nil {
		value = e.emitExpr(s.Value)
	}

	return &ir.VarDeclStmt{
		Name:  goName,
		Type:  goType,
		Value: value,
	}
}

func (e *IREmitter) emitAssignStmt(s *parser.AssignStmt) ir.Stmt {
	target := e.emitExpr(s.Target)
	op := s.Op

	// Use := for first-time variable assignments
	if op == "=" {
		if ident, ok := target.(*ir.Ident); ok {
			if !e.declared[ident.Name] {
				op = ":="
				e.declared[ident.Name] = true
			}
		}
	}

	return &ir.AssignStmt{
		Target: target,
		Op:     op,
		Value:  e.emitExpr(s.Value),
	}
}

func (e *IREmitter) emitReturnStmt(s *parser.ReturnStmt) ir.Stmt {
	var values []ir.Expr
	for _, v := range s.Values {
		values = append(values, e.emitExpr(v))
	}
	return &ir.ReturnStmt{Values: values}
}

func (e *IREmitter) emitIfStmt(s *parser.IfStmt) ir.Stmt {
	result := &ir.IfStmt{
		Condition: e.emitExpr(s.Condition),
		Body:      e.emitStmts(s.Body.Statements),
	}

	for _, eif := range s.ElseIfs {
		result.ElseIfs = append(result.ElseIfs, &ir.ElseIfClause{
			Condition: e.emitExpr(eif.Condition),
			Body:      e.emitStmts(eif.Body.Statements),
		})
	}

	if s.ElseBody != nil {
		result.ElseBody = e.emitStmts(s.ElseBody.Statements)
	}

	return result
}

func (e *IREmitter) emitForStmt(s *parser.ForStmt) ir.Stmt {
	return &ir.ForStmt{
		Init:   e.emitStmt(s.Init),
		Cond:   e.emitExpr(s.Condition),
		Update: e.emitStmt(s.Update),
		Body:   e.emitStmts(s.Body.Statements),
	}
}

func (e *IREmitter) emitForeachStmt(s *parser.ForeachStmt) ir.Stmt {
	keyName := "_"
	if s.KeyVar != "" {
		keyName = e.resolveGoName(s.KeyVar)
	}
	valueName := e.resolveGoName(s.ValueVar)

	return &ir.RangeStmt{
		Key:      keyName,
		Value:    valueName,
		Iterable: e.emitExpr(s.Iterable),
		Body:     e.emitStmts(s.Body.Statements),
	}
}

func (e *IREmitter) emitWhileStmt(s *parser.WhileStmt) ir.Stmt {
	// Go has no while — use for { if !cond { break } }
	return &ir.ForStmt{
		Init:   nil,
		Cond:   e.emitExpr(s.Condition),
		Update: nil,
		Body:   e.emitStmts(s.Body.Statements),
	}
}

func (e *IREmitter) emitMatchStmt(s *parser.MatchStmt) ir.Stmt {
	result := &ir.SwitchStmt{
		Subject: e.emitExpr(s.Subject),
	}

	for _, arm := range s.Arms {
		if _, isWild := arm.Pattern.(*parser.Wildcard); isWild {
			result.Default = e.emitStmtsFromSlice(arm.Body)
		} else {
			caseStmt := &ir.SwitchCase{
				Value: e.emitExpr(arm.Pattern),
				Body:  e.emitStmtsFromSlice(arm.Body),
			}
			result.Cases = append(result.Cases, caseStmt)
		}
	}

	return result
}

func (e *IREmitter) emitTryCatch(s *parser.TryCatchStmt) ir.Stmt {
	// Go doesn't have try/catch — emit as defer/recover pattern
	// Full implementation deferred to Week 7 (needs error interface wiring)
	e.imports["fmt"] = true

	catchVarName := "err"
	if s.CatchVar != "" {
		catchVarName = e.resolveGoName(s.CatchVar)
	}

	// Emit try body as-is, catch as a comment placeholder
	stmts := e.emitStmts(s.TryBody.Statements)
	stmts = append(stmts, &ir.RawStmt{
		Code: "// catch(" + catchVarName + "): Wolf try/catch → Go recover pattern",
	})

	return &ir.BlockStmt{Stmts: stmts}
}

func (e *IREmitter) emitParallel(s *parser.ParallelStmt) ir.Stmt {
	// Each statement in parallel block becomes a goroutine
	var goStmts []ir.Stmt
	for _, stmt := range s.Body.Statements {
		goStmts = append(goStmts, &ir.GoStmt{
			Body: []ir.Stmt{e.emitStmt(stmt)},
		})
	}
	return &ir.BlockStmt{Stmts: goStmts}
}

func (e *IREmitter) emitDestructure(s *parser.DestructureAssign) ir.Stmt {
	// [$a, $b] = expr  →  a, b := expr (in Go)
	// Emit as raw for now since multi-return assignment is complex
	var names []string
	for _, n := range s.Names {
		names = append(names, e.resolveGoName(n))
	}
	value := e.emitExpr(s.Value)
	return &ir.AssignStmt{
		Target: &ir.Ident{Name: joinNames(names)},
		Op:     ":=",
		Value:  value,
	}
}

func (e *IREmitter) emitBlock(s *parser.BlockStmt) ir.Stmt {
	return &ir.BlockStmt{Stmts: e.emitStmts(s.Statements)}
}

// ========== Function & Class Emission ==========

func (e *IREmitter) emitFunction(f *parser.FuncDecl) *ir.Function {
	fn := &ir.Function{
		Name: f.Name,
	}

	// If the function has a return type, use it as default for untyped params
	// This allows `func add($a, $b) -> float { return $a + $b }` to work
	defaultParamType := "interface{}"
	if f.ReturnType != nil && len(f.ReturnType.Types) == 1 {
		inferredType := e.wolfTypeToGo(f.ReturnType.Types[0])
		if inferredType != "" && inferredType != "error" {
			defaultParamType = inferredType
		}
	}

	for _, p := range f.Params {
		goName := e.resolveGoName(p.Name)
		goType := e.wolfTypeToGo(p.TypeName)
		if goType == "" {
			goType = defaultParamType
		}
		fn.Params = append(fn.Params, &ir.Param{Name: goName, Type: goType})
	}

	if f.ReturnType != nil {
		for _, rt := range f.ReturnType.Types {
			fn.ReturnTypes = append(fn.ReturnTypes, e.wolfTypeToGo(rt))
		}
	}

	if f.Body != nil {
		fn.Body = e.emitStmts(f.Body.Statements)
	}

	if f.ArrowExpr != nil {
		fn.Body = []ir.Stmt{
			&ir.ReturnStmt{Values: []ir.Expr{e.emitExpr(f.ArrowExpr)}},
		}
	}

	return fn
}

func (e *IREmitter) emitClass(c *parser.ClassDecl) *ir.Class {
	cls := &ir.Class{
		Name:    c.Name,
		Extends: c.Extends,
	}

	for _, prop := range c.Properties {
		goType := e.wolfTypeToGo(prop.TypeName)
		if goType == "" {
			goType = "interface{}"
		}
		var def ir.Expr
		if prop.Default != nil {
			def = e.emitExpr(prop.Default)
		}
		cls.Fields = append(cls.Fields, &ir.Field{
			Name:       e.resolveGoName(prop.Name),
			Type:       goType,
			Default:    def,
			Visibility: prop.Visibility,
		})
	}

	for _, method := range c.Methods {
		fn := e.emitFunction(method)
		fn.IsMethod = true
		fn.Receiver = c.Name
		if method.Name == "__construct" {
			fn.Name = "New" + c.Name
			fn.IsMethod = false // constructor is a standalone function
			cls.Constructor = fn
		} else {
			cls.Methods = append(cls.Methods, fn)
		}
	}

	return cls
}

// ========== Expression Emission ==========

func (e *IREmitter) emitExpr(expr parser.Expression) ir.Expr {
	if expr == nil {
		return &ir.NilLit{}
	}

	switch ex := expr.(type) {
	case *parser.DollarIdent:
		return &ir.Ident{Name: e.resolveGoName(ex.Name)}

	case *parser.Identifier:
		return &ir.Ident{Name: ex.Name}

	case *parser.IntLiteral:
		return &ir.IntLit{Value: ex.Value}

	case *parser.FloatLiteral:
		return &ir.FloatLit{Value: ex.Value}

	case *parser.StringLiteral:
		return &ir.StringLit{Value: ex.Value}

	case *parser.BoolLiteral:
		return &ir.BoolLit{Value: ex.Value}

	case *parser.NilLiteral:
		return &ir.NilLit{}

	case *parser.InterpolatedString:
		return e.emitInterpolatedString(ex)

	case *parser.BinaryExpr:
		return &ir.BinaryExpr{
			Left:  e.emitExpr(ex.Left),
			Op:    ex.Op,
			Right: e.emitExpr(ex.Right),
		}

	case *parser.UnaryExpr:
		return &ir.UnaryExpr{Op: ex.Op, Operand: e.emitExpr(ex.Operand)}

	case *parser.PostfixExpr:
		return &ir.PostfixExpr{Operand: e.emitExpr(ex.Operand), Op: ex.Op}

	case *parser.CallExpr:
		var args []ir.Expr
		for _, a := range ex.Args {
			args = append(args, e.emitExpr(a))
		}
		// Flatten named args into positional for now
		for _, na := range ex.NamedArgs {
			args = append(args, e.emitExpr(na.Value))
		}
		return &ir.CallExpr{Callee: e.emitExpr(ex.Callee), Args: args}

	case *parser.PropertyAccess:
		return &ir.FieldAccess{
			Object: e.emitExpr(ex.Object),
			Field:  ex.Property,
		}

	case *parser.MethodCall:
		var args []ir.Expr
		for _, a := range ex.Args {
			args = append(args, e.emitExpr(a))
		}
		if ident, ok := ex.Object.(*parser.Identifier); ok {
			if strings.HasPrefix(ident.Name, "Wolf_") {
				e.imports["github.com/wolflang/wolf/stdlib"] = true
			}
		}
		return &ir.MethodCallExpr{
			Object: e.emitExpr(ex.Object),
			Method: ex.Method,
			Args:   args,
		}

	case *parser.StaticCall:
		var args []ir.Expr
		for _, a := range ex.Args {
			args = append(args, e.emitExpr(a))
		}
		if strings.HasPrefix(ex.Class, "Wolf_") {
			e.imports["github.com/wolflang/wolf/stdlib"] = true
		}
		return &ir.StaticCall{
			Class:  ex.Class,
			Method: ex.Method,
			Args:   args,
		}

	case *parser.IndexExpr:
		return &ir.IndexExpr{
			Object: e.emitExpr(ex.Object),
			Index:  e.emitExpr(ex.Index),
		}

	case *parser.ArrayLiteral:
		var elems []ir.Expr
		for _, elem := range ex.Elements {
			elems = append(elems, e.emitExpr(elem))
		}
		return &ir.SliceLit{ElemType: "interface{}", Elements: elems}

	case *parser.MapLiteral:
		var keys, vals []ir.Expr
		for i := range ex.Keys {
			keys = append(keys, e.emitExpr(ex.Keys[i]))
			vals = append(vals, e.emitExpr(ex.Values[i]))
		}
		return &ir.MapLit{
			KeyType: "string", ValueType: "interface{}",
			Keys: keys, Values: vals,
		}

	case *parser.NewExpr:
		var args []ir.Expr
		for _, a := range ex.Args {
			args = append(args, e.emitExpr(a))
		}
		return &ir.CallExpr{
			Callee: &ir.Ident{Name: "New" + ex.ClassName},
			Args:   args,
		}

	case *parser.ClosureExpr:
		fn := &ir.FuncLit{}
		for _, p := range ex.Params {
			goType := e.wolfTypeToGo(p.TypeName)
			if goType == "" {
				goType = "interface{}"
			}
			fn.Params = append(fn.Params, &ir.Param{Name: e.resolveGoName(p.Name), Type: goType})
		}
		if ex.Body != nil {
			fn.Body = e.emitStmts(ex.Body.Statements)
		}
		if ex.ArrowExpr != nil {
			fn.Body = []ir.Stmt{&ir.ReturnStmt{Values: []ir.Expr{e.emitExpr(ex.ArrowExpr)}}}
		}
		return fn

	case *parser.AsyncExpr:
		// async { } → go func() { ... }() wrapped, returning a channel
		return &ir.FuncLit{
			Body: e.emitStmts(ex.Body.Statements),
		}

	case *parser.AwaitExpr:
		// await $task → <-task (channel receive)
		return &ir.ChanRecv{Channel: e.emitExpr(ex.Expr)}

	case *parser.ChannelExpr:
		return &ir.ChanMake{ElemType: e.wolfTypeToGo(ex.ElemType)}

	case *parser.SendExpr:
		return &ir.ChanSend{Channel: e.emitExpr(ex.Channel), Value: e.emitExpr(ex.Value)}

	case *parser.ReceiveExpr:
		return &ir.ChanRecv{Channel: e.emitExpr(ex.Channel)}

	case *parser.ErrorExpr:
		e.imports["errors"] = true
		return &ir.ErrorNew{Message: e.emitExpr(ex.Message)}

	case *parser.PrintExpr:
		e.imports["fmt"] = true
		return &ir.CallExpr{
			Callee: &ir.Ident{Name: "fmt.Println"},
			Args:   []ir.Expr{e.emitExpr(ex.Arg)},
		}

	case *parser.StringConcat:
		return &ir.StringConcat{
			Left:  e.emitExpr(ex.Left),
			Right: e.emitExpr(ex.Right),
		}

	case *parser.Wildcard:
		return &ir.Ident{Name: "_"}

	default:
		return &ir.NilLit{}
	}
}

func (e *IREmitter) emitInterpolatedString(s *parser.InterpolatedString) ir.Expr {
	e.imports["fmt"] = true

	format := ""
	var args []ir.Expr

	for _, part := range s.Parts {
		switch p := part.(type) {
		case *parser.StringLiteral:
			format += p.Value
		default:
			format += "%v"
			args = append(args, e.emitExpr(part))
		}
	}

	return &ir.FmtSprintf{Format: format, Args: args}
}

// ========== Helpers ==========

func (e *IREmitter) emitStmts(stmts []parser.Statement) []ir.Stmt {
	var result []ir.Stmt
	for _, s := range stmts {
		irStmt := e.emitStmt(s)
		if irStmt != nil {
			result = append(result, irStmt)
		}
	}
	return result
}

func (e *IREmitter) emitStmtsFromSlice(stmts []parser.Statement) []ir.Stmt {
	return e.emitStmts(stmts)
}

func (e *IREmitter) resolveGoName(wolfName string) string {
	if names := e.resolver.ResolvedNames(); names != nil {
		if goName, ok := names[wolfName]; ok {
			return goName
		}
	}
	return stripDollar(wolfName)
}

func stripDollar(name string) string {
	if len(name) > 0 && name[0] == '$' {
		return name[1:]
	}
	return name
}

func (e *IREmitter) wolfTypeToGo(wolfType string) string {
	switch wolfType {
	case "int":
		return "int"
	case "float", "float64":
		return "float64"
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "error":
		return "error"
	case "any", "":
		return ""
	default:
		return wolfType // class names pass through
	}
}

func joinNames(names []string) string {
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ", "
		}
		result += n
	}
	return result
}
