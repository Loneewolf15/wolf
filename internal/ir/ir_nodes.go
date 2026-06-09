// Package ir defines the Wolf Intermediate Representation (WIR).
// WIR is a simplified, Go-oriented representation of the program
// that sits between the AST and Go source code emission.
package ir

// Program is the top-level WIR node — a complete compilation unit.
type Program struct {
	Package    string
	Imports    []string
	Interfaces []*Interface
	Functions        []*Function
	Classes          []*Class
	InitStmts        []Stmt // top-level statements placed in main() or init()
	RequiresMLBridge bool   // true if @ml block is used
}

// Function represents a Go function.
type Function struct {
	Name        string
	TypeParams  []string // generic type params
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
	TypeParams  []string // generic type params
	Implements  []string // interface names
	Fields      []*Field
	Methods     []*Function
	Constructor *Function // __construct
}

// Interface represents an interface definition in WIR.
type Interface struct {
	Name    string
	Methods []*InterfaceMethodSig
}

// InterfaceMethodSig is a method signature within an interface.
type InterfaceMethodSig struct {
	Name        string
	Params      []*Param
	ReturnTypes []string
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

func (*VarDeclStmt) irStmt() { _ = 0 }

// AssignStmt: x = value
type AssignStmt struct {
	Target Expr
	Op     string // "=", "+=", etc.
	Value  Expr
}

func (*AssignStmt) irStmt() { _ = 0 }

// ExprStmt wraps an expression used as a statement.
type ExprStmt struct {
	Expr Expr
}

func (*ExprStmt) irStmt() { _ = 0 }

// ReturnStmt: return val1, val2
type ReturnStmt struct {
	Values []Expr
}

func (*ReturnStmt) irStmt() { _ = 0 }

// IfStmt: if cond { } else if { } else { }
type IfStmt struct {
	Condition Expr
	Body      []Stmt
	ElseIfs   []*ElseIfClause
	ElseBody  []Stmt // may be nil
}

func (*IfStmt) irStmt() { _ = 0 }

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

func (*ForStmt) irStmt() { _ = 0 }

// RangeStmt: for key, value := range iterable { }
type RangeStmt struct {
	Key      string
	Value    string
	Iterable Expr
	Body     []Stmt
}

func (*RangeStmt) irStmt() { _ = 0 }

// SwitchStmt: Go switch for Wolf's match.
type SwitchStmt struct {
	Subject Expr
	Cases   []*SwitchCase
	Default []Stmt
}

func (*SwitchStmt) irStmt() { _ = 0 }

// SwitchCase is one case in a switch.
type SwitchCase struct {
	Value Expr
	Body  []Stmt
}

// BlockStmt is a plain block of statements.
type BlockStmt struct {
	Stmts []Stmt
}

func (*BlockStmt) irStmt() { _ = 0 }

// TryCatchStmt: try { } catch (err) { }
type TryCatchStmt struct {
	TryBody   []Stmt
	CatchVar  string
	CatchBody []Stmt
}

func (*TryCatchStmt) irStmt() { _ = 0 }

// SpawnStmt: spawn fn(args...) inside a supervise module
type SpawnStmt struct {
	Name string // for tracking
	Call *CallExpr
}

func (*SpawnStmt) irStmt() { _ = 0 }

// WaitAllStmt represents the synchronization barrier at the end of @supervise
type WaitAllStmt struct{}

func (*WaitAllStmt) irStmt() { _ = 0 }

// DeferStmt: defer statement.
type DeferStmt struct {
	Call Expr
}

func (*DeferStmt) irStmt() { _ = 0 }

// RawStmt emits raw Go code verbatim.
type RawStmt struct {
	Code string
}

func (*RawStmt) irStmt() { _ = 0 }

// SuperviseStmt implements the Let It Crash block in WIR.
type SuperviseStmt struct {
	Strategy string
	Restart  string
	Max      int
	Body     []Stmt
}

func (*SuperviseStmt) irStmt() { _ = 0 }

// TraceStmt implements the observability trace block in WIR.
type TraceStmt struct {
	SpanName Expr
	Body     []Stmt
}

func (*TraceStmt) irStmt() { _ = 0 }

// MLBlockStmt implements the @ml Python block in WIR.
type MLBlockStmt struct {
	PythonCode string
	InputVars  []string
	OutputVars []string
}

func (*MLBlockStmt) irStmt() { _ = 0 }

// ========== Expressions ==========

// Expr is the interface for all IR expressions.
type Expr interface {
	irExpr()
}

// Ident is a Go identifier.
type Ident struct {
	Name string
}

func (*Ident) irExpr() { _ = 0 }

// IntLit is an integer literal.
type IntLit struct {
	Value string
}

func (*IntLit) irExpr() { _ = 0 }

// FloatLit is a float literal.
type FloatLit struct {
	Value string
}

func (*FloatLit) irExpr() { _ = 0 }

// StringLit is a string literal (Go-escaped).
type StringLit struct {
	Value string
}

func (*StringLit) irExpr() { _ = 0 }

// BoolLit is true/false.
type BoolLit struct {
	Value bool
}

func (*BoolLit) irExpr() { _ = 0 }

// NilLit is Go's nil.
type NilLit struct{}

func (*NilLit) irExpr() { _ = 0 }

// BinaryExpr: left op right.
type BinaryExpr struct {
	Left  Expr
	Op    string
	Right Expr
}

func (*BinaryExpr) irExpr() { _ = 0 }

// UnaryExpr: op operand.
type UnaryExpr struct {
	Op      string
	Operand Expr
}

func (*UnaryExpr) irExpr() { _ = 0 }

// CallExpr: callee(args).
type CallExpr struct {
	Callee   Expr
	TypeArgs []string
	Args     []Expr
}

func (*CallExpr) irExpr() { _ = 0 }

// FieldAccess: obj.field (Go dot notation).
type FieldAccess struct {
	Object Expr
	Field  string
}

func (*FieldAccess) irExpr() { _ = 0 }

// MethodCallExpr: obj.Method(args).
type MethodCallExpr struct {
	Object   Expr
	Method   string
	TypeArgs []string
	Args     []Expr
}

func (*MethodCallExpr) irExpr() { _ = 0 }

// StaticCall: Class::Method(args).
type StaticCall struct {
	Class  string
	Method string
	Args   []Expr
}

func (*StaticCall) irExpr() { _ = 0 }

// IndexExpr: obj[index].
type IndexExpr struct {
	Object Expr
	Index  Expr
}

func (*IndexExpr) irExpr() { _ = 0 }

// SliceLit: []Type{elems}.
type SliceLit struct {
	ElemType string
	Elements []Expr
}

func (*SliceLit) irExpr() { _ = 0 }

// MapLit: map[K]V{entries}.
type MapLit struct {
	KeyType   string
	ValueType string
	Keys      []Expr
	Values    []Expr
}

func (*MapLit) irExpr() { _ = 0 }

// StructLit: TypeName{fields}.
type StructLit struct {
	TypeName string
	Fields   map[string]Expr
}

func (*StructLit) irExpr() { _ = 0 }

// FuncLit: func(params) ReturnType { body }.
type FuncLit struct {
	Params      []*Param
	ReturnTypes []string
	Body        []Stmt
}

func (*FuncLit) irExpr() { _ = 0 }

// FmtSprintf: fmt.Sprintf(format, args...) for string interpolation.
type FmtSprintf struct {
	Format string
	Args   []Expr
}

func (*FmtSprintf) irExpr() { _ = 0 }

// ChanMake: make(chan Type).
type ChanMake struct {
	ElemType string
}

func (*ChanMake) irExpr() { _ = 0 }

// ChanSend: ch <- value.
type ChanSend struct {
	Channel Expr
	Value   Expr
}

func (*ChanSend) irExpr() { _ = 0 }

// ChanRecv: <-ch.
type ChanRecv struct {
	Channel Expr
}

func (*ChanRecv) irExpr() { _ = 0 }

// PostfixExpr: operand++ or operand--.
type PostfixExpr struct {
	Operand Expr
	Op      string
}

func (*PostfixExpr) irExpr() { _ = 0 }

// StringConcat: left + right (Go string concatenation).
type StringConcat struct {
	Left  Expr
	Right Expr
}

func (*StringConcat) irExpr() { _ = 0 }

// ErrorNew: errors.New(msg) or fmt.Errorf(msg).
type ErrorNew struct {
	Message Expr
}

func (*ErrorNew) irExpr() { _ = 0 }

// ========== HTTP Server Nodes ==========

// RouteStmt registers an HTTP route handler.
type RouteStmt struct {
	Method  string // "GET", "POST", etc.
	Path    string // "/health", "/rides/estimate"
	Handler string // function name to call
}

func (*RouteStmt) irStmt() { _ = 0 }

// ServeStmt starts the HTTP server.
type ServeStmt struct {
	Port    Expr // port number expression
	Handler Expr // handler function expression
}

func (*ServeStmt) irStmt() { _ = 0 }

// RespondStmt sends an HTTP JSON response.
type RespondStmt struct {
	Status Expr // HTTP status code
	Body   Expr // response body (map/value to JSON encode)
}

func (*RespondStmt) irStmt() { _ = 0 }
