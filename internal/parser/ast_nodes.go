// Package parser implements the Wolf language parser.
// It consumes tokens from the lexer and produces an Abstract Syntax Tree (AST).
package parser

import (
	"github.com/wolflang/wolf/internal/lexer"
)

// Node is the interface that all AST nodes implement.
type Node interface {
	nodeType() string
	Pos() lexer.Position
}

// Statement nodes do not produce a value.
type Statement interface {
	Node
	stmtNode()
}

// Expression nodes produce a value.
type Expression interface {
	Node
	exprNode()
}

// ---------- Program (root) ----------

// Program is the root AST node — a list of top-level statements.
type Program struct {
	Statements []Statement
	Pos_       lexer.Position
}

func (p *Program) nodeType() string    { return "Program" }
func (p *Program) Pos() lexer.Position { return p.Pos_ }

// ---------- Statements ----------

// ExpressionStmt wraps an expression used as a statement.
type ExpressionStmt struct {
	Expr Expression
	Pos_ lexer.Position
}

func (s *ExpressionStmt) nodeType() string    { return "ExpressionStmt" }
func (s *ExpressionStmt) Pos() lexer.Position { return s.Pos_ }
func (s *ExpressionStmt) stmtNode()           {}

// VarDecl represents a typed variable declaration: var $x: int = 42
type VarDecl struct {
	Name     string     // variable name including $
	TypeName string     // type annotation (may be empty)
	Value    Expression // initializer (may be nil)
	Pos_     lexer.Position
}

func (s *VarDecl) nodeType() string    { return "VarDecl" }
func (s *VarDecl) Pos() lexer.Position { return s.Pos_ }
func (s *VarDecl) stmtNode()           {}

// AssignStmt represents $var = expr or $obj->prop = expr
type AssignStmt struct {
	Target Expression // left side (DollarIdent or PropertyAccess)
	Op     string     // "=", "+=", "-=", "*=", "/="
	Value  Expression // right side
	Pos_   lexer.Position
}

func (s *AssignStmt) nodeType() string    { return "AssignStmt" }
func (s *AssignStmt) Pos() lexer.Position { return s.Pos_ }
func (s *AssignStmt) stmtNode()           {}

// ReturnStmt represents: return expr or return expr1, expr2
type ReturnStmt struct {
	Values []Expression // one or more return values (nil slice for bare return)
	Pos_   lexer.Position
}

func (s *ReturnStmt) nodeType() string    { return "ReturnStmt" }
func (s *ReturnStmt) Pos() lexer.Position { return s.Pos_ }
func (s *ReturnStmt) stmtNode()           {}

// IfStmt represents: if $cond { } else if { } else { }
type IfStmt struct {
	Condition Expression
	Body      *BlockStmt
	ElseIfs   []*ElseIfClause // zero or more else-if branches
	ElseBody  *BlockStmt      // may be nil
	Pos_      lexer.Position
}

func (s *IfStmt) nodeType() string    { return "IfStmt" }
func (s *IfStmt) Pos() lexer.Position { return s.Pos_ }
func (s *IfStmt) stmtNode()           {}

// ElseIfClause represents a single else-if branch.
type ElseIfClause struct {
	Condition Expression
	Body      *BlockStmt
	Pos_      lexer.Position
}

func (c *ElseIfClause) nodeType() string    { return "ElseIfClause" }
func (c *ElseIfClause) Pos() lexer.Position { return c.Pos_ }

// ForStmt represents C-style for: for $i = 0; $i < 10; $i++ { }
type ForStmt struct {
	Init      Statement  // initializer (assignment)
	Condition Expression // loop condition
	Update    Statement  // update (e.g., $i++)
	Body      *BlockStmt
	Pos_      lexer.Position
}

func (s *ForStmt) nodeType() string    { return "ForStmt" }
func (s *ForStmt) Pos() lexer.Position { return s.Pos_ }
func (s *ForStmt) stmtNode()           {}

// ForeachStmt represents: foreach $items as $item { }
// or: foreach $users as $key => $value { }
type ForeachStmt struct {
	Iterable Expression
	ValueVar string // the $item variable
	KeyVar   string // the $key variable (empty if not used)
	Body     *BlockStmt
	Pos_     lexer.Position
}

func (s *ForeachStmt) nodeType() string    { return "ForeachStmt" }
func (s *ForeachStmt) Pos() lexer.Position { return s.Pos_ }
func (s *ForeachStmt) stmtNode()           {}

// WhileStmt represents: while $cond { }
type WhileStmt struct {
	Condition Expression
	Body      *BlockStmt
	Pos_      lexer.Position
}

func (s *WhileStmt) nodeType() string    { return "WhileStmt" }
func (s *WhileStmt) Pos() lexer.Position { return s.Pos_ }
func (s *WhileStmt) stmtNode()           {}

// MatchStmt represents: match $expr { "val" => ..., _ => ... }
type MatchStmt struct {
	Subject Expression
	Arms    []*MatchArm
	Pos_    lexer.Position
}

func (s *MatchStmt) nodeType() string    { return "MatchStmt" }
func (s *MatchStmt) Pos() lexer.Position { return s.Pos_ }
func (s *MatchStmt) stmtNode()           {}

// MatchArm represents one arm: pattern => body
type MatchArm struct {
	Pattern Expression // literal, identifier, or _ (Wildcard)
	Body    []Statement
	Pos_    lexer.Position
}

func (a *MatchArm) nodeType() string    { return "MatchArm" }
func (a *MatchArm) Pos() lexer.Position { return a.Pos_ }

// BlockStmt is a { } delimited list of statements.
type BlockStmt struct {
	Statements []Statement
	Pos_       lexer.Position
}

func (s *BlockStmt) nodeType() string    { return "BlockStmt" }
func (s *BlockStmt) Pos() lexer.Position { return s.Pos_ }
func (s *BlockStmt) stmtNode()           {}

// FuncDecl represents a function declaration: func name($args) { }
type FuncDecl struct {
	Name       string
	Params     []*Param
	ReturnType *ReturnTypeSpec // may be nil
	TypeParams []string        // generic type params, e.g., ["T", "U"]
	Body       *BlockStmt      // nil for arrow functions
	ArrowExpr  Expression      // non-nil for => shorthand
	Pos_       lexer.Position
}

func (s *FuncDecl) nodeType() string    { return "FuncDecl" }
func (s *FuncDecl) Pos() lexer.Position { return s.Pos_ }
func (s *FuncDecl) stmtNode()           {}

// Param represents a function parameter.
type Param struct {
	Name     string     // including $ prefix
	TypeName string     // type annotation (empty if dynamic)
	Default  Expression // default value (may be nil)
	Pos_     lexer.Position
}

func (p *Param) nodeType() string    { return "Param" }
func (p *Param) Pos() lexer.Position { return p.Pos_ }

// ReturnTypeSpec represents -> (type) or -> type
type ReturnTypeSpec struct {
	Types []string // one or more return types
	Pos_  lexer.Position
}

func (r *ReturnTypeSpec) nodeType() string    { return "ReturnTypeSpec" }
func (r *ReturnTypeSpec) Pos() lexer.Position { return r.Pos_ }

// ClassDecl represents a class definition.
type ClassDecl struct {
	Name       string
	Extends    string
	TypeParams []string // generic type params, e.g., ["T", "U"]
	Implements []string // interface names this class implements
	Properties []*PropertyDecl
	Methods    []*FuncDecl
	Pos_       lexer.Position
}

func (s *ClassDecl) nodeType() string    { return "ClassDecl" }
func (s *ClassDecl) Pos() lexer.Position { return s.Pos_ }
func (s *ClassDecl) stmtNode()           {}

// InterfaceDecl represents: interface Foo { func bar() -> string }
type InterfaceDecl struct {
	Name    string
	Methods []*InterfaceMethod // signatures only — no body
	Pos_    lexer.Position
}

func (s *InterfaceDecl) nodeType() string    { return "InterfaceDecl" }
func (s *InterfaceDecl) Pos() lexer.Position { return s.Pos_ }
func (s *InterfaceDecl) stmtNode()           {}

// InterfaceMethod represents a single method signature inside an interface.
type InterfaceMethod struct {
	Name       string
	Params     []*Param
	ReturnType *ReturnTypeSpec
	Pos_       lexer.Position
}

func (m *InterfaceMethod) nodeType() string    { return "InterfaceMethod" }
func (m *InterfaceMethod) Pos() lexer.Position { return m.Pos_ }

// EnumDecl represents an enum definition: enum Name { Var1, Var2 }
type EnumDecl struct {
	Name     string
	Variants []string
	Pos_     lexer.Position
}

func (s *EnumDecl) nodeType() string    { return "EnumDecl" }
func (s *EnumDecl) Pos() lexer.Position { return s.Pos_ }
func (s *EnumDecl) stmtNode()           {}

// PropertyDecl represents a class property: $name: type = default
type PropertyDecl struct {
	Name       string     // including $
	TypeName   string     // may be empty
	Default    Expression // may be nil
	Visibility string     // "private", "public", "static", or ""
	Pos_       lexer.Position
}

func (p *PropertyDecl) nodeType() string    { return "PropertyDecl" }
func (p *PropertyDecl) Pos() lexer.Position { return p.Pos_ }

// TryCatchStmt represents try { } catch ($e) { }
type TryCatchStmt struct {
	TryBody   *BlockStmt
	CatchVar  string // the $e variable
	CatchBody *BlockStmt
	Pos_      lexer.Position
}

func (s *TryCatchStmt) nodeType() string    { return "TryCatchStmt" }
func (s *TryCatchStmt) Pos() lexer.Position { return s.Pos_ }
func (s *TryCatchStmt) stmtNode()           {}

// ImportStmt represents: import "module"
type ImportStmt struct {
	Path string
	Pos_ lexer.Position
}

func (s *ImportStmt) nodeType() string    { return "ImportStmt" }
func (s *ImportStmt) Pos() lexer.Position { return s.Pos_ }
func (s *ImportStmt) stmtNode()           {}

// MLBlockStmt represents the @ml { python code } block.
type MLBlockStmt struct {
	InVars    []string   // explicit in: [$var1, $var2]
	OutVars   []string   // explicit out: [$var1, $var2]
	ModelName Expression // for @ml model("name") — may be nil
	IsAsync   bool       // @ml async
	Body      string     // raw Python source code
	Pos_      lexer.Position
}

func (s *MLBlockStmt) nodeType() string    { return "MLBlockStmt" }
func (s *MLBlockStmt) Pos() lexer.Position { return s.Pos_ }
func (s *MLBlockStmt) stmtNode()           {}

// SuperviseBlockStmt represents: @supervise(strategy: "...", restart: "...", max: X) { ... }
type SuperviseBlockStmt struct {
	Strategy string // "one_for_one", "one_for_all", etc.
	Restart  string // "exponential", "always", "never"
	Max      int    // max retries (e.g. 5)
	Body     *BlockStmt
	Pos_     lexer.Position
}

func (s *SuperviseBlockStmt) nodeType() string    { return "SuperviseBlockStmt" }
func (s *SuperviseBlockStmt) Pos() lexer.Position { return s.Pos_ }
func (s *SuperviseBlockStmt) stmtNode()           {}

// SpawnStmt represents: spawn fn(args...)
type SpawnStmt struct {
	Call *CallExpr
	Pos_ lexer.Position
}

func (s *SpawnStmt) nodeType() string    { return "SpawnStmt" }
func (s *SpawnStmt) Pos() lexer.Position { return s.Pos_ }
func (s *SpawnStmt) stmtNode()           {}

// TraceBlockStmt represents: @trace("span.name") { ... }
type TraceBlockStmt struct {
	SpanName Expression // usually a StringLiteral
	Body     *BlockStmt
	Pos_     lexer.Position
}

func (s *TraceBlockStmt) nodeType() string    { return "TraceBlockStmt" }
func (s *TraceBlockStmt) Pos() lexer.Position { return s.Pos_ }
func (s *TraceBlockStmt) stmtNode()           {}

// ParallelStmt represents: parallel { ... }
type ParallelStmt struct {
	Body *BlockStmt
	Pos_ lexer.Position
}

func (s *ParallelStmt) nodeType() string    { return "ParallelStmt" }
func (s *ParallelStmt) Pos() lexer.Position { return s.Pos_ }
func (s *ParallelStmt) stmtNode()           {}

// DestructureAssign represents: [$a, $b] = expr
type DestructureAssign struct {
	Names []string // [$data, $err]
	Value Expression
	Pos_  lexer.Position
}

func (s *DestructureAssign) nodeType() string    { return "DestructureAssign" }
func (s *DestructureAssign) Pos() lexer.Position { return s.Pos_ }
func (s *DestructureAssign) stmtNode()           {}

// ---------- Expressions ----------

// DollarIdent represents a $variable reference.
type DollarIdent struct {
	Name string // including $ prefix
	Pos_ lexer.Position
}

func (e *DollarIdent) nodeType() string    { return "DollarIdent" }
func (e *DollarIdent) Pos() lexer.Position { return e.Pos_ }
func (e *DollarIdent) exprNode()           {}

// Identifier represents a bare identifier (type names, function names, etc).
type Identifier struct {
	Name string
	Pos_ lexer.Position
}

func (e *Identifier) nodeType() string    { return "Identifier" }
func (e *Identifier) Pos() lexer.Position { return e.Pos_ }
func (e *Identifier) exprNode()           {}

// IntLiteral represents an integer literal.
type IntLiteral struct {
	Value string
	Pos_  lexer.Position
}

func (e *IntLiteral) nodeType() string    { return "IntLiteral" }
func (e *IntLiteral) Pos() lexer.Position { return e.Pos_ }
func (e *IntLiteral) exprNode()           {}

// FloatLiteral represents a float literal.
type FloatLiteral struct {
	Value string
	Pos_  lexer.Position
}

func (e *FloatLiteral) nodeType() string    { return "FloatLiteral" }
func (e *FloatLiteral) Pos() lexer.Position { return e.Pos_ }
func (e *FloatLiteral) exprNode()           {}

// StringLiteral represents a simple (non-interpolated) string.
type StringLiteral struct {
	Value string
	Pos_  lexer.Position
}

func (e *StringLiteral) nodeType() string    { return "StringLiteral" }
func (e *StringLiteral) Pos() lexer.Position { return e.Pos_ }
func (e *StringLiteral) exprNode()           {}

// InterpolatedString represents a string with interpolation segments.
type InterpolatedString struct {
	Parts []Expression // StringLiteral parts and expressions interleaved
	Pos_  lexer.Position
}

func (e *InterpolatedString) nodeType() string    { return "InterpolatedString" }
func (e *InterpolatedString) Pos() lexer.Position { return e.Pos_ }
func (e *InterpolatedString) exprNode()           {}

// BoolLiteral represents true or false.
type BoolLiteral struct {
	Value bool
	Pos_  lexer.Position
}

func (e *BoolLiteral) nodeType() string    { return "BoolLiteral" }
func (e *BoolLiteral) Pos() lexer.Position { return e.Pos_ }
func (e *BoolLiteral) exprNode()           {}

// NilLiteral represents nil.
type NilLiteral struct {
	Pos_ lexer.Position
}

func (e *NilLiteral) nodeType() string    { return "NilLiteral" }
func (e *NilLiteral) Pos() lexer.Position { return e.Pos_ }
func (e *NilLiteral) exprNode()           {}

// Wildcard represents the _ pattern in match arms.
type Wildcard struct {
	Pos_ lexer.Position
}

func (e *Wildcard) nodeType() string    { return "Wildcard" }
func (e *Wildcard) Pos() lexer.Position { return e.Pos_ }
func (e *Wildcard) exprNode()           {}

// ArrayLiteral represents [1, 2, 3]
type ArrayLiteral struct {
	Elements []Expression
	Pos_     lexer.Position
}

func (e *ArrayLiteral) nodeType() string    { return "ArrayLiteral" }
func (e *ArrayLiteral) Pos() lexer.Position { return e.Pos_ }
func (e *ArrayLiteral) exprNode()           {}

// MapLiteral represents {"key": value, ...}
type MapLiteral struct {
	Keys   []Expression
	Values []Expression
	Pos_   lexer.Position
}

func (e *MapLiteral) nodeType() string    { return "MapLiteral" }
func (e *MapLiteral) Pos() lexer.Position { return e.Pos_ }
func (e *MapLiteral) exprNode()           {}

// BinaryExpr represents a binary operation: left op right
type BinaryExpr struct {
	Left  Expression
	Op    string
	Right Expression
	Pos_  lexer.Position
}

func (e *BinaryExpr) nodeType() string    { return "BinaryExpr" }
func (e *BinaryExpr) Pos() lexer.Position { return e.Pos_ }
func (e *BinaryExpr) exprNode()           {}

// UnaryExpr represents a unary operation: !expr, -expr
type UnaryExpr struct {
	Op      string
	Operand Expression
	Pos_    lexer.Position
}

func (e *UnaryExpr) nodeType() string    { return "UnaryExpr" }
func (e *UnaryExpr) Pos() lexer.Position { return e.Pos_ }
func (e *UnaryExpr) exprNode()           {}

// PostfixExpr represents $i++ or $i--
type PostfixExpr struct {
	Operand Expression
	Op      string // "++" or "--"
	Pos_    lexer.Position
}

func (e *PostfixExpr) nodeType() string    { return "PostfixExpr" }
func (e *PostfixExpr) Pos() lexer.Position { return e.Pos_ }
func (e *PostfixExpr) exprNode()           {}

// CallExpr represents a function call: name(args)
type CallExpr struct {
	Callee    Expression
	Args      []Expression
	TypeArgs  []string    // generic type arguments, e.g., ["int"]
	NamedArgs []*NamedArg // for named parameters
	Pos_      lexer.Position
}

func (e *CallExpr) nodeType() string    { return "CallExpr" }
func (e *CallExpr) Pos() lexer.Position { return e.Pos_ }
func (e *CallExpr) exprNode()           {}

// NamedArg represents name: value in a function call.
type NamedArg struct {
	Name  string
	Value Expression
	Pos_  lexer.Position
}

func (a *NamedArg) nodeType() string    { return "NamedArg" }
func (a *NamedArg) Pos() lexer.Position { return a.Pos_ }

// PropertyAccess represents $obj->property
type PropertyAccess struct {
	Object   Expression
	Property string
	Pos_     lexer.Position
}

func (e *PropertyAccess) nodeType() string    { return "PropertyAccess" }
func (e *PropertyAccess) Pos() lexer.Position { return e.Pos_ }
func (e *PropertyAccess) exprNode()           {}

// MethodCall represents $obj->method(args) — sugar over PropertyAccess + CallExpr
type MethodCall struct {
	Object Expression
	Method string
	Args   []Expression
	Pos_   lexer.Position
}

func (e *MethodCall) nodeType() string    { return "MethodCall" }
func (e *MethodCall) Pos() lexer.Position { return e.Pos_ }
func (e *MethodCall) exprNode()           {}

// StaticCall represents Class::method(args)
type StaticCall struct {
	Class  string
	Method string
	Args   []Expression
	Pos_   lexer.Position
}

func (e *StaticCall) nodeType() string    { return "StaticCall" }
func (e *StaticCall) Pos() lexer.Position { return e.Pos_ }
func (e *StaticCall) exprNode()           {}

// EnumAccess represents EnumName::Variant
type EnumAccess struct {
	Enum    string
	Variant string
	Pos_    lexer.Position
}

func (e *EnumAccess) nodeType() string    { return "EnumAccess" }
func (e *EnumAccess) Pos() lexer.Position { return e.Pos_ }
func (e *EnumAccess) exprNode()           {}

// IndexExpr represents $arr[index] or $map["key"]
type IndexExpr struct {
	Object Expression
	Index  Expression
	Pos_   lexer.Position
}

func (e *IndexExpr) nodeType() string    { return "IndexExpr" }
func (e *IndexExpr) Pos() lexer.Position { return e.Pos_ }
func (e *IndexExpr) exprNode()           {}

// NewExpr represents: new ClassName(args) or new ClassName<T>(args)
type NewExpr struct {
	ClassName string
	TypeArgs  []string // generic type arguments, e.g., ["int"]
	Args      []Expression
	Pos_      lexer.Position
}

func (e *NewExpr) nodeType() string    { return "NewExpr" }
func (e *NewExpr) Pos() lexer.Position { return e.Pos_ }
func (e *NewExpr) exprNode()           {}

// ClosureExpr represents an anonymous function: func($a, $b) { return $a + $b }
type ClosureExpr struct {
	Params    []*Param
	Body      *BlockStmt
	ArrowExpr Expression // for arrow shorthand
	Pos_      lexer.Position
}

func (e *ClosureExpr) nodeType() string    { return "ClosureExpr" }
func (e *ClosureExpr) Pos() lexer.Position { return e.Pos_ }
func (e *ClosureExpr) exprNode()           {}

// AsyncExpr represents: async { ... }
type AsyncExpr struct {
	Body *BlockStmt
	Pos_ lexer.Position
}

func (e *AsyncExpr) nodeType() string    { return "AsyncExpr" }
func (e *AsyncExpr) Pos() lexer.Position { return e.Pos_ }
func (e *AsyncExpr) exprNode()           {}

// AwaitExpr represents: await $task
type AwaitExpr struct {
	Expr Expression
	Pos_ lexer.Position
}

func (e *AwaitExpr) nodeType() string    { return "AwaitExpr" }
func (e *AwaitExpr) Pos() lexer.Position { return e.Pos_ }
func (e *AwaitExpr) exprNode()           {}

// ChannelExpr represents: channel(type)
type ChannelExpr struct {
	ElemType string
	Pos_     lexer.Position
}

func (e *ChannelExpr) nodeType() string    { return "ChannelExpr" }
func (e *ChannelExpr) Pos() lexer.Position { return e.Pos_ }
func (e *ChannelExpr) exprNode()           {}

// SendExpr represents: send($ch, value)
type SendExpr struct {
	Channel Expression
	Value   Expression
	Pos_    lexer.Position
}

func (e *SendExpr) nodeType() string    { return "SendExpr" }
func (e *SendExpr) Pos() lexer.Position { return e.Pos_ }
func (e *SendExpr) exprNode()           {}

// ReceiveExpr represents: receive($ch)
type ReceiveExpr struct {
	Channel Expression
	Pos_    lexer.Position
}

func (e *ReceiveExpr) nodeType() string    { return "ReceiveExpr" }
func (e *ReceiveExpr) Pos() lexer.Position { return e.Pos_ }
func (e *ReceiveExpr) exprNode()           {}

// ErrorExpr represents: error("message")
type ErrorExpr struct {
	Message Expression
	Pos_    lexer.Position
}

func (e *ErrorExpr) nodeType() string    { return "ErrorExpr" }
func (e *ErrorExpr) Pos() lexer.Position { return e.Pos_ }
func (e *ErrorExpr) exprNode()           {}

// StringConcat represents the . (dot-dot) string concatenation operator.
type StringConcat struct {
	Left  Expression
	Right Expression
	Pos_  lexer.Position
}

func (e *StringConcat) nodeType() string    { return "StringConcat" }
func (e *StringConcat) Pos() lexer.Position { return e.Pos_ }
func (e *StringConcat) exprNode()           {}

// PrintExpr represents print(expr) — treated as a built-in expression.
type PrintExpr struct {
	Arg  Expression
	Pos_ lexer.Position
}

func (e *PrintExpr) nodeType() string    { return "PrintExpr" }
func (e *PrintExpr) Pos() lexer.Position { return e.Pos_ }
func (e *PrintExpr) exprNode()           {}
