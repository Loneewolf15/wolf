# Database and Redis 🐺

Wolf provides TraversyMVC-inspired database wrappers for familiar, safe, and powerful data access. Instead of raw drivers, Wolf wraps Go's `database/sql` and `go-redis/redis` for zero-friction backend development.

## Configuring Connections

Database connections are defined in your `config/wolf.config.json` file. The standard library automatically boots these configurations when you call `Wolf\Database.Connect()` or `Wolf\Redis.Connect()`.

**`wolf.config.json`**
```json
{
  "database": {
    "driver": "mysql",
    "host": "localhost",
    "port": 3306,
    "user": "root",
    "pass": "secret",
    "name": "wolfdb"
  },
  "redis": {
    "host": "localhost",
    "port": 6379,
    "pass": "",
    "db": 0
  }
}
```

## Relational Databases (MySQL, Postgres, SQLite)

Wolf's database library uses named parameters (`:id`, `:name`) to prevent SQL injection rather than raw positional arguments (`?`).

```wolf
# 1. Connect (reads from JSON config automatically)
Wolf\Database.Connect()

# 2. Prepare a query
Wolf\Database.Query("SELECT * FROM users WHERE status = :status AND age > :age")

# 3. Bind named parameters
Wolf\Database.Bind(":status", "active")
Wolf\Database.Bind(":age", 18)

# 4. Execute
$users = Wolf\Database.ResultSet()
foreach $users as $user {
    print("User: {$user['name']}")
}

# Fetching a single row
Wolf\Database.Query("SELECT * FROM users WHERE id = :id")
Wolf\Database.Bind(":id", 1)
$single = Wolf\Database.Single()

# Row count
$count = Wolf\Database.RowCount()
```

### Database Methods

| Method | Description |
|---|---|
| `Connect()` | Establishes the connection pool. |
| `Query(sql)` | Prepares an SQL statement for execution. |
| `Bind(param, val)` | Safely binds a value to a named parameter. |
| `Execute()` | Executes query without returning rows (INSERT, UPDATE, DELETE). |
| `ResultSet()` | Returns an array of associative arrays (all rows). |
| `Single()` | Returns a single associative array (one row). |
| `RowCount()` | Returns the number of rows affected by the last operation. |


## Redis

Wolf also includes native Redis bindings and a **Memory Mock** out-of-the-box for seamless testing without a live Redis server.

```wolf
# 1. Connect (reads config)
Wolf\Redis.Connect()

# 2. String operations
Wolf\Redis.Set("session:123", "active", 3600)  # expires in 3600s
$status = Wolf\Redis.Get("session:123")

# 3. Expiration and Deletion
Wolf\Redis.Expire("session:123", 60)
Wolf\Redis.Del("session:123")
```

### Hash Operations
Redis Hashes map perfectly to Wolf Maps.

```wolf
# Set multiple fields at once
Wolf\Redis.HSet("user:456", {"name": "Ghost", "level": "99"})

# Get a specific field
$name = Wolf\Redis.HGet("user:456", "name")

# Get all fields
$data = Wolf\Redis.HGetAll("user:456")
print("User level: {$data['level']}")
```

### Auto-Mocking
During testing or local development, if Redis is unreachable, the wrapper will log a warning and fallback gracefully if configured, or you can inject `MemoryRedis` explicitly in tests.
