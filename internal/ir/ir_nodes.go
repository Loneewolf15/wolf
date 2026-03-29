// Package ir defines the Wolf Intermediate Representation (WIR).
// WIR is a simplified, Go-oriented representation of the program
// that sits between the AST and Go source code emission.
package ir

// Program is the top-level WIR node — a complete compilation unit.
type Program struct {
	Package   string
	Imports   []string
	Functions []*Function
	Classes   []*Class
	InitStmts []Stmt // top-level statements placed in main() or init()
}

// Function represents a Go function.
type Function struct {
	Name        string
	Params      []*Param
	ReturnTypes []string // Go type names
	Body        []Stmt
	IsMethod    bool
	Receiver    string // struct name if IsMethod
}

// Param is a function parameter with a Go type.
type Param struct {
	Name string // Go-safe name (no $)
	Type string // Go type
}

// Class maps to a Go struct + methods.
type Class struct {
	Name        string
	Extends     string
	Fields      []*Field
	Methods     []*Function
	Constructor *Function // __construct
}

// Field is a struct field.
type Field struct {
	Name       string
	Type       string
	Default    Expr // may be nil
	Visibility string
}

// ========== Statements ==========

// Stmt is the interface for all IR statements.
type Stmt interface {
	irStmt()
}

// VarDeclStmt: var x Type = value
type VarDeclStmt struct {
	Name  string
	Type  string
	Value Expr // may be nil
}

func (*VarDeclStmt) irStmt() {}

// AssignStmt: x = value
type AssignStmt struct {
	Target Expr
	Op     string // "=", "+=", etc.
	Value  Expr
}

func (*AssignStmt) irStmt() {}

// ExprStmt wraps an expression used as a statement.
type ExprStmt struct {
	Expr Expr
}

func (*ExprStmt) irStmt() {}

// ReturnStmt: return val1, val2
type ReturnStmt struct {
	Values []Expr
}

func (*ReturnStmt) irStmt() {}

// IfStmt: if cond { } else if { } else { }
type IfStmt struct {
	Condition Expr
	Body      []Stmt
	ElseIfs   []*ElseIfClause
	ElseBody  []Stmt // may be nil
}

func (*IfStmt) irStmt() {}

// ElseIfClause is a single else-if branch.
type ElseIfClause struct {
	Condition Expr
	Body      []Stmt
}

// ForStmt: for init; cond; update { }
type ForStmt struct {
	Init   Stmt
	Cond   Expr
	Update Stmt
	Body   []Stmt
}

func (*ForStmt) irStmt() {}

// RangeStmt: for key, value := range iterable { }
type RangeStmt struct {
	Key      string
	Value    string
	Iterable Expr
	Body     []Stmt
}

func (*RangeStmt) irStmt() {}

// SwitchStmt: Go switch for Wolf's match.
type SwitchStmt struct {
	Subject Expr
	Cases   []*SwitchCase
	Default []Stmt
}

func (*SwitchStmt) irStmt() {}

// SwitchCase is one case in a switch.
type SwitchCase struct {
	Value Expr
	Body  []Stmt
}

// BlockStmt is a plain block of statements.
type BlockStmt struct {
	Stmts []Stmt
}

func (*BlockStmt) irStmt() {}

// GoStmt: go func() { ... }() for parallel blocks.
type GoStmt struct {
	Body []Stmt
}

func (*GoStmt) irStmt() {}

// DeferStmt: defer statement.
type DeferStmt struct {
	Call Expr
}

func (*DeferStmt) irStmt() {}

// RawStmt emits raw Go code verbatim.
type RawStmt struct {
	Code string
}

func (*RawStmt) irStmt() {}

// ========== Expressions ==========

// Expr is the interface for all IR expressions.
type Expr interface {
	irExpr()
}

// Ident is a Go identifier.
type Ident struct {
	Name string
}

func (*Ident) irExpr() {}

// IntLit is an integer literal.
type IntLit struct {
	Value string
}

func (*IntLit) irExpr() {}

// FloatLit is a float literal.
type FloatLit struct {
	Value string
}

func (*FloatLit) irExpr() {}

// StringLit is a string literal (Go-escaped).
type StringLit struct {
	Value string
}

func (*StringLit) irExpr() {}

// BoolLit is true/false.
type BoolLit struct {
	Value bool
}

func (*BoolLit) irExpr() {}

// NilLit is Go's nil.
type NilLit struct{}

func (*NilLit) irExpr() {}

// BinaryExpr: left op right.
type BinaryExpr struct {
	Left  Expr
	Op    string
	Right Expr
}

func (*BinaryExpr) irExpr() {}

// UnaryExpr: op operand.
type UnaryExpr struct {
	Op      string
	Operand Expr
}

func (*UnaryExpr) irExpr() {}

// CallExpr: callee(args).
type CallExpr struct {
	Callee Expr
	Args   []Expr
}

func (*CallExpr) irExpr() {}

// FieldAccess: obj.field (Go dot notation).
type FieldAccess struct {
	Object Expr
	Field  string
}

func (*FieldAccess) irExpr() {}

// MethodCallExpr: obj.Method(args).
type MethodCallExpr struct {
	Object Expr
	Method string
	Args   []Expr
}

func (*MethodCallExpr) irExpr() {}

// StaticCall: Class::Method(args).
type StaticCall struct {
	Class  string
	Method string
	Args   []Expr
}

func (*StaticCall) irExpr() {}

// IndexExpr: obj[index].
type IndexExpr struct {
	Object Expr
	Index  Expr
}

func (*IndexExpr) irExpr() {}

// SliceLit: []Type{elems}.
type SliceLit struct {
	ElemType string
	Elements []Expr
}

func (*SliceLit) irExpr() {}

// MapLit: map[K]V{entries}.
type MapLit struct {
	KeyType   string
	ValueType string
	Keys      []Expr
	Values    []Expr
}

func (*MapLit) irExpr() {}

// StructLit: TypeName{fields}.
type StructLit struct {
	TypeName string
	Fields   map[string]Expr
}

func (*StructLit) irExpr() {}

// FuncLit: func(params) ReturnType { body }.
type FuncLit struct {
	Params      []*Param
	ReturnTypes []string
	Body        []Stmt
}

func (*FuncLit) irExpr() {}

// FmtSprintf: fmt.Sprintf(format, args...) for string interpolation.
type FmtSprintf struct {
	Format string
	Args   []Expr
}

func (*FmtSprintf) irExpr() {}

// ChanMake: make(chan Type).
type ChanMake struct {
	ElemType string
}

func (*ChanMake) irExpr() {}

// ChanSend: ch <- value.
type ChanSend struct {
	Channel Expr
	Value   Expr
}

func (*ChanSend) irExpr() {}

// ChanRecv: <-ch.
type ChanRecv struct {
	Channel Expr
}

func (*ChanRecv) irExpr() {}

// PostfixExpr: operand++ or operand--.
type PostfixExpr struct {
	Operand Expr
	Op      string
}

func (*PostfixExpr) irExpr() {}

// StringConcat: left + right (Go string concatenation).
type StringConcat struct {
	Left  Expr
	Right Expr
}

func (*StringConcat) irExpr() {}

// ErrorNew: errors.New(msg) or fmt.Errorf(msg).
type ErrorNew struct {
	Message Expr
}

func (*ErrorNew) irExpr() {}

// ========== HTTP Server Nodes ==========

// RouteStmt registers an HTTP route handler.
type RouteStmt struct {
	Method  string // "GET", "POST", etc.
	Path    string // "/health", "/rides/estimate"
	Handler string // function name to call
}

func (*RouteStmt) irStmt() {}

// ServeStmt starts the HTTP server.
type ServeStmt struct {
	Port    Expr // port number expression
	Handler Expr // handler function expression
}

func (*ServeStmt) irStmt() {}

// RespondStmt sends an HTTP JSON response.
type RespondStmt struct {
	Status Expr // HTTP status code
	Body   Expr // response body (map/value to JSON encode)
}

func (*RespondStmt) irStmt() {}
