# 🐺 Wolf Language

![Wolf Language](https://img.shields.io/badge/Language-Wolf-blue.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![Version](https://img.shields.io/badge/version-v0.1.0--dev-orange)

Wolf is a compiled, statically-typed (with dynamic fallback) programming language designed specifically for **AI-native applications**. 

Built entirely in Go, Wolf compiles down to native binaries. It combines the speed, strictness, and concurrency of Go with an elegant, expressive syntax, and introduces the **Machine Learning Bridge (`@ml`)** — allowing you to write embedded Python within your native code with zero FFI boilerplate.

---

## 🚀 Features

- **No FFI Setup:** Write Python inside Go natively via the `@ml` block.
- **Auto-Environment:** Automatic Python virtual environment creation and pip dependency management.
- **Compiled to Go:** Transforms Wolf Intermediate Representation (WIR) into blazing-fast Go standard code.
- **Built-in HTTP:** First-class syntax for HTTP servers (`route()`, `serve()`, `respond()`).
- **TraversyMVC Core:** Built-in SQL and Redis wrappers utilizing named parameters and clean PDO-like patterns.
- **Object-Oriented:** Classes, properties, constructors, and methods backed by Go structs.

## ⚡ Quick Look

```wolf
# Pure Wolf Syntax
func handle_prediction() {
    $text = "The neural net learns quickly."

    # Native Python execution inside Wolf!
    @ml in: [$text] out: [$score] {
        from textblob import TextBlob
        score = TextBlob(text).sentiment.polarity
    }

    respond(200, {"sentiment": $score})
}

route("GET", "/predict", handle_prediction)
serve(8080)
```

## 📦 Installation

Wolf is currently distributed via source. Requirements: **Go 1.21+** and **Python 3.10+**.

```bash
git clone https://github.com/wolflang/wolf.git
cd wolf
go build -o wolf ./cmd/wolf
sudo cp wolf /usr/local/bin/wolf
```

## 📚 Documentation

1. [Getting Started](docs/getting_started.md)
2. [Syntax Reference](docs/syntax.md)
3. [The ML Bridge (`@ml`)](docs/ml_bridge.md)
4. [Database & Redis](docs/database.md)
5. [Standard Library](docs/stdlib.md)

## 🛠 Usage

Create a new application skeleton:
```bash
wolf new my-ai-app
cd my-ai-app
```

Run a Wolf script directly:
```bash
wolf run src/main.wolf
```

Compile a Wolf script to a native binary:
```bash
wolf build src/main.wolf
./wolf_out/my_app
```

## 🧪 Testing

Wolf's compiler has 379 passing tests with 0 data races.
```bash
make test
```

## 📄 License
MIT License. See [LICENSE](LICENSE) for details.
