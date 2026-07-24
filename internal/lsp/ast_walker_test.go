package lsp

import (
	"testing"

	"wolf/internal/lexer"
	"wolf/internal/parser"
)

func TestFindNodeAtPosition(t *testing.T) {
	// Let's create a manual AST structure representing:
	// 1: func main() {
	// 2:    $myVar = 42
	// 3: }
	//
	// We want to query Line 2, Col 7 (the 'V' in $myVar)

	ident := &parser.DollarIdent{
		Name: "$myVar",
		Pos_: lexer.Position{Line: 2, Col: 4}, // starts at col 4
	}

	assign := &parser.AssignStmt{
		Target: ident,
		Value: &parser.IntLiteral{
			Value: "42",
			Pos_:  lexer.Position{Line: 2, Col: 13},
		},
		Pos_: lexer.Position{Line: 2, Col: 4},
	}

	funcDecl := &parser.FuncDecl{
		Name: "main",
		Body: &parser.BlockStmt{
			Statements: []parser.Statement{assign},
			Pos_:       lexer.Position{Line: 1, Col: 13},
		},
		Pos_: lexer.Position{Line: 1, Col: 1},
	}

	program := &parser.Program{
		Statements: []parser.Statement{funcDecl},
		Pos_:       lexer.Position{Line: 1, Col: 1},
	}

	// Query line 2, col 7
	node := FindNodeAtPosition(program, 2, 7)

	if node == nil {
		t.Fatalf("Expected a node, got nil")
	}

	foundIdent, ok := node.(*parser.DollarIdent)
	if !ok {
		t.Fatalf("Expected a DollarIdent, got %T", node)
	}

	if foundIdent.Name != "$myVar" {
		t.Errorf("Expected $myVar, got %s", foundIdent.Name)
	}
}
