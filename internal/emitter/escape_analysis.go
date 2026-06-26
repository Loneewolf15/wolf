package emitter

import (
	"github.com/wolflang/wolf/internal/ir"
)

// scanCaptured scans an AST/IR block for usages of variables that were declared OUTSIDE
// the current closure, and adds them to the captured map.
func scanCaptured(stmts []ir.Stmt, outerVars map[string]string, captured map[string]bool) {
	for _, stmt := range stmts {
		scanStmt(stmt, outerVars, captured)
	}
}

func scanStmt(stmt ir.Stmt, outerVars map[string]string, captured map[string]bool) {
	if stmt == nil {
		return
	}
	switch s := stmt.(type) {
	case *ir.ExprStmt:
		scanExpr(s.Expr, outerVars, captured)
	case *ir.VarDeclStmt:
		scanExpr(s.Value, outerVars, captured)
	case *ir.AssignStmt:
		scanExpr(s.Target, outerVars, captured)
		scanExpr(s.Value, outerVars, captured)
	case *ir.ReturnStmt:
		for _, v := range s.Values {
			scanExpr(v, outerVars, captured)
		}
	case *ir.IfStmt:
		scanExpr(s.Condition, outerVars, captured)
		scanCaptured(s.Body, outerVars, captured)
		for _, ei := range s.ElseIfs {
			scanExpr(ei.Condition, outerVars, captured)
			scanCaptured(ei.Body, outerVars, captured)
		}
		if s.ElseBody != nil {
			scanCaptured(s.ElseBody, outerVars, captured)
		}
	case *ir.ForStmt:
		scanStmt(s.Init, outerVars, captured)
		scanExpr(s.Cond, outerVars, captured)
		scanStmt(s.Update, outerVars, captured)
		scanCaptured(s.Body, outerVars, captured)
	case *ir.RangeStmt:
		scanExpr(s.Iterable, outerVars, captured)
		scanCaptured(s.Body, outerVars, captured)
	case *ir.SwitchStmt:
		scanExpr(s.Subject, outerVars, captured)
		for _, c := range s.Cases {
			scanExpr(c.Value, outerVars, captured)
			scanCaptured(c.Body, outerVars, captured)
		}
		if s.Default != nil {
			scanCaptured(s.Default, outerVars, captured)
		}
	case *ir.TryCatchStmt:
		scanCaptured(s.TryBody, outerVars, captured)
		scanCaptured(s.CatchBody, outerVars, captured)
	case *ir.BlockStmt:
		scanCaptured(s.Stmts, outerVars, captured)
	case *ir.RouteStmt:
		// string method/path, string handler
	case *ir.ServeStmt:
		scanExpr(s.Port, outerVars, captured)
		scanExpr(s.Handler, outerVars, captured)
	case *ir.RespondStmt:
		scanExpr(s.Status, outerVars, captured)
		scanExpr(s.Body, outerVars, captured)
	}
}

func scanExpr(expr ir.Expr, outerVars map[string]string, captured map[string]bool) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *ir.Ident:
		if _, exists := outerVars[e.Name]; exists {
			captured[e.Name] = true
		}
	case *ir.BinaryExpr:
		scanExpr(e.Left, outerVars, captured)
		scanExpr(e.Right, outerVars, captured)
	case *ir.UnaryExpr:
		scanExpr(e.Operand, outerVars, captured)
	case *ir.PostfixExpr:
		scanExpr(e.Operand, outerVars, captured)
	case *ir.CallExpr:
		scanExpr(e.Callee, outerVars, captured)
		for _, arg := range e.Args {
			scanExpr(arg, outerVars, captured)
		}
	case *ir.FieldAccess:
		scanExpr(e.Object, outerVars, captured)
	case *ir.MethodCallExpr:
		scanExpr(e.Object, outerVars, captured)
		for _, arg := range e.Args {
			scanExpr(arg, outerVars, captured)
		}
	case *ir.StaticCall:
		for _, arg := range e.Args {
			scanExpr(arg, outerVars, captured)
		}
	case *ir.IndexExpr:
		scanExpr(e.Object, outerVars, captured)
		scanExpr(e.Index, outerVars, captured)
	case *ir.SliceLit:
		for _, elem := range e.Elements {
			scanExpr(elem, outerVars, captured)
		}
	case *ir.MapLit:
		for _, k := range e.Keys {
			scanExpr(k, outerVars, captured)
		}
		for _, v := range e.Values {
			scanExpr(v, outerVars, captured)
		}
	case *ir.FuncLit:
		// Closures inside closures: they can capture outer variables too!
		scanCaptured(e.Body, outerVars, captured)
	case *ir.ChanMake, *ir.ChanRecv:
		// if channel has expr
		if recv, ok := e.(*ir.ChanRecv); ok {
			scanExpr(recv.Channel, outerVars, captured)
		}
	case *ir.ChanSend:
		scanExpr(e.Channel, outerVars, captured)
		scanExpr(e.Value, outerVars, captured)
	case *ir.StringConcat:
		scanExpr(e.Left, outerVars, captured)
		scanExpr(e.Right, outerVars, captured)
	case *ir.FmtSprintf:
		for _, arg := range e.Args {
			scanExpr(arg, outerVars, captured)
		}
	}
}
