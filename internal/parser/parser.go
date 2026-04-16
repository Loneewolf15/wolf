package parser

import (
	"fmt"

	"github.com/wolflang/wolf/internal/lexer"
)

// Parser produces an AST from a token stream.
type Parser struct {
	tokens  []lexer.Token
	current int
	errors  []*lexer.WolfError
	file    string
}

// New creates a new Parser for the given token slice.
func New(tokens []lexer.Token, file string) *Parser {
	return &Parser{
		tokens: tokens,
		file:   file,
	}
}

// Parse parses the token stream and returns the program AST.
func (p *Parser) Parse() (*Program, []*lexer.WolfError) {
	program := &Program{
		Pos_: p.currentPos(),
	}

	for !p.isAtEnd() {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
	}

	return program, p.errors
}

// Errors returns accumulated parse errors.
func (p *Parser) Errors() []*lexer.WolfError {
	return p.errors
}

// ========== Statement Parsing ==========

func (p *Parser) parseTypeParams() []string {
	var params []string
	if !p.check(lexer.TOKEN_LESS) {
		return params
	}
	p.advance() // consume '<'

	for !p.check(lexer.TOKEN_GREATER) && !p.isAtEnd() {
		if p.check(lexer.TOKEN_IDENT) {
			params = append(params, p.advance().Literal)
		} else {
			p.addError("expected type parameter identifier")
			p.advance()
		}
		if p.check(lexer.TOKEN_COMMA) {
			p.advance()
		}
	}
	p.expect(lexer.TOKEN_GREATER, "expected '>' after type parameters")
	return params
}

func (p *Parser) parseStatement() Statement {
	switch p.peek().Type {

	case lexer.TOKEN_FUNC:
		return p.parseFuncDecl()

	case lexer.TOKEN_CLASS:
		return p.parseClassDecl()

	case lexer.TOKEN_INTERFACE:
		return p.parseInterfaceDecl()

	case lexer.TOKEN_ENUM:
		return p.parseEnumDecl()

	case lexer.TOKEN_VAR:
		return p.parseVarDecl()

	case lexer.TOKEN_IF:
		return p.parseIfStmt()

	case lexer.TOKEN_FOR:
		return p.parseForStmt()

	case lexer.TOKEN_FOREACH:
		return p.parseForeachStmt()

	case lexer.TOKEN_WHILE:
		return p.parseWhileStmt()

	case lexer.TOKEN_MATCH:
		return p.parseMatchStmt()

	case lexer.TOKEN_RETURN:
		return p.parseReturnStmt()

	case lexer.TOKEN_TRY:
		return p.parseTryCatchStmt()

	case lexer.TOKEN_IMPORT:
		return p.parseImportStmt()

	case lexer.TOKEN_AT_ML:
		return p.parseMLBlockStmt()

	case lexer.TOKEN_AT_SUPERVISE:
		return p.parseSuperviseBlockStmt()

	case lexer.TOKEN_AT_TRACE:
		return p.parseTraceBlockStmt()

	case lexer.TOKEN_PARALLEL:
		return p.parseParallelStmt()

	case lexer.TOKEN_LBRACKET:
		// Could be [$a, $b] = expr (destructuring) or array expression
		return p.parseDestructureOrExprStmt()

	default:
		return p.parseExpressionStmt()
	}
}

// ---- func declaration ----

func (p *Parser) parseFuncDecl() *FuncDecl {
	pos := p.currentPos()
	p.expect(lexer.TOKEN_FUNC, "expected 'func'")

	// Optional name (closures have no name, but declared at statement level they do)
	name := ""
	if p.check(lexer.TOKEN_IDENT) {
		name = p.advance().Literal
	}

	typeParams := p.parseTypeParams()

	params := p.parseParamList()

	var returnType *ReturnTypeSpec
	if p.check(lexer.TOKEN_ARROW) {
		returnType = p.parseReturnType()
	}

	// Arrow shorthand: func name($x) => $x * 2
	if p.check(lexer.TOKEN_FAT_ARROW) {
		p.advance()
		expr := p.parseExpression()
		return &FuncDecl{
			Name:       name,
			TypeParams: typeParams,
			Params:     params,
			ReturnType: returnType,
			ArrowExpr:  expr,
			Pos_:       pos,
		}
	}

	body := p.parseBlock()

	return &FuncDecl{
		Name:       name,
		TypeParams: typeParams,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
		Pos_:       pos,
	}
}

func (p *Parser) parseParamList() []*Param {
	var params []*Param
	if !p.check(lexer.TOKEN_LPAREN) {
		return params
	}
	p.advance() // (

	for !p.check(lexer.TOKEN_RPAREN) && !p.isAtEnd() {
		param := p.parseParam()
		params = append(params, param)
		if !p.check(lexer.TOKEN_RPAREN) {
			p.expect(lexer.TOKEN_COMMA, "expected ',' between parameters")
		}
	}

	p.expect(lexer.TOKEN_RPAREN, "expected ')' after parameters")
	return params
}

func (p *Parser) parseParam() *Param {
	pos := p.currentPos()

	// Named parameter form: name: $var or just $var
	// Check if it's a named param by looking for ident: pattern
	if p.check(lexer.TOKEN_IDENT) && p.peekNext().Type == lexer.TOKEN_COLON {
		// This is actually a named parameter definition: name: $var = default
		// But in function declarations, params are just $var
		// Named params are: func foo(name: $name, email: $email)
		// But actually in Wolf PRD, the param declaration is still $var with optional default
		// name: is used at call site. Let's just parse $var style.
	}

	name := ""
	if p.check(lexer.TOKEN_DOLLAR_IDENT) {
		name = p.advance().Literal
	} else {
		p.addError("expected parameter name ($variable)")
		p.advance()
		return &Param{Name: "", Pos_: pos}
	}

	typeName := ""
	if p.check(lexer.TOKEN_COLON) {
		p.advance()
		if p.check(lexer.TOKEN_IDENT) {
			typeName = p.advance().Literal
		}
	}

	var defaultVal Expression
	if p.check(lexer.TOKEN_ASSIGN) {
		p.advance()
		defaultVal = p.parseExpression()
	}

	return &Param{
		Name:     name,
		TypeName: typeName,
		Default:  defaultVal,
		Pos_:     pos,
	}
}

func (p *Parser) parseReturnType() *ReturnTypeSpec {
	pos := p.currentPos()
	p.advance() // consume ->

	var types []string
	if p.check(lexer.TOKEN_LPAREN) {
		p.advance()
		for !p.check(lexer.TOKEN_RPAREN) && !p.isAtEnd() {
			if p.check(lexer.TOKEN_IDENT) || p.check(lexer.TOKEN_ERROR) {
				types = append(types, p.advance().Literal)
			} else {
				p.addError("expected type name in return type")
				p.advance()
			}
			if !p.check(lexer.TOKEN_RPAREN) {
				p.expect(lexer.TOKEN_COMMA, "expected ',' between return types")
			}
		}
		p.expect(lexer.TOKEN_RPAREN, "expected ')' after return types")
	} else if p.check(lexer.TOKEN_IDENT) {
		types = append(types, p.advance().Literal)
	}

	return &ReturnTypeSpec{Types: types, Pos_: pos}
}

// ---- enum declaration ----

func (p *Parser) parseEnumDecl() *EnumDecl {
	pos := p.currentPos()
	p.expect(lexer.TOKEN_ENUM, "expected 'enum'")

	name := ""
	if p.check(lexer.TOKEN_IDENT) {
		name = p.advance().Literal
	} else {
		p.addError("expected enum name")
	}

	p.expect(lexer.TOKEN_LBRACE, "expected '{' after enum name")

	var variants []string
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		if p.check(lexer.TOKEN_IDENT) {
			variants = append(variants, p.advance().Literal)
		} else {
			p.addError("expected enum variant identifier")
			p.advance()
		}
		if p.check(lexer.TOKEN_COMMA) {
			p.advance()
		}
	}

	p.expect(lexer.TOKEN_RBRACE, "expected '}' to close enum body")

	return &EnumDecl{
		Name:     name,
		Variants: variants,
		Pos_:     pos,
	}
}

// ---- class declaration ----

func (p *Parser) parseClassDecl() *ClassDecl {
	pos := p.currentPos()
	p.expect(lexer.TOKEN_CLASS, "expected 'class'")

	name := ""
	if p.check(lexer.TOKEN_IDENT) {
		name = p.advance().Literal
	} else {
		p.addError("expected class name")
	}

	typeParams := p.parseTypeParams()

	extends := ""
	if p.check(lexer.TOKEN_IDENT) && p.peek().Literal == "extends" {
		p.advance() // consume "extends"
		if p.check(lexer.TOKEN_IDENT) {
			extends = p.advance().Literal
		} else {
			p.addError("expected base class name after 'extends'")
		}
	}

	// implements Foo, Bar
	var implements []string
	if p.check(lexer.TOKEN_IMPLEMENTS) {
		p.advance() // consume "implements"
		for {
			if p.check(lexer.TOKEN_IDENT) {
				implements = append(implements, p.advance().Literal)
			} else {
				p.addError("expected interface name after 'implements'")
			}
			if p.check(lexer.TOKEN_COMMA) {
				p.advance()
			} else {
				break
			}
		}
	}

	p.expect(lexer.TOKEN_LBRACE, "expected '{' after class name")

	var properties []*PropertyDecl
	var methods []*FuncDecl

	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		visibility := ""
		if p.check(lexer.TOKEN_PRIVATE) || p.check(lexer.TOKEN_PUBLIC) || p.check(lexer.TOKEN_STATIC) {
			visibility = p.advance().Literal
		}

		if p.check(lexer.TOKEN_FUNC) {
			method := p.parseFuncDecl()
			methods = append(methods, method)
		} else if p.check(lexer.TOKEN_DOLLAR_IDENT) {
			prop := p.parsePropertyDecl(visibility)
			properties = append(properties, prop)
		} else {
			p.addError("expected property or method declaration in class body")
			p.advance()
		}
	}

	p.expect(lexer.TOKEN_RBRACE, "expected '}' to close class body")

	return &ClassDecl{
		Name:       name,
		Extends:    extends,
		TypeParams: typeParams,
		Implements: implements,
		Properties: properties,
		Methods:    methods,
		Pos_:       pos,
	}
}

// ---- interface declaration ----

func (p *Parser) parseInterfaceDecl() *InterfaceDecl {
	pos := p.currentPos()
	p.expect(lexer.TOKEN_INTERFACE, "expected 'interface'")

	name := ""
	if p.check(lexer.TOKEN_IDENT) {
		name = p.advance().Literal
	} else {
		p.addError("expected interface name")
	}

	p.expect(lexer.TOKEN_LBRACE, "expected '{' after interface name")

	var methods []*InterfaceMethod
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		if p.check(lexer.TOKEN_FUNC) {
			m := p.parseInterfaceMethod()
			methods = append(methods, m)
		} else {
			p.addError("expected method signature in interface body")
			p.advance()
		}
	}

	p.expect(lexer.TOKEN_RBRACE, "expected '}' to close interface body")

	return &InterfaceDecl{
		Name:    name,
		Methods: methods,
		Pos_:    pos,
	}
}

// parseInterfaceMethod parses a method signature (no body) inside an interface.
func (p *Parser) parseInterfaceMethod() *InterfaceMethod {
	pos := p.currentPos()
	p.expect(lexer.TOKEN_FUNC, "expected 'func'")

	name := ""
	if p.check(lexer.TOKEN_IDENT) {
		name = p.advance().Literal
	} else {
		p.addError("expected method name in interface")
	}

	params := p.parseParamList()

	var returnType *ReturnTypeSpec
	if p.check(lexer.TOKEN_ARROW) {
		returnType = p.parseReturnType()
	}

	return &InterfaceMethod{
		Name:       name,
		Params:     params,
		ReturnType: returnType,
		Pos_:       pos,
	}
}

func (p *Parser) parsePropertyDecl(visibility string) *PropertyDecl {
	pos := p.currentPos()
	name := p.advance().Literal // $propName

	typeName := ""
	if p.check(lexer.TOKEN_COLON) {
		p.advance()
		if p.check(lexer.TOKEN_IDENT) {
			typeName = p.advance().Literal
		}
	}

	var defaultVal Expression
	if p.check(lexer.TOKEN_ASSIGN) {
		p.advance()
		defaultVal = p.parseExpression()
	}

	return &PropertyDecl{
		Name:       name,
		TypeName:   typeName,
		Default:    defaultVal,
		Visibility: visibility,
		Pos_:       pos,
	}
}

// ---- var declaration ----

func (p *Parser) parseVarDecl() *VarDecl {
	pos := p.currentPos()
	p.advance() // consume 'var'

	name := ""
	if p.check(lexer.TOKEN_DOLLAR_IDENT) {
		name = p.advance().Literal
	} else {
		p.addError("expected variable name after 'var'")
	}

	typeName := ""
	if p.check(lexer.TOKEN_COLON) {
		p.advance()
		if p.check(lexer.TOKEN_IDENT) {
			typeName = p.advance().Literal
		}
		// Also handle []type arrays
		if p.check(lexer.TOKEN_LBRACKET) {
			p.advance()
			p.expect(lexer.TOKEN_RBRACKET, "expected ']'")
			if p.check(lexer.TOKEN_IDENT) {
				typeName = "[]" + p.advance().Literal
			}
		}
	}

	var value Expression
	if p.check(lexer.TOKEN_ASSIGN) {
		p.advance()
		value = p.parseExpression()
	}

	return &VarDecl{
		Name:     name,
		TypeName: typeName,
		Value:    value,
		Pos_:     pos,
	}
}

// ---- if statement ----

func (p *Parser) parseIfStmt() *IfStmt {
	pos := p.currentPos()
	p.advance() // consume 'if'

	condition := p.parseExpression()
	body := p.parseBlock()

	var elseIfs []*ElseIfClause
	var elseBody *BlockStmt

	for p.check(lexer.TOKEN_ELSE) {
		p.advance() // consume 'else'
		if p.check(lexer.TOKEN_IF) {
			elseIfPos := p.currentPos()
			p.advance() // consume 'if'
			eifCond := p.parseExpression()
			eifBody := p.parseBlock()
			elseIfs = append(elseIfs, &ElseIfClause{
				Condition: eifCond,
				Body:      eifBody,
				Pos_:      elseIfPos,
			})
		} else {
			elseBody = p.parseBlock()
			break
		}
	}

	return &IfStmt{
		Condition: condition,
		Body:      body,
		ElseIfs:   elseIfs,
		ElseBody:  elseBody,
		Pos_:      pos,
	}
}

// ---- for statement (C-style) ----

func (p *Parser) parseForStmt() *ForStmt {
	pos := p.currentPos()
	p.advance() // consume 'for'

	init := p.parseAssignOrExprStmt()
	p.expect(lexer.TOKEN_SEMICOLON, "expected ';' after for initializer")

	condition := p.parseExpression()
	p.expect(lexer.TOKEN_SEMICOLON, "expected ';' after for condition")

	update := p.parseAssignOrExprStmt()

	body := p.parseBlock()

	return &ForStmt{
		Init:      init,
		Condition: condition,
		Update:    update,
		Body:      body,
		Pos_:      pos,
	}
}

// ---- foreach statement ----

func (p *Parser) parseForeachStmt() *ForeachStmt {
	pos := p.currentPos()
	p.advance() // consume 'foreach'

	iterable := p.parseExpression()
	p.expect(lexer.TOKEN_AS, "expected 'as' after foreach iterable")

	// First var
	firstVar := ""
	if p.check(lexer.TOKEN_DOLLAR_IDENT) {
		firstVar = p.advance().Literal
	} else {
		p.addError("expected variable after 'as'")
	}

	keyVar := ""
	valueVar := firstVar
	// Check for $key => $value pattern
	if p.check(lexer.TOKEN_FAT_ARROW) {
		p.advance()
		keyVar = firstVar
		if p.check(lexer.TOKEN_DOLLAR_IDENT) {
			valueVar = p.advance().Literal
		} else {
			p.addError("expected value variable after '=>'")
		}
	}

	body := p.parseBlock()

	return &ForeachStmt{
		Iterable: iterable,
		ValueVar: valueVar,
		KeyVar:   keyVar,
		Body:     body,
		Pos_:     pos,
	}
}

// ---- while statement ----

func (p *Parser) parseWhileStmt() *WhileStmt {
	pos := p.currentPos()
	p.advance() // consume 'while'

	condition := p.parseExpression()
	body := p.parseBlock()

	return &WhileStmt{
		Condition: condition,
		Body:      body,
		Pos_:      pos,
	}
}

// ---- match statement ----

func (p *Parser) parseMatchStmt() *MatchStmt {
	pos := p.currentPos()
	p.advance() // consume 'match'

	subject := p.parseExpression()
	p.expect(lexer.TOKEN_LBRACE, "expected '{' after match expression")

	var arms []*MatchArm
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		arm := p.parseMatchArm()
		arms = append(arms, arm)
	}

	p.expect(lexer.TOKEN_RBRACE, "expected '}' to close match")

	return &MatchStmt{
		Subject: subject,
		Arms:    arms,
		Pos_:    pos,
	}
}

func (p *Parser) parseMatchArm() *MatchArm {
	pos := p.currentPos()

	// Pattern: literal, identifier, or _ (wildcard)
	var pattern Expression
	if p.check(lexer.TOKEN_IDENT) && p.peek().Literal == "_" {
		p.advance()
		pattern = &Wildcard{Pos_: pos}
	} else {
		pattern = p.parseExpression()
	}

	p.expect(lexer.TOKEN_FAT_ARROW, "expected '=>' after match pattern")

	// Body: one or more statements (usually a single expression/call)
	var body []Statement
	if p.check(lexer.TOKEN_LBRACE) {
		block := p.parseBlock()
		body = block.Statements
	} else {
		stmt := p.parseExpressionStmt()
		body = []Statement{stmt}
	}

	return &MatchArm{
		Pattern: pattern,
		Body:    body,
		Pos_:    pos,
	}
}

// ---- return ----

func (p *Parser) parseReturnStmt() *ReturnStmt {
	pos := p.currentPos()
	p.advance() // consume 'return'

	var values []Expression
	if !p.check(lexer.TOKEN_RBRACE) && !p.check(lexer.TOKEN_EOF) {
		values = append(values, p.parseExpression())
		// Support multiple return values: return expr1, expr2
		for p.check(lexer.TOKEN_COMMA) {
			p.advance()
			values = append(values, p.parseExpression())
		}
	}

	return &ReturnStmt{
		Values: values,
		Pos_:   pos,
	}
}

// ---- try/catch ----

func (p *Parser) parseTryCatchStmt() *TryCatchStmt {
	pos := p.currentPos()
	p.advance() // consume 'try'

	tryBody := p.parseBlock()

	p.expect(lexer.TOKEN_CATCH, "expected 'catch' after try block")

	catchVar := ""
	if p.check(lexer.TOKEN_LPAREN) {
		p.advance()
		if p.check(lexer.TOKEN_DOLLAR_IDENT) {
			catchVar = p.advance().Literal
		}
		p.expect(lexer.TOKEN_RPAREN, "expected ')' after catch variable")
	}

	catchBody := p.parseBlock()

	return &TryCatchStmt{
		TryBody:   tryBody,
		CatchVar:  catchVar,
		CatchBody: catchBody,
		Pos_:      pos,
	}
}

// ---- import ----

func (p *Parser) parseImportStmt() *ImportStmt {
	pos := p.currentPos()
	p.advance() // consume 'import'

	path := ""
	if p.check(lexer.TOKEN_STRING) {
		path = p.advance().Literal
	} else {
		p.addError("expected string after 'import'")
	}

	return &ImportStmt{Path: path, Pos_: pos}
}

// ---- @ml block ----

func (p *Parser) parseMLBlockStmt() *MLBlockStmt {
	pos := p.currentPos()
	p.advance() // consume @ml

	ml := &MLBlockStmt{Pos_: pos}

	// Check for optional modifiers: model("name"), async, (in: [...], out: [...])
	if p.check(lexer.TOKEN_IDENT) && p.peek().Literal == "model" {
		p.advance()
		if p.check(lexer.TOKEN_LPAREN) {
			p.advance()
			ml.ModelName = p.parseExpression()
			p.expect(lexer.TOKEN_RPAREN, "expected ')' after model name")
		}
	}

	if p.check(lexer.TOKEN_ASYNC) {
		p.advance()
		ml.IsAsync = true
	}

	// Parse optional (in: [...], out: [...])
	if p.check(lexer.TOKEN_LPAREN) {
		p.advance()
		for !p.check(lexer.TOKEN_RPAREN) && !p.isAtEnd() {
			if p.check(lexer.TOKEN_IDENT) {
				label := p.advance().Literal
				p.expect(lexer.TOKEN_COLON, "expected ':' after in/out label")
				vars := p.parseBracketedVarList()
				if label == "in" {
					ml.InVars = vars
				} else if label == "out" {
					ml.OutVars = vars
				}
			}
			if p.check(lexer.TOKEN_COMMA) {
				p.advance()
			}
		}
		p.expect(lexer.TOKEN_RPAREN, "expected ')' after @ml options")
	}

	// The body is raw Python code inside { }
	p.expect(lexer.TOKEN_LBRACE, "expected '{' to start @ml block")
	ml.Body = p.consumeRawBlock()

	return ml
}

// consumeRawBlock reads tokens until a matching } and returns their literals as raw text.
func (p *Parser) consumeRawBlock() string {
	depth := 1
	raw := ""
	for !p.isAtEnd() && depth > 0 {
		tok := p.advance()
		if tok.Type == lexer.TOKEN_LBRACE {
			depth++
		}
		if tok.Type == lexer.TOKEN_RBRACE {
			depth--
			if depth == 0 {
				break
			}
		}
		raw += tok.Literal + " "
	}
	return raw
}

func (p *Parser) parseBracketedVarList() []string {
	var vars []string
	p.expect(lexer.TOKEN_LBRACKET, "expected '['")
	for !p.check(lexer.TOKEN_RBRACKET) && !p.isAtEnd() {
		if p.check(lexer.TOKEN_DOLLAR_IDENT) {
			vars = append(vars, p.advance().Literal)
		} else {
			p.advance()
		}
		if p.check(lexer.TOKEN_COMMA) {
			p.advance()
		}
	}
	p.expect(lexer.TOKEN_RBRACKET, "expected ']'")
	return vars
}

// ---- @supervise block ----

func (p *Parser) parseSuperviseBlockStmt() *SuperviseBlockStmt {
	pos := p.currentPos()
	p.advance() // consume @supervise

	sup := &SuperviseBlockStmt{Pos_: pos}

	if p.check(lexer.TOKEN_LPAREN) {
		p.advance()
		for !p.check(lexer.TOKEN_RPAREN) && !p.isAtEnd() {
			if p.check(lexer.TOKEN_IDENT) {
				key := p.advance().Literal
				p.expect(lexer.TOKEN_COLON, "expected ':' after supervise option key")

				if key == "strategy" {
					if p.check(lexer.TOKEN_STRING) {
						sup.Strategy = p.advance().Literal
					} else {
						p.addError("expected string for strategy option")
						p.advance()
					}
				} else if key == "restart" {
					if p.check(lexer.TOKEN_STRING) {
						sup.Restart = p.advance().Literal
					} else {
						p.addError("expected string for restart option")
						p.advance()
					}
				} else if key == "max" {
					if p.check(lexer.TOKEN_INT) {
						fmt.Sscanf(p.advance().Literal, "%d", &sup.Max)
					} else {
						p.addError("expected integer for max option")
						p.advance()
					}
				} else {
					p.addError("unknown supervise option: " + key)
				}
			} else {
				p.advance()
			}
			if p.check(lexer.TOKEN_COMMA) {
				p.advance()
			}
		}
		p.expect(lexer.TOKEN_RPAREN, "expected ')' after supervise options")
	}

	sup.Body = p.parseBlock()
	return sup
}

// ---- @trace block ----

func (p *Parser) parseTraceBlockStmt() *TraceBlockStmt {
	pos := p.currentPos()
	p.advance() // consume @trace

	trace := &TraceBlockStmt{Pos_: pos}

	if p.check(lexer.TOKEN_LPAREN) {
		p.advance()
		trace.SpanName = p.parseExpression()
		p.expect(lexer.TOKEN_RPAREN, "expected ')' after trace span name")
	}

	trace.Body = p.parseBlock()
	return trace
}

// ---- parallel ----

func (p *Parser) parseParallelStmt() *ParallelStmt {
	pos := p.currentPos()
	p.advance() // consume 'parallel'
	body := p.parseBlock()
	return &ParallelStmt{Body: body, Pos_: pos}
}

// ---- destructure or expression statement ----

func (p *Parser) parseDestructureOrExprStmt() Statement {
	pos := p.currentPos()

	// Peek ahead: if [ $ident, ... ] = , it's a destructure
	if p.isDestructureAssign() {
		return p.parseDestructureAssign(pos)
	}

	return p.parseExpressionStmt()
}

func (p *Parser) isDestructureAssign() bool {
	// Save state
	saved := p.current

	if !p.check(lexer.TOKEN_LBRACKET) {
		return false
	}
	p.current++

	// Check for $ident pattern inside brackets
	for p.current < len(p.tokens) {
		tok := p.tokens[p.current]
		if tok.Type == lexer.TOKEN_RBRACKET {
			p.current++
			// Check if next is =
			if p.current < len(p.tokens) && p.tokens[p.current].Type == lexer.TOKEN_ASSIGN {
				p.current = saved
				return true
			}
			p.current = saved
			return false
		}
		if tok.Type == lexer.TOKEN_DOLLAR_IDENT || tok.Type == lexer.TOKEN_COMMA {
			p.current++
			continue
		}
		break
	}

	p.current = saved
	return false
}

func (p *Parser) parseDestructureAssign(pos lexer.Position) *DestructureAssign {
	p.advance() // consume [
	var names []string
	for !p.check(lexer.TOKEN_RBRACKET) && !p.isAtEnd() {
		if p.check(lexer.TOKEN_DOLLAR_IDENT) {
			names = append(names, p.advance().Literal)
		}
		if p.check(lexer.TOKEN_COMMA) {
			p.advance()
		}
	}
	p.expect(lexer.TOKEN_RBRACKET, "expected ']'")
	p.expect(lexer.TOKEN_ASSIGN, "expected '=' after destructure target")
	value := p.parseExpression()

	return &DestructureAssign{
		Names: names,
		Value: value,
		Pos_:  pos,
	}
}

// ---- expression statement ----

func (p *Parser) parseExpressionStmt() *ExpressionStmt {
	pos := p.currentPos()
	expr := p.parseExpression()

	// Check for assignment: $var = expr, $obj->prop += expr
	if p.isAssignOp() {
		op := p.advance().Literal
		value := p.parseExpression()
		return &ExpressionStmt{
			Expr: &BinaryExpr{Left: expr, Op: op, Right: value, Pos_: pos},
			Pos_: pos,
		}
	}

	return &ExpressionStmt{Expr: expr, Pos_: pos}
}

func (p *Parser) isAssignOp() bool {
	t := p.peek().Type
	return t == lexer.TOKEN_ASSIGN ||
		t == lexer.TOKEN_PLUS_ASSIGN ||
		t == lexer.TOKEN_MINUS_ASSIGN ||
		t == lexer.TOKEN_STAR_ASSIGN ||
		t == lexer.TOKEN_SLASH_ASSIGN
}

// ---- assignment or expression (for for-loop init/update) ----

func (p *Parser) parseAssignOrExprStmt() Statement {
	pos := p.currentPos()
	expr := p.parseExpression()

	if p.isAssignOp() {
		op := p.advance().Literal
		value := p.parseExpression()
		return &AssignStmt{
			Target: expr,
			Op:     op,
			Value:  value,
			Pos_:   pos,
		}
	}

	return &ExpressionStmt{Expr: expr, Pos_: pos}
}

// ---- block ----

func (p *Parser) parseBlock() *BlockStmt {
	pos := p.currentPos()
	p.expect(lexer.TOKEN_LBRACE, "expected '{'")

	var stmts []Statement
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	p.expect(lexer.TOKEN_RBRACE, "expected '}'")

	return &BlockStmt{Statements: stmts, Pos_: pos}
}

// ========== Expression Parsing (with precedence) ==========

// Precedence climbing: lowest → highest
// 1. || (logical OR)
// 2. && (logical AND)
// 3. == != (equality)
// 4. < <= > >= (comparison)
// 5. .. (string concat)
// 6. + - (addition)
// 7. * / % (multiplication)
// 8. ! - (unary prefix)
// 9. ++ -- (postfix)
// 10. () [] -> :: . (call, index, access)

func (p *Parser) parseExpression() Expression {
	return p.parseOr()
}

func (p *Parser) parseOr() Expression {
	left := p.parseAnd()
	for p.check(lexer.TOKEN_OR) {
		pos := p.currentPos()
		p.advance()
		right := p.parseAnd()
		left = &BinaryExpr{Left: left, Op: "||", Right: right, Pos_: pos}
	}
	return left
}

func (p *Parser) parseAnd() Expression {
	left := p.parseEquality()
	for p.check(lexer.TOKEN_AND) {
		pos := p.currentPos()
		p.advance()
		right := p.parseEquality()
		left = &BinaryExpr{Left: left, Op: "&&", Right: right, Pos_: pos}
	}
	return left
}

func (p *Parser) parseEquality() Expression {
	left := p.parseComparison()
	for p.check(lexer.TOKEN_EQUAL) || p.check(lexer.TOKEN_NOT_EQUAL) {
		pos := p.currentPos()
		op := p.advance().Literal
		right := p.parseComparison()
		left = &BinaryExpr{Left: left, Op: op, Right: right, Pos_: pos}
	}
	return left
}

func (p *Parser) parseComparison() Expression {
	left := p.parseConcat()
	for p.check(lexer.TOKEN_LESS) || p.check(lexer.TOKEN_LESS_EQUAL) ||
		p.check(lexer.TOKEN_GREATER) || p.check(lexer.TOKEN_GREATER_EQUAL) {
		pos := p.currentPos()
		op := p.advance().Literal
		right := p.parseConcat()
		left = &BinaryExpr{Left: left, Op: op, Right: right, Pos_: pos}
	}
	return left
}

func (p *Parser) parseConcat() Expression {
	left := p.parseAddition()
	for p.check(lexer.TOKEN_DOT_DOT) {
		pos := p.currentPos()
		p.advance()
		right := p.parseAddition()
		left = &StringConcat{Left: left, Right: right, Pos_: pos}
	}
	return left
}

func (p *Parser) parseAddition() Expression {
	left := p.parseMultiplication()
	for p.check(lexer.TOKEN_PLUS) || p.check(lexer.TOKEN_MINUS) {
		pos := p.currentPos()
		op := p.advance().Literal
		right := p.parseMultiplication()
		left = &BinaryExpr{Left: left, Op: op, Right: right, Pos_: pos}
	}
	return left
}

func (p *Parser) parseMultiplication() Expression {
	left := p.parseUnary()
	for p.check(lexer.TOKEN_STAR) || p.check(lexer.TOKEN_SLASH) || p.check(lexer.TOKEN_MODULO) {
		pos := p.currentPos()
		op := p.advance().Literal
		right := p.parseUnary()
		left = &BinaryExpr{Left: left, Op: op, Right: right, Pos_: pos}
	}
	return left
}

func (p *Parser) parseUnary() Expression {
	if p.check(lexer.TOKEN_NOT) || p.check(lexer.TOKEN_MINUS) {
		pos := p.currentPos()
		op := p.advance().Literal
		operand := p.parseUnary()
		return &UnaryExpr{Op: op, Operand: operand, Pos_: pos}
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() Expression {
	expr := p.parseCallAndAccess()

	// Postfix ++ or --
	if p.check(lexer.TOKEN_INCREMENT) || p.check(lexer.TOKEN_DECREMENT) {
		pos := p.currentPos()
		op := p.advance().Literal
		return &PostfixExpr{Operand: expr, Op: op, Pos_: pos}
	}

	return expr
}

func (p *Parser) parseCallAndAccess() Expression {
	expr := p.parsePrimary()

	for {
		if p.check(lexer.TOKEN_LPAREN) {
			// Function call
			expr = p.parseCallExpr(expr)
		} else if p.check(lexer.TOKEN_ARROW) {
			// Property access or method call: $obj->prop or $obj->method()
			pos := p.currentPos()
			p.advance() // consume ->
			propName := ""
			if p.check(lexer.TOKEN_IDENT) {
				propName = p.advance().Literal
			} else if p.isKeywordIdent() {
				// Allow keywords as property names: $this->error, $this->class, etc.
				propName = p.advance().Literal
			} else {
				p.addError("expected property name after '->'")
			}

			if p.check(lexer.TOKEN_LPAREN) {
				// Method call
				args := p.parseArgList()
				expr = &MethodCall{
					Object: expr,
					Method: propName,
					Args:   args,
					Pos_:   pos,
				}
			} else {
				expr = &PropertyAccess{
					Object:   expr,
					Property: propName,
					Pos_:     pos,
				}
			}
		} else if p.check(lexer.TOKEN_DOT) {
			// Module property access or module method call: math.random()
			pos := p.currentPos()
			p.advance() // consume .
			propName := ""
			if p.check(lexer.TOKEN_IDENT) {
				propName = p.advance().Literal
			} else {
				p.addError("expected property name after '.'")
			}

			if p.check(lexer.TOKEN_LPAREN) {
				// Method call mapped as StaticCall for module.method()
				args := p.parseArgList()
				// Unwrap the module name
				moduleName := ""
				if ident, ok := expr.(*Identifier); ok {
					moduleName = ident.Name
				}
				expr = &StaticCall{
					Class:  moduleName,
					Method: propName,
					Args:   args,
					Pos_:   pos,
				}
			} else {
				// Property access mapped dynamically
				expr = &PropertyAccess{
					Object:   expr,
					Property: propName,
					Pos_:     pos,
				}
			}
		} else if p.check(lexer.TOKEN_DOUBLE_COLON) {
			// Static call: Class::method() or Enum::Variant
			pos := p.currentPos()
			p.advance()
			methodName := ""
			if p.check(lexer.TOKEN_IDENT) {
				methodName = p.advance().Literal
			}
			// Unwrap the class name
			className := ""
			if ident, ok := expr.(*Identifier); ok {
				className = ident.Name
			}
			
			// If followed by (, it is a StaticCall
			if p.check(lexer.TOKEN_LPAREN) {
				args := p.parseArgList()
				expr = &StaticCall{
					Class:  className,
					Method: methodName,
					Args:   args,
					Pos_:   pos,
				}
			} else {
				// Otherwise it is an enum access: Enum::Variant
				expr = &EnumAccess{
					Enum:    className,
					Variant: methodName,
					Pos_:    pos,
				}
			}
		} else if p.check(lexer.TOKEN_LBRACKET) {
			// Index: $arr[i]
			pos := p.currentPos()
			p.advance()
			index := p.parseExpression()
			p.expect(lexer.TOKEN_RBRACKET, "expected ']'")
			expr = &IndexExpr{Object: expr, Index: index, Pos_: pos}
		} else {
			break
		}
	}

	return expr
}

func (p *Parser) parseCallExpr(callee Expression) *CallExpr {
	pos := p.currentPos()
	args, namedArgs := p.parseCallArgs()
	return &CallExpr{
		Callee:    callee,
		Args:      args,
		NamedArgs: namedArgs,
		Pos_:      pos,
	}
}

func (p *Parser) parseCallArgs() ([]Expression, []*NamedArg) {
	p.advance() // consume (
	var args []Expression
	var namedArgs []*NamedArg

	for !p.check(lexer.TOKEN_RPAREN) && !p.isAtEnd() {
		// Check for named argument: name: value
		if p.check(lexer.TOKEN_IDENT) && p.peekNext().Type == lexer.TOKEN_COLON {
			pos := p.currentPos()
			name := p.advance().Literal
			p.advance() // consume :
			value := p.parseExpression()
			namedArgs = append(namedArgs, &NamedArg{Name: name, Value: value, Pos_: pos})
		} else {
			args = append(args, p.parseExpression())
		}

		if !p.check(lexer.TOKEN_RPAREN) {
			p.expect(lexer.TOKEN_COMMA, "expected ',' between arguments")
		}
	}

	p.expect(lexer.TOKEN_RPAREN, "expected ')'")
	return args, namedArgs
}

func (p *Parser) parseArgList() []Expression {
	args, _ := p.parseCallArgs()
	return args
}

// ========== Primary Expressions ==========

func (p *Parser) parsePrimary() Expression {
	pos := p.currentPos()

	switch p.peek().Type {

	case lexer.TOKEN_DOLLAR_IDENT:
		tok := p.advance()
		return &DollarIdent{Name: tok.Literal, Pos_: pos}

	case lexer.TOKEN_INT:
		tok := p.advance()
		return &IntLiteral{Value: tok.Literal, Pos_: pos}

	case lexer.TOKEN_FLOAT:
		tok := p.advance()
		return &FloatLiteral{Value: tok.Literal, Pos_: pos}

	case lexer.TOKEN_STRING:
		tok := p.advance()
		return &StringLiteral{Value: tok.Literal, Pos_: pos}

	case lexer.TOKEN_STRING_INTERP_START:
		return p.parseInterpolatedString()

	case lexer.TOKEN_BOOL_TRUE:
		p.advance()
		return &BoolLiteral{Value: true, Pos_: pos}

	case lexer.TOKEN_BOOL_FALSE:
		p.advance()
		return &BoolLiteral{Value: false, Pos_: pos}

	case lexer.TOKEN_NIL:
		p.advance()
		return &NilLiteral{Pos_: pos}

	case lexer.TOKEN_LBRACKET:
		return p.parseArrayLiteral()

	case lexer.TOKEN_LBRACE:
		return p.parseMapLiteral()

	case lexer.TOKEN_LPAREN:
		p.advance()
		expr := p.parseExpression()
		p.expect(lexer.TOKEN_RPAREN, "expected ')'")
		return expr

	case lexer.TOKEN_NEW:
		return p.parseNewExpr()

	case lexer.TOKEN_FUNC:
		return p.parseClosureExpr()

	case lexer.TOKEN_ASYNC:
		return p.parseAsyncExpr()

	case lexer.TOKEN_AWAIT:
		return p.parseAwaitExpr()

	case lexer.TOKEN_CHANNEL:
		return p.parseChannelExpr()

	case lexer.TOKEN_SEND:
		return p.parseSendExpr()

	case lexer.TOKEN_RECEIVE:
		return p.parseReceiveExpr()

	case lexer.TOKEN_ERROR:
		return p.parseErrorExpr()

	case lexer.TOKEN_PRINT:
		return p.parsePrintExpr()

	case lexer.TOKEN_IDENT:
		tok := p.advance()
		// Handle _ as wildcard
		if tok.Literal == "_" {
			return &Wildcard{Pos_: pos}
		}
		return &Identifier{Name: tok.Literal, Pos_: pos}

	default:
		p.addError(fmt.Sprintf("unexpected token: %s (%q)", p.peek().Type, p.peek().Literal))
		p.advance() // skip to avoid infinite loop
		return &NilLiteral{Pos_: pos}
	}
}

// ---- Interpolated String ----

func (p *Parser) parseInterpolatedString() Expression {
	pos := p.currentPos()
	p.advance() // consume STRING_INTERP_START

	var parts []Expression

	for !p.check(lexer.TOKEN_STRING_INTERP_END) && !p.isAtEnd() {
		switch p.peek().Type {
		case lexer.TOKEN_STRING_LITERAL_PART:
			tok := p.advance()
			parts = append(parts, &StringLiteral{Value: tok.Literal, Pos_: tok.Pos})

		case lexer.TOKEN_INTERP_LBRACE:
			p.advance() // consume {$
			// The next token(s) form the interpolation expression (usually $var or $var->prop chain)
			expr := p.parseExpression()
			parts = append(parts, expr)
			p.expect(lexer.TOKEN_RBRACE, "expected '}' to close interpolation")

		case lexer.TOKEN_DOLLAR_LBRACE:
			p.advance() // consume ${
			expr := p.parseExpression()
			parts = append(parts, expr)
			p.expect(lexer.TOKEN_RBRACE, "expected '}' to close expression interpolation")

		default:
			// Unexpected token in interpolated string
			p.advance()
		}
	}

	p.expect(lexer.TOKEN_STRING_INTERP_END, "expected end of interpolated string")

	return &InterpolatedString{Parts: parts, Pos_: pos}
}

// ---- Array literal ----

func (p *Parser) parseArrayLiteral() Expression {
	pos := p.currentPos()
	p.advance() // consume [

	var elements []Expression
	for !p.check(lexer.TOKEN_RBRACKET) && !p.isAtEnd() {
		elements = append(elements, p.parseExpression())
		if !p.check(lexer.TOKEN_RBRACKET) {
			p.expect(lexer.TOKEN_COMMA, "expected ',' between array elements")
		}
	}

	p.expect(lexer.TOKEN_RBRACKET, "expected ']'")
	return &ArrayLiteral{Elements: elements, Pos_: pos}
}

// ---- Map literal ----

func (p *Parser) parseMapLiteral() Expression {
	pos := p.currentPos()
	p.advance() // consume {

	var keys, values []Expression
	for !p.check(lexer.TOKEN_RBRACE) && !p.isAtEnd() {
		key := p.parseExpression()
		p.expect(lexer.TOKEN_COLON, "expected ':' between map key and value")
		value := p.parseExpression()
		keys = append(keys, key)
		values = append(values, value)
		if !p.check(lexer.TOKEN_RBRACE) {
			p.expect(lexer.TOKEN_COMMA, "expected ',' between map entries")
		}
	}

	p.expect(lexer.TOKEN_RBRACE, "expected '}'")
	return &MapLiteral{Keys: keys, Values: values, Pos_: pos}
}

// ---- new expression ----

func (p *Parser) parseNewExpr() Expression {
	pos := p.currentPos()
	p.advance() // consume 'new'

	className := ""
	if p.check(lexer.TOKEN_IDENT) {
		className = p.advance().Literal
	} else {
		p.addError("expected class name after 'new'")
	}

	typeArgs := p.parseTypeParams()

	var args []Expression
	if p.check(lexer.TOKEN_LPAREN) {
		args = p.parseArgList()
	}

	return &NewExpr{ClassName: className, TypeArgs: typeArgs, Args: args, Pos_: pos}
}

// ---- closure expression ----

func (p *Parser) parseClosureExpr() Expression {
	pos := p.currentPos()
	p.advance() // consume 'func'

	params := p.parseParamList()

	// Arrow shorthand
	if p.check(lexer.TOKEN_FAT_ARROW) {
		p.advance()
		expr := p.parseExpression()
		return &ClosureExpr{Params: params, ArrowExpr: expr, Pos_: pos}
	}

	body := p.parseBlock()
	return &ClosureExpr{Params: params, Body: body, Pos_: pos}
}

// ---- async expression ----

func (p *Parser) parseAsyncExpr() Expression {
	pos := p.currentPos()
	p.advance() // consume 'async'
	body := p.parseBlock()
	return &AsyncExpr{Body: body, Pos_: pos}
}

// ---- await expression ----

func (p *Parser) parseAwaitExpr() Expression {
	pos := p.currentPos()
	p.advance() // consume 'await'
	expr := p.parseExpression()
	return &AwaitExpr{Expr: expr, Pos_: pos}
}

// ---- channel expression ----

func (p *Parser) parseChannelExpr() Expression {
	pos := p.currentPos()
	p.advance() // consume 'channel'

	elemType := ""
	if p.check(lexer.TOKEN_LPAREN) {
		p.advance()
		if p.check(lexer.TOKEN_IDENT) {
			elemType = p.advance().Literal
		}
		p.expect(lexer.TOKEN_RPAREN, "expected ')' after channel type")
	}

	return &ChannelExpr{ElemType: elemType, Pos_: pos}
}

// ---- send expression ----

func (p *Parser) parseSendExpr() Expression {
	pos := p.currentPos()
	p.advance() // consume 'send'
	p.expect(lexer.TOKEN_LPAREN, "expected '(' after 'send'")
	ch := p.parseExpression()
	p.expect(lexer.TOKEN_COMMA, "expected ',' in send(channel, value)")
	val := p.parseExpression()
	p.expect(lexer.TOKEN_RPAREN, "expected ')'")
	return &SendExpr{Channel: ch, Value: val, Pos_: pos}
}

// ---- receive expression ----

func (p *Parser) parseReceiveExpr() Expression {
	pos := p.currentPos()
	p.advance() // consume 'receive'
	p.expect(lexer.TOKEN_LPAREN, "expected '(' after 'receive'")
	ch := p.parseExpression()
	p.expect(lexer.TOKEN_RPAREN, "expected ')'")
	return &ReceiveExpr{Channel: ch, Pos_: pos}
}

// ---- error expression ----

func (p *Parser) parseErrorExpr() Expression {
	pos := p.currentPos()
	p.advance() // consume 'error'
	p.expect(lexer.TOKEN_LPAREN, "expected '(' after 'error'")
	msg := p.parseExpression()
	p.expect(lexer.TOKEN_RPAREN, "expected ')'")
	return &ErrorExpr{Message: msg, Pos_: pos}
}

// ---- print expression ----

func (p *Parser) parsePrintExpr() Expression {
	pos := p.currentPos()
	p.advance() // consume 'print'
	p.expect(lexer.TOKEN_LPAREN, "expected '(' after 'print'")
	arg := p.parseExpression()
	p.expect(lexer.TOKEN_RPAREN, "expected ')'")
	return &PrintExpr{Arg: arg, Pos_: pos}
}

// ========== Token Helpers ==========

func (p *Parser) peek() lexer.Token {
	if p.current >= len(p.tokens) {
		return lexer.Token{Type: lexer.TOKEN_EOF}
	}
	return p.tokens[p.current]
}

func (p *Parser) peekNext() lexer.Token {
	next := p.current + 1
	if next >= len(p.tokens) {
		return lexer.Token{Type: lexer.TOKEN_EOF}
	}
	return p.tokens[next]
}

func (p *Parser) advance() lexer.Token {
	tok := p.peek()
	if !p.isAtEnd() {
		p.current++
	}
	return tok
}

func (p *Parser) check(t lexer.TokenType) bool {
	return p.peek().Type == t
}

func (p *Parser) expect(t lexer.TokenType, msg string) lexer.Token {
	if p.check(t) {
		return p.advance()
	}
	p.addError(fmt.Sprintf("%s, got %s (%q)", msg, p.peek().Type, p.peek().Literal))
	return p.peek()
}

func (p *Parser) isAtEnd() bool {
	return p.peek().Type == lexer.TOKEN_EOF
}

// isKeywordIdent returns true if the current token is a keyword that can
// also be used as an identifier in property/method positions (after ->).
// This mirrors PHP where $this->error, $this->class, etc. are valid.
func (p *Parser) isKeywordIdent() bool {
	switch p.peek().Type {
	case lexer.TOKEN_ERROR, lexer.TOKEN_CLASS, lexer.TOKEN_NEW,
		lexer.TOKEN_MATCH, lexer.TOKEN_PRINT,
		lexer.TOKEN_TRY, lexer.TOKEN_CATCH, lexer.TOKEN_IMPORT:
		return true
	}
	return false
}

func (p *Parser) currentPos() lexer.Position {
	return p.peek().Pos
}

func (p *Parser) addError(msg string) {
	pos := p.currentPos()
	p.errors = append(p.errors, &lexer.WolfError{
		Pos:     pos,
		Message: msg,
		Phase:   "parser",
	})
}
