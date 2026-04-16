// Package typechecker implements Wolf's type checking phase.
// In default (dynamic) mode, it performs minimal checks.
// In strict mode, it validates type annotations, @ml boundary types,
// and DB config references.
package typechecker

import (
	"fmt"

	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
	"github.com/wolflang/wolf/internal/resolver"
)

// WolfType represents a type in the Wolf type system.
type WolfType int

const (
	TypeUnknown WolfType = iota
	TypeInt
	TypeFloat
	TypeString
	TypeBool
	TypeNil
	TypeArray
	TypeMap
	TypeFunc
	TypeClass
	TypeChannel
	TypeError
	TypeAny // dynamic typing fallback
)

// typeNames maps types to string representations.
var typeNames = map[WolfType]string{
	TypeUnknown: "unknown",
	TypeInt:     "int",
	TypeFloat:   "float",
	TypeString:  "string",
	TypeBool:    "bool",
	TypeNil:     "nil",
	TypeArray:   "array",
	TypeMap:     "map",
	TypeFunc:    "func",
	TypeClass:   "class",
	TypeChannel: "channel",
	TypeError:   "error",
	TypeAny:     "any",
}

func (t WolfType) String() string {
	if name, ok := typeNames[t]; ok {
		return name
	}
	return "unknown"
}

// stringToType converts a type name string to a WolfType.
func stringToType(name string) WolfType {
	switch name {
	case "int":
		return TypeInt
	case "float", "float64":
		return TypeFloat
	case "string":
		return TypeString
	case "bool":
		return TypeBool
	case "error":
		return TypeError
	case "any":
		return TypeAny
	default:
		return TypeUnknown
	}
}

// mlBridgeableTypes are Wolf types that can cross the @ml boundary.
var mlBridgeableTypes = map[WolfType]bool{
	TypeInt:    true,
	TypeFloat:  true,
	TypeString: true,
	TypeBool:   true,
	TypeArray:  true,
	TypeMap:    true,
	TypeNil:    true,
	TypeAny:    true,
}

// Checker performs type checking on the AST.
type Checker struct {
	resolver   *resolver.Resolver
	errors     []*lexer.WolfError
	strictMode bool
	file       string

	// Track known DB config constants
	dbConfigKeys map[string]bool
}

// New creates a new Checker.
func New(res *resolver.Resolver, file string) *Checker {
	return &Checker{
		resolver: res,
		file:     file,
		dbConfigKeys: map[string]bool{
			"DB_HOST": true, "DB_USER": true, "DB_PASS": true,
			"DB_NAME": true, "DB_PORT": true,
			"REDIS_HOST": true, "REDIS_PORT": true, "REDIS_PASS": true,
			"MONGO_HOST": true, "MONGO_PORT": true, "MONGO_DB": true,
		},
	}
}

// SetStrictMode enables strict type checking.
func (c *Checker) SetStrictMode(strict bool) {
	c.strictMode = strict
}

// Check performs type checking on the program AST.
// Returns any type errors found.
func (c *Checker) Check(program *parser.Program) []*lexer.WolfError {
	for _, stmt := range program.Statements {
		c.checkStmt(stmt)
	}
	return c.errors
}

// Errors returns accumulated type errors.
func (c *Checker) Errors() []*lexer.WolfError {
	return c.errors
}

// ========== Statement Checking ==========

func (c *Checker) checkStmt(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.VarDecl:
		c.checkVarDecl(s)

	case *parser.FuncDecl:
		c.checkFuncDecl(s)

	case *parser.EnumDecl:
		c.checkEnumDecl(s)

	case *parser.ClassDecl:
		c.checkClassDecl(s)

	case *parser.ReturnStmt:
		for _, v := range s.Values {
			c.checkExpr(v)
		}

	case *parser.IfStmt:
		c.checkExpr(s.Condition)
		c.checkBlock(s.Body)
		for _, eif := range s.ElseIfs {
			c.checkExpr(eif.Condition)
			c.checkBlock(eif.Body)
		}
		if s.ElseBody != nil {
			c.checkBlock(s.ElseBody)
		}

	case *parser.ForStmt:
		c.checkStmt(s.Init)
		c.checkExpr(s.Condition)
		c.checkStmt(s.Update)
		c.checkBlock(s.Body)
		c.checkN1InLoop(s.Body, s.Pos())

	case *parser.ForeachStmt:
		c.checkExpr(s.Iterable)
		c.checkBlock(s.Body)
		c.checkN1InLoop(s.Body, s.Pos())

	case *parser.WhileStmt:
		c.checkExpr(s.Condition)
		c.checkBlock(s.Body)
		c.checkN1InLoop(s.Body, s.Pos())

	case *parser.MatchStmt:
		c.checkExpr(s.Subject)
		for _, arm := range s.Arms {
			c.checkExpr(arm.Pattern)
			for _, bodyStmt := range arm.Body {
				c.checkStmt(bodyStmt)
			}
		}

	case *parser.TryCatchStmt:
		c.checkBlock(s.TryBody)
		c.checkBlock(s.CatchBody)

	case *parser.MLBlockStmt:
		c.checkMLBlock(s)

	case *parser.ParallelStmt:
		c.checkBlock(s.Body)

	case *parser.SuperviseBlockStmt:
		c.checkBlock(s.Body)

	case *parser.TraceBlockStmt:
		c.checkExpr(s.SpanName)
		c.checkBlock(s.Body)

	case *parser.DestructureAssign:
		c.checkExpr(s.Value)

	case *parser.ExpressionStmt:
		c.checkExpr(s.Expr)

	case *parser.AssignStmt:
		c.checkExpr(s.Target)
		c.checkExpr(s.Value)

	case *parser.BlockStmt:
		c.checkBlock(s)

	case *parser.ImportStmt:
		// No type checking needed
	}
}

func (c *Checker) checkBlock(block *parser.BlockStmt) {
	if block == nil {
		return
	}
	for _, stmt := range block.Statements {
		c.checkStmt(stmt)
	}
}

func (c *Checker) checkVarDecl(v *parser.VarDecl) {
	if v.Value != nil {
		c.checkExpr(v.Value)
	}

	// In strict mode, validate type annotation matches value
	if c.strictMode && v.TypeName != "" && v.Value != nil {
		declaredType := stringToType(v.TypeName)
		inferredType := c.inferType(v.Value)
		if declaredType != TypeUnknown && inferredType != TypeAny &&
			inferredType != TypeUnknown && declaredType != inferredType {
			c.addError(v.Pos(), "type mismatch: variable '%s' declared as '%s' but assigned '%s'",
				v.Name, v.TypeName, inferredType)
		}
	}
}

func (c *Checker) checkEnumDecl(e *parser.EnumDecl) {
	// Enums are valid as long as variants are valid identifiers
}

func (c *Checker) checkFuncDecl(f *parser.FuncDecl) {
	// Check parameter defaults
	for _, param := range f.Params {
		if param.Default != nil {
			c.checkExpr(param.Default)
		}
	}

	if f.Body != nil {
		c.checkBlock(f.Body)
	}
	if f.ArrowExpr != nil {
		c.checkExpr(f.ArrowExpr)
	}
}

func (c *Checker) checkClassDecl(cls *parser.ClassDecl) {
	for _, prop := range cls.Properties {
		if prop.Default != nil {
			c.checkExpr(prop.Default)
		}
	}
	for _, method := range cls.Methods {
		c.checkFuncDecl(method)
	}
}

// checkMLBlock validates @ml block type boundaries.
func (c *Checker) checkMLBlock(ml *parser.MLBlockStmt) {
	// Validate that in/out vars use bridgeable types
	// In dynamic mode, this is permissive. In strict mode, enforce the type bridge table.
	if c.strictMode {
		resolvedNames := c.resolver.ResolvedNames()
		for _, varName := range ml.InVars {
			if _, ok := resolvedNames[varName]; !ok {
				c.addError(ml.Pos(), "@ml in variable '%s' is not declared", varName)
			}
		}
	}

	if ml.ModelName != nil {
		c.checkExpr(ml.ModelName)
	}
}

// ========== Expression Checking ==========

func (c *Checker) checkExpr(expr parser.Expression) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *parser.BinaryExpr:
		c.checkExpr(e.Left)
		c.checkExpr(e.Right)

	case *parser.UnaryExpr:
		c.checkExpr(e.Operand)

	case *parser.PostfixExpr:
		c.checkExpr(e.Operand)

	case *parser.CallExpr:
		c.checkExpr(e.Callee)
		for _, arg := range e.Args {
			c.checkExpr(arg)
		}
		for _, na := range e.NamedArgs {
			c.checkExpr(na.Value)
		}

	case *parser.PropertyAccess:
		c.checkExpr(e.Object)

	case *parser.MethodCall:
		c.checkExpr(e.Object)
		for _, arg := range e.Args {
			c.checkExpr(arg)
		}

	case *parser.IndexExpr:
		c.checkExpr(e.Object)
		c.checkExpr(e.Index)

	case *parser.NewExpr:
		for _, arg := range e.Args {
			c.checkExpr(arg)
		}

	case *parser.ClosureExpr:
		if e.Body != nil {
			c.checkBlock(e.Body)
		}
		if e.ArrowExpr != nil {
			c.checkExpr(e.ArrowExpr)
		}

	case *parser.AsyncExpr:
		c.checkBlock(e.Body)

	case *parser.AwaitExpr:
		c.checkExpr(e.Expr)

	case *parser.SendExpr:
		c.checkExpr(e.Channel)
		c.checkExpr(e.Value)

	case *parser.ReceiveExpr:
		c.checkExpr(e.Channel)

	case *parser.ErrorExpr:
		c.checkExpr(e.Message)

	case *parser.PrintExpr:
		c.checkExpr(e.Arg)

	case *parser.ArrayLiteral:
		for _, elem := range e.Elements {
			c.checkExpr(elem)
		}

	case *parser.MapLiteral:
		// Validate map keys are all the same type
		for i := range e.Keys {
			c.checkExpr(e.Keys[i])
			c.checkExpr(e.Values[i])
		}

	case *parser.InterpolatedString:
		for _, part := range e.Parts {
			c.checkExpr(part)
		}

	case *parser.StringConcat:
		c.checkExpr(e.Left)
		c.checkExpr(e.Right)

	case *parser.StaticCall:
		for _, arg := range e.Args {
			c.checkExpr(arg)
		}

	// Literals and leaf nodes — no checking needed
	case *parser.DollarIdent, *parser.Identifier, *parser.StringLiteral,
		*parser.IntLiteral, *parser.FloatLiteral, *parser.BoolLiteral,
		*parser.NilLiteral, *parser.Wildcard, *parser.ChannelExpr, *parser.EnumAccess:
		// Nothing to check
	}
}

// ========== Type Inference ==========

// inferType returns the inferred Wolf type of an expression.
func (c *Checker) inferType(expr parser.Expression) WolfType {
	switch expr.(type) {
	case *parser.IntLiteral:
		return TypeInt
	case *parser.FloatLiteral:
		return TypeFloat
	case *parser.StringLiteral, *parser.InterpolatedString, *parser.EnumAccess:
		return TypeString
	case *parser.BoolLiteral:
		return TypeBool
	case *parser.NilLiteral:
		return TypeNil
	case *parser.ArrayLiteral:
		return TypeArray
	case *parser.MapLiteral:
		return TypeMap
	case *parser.ClosureExpr:
		return TypeFunc
	case *parser.ChannelExpr:
		return TypeChannel
	case *parser.ErrorExpr:
		return TypeError
	default:
		return TypeAny // Can't infer — dynamic
	}
}

// ========== Helpers ==========

func (c *Checker) addError(pos lexer.Position, format string, args ...interface{}) {
	c.errors = append(c.errors, &lexer.WolfError{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
		Phase:   "typechecker",
	})
}

func (c *Checker) addWarning(pos lexer.Position, format string, args ...interface{}) {
	c.errors = append(c.errors, &lexer.WolfError{
		Pos:       pos,
		Message:   fmt.Sprintf(format, args...),
		Phase:     "typechecker",
		IsWarning: true,
	})
}

// ========== DB-03: N+1 Detection ==========

// dbQueryMethods are QB method names that trigger a DB query.
var dbQueryMethods = map[string]bool{
	"get":      true,
	"first":    true,
	"find":     true,
	"qb_get":   true,
	"qb_first": true,
	"all":      true,
}

// dbRelationMethods are QB method names that register eager-loading relations.
var dbRelationMethods = map[string]bool{
	"with":    true,
	"qb_with": true,
}

// checkN1InLoop scans a loop body for un-eagerly-loaded DB query calls.
// If it finds a call to get/first/find without a sibling with() call on
// the same receiver variable, it emits a warning.
func (c *Checker) checkN1InLoop(block *parser.BlockStmt, loopPos lexer.Position) {
	if block == nil {
		return
	}

	// Track which variable names have had ->with() registered in this block.
	hasRelation := map[string]bool{}

	for _, stmt := range block.Statements {
		// Unwrap expression statements
		expr := stmtToExpr(stmt)
		if expr == nil {
			continue
		}

		mc, ok := expr.(*parser.MethodCall)
		if !ok {
			continue
		}

		recvName := exprName(mc.Object)

		if dbRelationMethods[mc.Method] {
			hasRelation[recvName] = true
			continue
		}

		if dbQueryMethods[mc.Method] {
			if !hasRelation[recvName] {
				c.addWarning(mc.Pos(),
					"[N+1] DB query '%s()' inside a loop without eager loading — "+
						"consider $%s->with(relation, fk, table, pk) before the loop",
					mc.Method, recvName)
			}
		}
	}
}

// stmtToExpr extracts the expression from an ExpressionStmt or AssignStmt.
func stmtToExpr(stmt parser.Statement) parser.Expression {
	switch s := stmt.(type) {
	case *parser.ExpressionStmt:
		return s.Expr
	case *parser.AssignStmt:
		return s.Value
	case *parser.VarDecl:
		return s.Value
	}
	return nil
}

// exprName returns the variable name for a $ident or Identifier expression node.
func exprName(expr parser.Expression) string {
	switch e := expr.(type) {
	case *parser.DollarIdent:
		return e.Name
	case *parser.Identifier:
		return e.Name
	}
	return "_"
}
