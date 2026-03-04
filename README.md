# 🐺 Wolf Language

![Wolf Language](https://img.shields.io/badge/Language-Wolf-blue.svg)
![Build Status](https://github.com/Loneewolf15/wolf/actions/workflows/ci.yml/badge.svg)
![Version](https://img.shields.io/badge/version-v0.1.0--dev-orange)
![License](https://img.shields.io/badge/license-MIT-green)

**Wolf** is a natively compiled backend programming language that compiles to LLVM IR. It gives you **PHP familiarity** — `$variables`, `$this->method()`, curly braces — with performance **faster than Go** because there's no garbage collector and LLVM handles optimization.

> **The goal:** A Wolf developer should never need to install a package for common backend work. Everything they reach for is already there.

---

## ⚡ Quick Look

```wolf
class Auth extends Controller {
    $userModel

    func __construct() {
        $this->userModel = $this->model("User")
    }

    func sendResponse($status, $message, $data, $httpCode: int) {
        http_response_code($httpCode)
        echo json_encode({
            "status": $status,
            "message": $message,
            "data": $data
        })
    }

    func login() {
        $postData = $this->getData()
        $email    = $postData["email"]
        $password = $postData["password"]

        if !is_email($email) {
            $this->sendResponse(false, "Invalid email", {}, 400)
            return
        }

        $user = $this->userModel->findByEmail($email)
        if password_verify($password, $user->password_hash) {
            $token = uuid_v4()
            $this->sendResponse(true, "Login successful", {
                "token": $token,
                "user":  $user
            }, 200)
        } else {
            $this->sendResponse(false, "Invalid credentials", {}, 401)
        }
    }
}
```

---

## 🚀 Features

| Feature | Description |
|---|---|
| **PHP-familiar syntax** | `$variables`, `$this->`, `{}`, `foreach as`, `match` — no new paradigms to learn |
| **LLVM native compilation** | Compiles to `.ll` → native binary via `llc` + `clang`. No runtime, no GC |
| **110+ built-in functions** | Strings, arrays, math, crypto, validation, file I/O, date/time — all included |
| **Zero-config routing** | Drop a controller in `controllers/`, it's auto-discovered |
| **Built-in database** | `new Database` for MySQL/PostgreSQL. `new Redis` for caching. No facades |
| **Scaffolding CLI** | `wolf generate controller Users` creates production-ready boilerplate |
| **@ml Bridge** | Write Python inside `@ml {}` blocks — Wolf manages the venv and exchanges data via CPython C API |

---

## 📦 Installation

Requirements: **Go 1.21+**, **LLVM/Clang 15+**, **Python 3.10+** (for `@ml` bridge only).

```bash
git clone https://github.com/Loneewolf15/wolf.git
cd wolf
make build
sudo cp wolf /usr/local/bin/wolf
```

---

## 🛠 Usage

```bash
# Create a new project
wolf new my-api
cd my-api

# Generate components
wolf generate controller Users
wolf generate model User
wolf generate service AuthService

# Run your app
wolf run main.wolf

# Compile to native binary
wolf build main.wolf
./wolf_out/my_app
```

---

## 📚 Standard Library (110+ functions)

Wolf ships with a comprehensive standard library. No packages needed.

### Strings
```wolf
strtoupper("hello")             # "HELLO"
strtolower("HELLO")             # "hello"
ucfirst("hello world")          # "Hello world"
trim("  hello  ")               # "hello"
str_contains("hello", "ell")    # true
str_replace("world", "wolf", $s)
substr("hello", 1, 3)           # "ell"
strlen("hello")                 # 5
explode(",", "a,b,c")           # ["a","b","c"]
htmlspecialchars("<b>hi</b>")   # "&lt;b&gt;hi&lt;/b&gt;"
base64_encode("data")           # "ZGF0YQ=="
url_encode("hello world")       # "hello%20world"
```

### Math
```wolf
sin(1.5)   cos(0.5)   tan(0.8)        # trig — no math. prefix needed
sqrt(16)   pow(2, 10)  log(100)        # power & roots
round(3.7)  ceil(3.1)   floor(3.9)     # rounding
abs(-5)     fmod(7.5, 2.0)             # basics
pi()        deg2rad(90)                 # constants
rand(1, 100)  clamp(15, 0, 10)         # random & clamping
number_format(1234567.89, 2, ".", ",")  # "1,234,567.89"
```

### Arrays
```wolf
count($arr)                     # length
in_array("banana", $arr)        # search
sort($arr)                      # sort ascending
array_merge($a, $b)             # merge
array_unique($arr)              # deduplicate
array_reverse($arr)             # reverse
array_diff($a, $b)              # set difference
range(1, 10)                    # [1..10]
```

### Validation
```wolf
is_email("user@wolf.dev")      # true
is_url("https://wolf.dev")     # true
is_phone("+2349012345678")     # true
is_uuid("a1b2c3d4-...")        # true
is_ip("192.168.1.1")           # true
is_numeric("42.5")             # true
```

### Date & Time
```wolf
time()                          # Unix timestamp
time_ms()                       # Milliseconds
date("%Y-%m-%d", time())        # "2026-03-04"
strtotime("tomorrow")           # +86400
strtotime("+7 days")            # A week from now
days_in_month(2, 2024)          # 29
is_leap_year(2024)              # true
```

### Security & Encoding
```wolf
md5("data")                     # Hash
sha256("data")                  # SHA-256
password_hash("secret")         # Bcrypt hash
password_verify("secret", $h)   # Verify
uuid_v4()                       # Generate UUID
base64_encode($data)            # Encode
```

### File System
```wolf
file_read("/path/to/file")     # Read entire file
file_write("/path", "data")    # Write
file_exists("/path")           # Check existence
file_size("/path")             # Bytes
file_extension("img.jpg")     # "jpg"
slug("Hello World!")           # "hello-world"
```

---

## 🏗 Project Structure

```
my-api/
├── config/
│   └── config.wolf          # define() constants, env() vars
├── controllers/
│   ├── Auth.wolf            # Auto-routed to /auth/*
│   └── Users.wolf           # Auto-routed to /users/*
├── models/
│   └── User.wolf            # Database model
├── services/
│   └── AuthService.wolf     # Business logic
├── libraries/
│   ├── Controller.wolf      # Base controller (model, getData)
│   ├── Database.wolf        # PDO-style database wrapper
│   ├── Redis.wolf           # Redis client wrapper
│   └── Core.wolf            # Router & auto-discovery
├── helpers/
└── main.wolf                # Entry point
```

---

## 🧪 Testing

```bash
# Run the full E2E test suite (20 tests)
make test

# Run specific tests
go test ./e2e/... -run TestHelloWorld -v
```

---

## 🔧 Compiler Pipeline

```
Wolf Source (.wolf)
    ↓  Lexer
  Token Stream
    ↓  Parser
  Abstract Syntax Tree
    ↓  Resolver + Type Checker
  Wolf IR (WIR)
    ↓  LLVM Emitter
  LLVM IR (.ll)
    ↓  llc + clang
  Native Binary
```

The compiler is written in **Go**. The runtime (`wolf_runtime.c`) is a portable C library linked at compile time — it provides the 110+ stdlib functions, memory management, HTTP server, and database drivers.

---

## 🗺 Roadmap

- [x] LLVM IR compilation pipeline
- [x] Native HTTP server (C runtime)
- [x] Controller auto-discovery routing
- [x] `wolf generate` scaffolding CLI
- [x] 110+ stdlib functions (strings, arrays, math, validation, crypto, filesystem, date/time)
- [x] Config system (`define()`, `env()`)
- [x] Database & Redis drivers
- [ ] `@ml` Python bridge (CPython C API)
- [ ] Production MySQL/PostgreSQL drivers
- [ ] Async/parallel concurrency
- [ ] Package manager (`wolf install`)
- [ ] Rust-style error messages
- [ ] macOS + Windows builds

---

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.

---

**Wolf** — Write PHP. Compile native. Ship fast. 🐺
