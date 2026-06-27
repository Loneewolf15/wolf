package lsp

import (
	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
)

// FindNodeAtPosition returns the deepest AST Node that encloses the given line and column.
func FindNodeAtPosition(program *parser.Program, line, col int) parser.Node {
	if program == nil {
		return nil
	}

	var found parser.Node
	var bestDist int = 999999

	// A helper to check if a node's start position is before or at the cursor
	checkNode := func(n parser.Node) {
		if n == nil {
			return
		}
		pos := n.Pos()
		if pos.Line < line || (pos.Line == line && pos.Col <= col) {
			dist := (line-pos.Line)*1000 + (col - pos.Col)
			if dist >= 0 && dist <= bestDist {
				bestDist = dist
				found = n
			}
		}
	}

	// We simply traverse the entire tree and check every node
	// The deepest node that is closest to the cursor before it will win.
	var walk func(n parser.Node)
	walk = func(n parser.Node) {
		if n == nil {
			return
		}
		checkNode(n)

		switch node := n.(type) {
		case *parser.Program:
			for _, stmt := range node.Statements {
				walk(stmt)
			}
		case *parser.ExpressionStmt:
			walk(node.Expr)
		case *parser.VarDecl:
			walk(node.Value)
		case *parser.AssignStmt:
			walk(node.Target)
			walk(node.Value)
		case *parser.ReturnStmt:
			for _, v := range node.Values {
				walk(v)
			}
		case *parser.IfStmt:
			walk(node.Condition)
			walk(node.Body)
			for _, eif := range node.ElseIfs {
				walk(eif.Condition)
				walk(eif.Body)
			}
			walk(node.ElseBody)
		case *parser.ForStmt:
			walk(node.Init)
			walk(node.Condition)
			walk(node.Update)
			walk(node.Body)
		case *parser.ForeachStmt:
			walk(node.Iterable)
			walk(node.Body)
		case *parser.WhileStmt:
			walk(node.Condition)
			walk(node.Body)
		case *parser.MatchStmt:
			walk(node.Subject)
			for _, arm := range node.Arms {
				walk(arm.Pattern)
				for _, stmt := range arm.Body {
					walk(stmt)
				}
			}
		case *parser.TryCatchStmt:
			walk(node.TryBody)
			walk(node.CatchBody)
		case *parser.BlockStmt:
			for _, stmt := range node.Statements {
				walk(stmt)
			}
		case *parser.FuncDecl:
			walk(node.Body)
			walk(node.ArrowExpr)
		case *parser.ClassDecl:
			for _, prop := range node.Properties {
				walk(prop.Default)
			}
			for _, method := range node.Methods {
				walk(method)
			}
		case *parser.TraceBlockStmt:
			walk(node.SpanName)
			walk(node.Body)
		case *parser.SuperviseBlockStmt:
			walk(node.Body)
		case *parser.ParallelStmt:
			walk(node.Body)
		case *parser.AsyncExpr:
			walk(node.Body)
		case *parser.MLBlockStmt:
			walk(node.ModelName)
		case *parser.DestructureAssign:
			walk(node.Value)

		// Expressions
		case *parser.BinaryExpr:
			walk(node.Left)
			walk(node.Right)
		case *parser.UnaryExpr:
			walk(node.Operand)
		case *parser.PostfixExpr:
			walk(node.Operand)
		case *parser.CallExpr:
			walk(node.Callee)
			for _, arg := range node.Args {
				walk(arg)
			}
			for _, na := range node.NamedArgs {
				walk(na.Value)
			}
		case *parser.PropertyAccess:
			walk(node.Object)
		case *parser.MethodCall:
			walk(node.Object)
			for _, arg := range node.Args {
				walk(arg)
			}
		case *parser.IndexExpr:
			walk(node.Object)
			walk(node.Index)
		case *parser.NewExpr:
			walk(node.ClassExpr)
			for _, arg := range node.Args {
				walk(arg)
			}
		case *parser.ClosureExpr:
			walk(node.Body)
			walk(node.ArrowExpr)
		case *parser.AwaitExpr:
			walk(node.Expr)
		case *parser.SendExpr:
			walk(node.Channel)
			walk(node.Value)
		case *parser.ReceiveExpr:
			walk(node.Channel)
		case *parser.ErrorExpr:
			walk(node.Message)
		case *parser.PrintExpr:
			walk(node.Arg)
		case *parser.ArrayLiteral:
			for _, elem := range node.Elements {
				walk(elem)
			}
		case *parser.MapLiteral:
			for _, k := range node.Keys {
				walk(k)
			}
			for _, v := range node.Values {
				walk(v)
			}
		case *parser.InterpolatedString:
			for _, part := range node.Parts {
				walk(part)
			}
		case *parser.StringConcat:
			walk(node.Left)
			walk(node.Right)
		case *parser.StaticCall:
			for _, arg := range node.Args {
				walk(arg)
			}
		}
	}

	walk(program)
	return found
}

// ExprName returns the name of an identifier expression, or empty.
func ExprName(expr parser.Expression) string {
	switch e := expr.(type) {
	case *parser.DollarIdent:
		return e.Name
	case *parser.Identifier:
		return e.Name
	case *parser.PropertyAccess:
		return e.Property
	}
	return ""
}

// ConvertPos translates internal Lexer Position (1-indexed) to LSP Position (0-indexed).
func ConvertPos(pos lexer.Position) Position {
	l := pos.Line - 1
	if l < 0 {
		l = 0
	}
	c := pos.Col - 1
	if c < 0 {
		c = 0
	}
	return Position{Line: l, Character: c}
}
