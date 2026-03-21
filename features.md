# Wolf Language — Features Backlog

> Implementation order: stdlib first, then language features, then tooling, then platform.
> Each feature is self-contained. Work through them top to bottom.

---

## Phase 1 — Stdlib Completion (Start Here)

These are functions documented in the PRD that are not yet in the functional stdlib.
The goal: a Wolf developer never installs a package for common backend work.

---

### STDLIB-01 — String Functions

**Missing from current stdlib:**

- `ucfirst($str)` — capitalise first character
- `ucwords($str)` — capitalise first letter of each word
- `lcfirst($str)` — lowercase first character
- `str_contains($str, $sub)` — boolean substring check
- `str_starts_with($str, $prefix)` — boolean prefix check
- `str_ends_with($str, $suffix)` — boolean suffix check
- `str_ireplace($find, $rep, $str)` — case-insensitive replace
- `str_repeat($str, $times)` — repeat string N times
- `str_pad($str, $len, $pad)` — pad string to length
- `wordwrap($str, $width, $break)` — wrap long strings at word boundary
- `str_word_count($str)` — count words in string
- `strrpos($str, $sub)` — last position of substring
- `similar_text($a, $b)` — similarity percentage between two strings
- `number_format($num, $dec)` — format number with thousands separator
- `nl2br($str)` — convert newlines to `<br>` tags
- `strip_tags($str)` — remove HTML tags
- `htmlspecialchars($str)` — escape HTML entities
- `htmlspecialchars_decode($str)` — decode HTML entities
- `addslashes($str)` — escape quotes with backslash
- `stripslashes($str)` — remove escape slashes
- `quoted_printable_encode($str)` — email-safe encoding
- `preg_match($pattern, $str)` — regex match, returns bool
- `preg_match($pattern, $str, &$matches)` — regex match with capture groups
- `preg_match_all($pattern, $str, &$matches)` — all regex matches
- `preg_replace($pattern, $rep, $str)` — regex replace
- `preg_split($pattern, $str)` — split string by regex pattern

---

### STDLIB-02 — Array Functions

**Missing from current stdlib:**

- `range($start, $end)` — generate array of integers
- `range($start, $end, $step)` — range with step
- `array_fill($start, $num, $value)` — fill array with value
- `array_combine($keys, $values)` — zip keys and values into map
- `array_push($arr, $val)` — append to end
- `array_pop($arr)` — remove and return last element
- `array_unshift($arr, $val)` — prepend to array
- `array_shift($arr)` — remove and return first element
- `array_splice($arr, $offset, $len)` — remove/insert at position
- `array_search($val, $arr)` — find key of value, false if not found
- `array_key_exists($key, $arr)` — check map key exists
- `array_map($fn, $arr)` — transform each element
- `array_filter($arr, $fn)` — keep elements where fn returns true
- `array_reduce($arr, $fn, $init)` — reduce array to single value
- `array_walk($arr, $fn)` — apply function in-place
- `rsort($arr)` — sort descending
- `asort($arr)` — sort preserving keys
- `arsort($arr)` — sort descending preserving keys
- `ksort($arr)` — sort by key ascending
- `krsort($arr)` — sort by key descending
- `usort($arr, $fn)` — sort with custom comparator
- `uasort($arr, $fn)` — sort with custom comparator, preserve keys
- `array_chunk($arr, $size)` — split into chunks of size
- `array_unique($arr)` — remove duplicate values
- `array_flip($arr)` — swap keys and values
- `array_reverse($arr)` — reverse array order
- `array_sum($arr)` — sum all values
- `array_product($arr)` — multiply all values
- `array_keys($arr)` — get all keys
- `array_values($arr)` — get all values
- `array_column($arr, $col)` — pluck column from 2D array
- `compact(...$vars)` — create map from variable names
- `extract($arr)` — create variables from map keys
- `array_diff($a, $b)` — values in a not in b
- `array_intersect($a, $b)` — values in both arrays
- `array_diff_key($a, $b)` — keys in a not in b

---

### STDLIB-03 — Math Functions

**Missing from current stdlib:**

- `fmod($a, $b)` — floating point modulo
- `log($n)` — natural logarithm
- `log($n, $base)` — logarithm with specified base
- `exp($n)` — e raised to power n
- `clamp($n, $min, $max)` — constrain value to range (Wolf addition)
- `rand_float()` — random float between 0.0 and 1.0
- `rand_secure()` — cryptographically secure random bytes
- `array_rand($arr)` — random key from array
- `array_rand($arr, $count)` — multiple random keys from array
- `sin($n)`, `cos($n)`, `tan($n)` — trigonometry
- `asin($n)`, `acos($n)`, `atan($n)`, `atan2($y, $x)` — inverse trig
- `deg2rad($deg)`, `rad2deg($rad)` — angle conversion
- `pi()` — π constant
- `INF` — infinity constant
- `NAN` — not a number constant
- `array_mean($arr)` — arithmetic mean
- `array_median($arr)` — median value
- `array_mode($arr)` — most common value
- `array_variance($arr)` — statistical variance
- `array_std_dev($arr)` — standard deviation
- `array_percentile($arr, $p)` — nth percentile value

---

### STDLIB-04 — Date & Time Functions

**Missing from current stdlib:**

- `time_ms()` — Unix timestamp in milliseconds
- `time_ns()` — Unix timestamp in nanoseconds
- `strtotime($str)` — parse human date string to timestamp
- `strtotime($str, $base)` — relative to base timestamp
- `mktime($h, $m, $s, $mo, $d, $y)` — create timestamp from parts
- `date_create($str)` — create date object from string
- `date_format($date, $format)` — format a date object
- `date_diff($date1, $date2)` — difference between two dates

**Date object methods (Wolf addition):**

- `$d->format($fmt)` — format date as string
- `$d->addDays($n)` — return new date N days ahead
- `$d->addMonths($n)` — return new date N months ahead
- `$d->addHours($n)` — return new date N hours ahead
- `$d->diffInDays($other)` — integer days between dates
- `$d->diffInHours($other)` — integer hours between dates
- `$d->isPast()` — true if date is in the past
- `$d->isFuture()` — true if date is in the future
- `$d->isToday()` — true if date is today
- `$d->startOfDay()` — set time to 00:00:00
- `$d->endOfDay()` — set time to 23:59:59
- `$d->startOfMonth()` — first day of month
- `$d->endOfMonth()` — last day of month
- `$d->toTimestamp()` — convert to Unix timestamp
- `$d->toISO()` — convert to ISO 8601 string
- `$d->timezone($tz)` — convert to specified timezone

**Timezone constants:**
`TZ_UTC`, `TZ_LAGOS`, `TZ_BERLIN`, `TZ_LONDON`, `TZ_NEW_YORK`, `TZ_DUBAI`

---

### STDLIB-05 — Security & Crypto Functions

**Missing from current stdlib:**

- `sha256($str)` — SHA-256 hash
- `sha512($str)` — SHA-512 hash
- `hash($algo, $str)` — hash with any algorithm
- `hash_hmac($algo, $str, $key)` — HMAC signing
- `hash_equals($known, $user)` — timing-safe string comparison
- `password_hash($password)` — bcrypt hash (default cost)
- `password_hash($password, $cost)` — bcrypt with custom cost
- `password_verify($password, $hash)` — verify bcrypt password
- `password_needs_rehash($hash)` — true if cost factor outdated
- `jwt_encode($payload, $secret)` — sign JWT
- `jwt_encode($payload, $secret, $exp)` — sign JWT with expiry
- `jwt_decode($token, $secret)` — verify and decode JWT, false on failure
- `jwt_decode_unverified($token)` — decode without verification
- `jwt_expired($token)` — true if token is expired
- `encrypt($data, $key)` — AES-256-GCM encrypt
- `decrypt($data, $key)` — AES-256-GCM decrypt
- `encrypt_asymmetric($data, $pubkey)` — RSA encrypt
- `decrypt_asymmetric($data, $privkey)` — RSA decrypt
- `rand_bytes($length)` — cryptographically secure random bytes
- `rand_hex($length)` — random hex string of specified length
- `rand_token()` — URL-safe 32-byte random token
- `sign($data, $privkey)` — RSA/ECDSA sign data
- `verify($data, $signature, $pubkey)` — verify signature
- `base64_url_encode($str)` — URL-safe base64 encoding
- `base64_url_decode($str)` — URL-safe base64 decoding
- `hex_encode($str)` — encode bytes as hex string
- `hex_decode($str)` — decode hex string to bytes

---

### STDLIB-06 — HTTP Client Functions

**Missing from current stdlib:**

- `http_get($url)` — GET request
- `http_get($url, $headers)` — GET with headers
- `http_post($url, $body)` — POST request
- `http_post($url, $body, $headers)` — POST with headers
- `http_put($url, $body, $headers)` — PUT request
- `http_delete($url, $headers)` — DELETE request
- `http_patch($url, $body, $headers)` — PATCH request
- `http_request($method, $url, $body, $headers)` — generic request

**HTTP response object:**

- `$res->status` — HTTP status code integer
- `$res->body()` — raw response body string
- `$res->json()` — parse body as JSON, returns map
- `$res->header($name)` — get response header value
- `$res->ok()` — true if status 200–299
- `$res->failed()` — true if status 400+

**URL functions:**

- `parse_url($url)` — parse URL into object with parts
- `parse_query($querystring)` — parse query string into map
- `build_query($map)` — map to URL query string
- `build_url($parts)` — assemble URL from parts map
- `urljoin($base, $path)` — safe URL joining

**Network utilities:**

- `get_client_ip($req)` — real client IP, handles proxies
- `is_valid_ip($str)` — true/false
- `is_valid_ipv4($str)` — true/false
- `is_valid_ipv6($str)` — true/false
- `dns_lookup($hostname)` — resolve hostname to IP
- `reverse_dns($ip)` — reverse DNS lookup to hostname
- `geoip_lookup($ip)` — returns `{country, region, city, lat, lng}`

---

### STDLIB-07 — File System Functions

**Missing from current stdlib:**

- `file_read($path)` — read entire file as string
- `file_write($path, $data)` — write string to file
- `file_append($path, $data)` — append data to file
- `file_read_lines($path)` — read file as array of lines
- `file_write_lines($path, $lines)` — write array of lines to file
- `file_exists($path)` — true/false
- `file_size($path)` — file size in bytes
- `file_extension($path)` — extension without dot, e.g. `"jpg"`
- `file_basename($path)` — filename with extension
- `file_dirname($path)` — parent directory path
- `file_mime($path)` — MIME type string
- `file_modified_at($path)` — last modified timestamp
- `file_delete($path)` — delete file
- `file_copy($src, $dest)` — copy file
- `file_move($src, $dest)` — move or rename file
- `file_chmod($path, $mode)` — change file permissions
- `make_dir($path)` — create directory recursively
- `remove_dir($path)` — delete directory and contents
- `scan_dir($path)` — list files in directory
- `scan_dir_recursive($path)` — list all files recursively
- `dir_exists($path)` — true/false
- `app_root()` — project root path
- `temp_dir()` — system temp directory path
- `temp_file()` — create temp file, return path
- `path_join(...$parts)` — safe path joining
- `path_resolve($path)` — resolve relative to absolute path
- `path_relative($from, $to)` — relative path between two locations

---

### STDLIB-08 — Validation Functions

**Missing from current stdlib — the `validate()` function and rules engine:**

- `validate($data, $rules)` — main validator, returns validator object
- `$v->passes()` — true if all rules passed
- `$v->errors()` — map of field => error message array
- `$v->validated()` — returns only validated and sanitised fields

**Validation rules to implement:**

`required`, `nullable`, `string`, `integer`, `float`, `bool`, `array`,
`email`, `url`, `phone`, `uuid`, `date`, `date_format:Y-m-d`,
`min:n`, `max:n`, `between:n,m`, `in:a,b,c`, `not_in:a,b,c`,
`regex:/pattern/`, `confirmed`, `unique:table,column`, `exists:table,column`,
`file`, `image`, `max_size:5mb`, `mime:jpg,png`, `digits:n`,
`digits_between:n,m`, `alpha`, `alpha_num`, `alpha_dash`

**Standalone validators:**

- `is_email($str)` — true/false
- `is_url($str)` — true/false
- `is_phone($str)` — true/false
- `is_uuid($str)` — true/false
- `is_int($val)` — true/false
- `is_float($val)` — true/false
- `is_string($val)` — true/false
- `is_array($val)` — true/false
- `is_null($val)` — true/false
- `is_bool($val)` — true/false
- `is_numeric($str)` — true/false, "42" is numeric
- `is_json($str)` — true/false

---

### STDLIB-09 — Type Casting Functions

**Missing from current stdlib:**

- `intval($val)` — cast to integer
- `floatval($val)` — cast to float
- `strval($val)` — cast to string
- `boolval($val)` — cast to boolean
- `intdiv($a, $b)` — integer division, 7/2 → 3
- `gettype($val)` — returns type name as string
- `settype(&$val, $type)` — mutate variable type in-place

---

### STDLIB-10 — Wolf-Specific Functions (No PHP Equivalent)

**ID generation:**

- `uuid_v4()` — random UUID v4
- `uuid_v7()` — time-ordered UUID v7, better for DB indexes
- `nanoid()` — URL-safe compact ID
- `nanoid($size)` — custom size nanoid
- `snowflake_id()` — Twitter-style distributed integer ID
- `custom_id($prefix, $entropy)` — Stripe-style prefixed ID e.g. `usr_x7k2m9`

**Environment:**

- `env($key)` — get environment variable
- `env($key, $default)` — get with default value
- `env_required($key)` — get or throw if missing
- `is_production()` — true if APP_ENV=production
- `is_development()` — true if APP_ENV=development
- `is_testing()` — true if APP_ENV=test

**Functional utilities:**

- `pipeline($value)->through($fn)->result()` — chainable value transformation
- `retry($n, $fn)` — retry function up to N times
- `retry($n, $fn, $delayMs)` — retry with delay between attempts
- `memoize($fn)` — cache function result in memory for the process lifetime
- `rate_limit($key, $max, $window)` — in-memory rate limiter, returns limiter object
- `$limiter->exceeded()` — true if limit hit
- `$limiter->hit()` — record an attempt

**String utilities:**

- `slug($str)` — convert string to URL slug
- `truncate($str, $len)` — truncate with `...` suffix
- `truncate($str, $len, $suffix)` — truncate with custom suffix
- `pluralise($word, $count)` — pluralise word based on count, handles irregulars

**Money (decimal-safe — no float errors):**

- `money_format($amount, $currency)` — format with currency symbol
- `money_add($a, $b)` — safe decimal addition
- `money_subtract($a, $b)` — safe decimal subtraction
- `money_multiply($amount, $factor)` — safe decimal multiply
- `money_divide($amount, $divisor)` — safe decimal division
- `money_percentage($amount, $pct)` — calculate percentage of amount

**Output & debugging:**

- `log_info($msg)` — tagged info log
- `log_warning($msg)` — tagged warning log
- `log_error($msg)` — tagged error log
- `log_debug($msg)` — debug log, only output with `--debug` flag
- `dump($val)` — pretty-print any value to stdout
- `dd($val)` — dump and die
- `inspect($val)` — returns formatted string representation of any value
- `json_pretty($data)` — JSON encode with human-readable formatting

---

## Phase 2 — Language Features

These extend the Wolf language itself. Begin after stdlib is complete.

---

### LANG-01 — @supervise Block

Fault tolerance construct. Inspired by Elixir OTP supervision trees.

```wolf
@supervise {
    # If this block crashes, Wolf restarts it automatically
}

@supervise(strategy: "one_for_one") { }
@supervise(strategy: "one_for_all") { }
@supervise(restart: "exponential", max: 5) { }
```

Strategies: `one_for_one` (restart only crashed process), `one_for_all` (restart all if one crashes).
`restart: "exponential"` with `max:` caps retry attempts before giving up.

---

### LANG-02 — @safe Block

Error-safe execution. Returns false if block throws instead of propagating.

```wolf
$result = @safe { riskyOperation() }
if !$result { return $this->res->error("Operation failed", 500) }
```

---

### LANG-03 — @contract Block

API contract verification in CI. Define what endpoints guarantee and Wolf checks them automatically.

```wolf
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
    response_time_ms: 100
}

@contract POST /auth/login {
    requires: [
        "username": "string",
        "password": "string",
    ]
    returns_on_success: { "status": true, "data": { "token": "string" } }
    returns_on_failure: { "status": false, "message": "string" }
    rate_limit: "5 per 60s"
}
```

---

### LANG-04 — Enum Support

Top Go developer complaint. Wolf adds native enum types.

```wolf
enum RideStatus {
    PENDING
    ACTIVE
    COMPLETED
    CANCELLED
}

enum Currency {
    NGN = "NGN"
    USD = "USD"
    EUR = "EUR"
}

$status = RideStatus::PENDING
if $ride->status == RideStatus::COMPLETED { }
```

---

### LANG-05 — Built-in Pub/Sub

Internal event system without an external broker for simple cases.

```wolf
publish("rides.created", $rideData)
publish("payments.completed", $paymentData)

subscribe("rides.created", fn($event) {
    $notification = new NotificationService
    $notification->send($event->user_id, "ride_created", $event)
})
```

Driver is a config option: `redis` (default), `kafka`, `rabbitmq`, `sqs`.
Same `publish`/`subscribe` API regardless of driver — switching is one line in `.env`.

---

### LANG-06 — WebSocket Support

Built-in WebSocket server. No external library.

```wolf
class LiveController extends Controller {
    func connect() {
        @guard("auth")
        $userId = $this->req->user->user_id

        $this->res->websocket(fn($socket) {
            $socket->join("user:{$userId}")
            $socket->join("global:announcements")

            $socket->on("message", fn($data) {
                @validate($data, ["room" => "required|string", "content" => "required|string|max:1000"])
                broadcast($clean["room"], "message", ["from" => $userId, "content" => $clean["content"]])
            })

            $socket->on("connect",    fn() { presence_track($userId) })
            $socket->on("disconnect", fn() { presence_untrack($userId) })
        })
    }
}
```

**Broadcast functions:**

- `broadcast($room, $event, $data)` — send to all in room
- `broadcast_to_all($event, $data)` — send to all connected clients
- `presence_track($userId)` — mark user as online
- `presence_untrack($userId)` — mark user as offline
- `presence_list($room)` — list of online user IDs in room
- `presence_count($room)` — count of online users in room

---

### LANG-07 — GraphQL Auto-generation

Generate GraphQL schema from annotated model methods. No separate schema file.

```wolf
@graphql(type: "Ride", fields: ["ride_id", "pickup", "dropoff", "status", "fare"])
func findById($id) { }

@graphql(type: "[Ride]", args: {userId: "String!"})
func getByUser($userId) { }
```

Enable with `define("GRAPHQL_ENABLED", true)` in config. Wolf generates schema and resolver wiring automatically.

---

### LANG-08 — gRPC Support

Proto-like service definitions. Wolf generates client/server stubs.

```wolf
@grpc service RideService {
    rpc GetRide(RideRequest) returns (RideResponse)
    rpc CreateRide(CreateRideRequest) returns (RideResponse)
    rpc StreamRideUpdates(RideRequest) returns (stream RideUpdate)
}
```

---

## Phase 3 — Database Layer

Extends `$this->db->` beyond raw queries.

---

### DB-01 — Query Builder

Dynamic query construction without string concatenation.

```wolf
$query = $this->db->builder("rides")
if $status   { $query->where("status", $status) }
if $zone     { $query->where("zone", $zone) }
if $fromDate { $query->whereGte("created_at", $fromDate) }
$query->orderBy("created_at", "desc")
$query->limit(20)
$rides = $query->get()
```

---

### DB-02 — Eager Loading

Load relations in a fixed number of queries, not N+1.

```wolf
$orders = $this->db->with("items", "user")->query("SELECT * FROM orders")
// 3 queries total regardless of result count
```

---

### DB-03 — N+1 Detection

Compiler warning when a query is detected inside a loop.

```wolf
foreach $orders as $order {
    $items = $order->getItems()
    // wolf: warning: possible N+1 query — consider eager loading
}
```

---

## Phase 4 — Tooling

CLI tools. Begin after language features are stable.

---

### TOOL-01 — wolf dev

Start the entire project stack with one command. Zero config. Zero Docker knowledge required.

```bash
wolf dev
wolf dev --watch
wolf dev --port 9000
```

Reads `config/config.wolf`, detects required services (MySQL, Redis, etc), starts them, streams unified logs with service labels. New dev joins → `git clone` → `cp .env.example .env` → `wolf dev`. Done.

---

### TOOL-02 — wolf new (Enhanced)

Interactive project scaffold with feature selection.

Features available for selection: `auth`, `oauth`, `roles`, `kyc`, `wallet`, `transactions`,
`subscriptions`, `splits`, `listings`, `orders`, `reviews`, `media`, `notifications`,
`messaging`, `webhooks`, `audit`, `admin`, `analytics`, `jobs`.

Each selected feature generates controllers, models, migrations, and services with best practices pre-applied. Features are generation recipes — no runtime `Feature.wolf` files exist after scaffolding.

Best practices applied automatically by feature:
- Auth: rate limiting on all endpoints, audit trail, timing-safe forgot-password
- Wallet: idempotency, DB transactions, circuit breaker on payment provider
- KYC: `@encrypted` on document fields, file validation, state machine
- All: `@validate` on every POST/PUT, pagination on all list endpoints

---

### TOOL-03 — wolf generate

Add features to an existing project at any time.

```bash
wolf generate feature Notifications
wolf generate feature KYC
```

Produces identical output to selecting the feature in `wolf new`. Updates `wolf.mod` scaffolded list.

---

### TOOL-04 — wolf test

Built-in test runner. No external library, no configuration.

```bash
wolf test
wolf test --file AuthController
wolf test --watch
wolf test --coverage
```

Built-in: HTTP test client (`wolf_test_post`, `wolf_test_get`), database mocking (`mock(Database)`), assertions (`assert_status`, `assert_json`, `assert_has_key`, `assert_called`), JWT test helper (`test_jwt_token()`).

---

### TOOL-05 — wolf migrate

Database migration management.

```bash
wolf migrate up       # run pending migrations
wolf migrate down     # rollback last migration
wolf migrate fresh    # wipe and re-run all migrations
wolf migrate status   # show current migration state
```

---

### TOOL-06 — wolf docker init

Generate a production-quality `docker-compose.yml` from `config/config.wolf`. Never write YAML manually.

```bash
wolf docker init      # generate docker-compose.yml
wolf docker up        # docker compose up -d
wolf docker down      # docker compose down
wolf docker logs      # docker compose logs -f
wolf docker reset     # stop, wipe volumes, restart fresh
```

Regenerating after config changes updates the compose file correctly. Adding a new service (e.g. MongoDB) to config and running `wolf docker init` adds it with correct health checks and volumes.

---

### TOOL-07 — wolf explain

AI-powered error explanation. Analyses error message against project codebase, identifies likely source, suggests fix, links documentation.

```bash
wolf explain "Deadlock found when trying to get lock"
```

---

### TOOL-08 — wolf profile

Production profiler. No APM subscription, no external agent, no code changes required.

```bash
wolf profile --endpoint "/rides/estimate" --duration 60s
```

Shows P50/P95/P99 response times, time breakdown by component (DB, ML, serialisation), slowest query with index suggestion.

---

### TOOL-09 — wolf deploy --hot

Zero-downtime hot code reload. Routes new requests to new version, waits for in-flight requests to complete, retires old version. Under 100ms total. Zero dropped requests.

```bash
wolf deploy --hot
```

---

## Phase 5 — Observability

---

### OBS-01 — Built-in Metrics Functions

No Prometheus setup required.

- `metrics_increment($name)` — increment a counter
- `metrics_gauge($name, $value)` — set a gauge value
- `metrics_histogram($name, $value)` — record histogram observation
- `metrics_counter($name)` — alias for increment

---

### OBS-02 — Automatic /health Endpoint

Every Wolf app exposes `GET /health` with no configuration:

```json
{
  "status": "healthy",
  "uptime": 3600,
  "memory_mb": 45,
  "db_connected": true,
  "redis_connected": true,
  "version": "1.0.0"
}
```

---

### OBS-03 — wolf dev Dashboard

Available at `localhost:8081` during `wolf dev`. Real-time display of: request rates, response time P50/P95/P99, error rates, active connections, DB query times, cache hit/miss ratio. No external tool, no configuration.

---

### OBS-04 — OpenTelemetry Export

Export traces and metrics to external systems (Jaeger, Grafana, Datadog). One config line to enable.

---

## Phase 6 — Security Defaults

---

### SEC-01 — Automatic Security Headers

Every Wolf response includes these headers by default, zero configuration:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy: default-src 'self'`

---

### SEC-02 — SQL Injection Compiler Warning

Wolf detects string interpolation inside `query()` calls at compile time and refuses to compile.

```wolf
$this->db->query("SELECT * FROM users WHERE id = {$id}")
// wolf: error: potential SQL injection — use bind() instead
```

---

### SEC-03 — Secret Redaction in Logs

- `@encrypted` values are never logged
- JWT tokens in logs truncated to first 10 characters
- Common secret patterns (keys, tokens, passwords) redacted in error output

---

## Phase 7 — Package Manager (Post-Launch)

---

### PKG-01 — wolf.mod Module File

Identifies the module. Exists from day one. No package downloading in v0.1.

```wolf
module my-app
wolf   1.0.0
author Loneewolf
license MIT
```

---

### PKG-02 — wolfpkg CLI (v0.2.0, 3–6 months post launch)

```bash
wolfpkg install stripe-wolf
wolfpkg publish
wolfpkg search payment
wolfpkg info stripe-wolf
```

Generates `wolf.lock` with checksums. No install scripts — packages are Wolf code only. Strict single version per package, no silent duplication.

---

### PKG-03 — wolfpkg.dev Registry (v0.3.0, 6–12 months post launch)

Public registry. Community publishing. Permanent package mirroring so deleted packages never break builds.

---

## Implementation Order Summary

```
1. STDLIB-01  String functions
2. STDLIB-02  Array functions
3. STDLIB-03  Math functions
4. STDLIB-04  Date & time functions
5. STDLIB-05  Security & crypto functions
6. STDLIB-06  HTTP client functions
7. STDLIB-07  File system functions
8. STDLIB-08  Validation functions + rules engine
9. STDLIB-09  Type casting functions
10. STDLIB-10 Wolf-specific functions (IDs, money, env, pipeline, slug)

11. DB-01     Query builder
12. DB-02     Eager loading
13. DB-03     N+1 detection

14. LANG-01   @supervise block
15. LANG-02   @safe block
16. LANG-03   @contract block
17. LANG-04   Enums
18. LANG-05   Built-in pub/sub
19. LANG-06   WebSocket support
20. LANG-07   GraphQL auto-generation
21. LANG-08   gRPC support

22. TOOL-01   wolf dev
23. TOOL-02   wolf new (enhanced with feature selection)
24. TOOL-03   wolf generate
25. TOOL-04   wolf test
26. TOOL-05   wolf migrate
27. TOOL-06   wolf docker init
28. TOOL-07   wolf explain
29. TOOL-08   wolf profile
30. TOOL-09   wolf deploy --hot

31. OBS-01    Built-in metrics functions
32. OBS-02    Automatic /health endpoint
33. OBS-03    wolf dev dashboard
34. OBS-04    OpenTelemetry export

35. SEC-01    Automatic security headers
36. SEC-02    SQL injection compiler warning
37. SEC-03    Secret redaction in logs

38. PKG-01    wolf.mod
39. PKG-02    wolfpkg CLI
40. PKG-03    wolfpkg.dev registry
```