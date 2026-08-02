package lsp

import (
	"strings"

	"wolf/internal/compiler"
	"wolf/internal/parser"
	"wolf/internal/resolver"
)

type Handler struct {
	vfs      map[string]string
	program  *parser.Program
	resolver *resolver.Resolver
}

func NewHandler() *Handler {
	return &Handler{
		vfs: make(map[string]string),
	}
}

func (h *Handler) Initialize(params InitializeParams) InitializeResult {
	return InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:       TextDocumentSyncKindFull,
			HoverProvider:          true,
			DefinitionProvider:     true,
			DocumentSymbolProvider: true,
			CompletionProvider: &CompletionOptions{
				ResolveProvider:   false,
				TriggerCharacters: []string{".", ":", ">", "$"},
			},
		},
	}
}

func (h *Handler) DidOpen(params DidOpenTextDocumentParams) {
	uri := params.TextDocument.URI
	text := params.TextDocument.Text
	path := uriToPath(uri)
	h.vfs[path] = text
	h.checkFile(path, text)
}

func (h *Handler) DidChange(params DidChangeTextDocumentParams) {
	uri := params.TextDocument.URI
	if len(params.ContentChanges) > 0 {
		text := params.ContentChanges[0].Text
		path := uriToPath(uri)
		h.vfs[path] = text
		h.checkFile(path, text)
	}
}

func (h *Handler) checkFile(path, text string) {
	// Run the compiler up to type checking
	c := compiler.New()
	c.VFS = h.vfs

	res, _ := c.Check(text, path)
	if res.Program != nil {
		h.program = res.Program
	}
	if res.Resolver != nil {
		h.resolver = res.Resolver
	}

	// Map lexer.WolfError to LSP Diagnostics
	var diagnostics []Diagnostic
	for _, e := range res.Diagnostics {
		severity := DiagnosticSeverityError
		if e.IsWarning {
			severity = DiagnosticSeverityWarning
		}

		// WolfError lines are 1-indexed, cols are 1-indexed
		// LSP lines are 0-indexed, cols are 0-indexed
		line := e.Pos.Line - 1
		if line < 0 {
			line = 0
		}
		col := e.Pos.Col - 1
		if col < 0 {
			col = 0
		}

		diag := Diagnostic{
			Range: Range{
				Start: Position{Line: line, Character: col},
				End:   Position{Line: line, Character: col + 1}, // point to the character
			},
			Severity: severity,
			Source:   "wolf",
			Message:  e.Message,
		}
		diagnostics = append(diagnostics, diag)
	}

	// Always send diagnostics, even if empty (to clear them)
	params := PublishDiagnosticsParams{
		URI:         pathToURI(path),
		Diagnostics: diagnostics,
	}
	SendNotification("textDocument/publishDiagnostics", params)
}

func (h *Handler) Hover(params HoverParams) *Hover {
	if h.program == nil {
		return nil
	}

	line := params.Position.Line + 1
	col := params.Position.Character + 1

	node := FindNodeAtPosition(h.program, line, col)
	if node == nil {
		return nil
	}

	var name string
	var typeName string = "unknown"

	if expr, ok := node.(parser.Expression); ok {
		name = ExprName(expr)
		switch e := expr.(type) {
		case *parser.IntLiteral:
			typeName = "int"
		case *parser.StringLiteral:
			typeName = "string"
		case *parser.FloatLiteral:
			typeName = "float"
		case *parser.BoolLiteral:
			typeName = "bool"
		case *parser.DollarIdent:
			typeName = "variable"
		case *parser.Identifier:
			typeName = "identifier"
		case *parser.PropertyAccess:
			typeName = "property"
		case *parser.MethodCall:
			name = "func " + e.Method
			typeName = "method"
		case *parser.CallExpr:
			if ident, ok := e.Callee.(*parser.Identifier); ok {
				name = "func " + ident.Name
			}
			typeName = "function call"
		}
	} else if fd, ok := node.(*parser.FuncDecl); ok {
		name = "func " + fd.Name
		typeName = "func"
	} else if cd, ok := node.(*parser.ClassDecl); ok {
		name = "class " + cd.Name
		typeName = "class"
	} else if vd, ok := node.(*parser.VarDecl); ok {
		name = "var " + vd.Name
		typeName = "variable"
	}

	if name == "" {
		return nil
	}

	markdown := "```wolf\n" + name + ": " + typeName + "\n```"
	if h.resolver != nil {
		if resolvedGo, ok := h.resolver.ResolvedNames()[name]; ok {
			markdown += "\n*Resolved to Go variable: `" + resolvedGo + "`*"
		}
	}

	return &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: markdown,
		},
	}
}

func (h *Handler) Definition(params DefinitionParams) *Location {
	if h.program == nil {
		return nil
	}

	line := params.Position.Line + 1
	col := params.Position.Character + 1

	node := FindNodeAtPosition(h.program, line, col)
	if node == nil {
		return nil
	}

	name := ""
	if expr, ok := node.(parser.Expression); ok {
		name = ExprName(expr)
	}

	if name == "" {
		return nil
	}

	// Simple definition search across top-level statements
	for _, stmt := range h.program.Statements {
		if fd, ok := stmt.(*parser.FuncDecl); ok && fd.Name == name {
			return &Location{
				URI: params.TextDocument.URI,
				Range: Range{
					Start: ConvertPos(fd.Pos()),
					End:   ConvertPos(fd.Pos()),
				},
			}
		}
		if cd, ok := stmt.(*parser.ClassDecl); ok && cd.Name == name {
			return &Location{
				URI: params.TextDocument.URI,
				Range: Range{
					Start: ConvertPos(cd.Pos()),
					End:   ConvertPos(cd.Pos()),
				},
			}
		}
		if vd, ok := stmt.(*parser.VarDecl); ok && vd.Name == name {
			return &Location{
				URI: params.TextDocument.URI,
				Range: Range{
					Start: ConvertPos(vd.Pos()),
					End:   ConvertPos(vd.Pos()),
				},
			}
		}
	}

	return nil
}

func (h *Handler) Completion(params CompletionParams) *CompletionList {
	items := []CompletionItem{
		{Label: "func", Kind: CompletionItemKindKeyword, Detail: "function declaration"},
		{Label: "class", Kind: CompletionItemKindKeyword, Detail: "class declaration"},
		{Label: "if", Kind: CompletionItemKindKeyword},
		{Label: "else", Kind: CompletionItemKindKeyword},
		{Label: "return", Kind: CompletionItemKindKeyword},
		{Label: "match", Kind: CompletionItemKindKeyword},
		{Label: "for", Kind: CompletionItemKindKeyword},
		{Label: "foreach", Kind: CompletionItemKindKeyword},
		{Label: "while", Kind: CompletionItemKindKeyword},
		{Label: "async", Kind: CompletionItemKindKeyword},
		{Label: "await", Kind: CompletionItemKindKeyword},
		{Label: "spawn", Kind: CompletionItemKindKeyword},
		{Label: "parallel", Kind: CompletionItemKindKeyword},
		{Label: "import", Kind: CompletionItemKindKeyword},
		{Label: "namespace", Kind: CompletionItemKindKeyword},
		{Label: "var", Kind: CompletionItemKindKeyword},
		{Label: "print", Kind: CompletionItemKindKeyword},
		{Label: "try", Kind: CompletionItemKindKeyword},
		{Label: "catch", Kind: CompletionItemKindKeyword},
	}

	if h.program != nil {
		for _, stmt := range h.program.Statements {
			if fd, ok := stmt.(*parser.FuncDecl); ok && fd.Name != "" {
				items = append(items, CompletionItem{
					Label: fd.Name,
					Kind:  CompletionItemKindFunction,
				})
			}
			if cd, ok := stmt.(*parser.ClassDecl); ok && cd.Name != "" {
				items = append(items, CompletionItem{
					Label: cd.Name,
					Kind:  CompletionItemKindClass,
				})
			}
			if vd, ok := stmt.(*parser.VarDecl); ok && vd.Name != "" {
				items = append(items, CompletionItem{
					Label: vd.Name,
					Kind:  CompletionItemKindVariable,
				})
			}
		}
	}

	// Include known variables from resolver
	if h.resolver != nil {
		for name := range h.resolver.ResolvedNames() {
			if name != "" && name[0] == '$' {
				items = append(items, CompletionItem{
					Label: name,
					Kind:  CompletionItemKindVariable,
				})
			}
		}
	}

	return &CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}

func uriToPath(uri string) string {
	return strings.TrimPrefix(uri, "file://")
}

func pathToURI(path string) string {
	return "file://" + path
}

func (h *Handler) DocumentSymbol(params DocumentSymbolParams) []DocumentSymbol {
	if h.program == nil {
		return nil
	}

	var symbols []DocumentSymbol

	for _, stmt := range h.program.Statements {
		if fd, ok := stmt.(*parser.FuncDecl); ok && fd.Name != "" {
			symbols = append(symbols, DocumentSymbol{
				Name:           fd.Name,
				Kind:           SymbolKindFunction,
				Range:          Range{Start: ConvertPos(fd.Pos()), End: ConvertPos(fd.Pos())},
				SelectionRange: Range{Start: ConvertPos(fd.Pos()), End: ConvertPos(fd.Pos())},
			})
		} else if cd, ok := stmt.(*parser.ClassDecl); ok && cd.Name != "" {
			var children []DocumentSymbol
			for _, m := range cd.Methods {
				children = append(children, DocumentSymbol{
					Name:           m.Name,
					Kind:           SymbolKindMethod,
					Range:          Range{Start: ConvertPos(m.Pos()), End: ConvertPos(m.Pos())},
					SelectionRange: Range{Start: ConvertPos(m.Pos()), End: ConvertPos(m.Pos())},
				})
			}
			symbols = append(symbols, DocumentSymbol{
				Name:           cd.Name,
				Kind:           SymbolKindClass,
				Range:          Range{Start: ConvertPos(cd.Pos()), End: ConvertPos(cd.Pos())},
				SelectionRange: Range{Start: ConvertPos(cd.Pos()), End: ConvertPos(cd.Pos())},
				Children:       children,
			})
		} else if vd, ok := stmt.(*parser.VarDecl); ok && vd.Name != "" {
			symbols = append(symbols, DocumentSymbol{
				Name:           vd.Name,
				Kind:           SymbolKindVariable,
				Range:          Range{Start: ConvertPos(vd.Pos()), End: ConvertPos(vd.Pos())},
				SelectionRange: Range{Start: ConvertPos(vd.Pos()), End: ConvertPos(vd.Pos())},
			})
		}
	}

	return symbols
}
