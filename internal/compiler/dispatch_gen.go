package compiler

import (
	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
)

// generateDispatcherAST looks at all classes in the parsed program and creates a
// global Wolf function:
//
//	func __compiler_dispatch_controller($c: string, $m: string, $args: array, $req_id: int, $res_id: int) -> bool {
//	    if $c == "Users" {
//	        wolf_set_current_context($req_id, $res_id)
//	        $ctrl = new UsersController()
//	        if $m == "index" { $ctrl->index(); return true }
//	    }
//	    return false
//	}
//
// Controller methods take NO parameters — they access the request via
// thread-local context (wolf_get_request_body, wolf_get_request_header, etc.)
// This matches PHP's pattern where controller methods read from $_POST/$_SERVER globals.
func generateDispatcherAST(program *parser.Program) parser.Statement {
	// Find all classes that end with "Controller"
	var controllers []string
	for _, stmt := range program.Statements {
		if classDecl, ok := stmt.(*parser.ClassDecl); ok {
			if len(classDecl.Name) > 10 && classDecl.Name[len(classDecl.Name)-10:] == "Controller" {
				controllers = append(controllers, classDecl.Name)
			}
		}
	}

	// We generate the dispatcher as raw Wolf source code and parse it into an AST,
	// because building it AST node by AST node manually would be hundreds of lines.
	src := "func __compiler_dispatch_controller($c: string, $m: string, $args: array, $req: ptr, $res: ptr) -> bool {\n"

	if len(controllers) > 0 {
		for _, cName := range controllers {
			// e.g. "UsersController" -> "Users"
			baseName := cName[:len(cName)-10]

			src += "    if $c == \"" + baseName + "\" {\n"
			// Set thread-local context BEFORE instantiating controller
			// This way Controller.__construct() can also access the request if needed
			src += "        wolf_set_current_context($req, $res)\n"
			src += "        $ctrl = new " + cName + "()\n"

			// Find methods on this controller
			for _, stmt := range program.Statements {
				if classDecl, ok := stmt.(*parser.ClassDecl); ok && classDecl.Name == cName {
					for _, method := range classDecl.Methods {
						// skip constructor and inherited base class methods
						if method.Name == "__construct" {
							continue
						}

						// Controller methods take NO args from the dispatcher.
						// They access request data via thread-local context.
						src += "        if $m == \"" + method.Name + "\" {\n"
						src += "            $ctrl->" + method.Name + "()\n"
						src += "            return true\n"
						src += "        }\n"
					}
				}
			}
			src += "    }\n"
		}
	}

	src += "    return false\n"
	src += "}\n"

	// Parse this generated source
	l := lexer.New(src, "compiler_generated_dispatcher.wolf")
	tokens, _ := l.Tokenize()
	p := parser.New(tokens, "compiler_generated_dispatcher.wolf")
	ast, _ := p.Parse()

	return ast.Statements[0]
}

// generateModelFactoryAST finds all classes that don't end in "Controller" (which are assumed to be models)
// and generates a factory function to instantiate them by string name.
func generateModelFactoryAST(program *parser.Program) parser.Statement {
	var models []string
	for _, stmt := range program.Statements {
		if classDecl, ok := stmt.(*parser.ClassDecl); ok {
			// If it's not a Controller, assume it's a Model for now
			if len(classDecl.Name) < 10 || classDecl.Name[len(classDecl.Name)-10:] != "Controller" {
				if classDecl.Name != "Controller" && classDecl.Name != "Database" && classDecl.Name != "Request" && classDecl.Name != "Response" {
					models = append(models, classDecl.Name)
				}
			}
		}
	}

	src := "func __compiler_create_model($name: string) -> ptr {\n"

	if len(models) > 0 {
		for _, mName := range models {
			src += "    if $name == \"" + mName + "\" {\n"
			src += "        return new " + mName + "()\n"
			src += "    }\n"
		}
	}

	src += "    return nil\n"
	src += "}\n"

	// Parse this generated source
	l := lexer.New(src, "compiler_generated_factory.wolf")
	tokens, _ := l.Tokenize()
	p := parser.New(tokens, "compiler_generated_factory.wolf")
	ast, _ := p.Parse()

	return ast.Statements[0]
}
