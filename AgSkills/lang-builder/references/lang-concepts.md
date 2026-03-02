# Language Design Concepts Reference

## Table of Contents
1. [PRD Sections to Prompt For](#prd-sections-to-prompt-for)
2. [Paradigm Patterns](#paradigm-patterns)
3. [Grammar Formats](#grammar-formats)
4. [Implementation Patterns by Host Language](#implementation-patterns-by-host-language)
5. [Type System Cheat Sheet](#type-system-cheat-sheet)
6. [Common Pitfalls](#common-pitfalls)

---

## PRD Sections to Prompt For

If a PRD is missing, collect these from the user before Phase 1:

| Section | Key Questions |
|---------|--------------|
| **Goal** | What problem does this language solve? What can't existing languages do? |
| **Target Users** | Who writes in this language? Programmers, data scientists, domain experts? |
| **Paradigm** | Functional, OOP, procedural, logic, reactive, or mixed? |
| **Syntax Style** | C-like, Python-like, Lisp-like, custom? Significant whitespace? |
| **Type System** | Static, dynamic, gradual, inferred? Structural or nominal? |
| **Runtime Target** | Interpreted, compiled to native, transpiled to JS/Python/C, bytecode VM? |
| **Standard Library** | What built-ins are mandatory for MVP? I/O, collections, math, concurrency? |
| **Interop** | Must call C/Python/JS? FFI required? |
| **Non-Goals** | What is explicitly out of scope for v1? |
| **Example Programs** | At least 3 programs the language must be able to run |

---

## Paradigm Patterns

### Functional
- First-class functions, closures, immutable-by-default
- Key features: `map`, `filter`, `fold`, pattern matching, algebraic data types
- Grammar tip: expressions return values; statements are just unit-returning expressions
- Reference languages: Haskell, Elm, OCaml, Elixir

### Object-Oriented
- Classes/traits/interfaces, encapsulation, inheritance or mixins
- Key features: method dispatch, constructors, `self`/`this`
- Grammar tip: distinguish between method call syntax and function call syntax early
- Reference languages: Python, Ruby, Java, Swift

### Procedural / Scripting
- Statements, mutable state, imperative control flow
- Key features: loops, conditionals, functions as procedures
- Grammar tip: statement vs. expression distinction is critical
- Reference languages: Lua, Bash, C, Go

### DSL (Domain-Specific)
- Narrow expressiveness; optimize for the domain's vocabulary
- Key features: few keywords, declarative style, minimal boilerplate
- Grammar tip: consider operator overloading or special syntax extensions
- Examples: SQL, CSS, Make, HCL (Terraform), YAML-like config languages

---

## Grammar Formats

### EBNF (Extended Backus-Naur Form)
Best for: documentation, human readability, specs
```ebnf
program     = statement* EOF ;
statement   = expr_stmt | if_stmt | while_stmt | return_stmt ;
expr_stmt   = expression ";" ;
if_stmt     = "if" "(" expression ")" block ( "else" block )? ;
expression  = assignment ;
assignment  = IDENTIFIER "=" assignment | equality ;
equality    = comparison ( ("==" | "!=") comparison )* ;
```

### ANTLR4 (.g4)
Best for: generating lexers/parsers automatically
```antlr
grammar MyLang;

program     : statement* EOF ;
statement   : exprStmt | ifStmt ;
exprStmt    : expression SEMI ;
ifStmt      : IF LPAREN expression RPAREN block (ELSE block)? ;

IF          : 'if' ;
ELSE        : 'else' ;
IDENTIFIER  : [a-zA-Z_][a-zA-Z_0-9]* ;
NUMBER      : [0-9]+ ('.' [0-9]+)? ;
WS          : [ \t\r\n]+ -> skip ;
```

### PEG (Parsing Expression Grammar)
Best for: unambiguous grammars, recursive descent parsers
```peg
program  <- statement* !.
statement <- ifStmt / exprStmt
ifStmt   <- "if" "(" expr ")" block ("else" block)?
expr     <- term (("+"/"-") term)*
```

**Recommendation**: Use ANTLR4 `.g4` if the user wants auto-generated parsers; use EBNF for spec documentation; write PEG when hand-rolling a recursive descent parser.

---

## Implementation Patterns by Host Language

### Python
- **Lexer**: Use `re` module with a token regex master pattern; `re.Scanner` or manual loop
- **Parser**: Hand-written recursive descent (simple) or `lark-parser` / `PLY` (complex)
- **AST**: `dataclasses` or `namedtuple`; implement `__repr__` for debugging
- **Evaluator**: Tree-walk interpreter with an `Environment` dict for scopes
- **Starter structure**:
  ```
  mylang/
  ├── lexer.py       # Token types + Lexer class
  ├── parser.py      # Recursive descent Parser
  ├── ast_nodes.py   # AST node dataclasses
  ├── interpreter.py # Tree-walk Interpreter
  ├── environment.py # Scope / variable store
  └── main.py        # REPL entry point
  ```

### Rust
- **Lexer**: Manual `Lexer` struct iterating `&str` with `chars().peekable()`
- **Parser**: Recursive descent with `Result<T, ParseError>` returns
- **AST**: `enum` with variants per node type; derive `Debug`
- **Evaluator**: `match` on AST enum variants; `HashMap<String, Value>` for env
- Crates: `logos` (lexer gen), `pest` (PEG parser gen), `inkwell` (LLVM bindings)

### Go
- **Lexer**: `text/scanner` for simple cases; manual rune iteration for control
- **Parser**: Recursive descent; return `(Node, error)` pairs
- **AST**: Interfaces + structs; `Eval(env *Env) (Value, error)` method pattern
- Packages: `github.com/alecthomas/participle` for grammar-driven parsers

### TypeScript / JavaScript
- **Lexer**: Regex-heavy; use generator functions for lazy token streams
- **Parser**: Recursive descent or `nearley` / `pegjs`
- **AST**: TypeScript discriminated unions for node types
- **Evaluator**: Visitor pattern with a `visit(node: ASTNode): Value` dispatcher

---

## Type System Cheat Sheet

| System | Complexity | Key Feature | Example Languages |
|--------|-----------|-------------|-------------------|
| Dynamic | Low | Types checked at runtime | Python, Lua, JavaScript |
| Static Manifest | Medium | Types declared explicitly | Java, C, Go |
| Static Inferred | High | Types inferred by compiler | Rust, Haskell, OCaml, Swift |
| Gradual | Medium | Mix of static + dynamic | TypeScript, Python w/ hints |
| Dependent | Very High | Types depend on values | Idris, Agda |

**For MVP**: Start with **dynamic** typing (easiest) or **static manifest** (most teachable). Add inference as v2.

---

## Common Pitfalls

| Pitfall | Symptom | Fix |
|---------|---------|-----|
| Ambiguous grammar | Parser conflicts, unexpected parses | Rewrite using PEG or add precedence rules |
| Ignoring precedence early | `2+3*4` → wrong result | Model arithmetic precedence in grammar from day 1 |
| Mutable AST nodes | Hard-to-trace bugs in interpreter | Make AST nodes immutable/frozen |
| Single global scope | `let` inside `if` leaks out | Implement `Environment` with parent chain |
| No error recovery | First error crashes whole parse | Add `synchronize()` method that skips to next statement |
| Forgetting EOF token | Parser infinite loops | Always emit an `EOF` token at end of token stream |
| String interning skipped | Slow identifier lookup | Intern identifier strings in symbol table |
