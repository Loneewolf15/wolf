// Package resolver implements Wolf's variable scope resolution phase.
// It walks the AST, resolves variable scopes, strips the $ prefix for
// Go output, and detects undeclared variables.
package resolver

import (
	"fmt"

	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
)

// Scope tracks variable bindings at one nesting level.
type Scope struct {
	parent    *Scope
	variables map[string]*VarInfo
}

// VarInfo records information about a resolved variable.
type VarInfo struct {
	Name     string         // original name with $
	GoName   string         // name without $ prefix, for Go output
	TypeName string         // type annotation (may be empty)
	Mutable  bool           // true for var declarations
	Depth    int            // scope depth where declared
	Pos      lexer.Position // declaration position
}

// NewScope creates a new scope with an optional parent.
func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:    parent,
		variables: make(map[string]*VarInfo),
	}
}

// Define adds a variable to this scope.
func (s *Scope) Define(name string, info *VarInfo) {
	s.variables[name] = info
}

// Lookup searches for a variable in this scope and all parents.
func (s *Scope) Lookup(name string) (*VarInfo, bool) {
	if info, ok := s.variables[name]; ok {
		return info, true
	}
	if s.parent != nil {
		return s.parent.Lookup(name)
	}
	return nil, false
}

// LookupLocal searches only in the current scope (not parents).
func (s *Scope) LookupLocal(name string) (*VarInfo, bool) {
	info, ok := s.variables[name]
	return info, ok
}

// Resolver performs variable scope resolution on the AST.
type Resolver struct {
	scope      *Scope
	depth      int
	errors     []*lexer.WolfError
	file       string
	resolved   map[string]string        // $name -> goName mapping (accumulated)
	strictMode bool
	interfaces map[string]*parser.InterfaceDecl // registered interface definitions
}

// New creates a new Resolver.
func New(file string) *Resolver {
	return &Resolver{
		scope:      NewScope(nil), // global scope
		file:       file,
		resolved:   make(map[string]string),
		interfaces: make(map[string]*parser.InterfaceDecl),
	}
}

// SetStrictMode enables strict type checking mode.
func (r *Resolver) SetStrictMode(strict bool) {
	r.strictMode = strict
}

// Resolve walks the AST and produces resolution information.
// Returns errors for undeclared variables and other scope issues.
func (r *Resolver) Resolve(program *parser.Program) []*lexer.WolfError {
	// Pre-define built-in identifiers
	r.defineBuiltins()

	for _, stmt := range program.Statements {
		r.resolveStmt(stmt)
	}
	return r.errors
}

// Errors returns accumulated resolver errors.
func (r *Resolver) Errors() []*lexer.WolfError {
	return r.errors
}

// ResolvedNames returns the mapping of $var names to Go names.
func (r *Resolver) ResolvedNames() map[string]string {
	return r.resolved
}

// defineBuiltins pre-declares Wolf built-in identifiers.
func (r *Resolver) defineBuiltins() {
	builtins := []string{"$this", "$_SERVER", "$_GET", "$_POST", "$_ENV"}
	for _, name := range builtins {
		goName := stripDollar(name)
		info := &VarInfo{
			Name:   name,
			GoName: goName,
			Depth:  0,
		}
		r.scope.Define(name, info)
		r.resolved[name] = goName
	}
}

// ========== Statement Resolution ==========

func (r *Resolver) resolveStmt(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.ExpressionStmt:
		r.resolveExpr(s.Expr)

	case *parser.VarDecl:
		r.resolveVarDecl(s)

	case *parser.AssignStmt:
		r.resolveExpr(s.Target)
		r.resolveExpr(s.Value)

	case *parser.ReturnStmt:
		for _, v := range s.Values {
			r.resolveExpr(v)
		}

	case *parser.IfStmt:
		r.resolveExpr(s.Condition)
		r.resolveBlock(s.Body)
		for _, eif := range s.ElseIfs {
			r.resolveExpr(eif.Condition)
			r.resolveBlock(eif.Body)
		}
		if s.ElseBody != nil {
			r.resolveBlock(s.ElseBody)
		}

	case *parser.ForStmt:
		r.pushScope()
		r.resolveStmt(s.Init)
		r.resolveExpr(s.Condition)
		r.resolveStmt(s.Update)
		r.resolveBlock(s.Body)
		r.popScope()

	case *parser.ForeachStmt:
		r.resolveExpr(s.Iterable)
		r.pushScope()
		if s.KeyVar != "" {
			r.declareVar(s.KeyVar, "", s.Pos())
		}
		r.declareVar(s.ValueVar, "", s.Pos())
		r.resolveBlock(s.Body)
		r.popScope()

	case *parser.WhileStmt:
		r.resolveExpr(s.Condition)
		r.resolveBlock(s.Body)

	case *parser.MatchStmt:
		r.resolveExpr(s.Subject)
		for _, arm := range s.Arms {
			r.resolveExpr(arm.Pattern)
			for _, bodyStmt := range arm.Body {
				r.resolveStmt(bodyStmt)
			}
		}

	case *parser.FuncDecl:
		r.resolveFuncDecl(s)

	case *parser.EnumDecl:
		r.resolveEnumDecl(s)

	case *parser.ClassDecl:
		r.resolveClassDecl(s)

	case *parser.InterfaceDecl:
		// Register the interface so that resolveClassDecl can check it
		r.interfaces[s.Name] = s
		// Also declare it as a known identifier
		info := &VarInfo{Name: s.Name, GoName: s.Name, Depth: r.depth, Pos: s.Pos()}
		r.scope.Define(s.Name, info)
		r.resolved[s.Name] = s.Name

	case *parser.TryCatchStmt:
		r.resolveBlock(s.TryBody)
		r.pushScope()
		if s.CatchVar != "" {
			r.declareVar(s.CatchVar, "", s.Pos())
		}
		r.resolveBlock(s.CatchBody)
		r.popScope()

	case *parser.ImportStmt:
		// Imports don't need scope resolution currently

	case *parser.MLBlockStmt:
		// @ml block: in/out vars should be declared
		for _, v := range s.InVars {
			r.resolveVarReference(v, s.Pos())
		}
		for _, v := range s.OutVars {
			r.declareVar(v, "", s.Pos())
		}

	case *parser.ParallelStmt:
		r.resolveBlock(s.Body)

	case *parser.SuperviseBlockStmt:
		r.resolveBlock(s.Body)

	case *parser.TraceBlockStmt:
		r.resolveExpr(s.SpanName)
		r.resolveBlock(s.Body)

	case *parser.DestructureAssign:
		r.resolveExpr(s.Value)
		for _, name := range s.Names {
			r.declareVar(name, "", s.Pos())
		}

	case *parser.BlockStmt:
		r.resolveBlock(s)
	}
}

func (r *Resolver) resolveBlock(block *parser.BlockStmt) {
	if block == nil {
		return
	}
	r.pushScope()
	for _, stmt := range block.Statements {
		r.resolveStmt(stmt)
	}
	r.popScope()
}

func (r *Resolver) resolveVarDecl(v *parser.VarDecl) {
	if v.Value != nil {
		r.resolveExpr(v.Value)
	}
	r.declareVar(v.Name, v.TypeName, v.Pos())
}

func (r *Resolver) resolveFuncDecl(f *parser.FuncDecl) {
	// Declare the function name in the current scope (if it has a name)
	if f.Name != "" {
		goName := f.Name // function names don't have $
		info := &VarInfo{
			Name:   f.Name,
			GoName: goName,
			Depth:  r.depth,
			Pos:    f.Pos(),
		}
		r.scope.Define(f.Name, info)
		r.resolved[f.Name] = goName
	}

	r.pushScope()

	// Declare parameters
	for _, param := range f.Params {
		r.declareVar(param.Name, param.TypeName, param.Pos())
		if param.Default != nil {
			r.resolveExpr(param.Default)
		}
	}

	if f.Body != nil {
		for _, stmt := range f.Body.Statements {
			r.resolveStmt(stmt)
		}
	}

	if f.ArrowExpr != nil {
		r.resolveExpr(f.ArrowExpr)
	}

	r.popScope()
}

func (r *Resolver) resolveEnumDecl(e *parser.EnumDecl) {
	info := &VarInfo{
		Name:   e.Name,
		GoName: e.Name,
		Depth:  r.depth,
		Pos:    e.Pos(),
	}
	r.scope.Define(e.Name, info)
	r.resolved[e.Name] = e.Name
}

func (r *Resolver) resolveClassDecl(c *parser.ClassDecl) {
	// Declare the class name
	info := &VarInfo{
		Name:   c.Name,
		GoName: c.Name,
		Depth:  r.depth,
		Pos:    c.Pos(),
	}
	r.scope.Define(c.Name, info)
	r.resolved[c.Name] = c.Name

	r.pushScope()

	// Declare $this
	r.declareVar("$this", c.Name, c.Pos())

	// Declare properties
	for _, prop := range c.Properties {
		r.declareVar(prop.Name, prop.TypeName, prop.Pos())
		if prop.Default != nil {
			r.resolveExpr(prop.Default)
		}
	}

	// Resolve methods
	for _, method := range c.Methods {
		r.resolveFuncDecl(method)
	}

	r.popScope()

	// Interface compliance: verify every declared interface method is present
	for _, ifaceName := range c.Implements {
		iface, ok := r.interfaces[ifaceName]
		if !ok {
			r.addError(c.Pos(), "class '%s' implements unknown interface '%s'", c.Name, ifaceName)
			continue
		}
		// Build a set of method names in this class
		classMethods := make(map[string]bool)
		for _, m := range c.Methods {
			classMethods[m.Name] = true
		}
		for _, required := range iface.Methods {
			if !classMethods[required.Name] {
				r.addError(c.Pos(),
					"class '%s' does not implement method '%s' required by interface '%s'",
					c.Name, required.Name, ifaceName)
			}
		}
	}
}

// ========== Expression Resolution ==========

func (r *Resolver) resolveExpr(expr parser.Expression) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *parser.DollarIdent:
		r.resolveVarReference(e.Name, e.Pos())

	case *parser.Identifier:
		// Bare identifiers (type names, function refs) — lookup optional
		if _, ok := r.scope.Lookup(e.Name); ok {
			// Already resolved
		}
		// Not an error for bare identifiers — they may be type names or external refs

	case *parser.BinaryExpr:
		r.resolveExpr(e.Left)
		r.resolveExpr(e.Right)

	case *parser.UnaryExpr:
		r.resolveExpr(e.Operand)

	case *parser.PostfixExpr:
		r.resolveExpr(e.Operand)

	case *parser.CallExpr:
		r.resolveExpr(e.Callee)
		for _, arg := range e.Args {
			r.resolveExpr(arg)
		}
		for _, na := range e.NamedArgs {
			r.resolveExpr(na.Value)
		}

	case *parser.PropertyAccess:
		r.resolveExpr(e.Object)

	case *parser.MethodCall:
		r.resolveExpr(e.Object)
		for _, arg := range e.Args {
			r.resolveExpr(arg)
		}

	case *parser.StaticCall:
		for _, arg := range e.Args {
			r.resolveExpr(arg)
		}

	case *parser.IndexExpr:
		r.resolveExpr(e.Object)
		r.resolveExpr(e.Index)

	case *parser.NewExpr:
		for _, arg := range e.Args {
			r.resolveExpr(arg)
		}

	case *parser.ClosureExpr:
		r.pushScope()
		for _, param := range e.Params {
			r.declareVar(param.Name, param.TypeName, param.Pos())
		}
		if e.Body != nil {
			for _, stmt := range e.Body.Statements {
				r.resolveStmt(stmt)
			}
		}
		if e.ArrowExpr != nil {
			r.resolveExpr(e.ArrowExpr)
		}
		r.popScope()

	case *parser.AsyncExpr:
		r.resolveBlock(e.Body)

	case *parser.AwaitExpr:
		r.resolveExpr(e.Expr)

	case *parser.ChannelExpr, *parser.EnumAccess:
		// No inner expressions

	case *parser.SendExpr:
		r.resolveExpr(e.Channel)
		r.resolveExpr(e.Value)

	case *parser.ReceiveExpr:
		r.resolveExpr(e.Channel)

	case *parser.ErrorExpr:
		r.resolveExpr(e.Message)

	case *parser.PrintExpr:
		r.resolveExpr(e.Arg)

	case *parser.ArrayLiteral:
		for _, elem := range e.Elements {
			r.resolveExpr(elem)
		}

	case *parser.MapLiteral:
		for i := range e.Keys {
			r.resolveExpr(e.Keys[i])
			r.resolveExpr(e.Values[i])
		}

	case *parser.InterpolatedString:
		for _, part := range e.Parts {
			r.resolveExpr(part)
		}

	case *parser.StringConcat:
		r.resolveExpr(e.Left)
		r.resolveExpr(e.Right)

	case *parser.StringLiteral, *parser.IntLiteral, *parser.FloatLiteral,
		*parser.BoolLiteral, *parser.NilLiteral, *parser.Wildcard:
		// Literals need no resolution
	}
}

// ========== Helper Methods ==========

// declareVar declares a $variable in the current scope, mapping $name to goName.
func (r *Resolver) declareVar(name, typeName string, pos lexer.Position) {
	goName := stripDollar(name)
	info := &VarInfo{
		Name:     name,
		GoName:   goName,
		TypeName: typeName,
		Mutable:  true,
		Depth:    r.depth,
		Pos:      pos,
	}
	r.scope.Define(name, info)
	r.resolved[name] = goName
}

// resolveVarReference checks that a $variable is declared.
func (r *Resolver) resolveVarReference(name string, pos lexer.Position) {
	if _, ok := r.scope.Lookup(name); !ok {
		// In non-strict mode, auto-declare on first use (PHP-like)
		if r.strictMode {
			r.addError(pos, "undeclared variable '%s'", name)
		} else {
			// Auto-declare (dynamic typing default)
			r.declareVar(name, "", pos)
		}
	}
}

func (r *Resolver) pushScope() {
	r.scope = NewScope(r.scope)
	r.depth++
}

func (r *Resolver) popScope() {
	if r.scope.parent != nil {
		r.scope = r.scope.parent
		r.depth--
	}
}

func (r *Resolver) addError(pos lexer.Position, format string, args ...interface{}) {
	r.errors = append(r.errors, &lexer.WolfError{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
		Phase:   "resolver",
	})
}

// stripDollar removes the $ prefix from a variable name for Go output.
func stripDollar(name string) string {
	if len(name) > 0 && name[0] == '$' {
		return name[1:]
	}
	return name
}
