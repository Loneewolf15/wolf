# Getting Started with Wolf 🐺

Wolf is a compiled programming language designed specifically for AI-native applications. It combines the speed and concurrency of Go with native syntax for embedded Python and Machine Learning workloads.

## Installation

Currently, Wolf is built directly from source using Go.

```bash
# Clone the repository
git clone https://github.com/wolflang/wolf.git
cd wolf

# Build the CLI
go build -o wolf ./cmd/wolf

# Make it globally available
sudo cp wolf /usr/local/bin/wolf
```

## Creating Your First Project

Use the `wolf new` scaffolding command to create a standard project structure:

```bash
wolf new my-app
cd my-app
```

This generates `src/main.wolf`, `config/wolf.config.json`, and `.gitignore`.

## Hello World

Create a file named `hello.wolf`:

```wolf
func main() {
    $name = "World"
    print("Hello, {$name}!")
}

main()
```

Run it:
```bash
$ wolf run hello.wolf
Hello, World!
```

## Your First HTTP Server

Wolf has built-in primitives for HTTP servers (`route()`, `serve()`, `respond()`).

```wolf
func handle_health() {
    respond(200, {"status": "healthy", "language": "Wolf"})
}

route("GET", "/health", handle_health)

print("Starting server on :8080")
serve(8080)
```

Run it, then test with curl:
```bash
$ curl http://localhost:8080/health
{"language":"Wolf","status":"healthy"}
```

## Your First Machine Learning Bridge (`@ml`)

Wolf allows you to write Python directly inside your code using the `@ml` block. Variables are automatically serialized and passed back and forth!

```wolf
# Define a Wolf variable
$name = "Wolf Developer"

# Process it in Python natively
@ml in: [$name] out: [$greeting] {
    greeting = f"Greeting from Python natively, {name}!"
}

# Print the result back in Wolf
print($greeting)
```

Wolf handles the background Python process and inter-process communication via JSON-RPC seamlessly. No FFI boilerplate required.
