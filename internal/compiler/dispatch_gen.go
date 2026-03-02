package compiler

import (
	"github.com/wolflang/wolf/internal/lexer"
	"github.com/wolflang/wolf/internal/parser"
)

// generateDispatcherAST looks at all classes in the parsed program and creates a
// global Wolf function:
//
// func __compiler_dispatch_controller($c: string, $m: string, $args: array, $req: Request, $res: Response) -> bool {
//     if $c == "Users" {
//         $ctrl = new UsersController()
//         if $m == "view" { $ctrl->view($args, $req, $res); return true }
//     }
//     return false
// }
//
// This bridges the dynamic request URLs (strings) to static compile-time LLVM class types.
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
	src := "func __compiler_dispatch_controller($c, $m, $args, $req, $res) -> bool {\n"
	
	if len(controllers) > 0 {
		for _, cName := range controllers {
			// e.g. "UsersController" -> "Users"
			baseName := cName[:len(cName)-10]
			
			src += "    if $c == \"" + baseName + "\" {\n"
			src += "        $ctrl = new " + cName + "()\n"
			
			// Find methods on this controller
			for _, stmt := range program.Statements {
				if classDecl, ok := stmt.(*parser.ClassDecl); ok && classDecl.Name == cName {
					for _, method := range classDecl.Methods {
						// skip constructor
						if method.Name == "__construct" { continue }
						
						src += "        if $m == \"" + method.Name + "\" {\n"
						src += "            $ctrl->" + method.Name + "($args, $req, $res)\n"
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
