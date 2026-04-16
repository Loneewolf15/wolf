# Wolf Backend Infrastructure — Comprehensive Research Document

> *"Everybody, when they're starting a project, there's so many decisions, there's so much fatigue,
> and there are so many tools that you need to start managing at day one of the project."*
> — José Valim, Creator of Elixir
>
> That sentence is Wolf's entire mission statement written by someone else.

## Table of Contents

1. [The Elixir Lesson — Most Important Study](#1-the-elixir-lesson--most-important-study)
2. [Category 1 — Concurrency & Process Management](#2-category-1--concurrency--process-management)
3. [Category 2 — Observability](#3-category-2--observability--the-biggest-gap-in-every-language)
4. [Category 3 — Error Handling & Error Messages](#4-category-3--error-handling--error-messages)
5. [Category 4 — Database](#5-category-4--database--what-every-language-gets-wrong)
6. [Category 5 — Real-time](#6-category-5--real-time--websockets-and-live-updates)
7. [Category 6 — Testing](#7-category-6--testing--what-every-language-makes-hard)
8. [Category 7 — Security](#8-category-7--security--what-every-language-leaves-to-the-developer)
9. [Category 8 — GraphQL and gRPC](#9-category-8--graphql-and-grpc--beyond-rest)
10. [Category 9 — Message Queues and Event Streaming](#10-category-9--message-queues-and-event-streaming)
11. [Category 10 — What Nobody Has Built — Wolf Originals](#11-category-10--what-nobody-has-built--wolf-originals)
12. [The Complete Map — What Wolf Takes From Each Language](#12-the-complete-map--what-wolf-takes-from-each-language)
13. [Priority Order for Implementation](#13-priority-order-for-implementation)

---

## 1. The Elixir Lesson — Most Important Study

Before the full list, Elixir deserves special attention because it is the most relevant language
to study for Wolf.

Elixir was the **second most desired programming language to learn** according to the 2024 Stack
Overflow survey with 65,000 respondents — just ahead of Zig and behind only Rust.

Why? Because Elixir solved the problem every other language ignores:

> *"Normally, designing an application that's resilient to failures is a tall order. With other
> languages, the best you can do is to code defensively. Elixir is different — it frees you to
> 'let it crash.' That's only possible because OTP Supervisors offer unparalleled support for
> detecting and recovering from process failures."*

In 2021, the Numerical Elixir effort was announced with the goal of bringing machine learning,
neural networks, GPU compilation, and data processing to the Elixir ecosystem — they are solving
the same `@ml` problem Wolf solved, just differently.

The single most important thing Elixir has that no other mainstream language has is **OTP —
specifically supervision trees and the "let it crash" philosophy.** Wolf needs its own version of
this.

Elixir also attracts experienced developers who have faced complexity issues in other stacks and
want a simpler solution:

> *"We keep everything simple."* — José Valim

**That sentence is Wolf's soul. Every feature in this document serves that principle — or it does
not ship.**

---

## 2. Category 1 — Concurrency & Process Management

### What Every Language Currently Does

| Language | Concurrency Model | Developer Pain |
|---|---|---|
| Go | Goroutines + channels | Manual — you write all supervision logic |
| Node.js | Single-threaded event loop | CPU-bound tasks block everything. No true parallelism |
| Python | GIL prevents true parallelism | Threading is painful. asyncio is confusing. Three different async models exist |
| PHP | None native | Every request is isolated. No shared state. No background processing |
| Elixir | BEAM processes + OTP | Best in class but functional paradigm is a steep learning curve |

### What Developers Complain About

Go developers interested in ML/AI expressed frustration with Python for reasons such as type
safety, code quality, and challenging deployments — but they were largely unified on what prevents
them from using Go with AI-powered services: **the ecosystem is centred around Python.**

Nobody has solved: write concurrent code that **supervises itself and recovers from failures
automatically.**

### What Wolf Implements

```wolf
# Wolf concurrency — already planned:
async { }          # non-blocking task
await $task        # wait for result
parallel { }       # run multiple tasks simultaneously
channel(int)       # typed channels like Go

# What Wolf adds from Elixir — supervision:
@supervise {
    # If this block crashes — Wolf restarts it automatically
    # Developer never writes restart logic
    $worker = new PaymentWorker
    $worker->processQueue()
}

# Supervision strategies:
@supervise(strategy: "one_for_one") {
    # restart only the crashed process
}

@supervise(strategy: "one_for_all") {
    # restart everything if one crashes — for interdependent processes
}

@supervise(restart: "exponential", max: 5) {
    # retry 5 times with exponential backoff before giving up
}
```

### The "Let It Crash" Philosophy for Wolf

The insight from Elixir is profound. Instead of writing defensive code for every possible
failure — you write happy-path code and let the supervisor handle failures. Wolf implements a
simpler version of this without requiring developers to learn functional programming.

```wolf
// Wolf supervised worker — crashes are handled automatically

class PaymentWorker {
    private $db

    func __construct() {
        $this->db = new Database
    }

    @supervise(restart: "exponential", max: 3)
    func processQueue() {
        $queue = new JobQueue
        while true {
            $job = $queue->pop()
            if !$job { sleep(1000); continue }

            # If this throws — @supervise restarts processQueue()
            # Developer never writes try/catch for infrastructure failures
            $this->processJob($job)
            $queue->delete($job["id"])
        }
    }
}
```

---

## 3. Category 2 — Observability — The Biggest Gap In Every Language

This is the category that costs developers the most time in production and that **no language has
solved natively.**

### What Every Language Currently Does

Every language makes you install external tools:

```bash
# Go:
install OpenTelemetry SDK + configure Jaeger + set up Prometheus
+ configure Grafana + wire everything together
= 2-3 days of setup before you see a single trace

# Node:
npm install @opentelemetry/sdk-node @opentelemetry/auto-instrumentations
+ configure exporters + set up collector
= same 2-3 days

# Python:
pip install opentelemetry-sdk opentelemetry-instrumentation
+ same configuration nightmare

# PHP:
no standard solution. Most PHP apps have no tracing at all.
```

### The Scale of the Problem

In a survey looking at the biggest challenges to observability, **58% of developers said that
identifying blind spots is a top concern.** A bug that could take 30 minutes to fix ends up
consuming days or weeks.

OpenTelemetry has become the de facto standard in observability. Beyond Trace, Metric, and Log,
OpenTelemetry introduced Profiling as a standard in 2024, aiming to standardize all data formats
in observability.

### What Wolf Implements — Observability as a Language Feature

```wolf
# @trace — already in our spec
@trace("payment.process") {
    $result = $payment->charge($data)
}

# Built-in metrics — no Prometheus setup
metrics_increment("rides.created")
metrics_gauge("active_connections", $count)
metrics_histogram("response_time_ms", $duration)
metrics_counter("payment.success")
metrics_counter("payment.failed")

# Built-in health endpoint — automatic on every Wolf app
# GET /health always returns:
# {
#   "status": "healthy",
#   "uptime": 3600,
#   "memory_mb": 45,
#   "db_connected": true,
#   "redis_connected": true,
#   "version": "1.0.0"
# }
```

### wolf dev Dashboard — The Killer Feature

Available at `localhost:8081` during development. No external tool. No configuration. Built in.

Real-time display of:
- Request rates
- Response time P50/P95/P99
- Error rates
- Active connections
- DB query times
- Cache hit/miss ratio

**Every developer who has stared at a blank terminal trying to figure out why their API is slow
will immediately understand the value.**

---

## 4. Category 3 — Error Handling & Error Messages

### What Developers Complain About Most

Go's syntax can be unintuitive and confusing. **The way it handles errors and null values can be
frustrating.** The 2025 Go Developer Survey showed frustrations centre on:

- Enforcing idioms (33%)
- Missing features like enums (28%)
- Vetting modules (26%)

Go's error handling pattern is the single most complained-about thing in the language:

```go
// Go — you write this pattern hundreds of times
result, err := doSomething()
if err != nil {
    return nil, err
}
result2, err := doSomethingElse(result)
if err != nil {
    return nil, err
}
```

### What Wolf Implements

```wolf
# Option 1 — try/catch (familiar to PHP developers)
try {
    $result  = doSomething()
    $result2 = doSomethingElse($result)
    return $this->res->json($result2)
} catch ($e) {
    return $this->res->error($e->message, 500)
}

# Option 2 — multiple return values (Go-style when needed)
[$data, $err] = riskyOperation()
if $err { return $this->res->error($err, 400) }

# Option 3 — @safe block (Wolf original)
# Runs the block, catches any error, returns false on failure
$result = @safe { riskyOperation() }
if !$result { return $this->res->error("Operation failed", 500) }
```

### Wolf Error Messages — Rust Quality

The most important part of error handling is not the syntax — it is the quality of the error
messages themselves. Rust set the standard. Wolf must match it.

```
# Bad error (what PHP/Go give you):
Parse error: syntax error, unexpected token on line 47

# Wolf error (Rust-quality):
wolf: error at controllers/RideController.wolf, line 47, col 12

    45 |     func create() {
    46 |         $data = $this->req->body()
    47 |         $ride = Ride
                         ^^^^ missing 'new' keyword

    Did you mean: new Ride

    Hint: Wolf classes are instantiated with 'new ClassName'
          See: wolflang.dev/docs/classes#instantiation
```

Every error includes:
- File, line, column
- The exact code that caused it
- A caret pointing to the problem
- A suggestion
- A documentation link

**Zero cryptic errors. Ever.**

---

## 5. Category 4 — Database — What Every Language Gets Wrong

### The Universal ORM Debate

Every language has the same debate: ORM vs raw SQL. The ORM camp makes simple queries easy and
complex queries impossible. The raw SQL camp makes everything verbose. Nobody has found the right
middle ground.

| Language | Options | Pain |
|---|---|---|
| Go | database/sql, GORM, sqlx | Verbose. GORM generates bad queries at scale |
| Node.js | Prisma, Sequelize, knex | Prisma generates massive queries. Sequelize is slow |
| Python | SQLAlchemy, Django ORM | SQLAlchemy has steep learning curve. Django ORM bad for complex |
| PHP | PDO, Eloquent, Doctrine | Eloquent generates N+1 queries. Doctrine needs XML config |

### What Wolf Implements

Wolf's `$this->db->` pattern is already better than most ORMs because it is honest — you write
SQL, Wolf executes it.

```wolf
// Simple query — clean
$this->db->query("SELECT * FROM rides WHERE user_id = :id")
$this->db->bind(":id", $userId)
$rides = $this->db->resultSet()

// Query builder for dynamic queries
$query = $this->db->builder("rides")
if $status   { $query->where("status", $status) }
if $zone     { $query->where("zone", $zone) }
if $fromDate { $query->whereGte("created_at", $fromDate) }
$query->orderBy("created_at", "desc")
$query->limit(20)
$rides = $query->get()
```

### N+1 Prevention — Built Into the Compiler

```wolf
// Wolf detects N+1 queries at compile time:
foreach $orders as $order {
    $items = $order->getItems()
    // wolf: warning: possible N+1 query
    // consider using eager loading
}

// Eager loading built in:
$orders = $this->db->with("items", "user")->query("SELECT * FROM orders")
// executes 3 queries total instead of N+1
```

---

## 6. Category 5 — Real-time — WebSockets and Live Updates

### What Developers Complain About

WebSockets in Go, Node, and PHP all require external libraries, manual connection management, and
complex state management.

**Phoenix Channels (Elixir) are considered the gold standard:**

> *"Phoenix Channels enable bidirectional communication between clients and servers, perfect for
> chat apps, live dashboards, and collaborative tools. Phoenix LiveView allows developers to build
> interactive, real-time UIs without writing JavaScript."*

### What Wolf Implements

```wolf
// controllers/LiveController.wolf

class LiveController extends Controller {

    # GET /live/connect
    func connect() {
        @guard("auth")

        $userId = $this->req->user->user_id

        $this->res->websocket(fn($socket) {

            # Join rooms
            $socket->join("user:{$userId}")
            $socket->join("global:announcements")

            # Handle incoming messages
            $socket->on("message", fn($data) {
                @validate($data, [
                    "room"    => "required|string",
                    "content" => "required|string|max:1000",
                ])
                broadcast($clean["room"], "message", [
                    "from"    => $userId,
                    "content" => $clean["content"],
                    "time"    => time(),
                ])
            })

            # Presence tracking — built in
            $socket->on("connect",    fn() { presence_track($userId) })
            $socket->on("disconnect", fn() { presence_untrack($userId) })
        })
    }
}

# Broadcasting from anywhere in the codebase
broadcast("user:{$userId}", "notification", $data)
broadcast("global:announcements", "alert", $message)
broadcast_to_all("system:maintenance", $notice)

# Presence — who is online right now
$online = presence_list("user:{$roomId}")
$count  = presence_count("global")
```

---

## 7. Category 6 — Testing — What Every Language Makes Hard

### The Universal Complaint

Testing in every language has the same problem: **mocking database calls and HTTP calls is
painful.**

- Go — verbose, no built-in HTTP test client
- Python — requires understanding fixtures, mocking, and pytest plugins
- Node — requires jest configuration, separate libraries
- PHP — PHPUnit is fine but verbose, mocking is painful

### What Wolf Implements — Built-in Test Runner

No external library. No configuration.

```wolf
# tests/AuthController.test.wolf

test "register creates user successfully" {
    # Mock the database — built in, no library
    $db = mock(Database)
    $db->expect("execute")->returns(true)
    $db->expect("lastInsertId")->returns("user_abc123")

    $res = wolf_test_post("/auth/register", [
        "first_name" => "Ade",
        "last_name"  => "Wolf",
        "email"      => "ade@wolf.dev",
        "phone"      => "08012345678",
        "password"   => "secret123",
        "requestID"  => "rid_2006",
    ])

    assert_status($res, 201)
    assert_json($res, "status", true)
    assert_json($res, "data.user_id", "user_abc123")
    assert_called($db, "execute", 1)
}

test "login fails with wrong password" {
    $res = wolf_test_post("/auth/login", [
        "username"  => "ade@wolf.dev",
        "password"  => "wrongpassword",
        "requestID" => "rid_2006",
    ])

    assert_status($res, 401)
    assert_json($res, "status", false)
    assert_json($res, "message", "Invalid credentials")
}

test "paginated rides returns correct structure" {
    $res = wolf_test_get("/rides/index", headers: [
        "Authorization" => "Bearer " . test_jwt_token()
    ])

    assert_status($res, 200)
    assert_has_key($res, "data.pagination")
    assert_has_key($res, "data.pagination.total_pages")
    assert_has_key($res, "data.pagination.current_page")
}
```

```bash
wolf test                       # run all tests
wolf test --file AuthController # run specific test file
wolf test --watch               # re-run on file change
wolf test --coverage            # show coverage report
```

---

## 8. Category 7 — Security — What Every Language Leaves To The Developer

### What Developers Consistently Get Wrong

Every security breach in backend systems comes from the same categories:

| Vulnerability | How It Happens | How Common |
|---|---|---|
| SQL injection | String concatenation instead of binds | Extremely common |
| XSS | Unescaped user output | Very common |
| CSRF | Missing CSRF tokens | Common |
| Insecure headers | No security headers set | Almost universal |
| Rate limiting | Not implemented on sensitive endpoints | Very common |
| Secret exposure | API keys in logs or error messages | Common |
| Mass assignment | Accepting all request fields without filtering | Very common |

### What Wolf Implements — Security as Default, Not Optional

```wolf
# SQL injection — structurally impossible in Wolf
# $this->db->bind() always uses parameterised queries
# The compiler warns if it detects raw string in query()

$this->db->query("SELECT * FROM users WHERE id = {$id}")
# wolf: error: potential SQL injection
# Use bind() instead: $this->db->bind(':id', $id)

# Security headers — automatic on every response
# Wolf adds these by default without any configuration:
# X-Content-Type-Options: nosniff
# X-Frame-Options: DENY
# X-XSS-Protection: 1; mode=block
# Referrer-Policy: strict-origin-when-cross-origin
# Content-Security-Policy: default-src 'self'

# Secret exposure prevention
# Wolf never logs values bound with @encrypted
# JWT tokens in logs are truncated to first 10 chars
# Common secret patterns are redacted in error messages

# Mass assignment protection — built into @validate
@validate($this->req->body(), [
    "first_name" => "required|string",
    "email"      => "required|email",
])
# $clean contains ONLY first_name and email
# Even if the request contained user_id, role, admin_flag
# they are stripped — mass assignment is structurally impossible
```

---

## 9. Category 8 — GraphQL and gRPC — Beyond REST

### What Developers Complain About

Every language requires external libraries for GraphQL and gRPC. The setup is complex, the
generated code is verbose, and maintaining schemas alongside code is painful.

### What Wolf Implements — Auto-generation from Controllers

```wolf
// GraphQL — Wolf generates schema from your models automatically

# Add to config/config.wolf:
define("GRAPHQL_ENABLED", true)
define("GRAPHQL_ENDPOINT", "/graphql")

# The developer annotates their model methods:
class Ride {
    private $db

    @graphql(type: "Ride", fields: ["ride_id", "pickup", "dropoff", "status", "fare"])
    func findById($id) {
        $this->db->query("SELECT * FROM rides WHERE ride_id = :id")
        $this->db->bind(":id", $id)
        return $this->db->single()
    }

    @graphql(type: "[Ride]", args: {userId: "String!"})
    func getByUser($userId) {
        $this->db->query("SELECT * FROM rides WHERE user_id = :uid")
        $this->db->bind(":uid", $userId)
        return $this->db->resultSet()
    }
}

# Wolf automatically generates:
# type Ride { ride_id: String, pickup: String, dropoff: String, status: String, fare: Float }
# type Query { ride(id: String!): Ride, ridesByUser(userId: String!): [Ride] }
```

```wolf
// gRPC — Wolf generates from proto-like definitions

@grpc service RideService {
    rpc GetRide(RideRequest) returns (RideResponse)
    rpc CreateRide(CreateRideRequest) returns (RideResponse)
    rpc StreamRideUpdates(RideRequest) returns (stream RideUpdate)
}

class RideService {
    func GetRide($req) {
        $ride = new Ride
        return $ride->findById($req->ride_id)
    }
}
```

---

## 10. Category 9 — Message Queues and Event Streaming

### What Developers Complain About

Kafka, RabbitMQ, Redis Pub/Sub — all require:
- External broker installation and management
- External client libraries per language
- Complex configuration
- Separate mental model from the rest of the codebase

### What Wolf Implements

```wolf
# Built-in pub/sub — no external broker for simple use cases
publish("rides.created", $rideData)
publish("payments.completed", $paymentData)
publish("users.kyc_approved", $userData)

# Subscribe — in a separate worker process
subscribe("rides.created", fn($event) {
    $notification = new NotificationService
    $notification->send($event->user_id, "ride_created", $event)
})

subscribe("payments.completed", fn($event) {
    $wallet = new Wallet
    $wallet->creditUser($event->user_id, $event->amount)
})

# For Kafka/RabbitMQ — same API, driver is a config change
# config/config.wolf:
define("QUEUE_DRIVER", Env::get("QUEUE_DRIVER", "redis"))
# Options: redis (default), kafka, rabbitmq, sqs

# The same publish/subscribe calls work with any driver
# Switching from Redis to Kafka = one line in .env
publish("rides.created", $data)       # works with redis, kafka, rabbitmq
subscribe("rides.created", fn($e) { }) # same
```

---

## 11. Category 10 — What Nobody Has Built — Wolf Originals

These do not exist in any language today. Wolf invents them.

### wolf explain — AI-Powered Error Explanation

```bash
# You see a cryptic error in production logs:
[ERROR] DB: Deadlock found when trying to get lock; try restarting transaction

wolf explain "Deadlock found when trying to get lock"

# Wolf responds:
# This error means two database transactions are waiting for each other,
# creating a circular dependency that MySQL cannot resolve.
#
# In your codebase, the most likely cause is:
# models/Wallet.wolf line 67 and models/Transaction.wolf line 34
# Both acquire locks on the wallets table in different orders.
#
# Fix: Always acquire locks in the same order across all transactions.
# Update models/Wallet.wolf to acquire locks in this order:
# 1. users table
# 2. wallets table
# 3. transactions table
#
# See: wolflang.dev/docs/database/deadlocks
```

### wolf profile — Production Profiling Without Overhead

```bash
wolf profile --endpoint "/rides/estimate" --duration 60s

# Samples your running production server for 60 seconds
# No APM subscription. No external agent. No code changes.

Results for GET /rides/estimate (1,247 requests):
┌─────────────────────────────────────────────┐
│ P50: 8ms   P95: 23ms   P99: 67ms           │
│                                             │
│ Time breakdown:                             │
│  DB queries:    12ms avg  ████████░░ 52%   │
│  @ml block:     8ms avg   ██████░░░░ 35%   │
│  Serialisation: 1ms avg   █░░░░░░░░░  4%   │
│  Other:         2ms avg   ██░░░░░░░░  9%   │
│                                             │
│ Slowest DB query (3ms avg):                 │
│  SELECT * FROM drivers WHERE zone = ?       │
│  → Add index on zone column                 │
└─────────────────────────────────────────────┘
```

### @contract — API Contract Testing Built In

```wolf
// Define what your API guarantees
// Wolf verifies these contracts in CI automatically

@contract GET /rides/index {
    requires: auth
    returns: {
        "status": bool,
        "data": {
            "data": array,
            "pagination": {
                "current_page": int,
                "total_pages":  int,
                "total":        int,
            }
        }
    }
    response_time_ms: 100  // fails CI if P95 exceeds this
}

@contract POST /auth/login {
    requires: [
        "username": "string",
        "password": "string",
    ]
    returns_on_success: {
        "status": true,
        "data": { "token": "string" }
    }
    returns_on_failure: {
        "status": false,
        "message": "string"
    }
    rate_limit: "5 per 60s"  // verified in tests automatically
}
```

### Hot Code Reloading in Production — From Elixir

Elixir has an interactive REPL inside running programs, including Phoenix web servers, with code
reloading and access to internal state. Wolf implements a safer version:

```bash
# Deploy a new version WITHOUT restarting the server
# Zero downtime. Zero dropped requests. Zero connection loss.

wolf deploy --hot

# What Wolf does internally:
# 1. Compiles the new version
# 2. Starts routing new requests to new version
# 3. Waits for in-flight requests on old version to complete
# 4. Swaps over completely
# 5. Old version retired
# Total time: under 100ms
# Dropped requests: 0
```

---

## 12. The Complete Map — What Wolf Takes From Each Language

### From Go ✅ Take | ❌ Fix

```
✅ goroutines → async/await + parallel blocks
✅ channels → channel(type)
✅ single binary deployment
✅ fast compile times
✅ simple concurrency model
❌ verbose error handling → Wolf uses try/catch + @safe
❌ no built-in DB pattern → Wolf has $this->db->
❌ no enums → Wolf adds enums (28% of Go devs want this)
❌ no built-in ML → Wolf has @ml
❌ blank slate — no framework choice → Wolf has wolf new with features
```

### From Python ✅ Take | ❌ Fix

```
✅ readable syntax → Wolf's PHP-familiar syntax achieves this differently
✅ ML ecosystem → @ml blocks give Wolf all of Python's ML
❌ slow performance → Wolf is 10-50x faster
❌ virtual env complexity → wolf.python handles automatically
❌ GIL → Wolf has no GIL
❌ deployment complexity → wolf build → one binary
❌ framework fragmentation (Django/Flask/FastAPI) → Wolf has one way
```

### From Node.js ✅ Take | ❌ Fix

```
✅ event-driven async → Wolf async is cleaner
✅ JSON-first → Wolf has json_encode/decode built in
❌ callback hell → Wolf has async/await
❌ npm dependency hell → Wolf stdlib covers 80% of npm use cases
❌ unstable APIs → Wolf syntax frozen at v1.0
❌ no binary deployment → wolf build → one binary
```

### From PHP ✅ Take | ❌ Fix

```
✅ $variables → locked in Wolf forever
✅ $this->method() → locked in Wolf forever
✅ curly braces → locked in Wolf forever
✅ global stdlib functions → Wolf stdlib works identically
✅ familiar to 77% of the web
❌ performance ceiling → Wolf is 10-50x faster via LLVM
❌ Apache/Nginx required → Wolf HTTP server is built in
❌ no native concurrency → Wolf has async/parallel
❌ shared-nothing model → Wolf has proper state management
❌ no binary deployment → wolf build → one binary
```

### From Elixir/OTP ✅ Take | ❌ Leave

```
✅ supervision trees → @supervise blocks
✅ let it crash philosophy → @supervise handles restarts
✅ hot code reloading → wolf deploy --hot
✅ pub/sub built in → publish/subscribe functions
✅ presence tracking → presence_track/presence_list
✅ operational simplicity → wolf dev / wolf deploy
❌ functional paradigm → Wolf stays OOP (intentional — lower barrier)
❌ BEAM VM → Wolf uses LLVM (more performance, simpler mental model)
❌ actor model → too complex for Wolf's target developer
```

### From Rust ✅ Take | ❌ Leave

```
✅ LLVM backend → Wolf uses same backend
✅ Rust-quality error messages → Wolf design goal
✅ single binary deployment
✅ no garbage collector → no GC pauses
❌ borrow checker → Wolf does not have this — intentional
❌ steep learning curve → Wolf's explicit goal is the opposite
❌ verbose syntax → Wolf is familiar to PHP developers
```

### What Wolf Has That NOBODY Has

```
✅ @ml blocks — Python ML inside compiled backend code
✅ @cache blocks — caching as language construct
✅ @queue blocks — background jobs as language construct
✅ @guard blocks — authentication as language construct
✅ @validate blocks — validation as language construct
✅ @supervise blocks — fault tolerance as language construct
✅ @contract — API contracts verified in CI
✅ @encrypted — column-level encryption annotation
✅ @safe — error-safe block execution
✅ $this->db->paginate() — pagination as database method
✅ $this->req->audit() — audit trail as request method
✅ $this->req->idempotent() — idempotency as request method
✅ $this->req->diff() — object diff as request method
✅ wolf dev — entire stack in one command, zero config
✅ wolf new — feature-based scaffold with best practices applied
✅ wolf explain — AI-powered error explanation
✅ wolf profile — built-in production profiler, no APM required
✅ wolf deploy --hot — zero-downtime hot code reload
✅ wolf docker init — generate docker-compose.yml from config
✅ money_format/money_multiply — decimal-safe money math
✅ phone_format("NG") — African-first phone normalisation
✅ Built-in observability dashboard in wolf dev
✅ Security headers automatic on every response
✅ SQL injection structurally impossible via compiler warning
✅ Mass assignment structurally impossible via @validate
```

---

## 13. Priority Order for Implementation

### Tier 1 — Ship with v0.1.0 (current sprint)

All @blocks (`@cache`, `@queue`, `@guard`, `@validate`, `@encrypted`, `@trace`), complete stdlib,
`wolf dev`, `wolf new` with feature selection, `$this->db->` full pattern, WebSocket support,
security headers automatic, SQL injection compiler warning.

### Tier 2 — Ship with v0.2.0 (3–6 months post launch)

```
@supervise blocks           — fault tolerance, let it crash
@safe blocks                — error-safe execution
@contract                   — API contract testing in CI
Built-in pub/sub            — publish/subscribe without external broker
wolf test                   — built-in test runner with mocking
wolf dev dashboard          — observability in browser at localhost:8081
Query builder               — $this->db->builder() for dynamic queries
N+1 detection               — compiler warning on queries in loops
Eager loading               — $this->db->with("relation")
Mass assignment protection  — structurally enforced via @validate
GraphQL auto-generation     — @graphql annotations on models
Enums                       — top Go developer complaint, Wolf adds them
```

### Tier 3 — Ship with v0.3.0 (6–12 months post launch)

```
wolf explain                — AI-powered error explanation
wolf profile                — production profiler without APM
wolf deploy --hot           — zero-downtime hot reload
gRPC support                — @grpc service definitions
OpenTelemetry export        — metrics/traces to external systems
wolfpkg v0.1                — package manager
Message queue drivers       — kafka, rabbitmq, sqs as driver options
```

### Tier 4 — v1.0 and beyond

```
Wolf self-hosted compiler   — Wolf compiles Wolf
WASM target                 — Wolf in the browser
wolfpkg.dev registry        — public package registry
Full distributed tracing    — spans across microservices
VS Code extension           — syntax highlighting + LSP
Windows support
```

---

## The Core Principle

Every feature in this document is measured against one sentence:

> **Wolf handles the infrastructure so you write the logic.**

If a feature helps the developer write more logic and less infrastructure — it ships.
If a feature adds infrastructure the developer has to think about — it gets cut.

This is Wolf's design philosophy. This is what makes a language a default.
