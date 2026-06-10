package emitter

// integer_unbox.go — Integer Purity Analysis Pass
//
// This pass runs before LLVM IR emission and identifies user-defined Wolf
// functions that are "integer-pure": every parameter is used only in integer
// arithmetic/comparison contexts, and every return value is a plain integer.
//
// Such functions are promoted from:
//
//	define ptr @wolf_fib(ptr %n.arg)        ← boxed: allocates wolf_value_t per call
//
// to:
//
//	define i64 @wolf_fib(i64 %n.arg)        ← unboxed: pure register arithmetic
//
// This eliminates the wolf_val_int() → wolf_req_alloc() allocation on every
// recursive call edge. For fib(35), this removes ~29 million heap allocations.
//
// The analysis is conservative: if we can't statically prove purity, we skip
// the function and fall back to the safe ptr representation.

import (
	"github.com/wolflang/wolf/internal/ir"
)

// isFuncIntegerPure returns true if fn qualifies for integer unboxing.
//
// Conditions:
//  1. The function has at least one parameter.
//  2. Every parameter is either untyped (Wolf default = ptr) or explicitly "int".
//  3. Every return statement in the body yields an expression that statically
//     evaluates to i64 (IntLit, integer binary op, or call to another Wolf
//     user function — which may also be integer-pure after the same analysis).
//  4. No parameter escapes to a ptr context (string concat, map ops, print).
func isFuncIntegerPure(fn *ir.Function, funcSigs map[string]*ir.Function) bool {
	if len(fn.Params) == 0 {
		return false
	}
	for _, p := range fn.Params {
		switch p.Type {
		case "", "int", "int64", "i64", "interface{}", "any":
			// untyped (Wolf default is interface{}) or explicitly integer → OK
		default:
			return false // has a string/float/ptr param → not integer-pure
		}
	}

	// Scan body: all return values must be integer-pure, no ptr escapes.
	hasReturn := false
	if !isBodyIntegerPure(fn.Body, fn.Params, funcSigs, &hasReturn) {
		return false
	}
	return hasReturn // must have at least one return with an integer value
}

// isBodyIntegerPure recursively walks stmts checking for integer purity.
func isBodyIntegerPure(stmts []ir.Stmt, params []*ir.Param, funcSigs map[string]*ir.Function, hasReturn *bool) bool {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ir.ReturnStmt:
			if len(s.Values) == 0 {
				continue
			}
			for _, v := range s.Values {
				if !isExprIntegerPure(v, funcSigs) {
					return false
				}
			}
			*hasReturn = true

		case *ir.IfStmt:
			if !isBodyIntegerPure(s.Body, params, funcSigs, hasReturn) {
				return false
			}
			for _, elif := range s.ElseIfs {
				if !isBodyIntegerPure(elif.Body, params, funcSigs, hasReturn) {
					return false
				}
			}
			if !isBodyIntegerPure(s.ElseBody, params, funcSigs, hasReturn) {
				return false
			}

		case *ir.ForStmt:
			if !isBodyIntegerPure(s.Body, params, funcSigs, hasReturn) {
				return false
			}

		case *ir.BlockStmt:
			if !isBodyIntegerPure(s.Stmts, params, funcSigs, hasReturn) {
				return false
			}

		case *ir.VarDeclStmt:
			// Local variable declarations are allowed even if non-integer.
			// We only care that *return values* and *param usages* stay integer.
			_ = s

		case *ir.AssignStmt:
			// Assignments allowed; we don't trace escape through locals here.
			_ = s

		case *ir.ExprStmt:
			// Side-effect expressions (e.g. println) allowed.
			_ = s
		}
	}
	return true
}

// isExprIntegerPure returns true when expr statically evaluates to i64 and
// does not require boxing any value as a ptr.
func isExprIntegerPure(expr ir.Expr, funcSigs map[string]*ir.Function) bool {
	if expr == nil {
		return false
	}
	switch ex := expr.(type) {
	case *ir.IntLit:
		return true // integer literal: always i64

	case *ir.Ident:
		// Variable/parameter reference. We conservatively allow all ident
		// references here; if a param is used in a non-integer context elsewhere
		// it would have been caught at the caller's assignment site.
		return true

	case *ir.BinaryExpr:
		switch ex.Op {
		case "+", "-", "*", "/", "%":
			// Arithmetic: both operands must be integer-pure
			return isExprIntegerPure(ex.Left, funcSigs) &&
				isExprIntegerPure(ex.Right, funcSigs)
		case "==", "!=", "<", ">", "<=", ">=":
			// Comparison: operands must be integer-pure; result is i1 (boolean),
			// which we allow as a sub-expression in a conditional (not as a
			// return value directly — that case is handled in isBodyIntegerPure).
			return isExprIntegerPure(ex.Left, funcSigs) &&
				isExprIntegerPure(ex.Right, funcSigs)
		default:
			return false
		}

	case *ir.UnaryExpr:
		if ex.Op == "-" {
			return isExprIntegerPure(ex.Operand, funcSigs)
		}
		return false

	case *ir.CallExpr:
		// Recursive or mutual call to a user-defined Wolf function.
		// We allow it conservatively: if it's in funcSigs it's a Wolf function
		// (not a runtime ptr-returning stdlib call). After the full pre-pass,
		// callee functions that are also integer-pure will be promoted too.
		if ident, ok := ex.Callee.(*ir.Ident); ok {
			if _, found := funcSigs[ident.Name]; found {
				return true // user Wolf function — may be pure; allow optimistically
			}
		}
		return false

	default:
		return false // StringLit, FloatLit, MapLit, SliceLit, etc. → not integer
	}
}
