import re

with open("src/compiler/AST.wolf", "r") as f:
    content = f.read()

new_class = """class ImportStmt extends ASTNode {
    public $path = ""
    public $alias = ""

    public func constructor() {
        $this->type = "ImportStmt"
    }
}

"""

if "class ImportStmt" not in content:
    content = content.replace("class ExpressionStmt extends ASTNode {", new_class + "class ExpressionStmt extends ASTNode {")
    with open("src/compiler/AST.wolf", "w") as f:
        f.write(content)
    print("Added ImportStmt")
else:
    print("Already exists")
