# Cache System Guide

## Overview

Your TraversyMVC backend uses a **file-based caching system** for high-performance data access. This guide explains how the cache works, patterns in use, and best practices.

---

## Cache Architecture

### File-Based Storage

**Location**: `app/cache/`
**Class**: `app/libraries/Cache.php`

```
app/cache/
├── <sha1_hash>.cache  # Serialized data with TTL
├── <sha1_hash>.cache
└── <sha1_hash>.cache
```

Each cache entry is stored as a separate file named after the SHA1 hash of the cache key.

---

## Core Operations

### 1. SET - Store Data

```php
$cache = new Cache();
$cache->set('user:123', $userData, 3600); // TTL = 1 hour (3600 seconds)
```

**What happens**:
1. Data is serialized with expiration timestamp
2. Saved to `app/cache/<sha1(key)>.cache`
3. TTL determines when data expires

### 2. GET - Retrieve Data

```php
$userData = $cache->get('user:123');
if ($userData !== false) {
    // Cache hit - use data
} else {
    // Cache miss - query database
}
```

**What happens**:
1. Reads cache file
2. Checks if current time < expiration
3. Returns data OR `false` if expired/missing
4. Automatically deletes expired files

### 3. DELETE - Invalidate Cache

```php
$cache->del('user:123'); // Remove from cache
```

**Use cases**:
- User updates profile → invalidate user cache
- User logs out → remove session data
- Data changed → clear stale cache

### 4. EXISTS - Check Without Reading

```php
if ($cache->exists('user:123')) {
    // Cache entry exists and not expired
}
```

### 5. INCREMENT - Atomic Counter

```php
$cache->increment('login_attempts:user@example.com', 1);
```

**Use cases**:
- Rate limiting counters
- Statistics tracking
- View counts

---

## Cache Patterns

### Pattern 1: Cache-Aside (Read-Through)

**Most common pattern** - Check cache first, query database on miss

```php
// Example from Users controller
public function getUser() {
    $userId = $this->RouteProtection()->user_id;
    
    // 1. Check cache
    $cacheKey = "user:id:{$userId}";
    $user = $this->cache->get($cacheKey);
    
    if ($user !== false) {
        // Cache hit
        return $this->sendResponse(true, 'User retrieved', $user);
    }
    
    // 2. Cache miss - query database
    $user = $this->userModel->findUserById($userId);
    
    // 3. Store in cache for future requests
    $this->cache->set($cacheKey, $user, 3600);
    
    return $this->sendResponse(true, 'User retrieved', $user);
}
```

**Performance**: First request ~10ms (DB), subsequent requests ~0.1ms (cache) = **100x faster**

---

### Pattern 2: Write-Through

**Update cache immediately after database write**

```php
// Example: Update user profile
public function updateProfile() {
    $userId = $this->RouteProtection()->user_id;
    $data = $this->getData();
    
    // 1. Update database
    $this->userModel->updateProfile($data);
    
    // 2. Immediately update cache
    $cacheKey = "user:id:{$userId}";
    $updatedUser = $this->userModel->findUserById($userId);
    $this->cache->set($cacheKey, $updatedUser, 3600);
    
    return $this->sendResponse(true, 'Profile updated');
}
```

**Benefit**: No stale data - cache always reflects database state

---

### Pattern 3: Cache Invalidation

**Remove cache when data changes**

```php
private function invalidateUserCaches($email = null, $userId = null) {
    if ($email) {
        $this->cache->del("user:email:{$email}");
        $this->cache->del("email_exists:{$email}");
    }
    if ($userId) {
        $this->cache->del("user:id:{$userId}");
    }
}
```

**When to invalidate**:
- User logs out → clear session
- Profile updated → clear user cache
- Delete operation → clear all related caches

---

### Pattern 4: Rate Limiting with Cache

**Track request attempts using cache counters**

```php
// Login rate limiting example
public function loginfunc() {
    $username = $postData['username'];
    
    // Check for account lockout
    $lockoutKey = "login_lockout:{$username}";
    if ($this->cache->exists($lockoutKey)) {
        return $this->sendResponse(false, 'Account locked', [], 429);
    }
    
    // Track failed attempts
    $attemptsKey = "login_attempts:{$username}";
    $attempts = (int)$this->cache->get($attemptsKey);
    
    if ($attempts >= 5) {
        // Lock account for 15 minutes
        $this->cache->set($lockoutKey, 1, 900);
        return $this->sendResponse(false, 'Too many attempts', [], 429);
    }
    
    // Attempt login...
    if (!$validCredentials) {
        // Increment attempts, expire after 1 hour
        $this->cache->increment($attemptsKey, 1);
        $this->cache->expire($attemptsKey, 3600);
    } else {
        // Success - clear attempts
        $this->cache->del($attemptsKey);
    }
}
```

---

## Cache Key Naming Conventions

**Use consistent, descriptive keys**:

| Entity | Pattern | Example | TTL |
|--------|---------|---------|-----|
| User by ID | `user:id:{id}` | `user:id:123` | 3600s (1h) |
| User by email | `user:email:{email}` | `user:email:test@example.com` | 3600s |
| Email existence | `email_exists:{email}` | `email_exists:test@example.com` | 3600s |
| Phone existence | `phone_exists:{phone}` | `phone_exists:+1234567890` | 3600s |
| Referral code | `referral:{code}` | `referral:ABC123` | 3600s |
| Login attempts | `login_attempts:{identifier}` | `login_attempts:user@example.com` | 3600s |
| Account lockout | `login_lockout:{identifier}` | `login_lockout:user@example.com` | 900s (15m) |

---

## TTL Guidelines

**How long should data be cached?**

| Data Type | Recommended TTL | Reason |
|-----------|-----------------|--------|
| User profile | 1 hour (3600s) | Changes infrequently |
| Email/phone checks | 1 hour | Registration validation |
| Rate limits | 1 hour | Security window |
| Account lockouts | 15 min (900s) | Temporary security measure |
| Session data | 24 hours (86400s) | User convenience |
| Static config | 1 day | Rarely changes |

---

## Performance Metrics

### Expected Performance

| Operation | Without Cache | With Cache | Improvement |
|-----------|---------------|------------|-------------|
| User lookup by email | 5-20ms | 0.1-0.5ms | **40x faster** |
| Email exists check | 3-10ms | 0.1ms | **50x faster** |
| Referral code lookup | 5-15ms | 0.1ms | **80x faster** |
| Rate limit check | 2-5ms | 0.1ms | **30x faster** |

### Cache Hit Ratio

**Target**: 80%+ hit rate

**Monitor with**:
```php
// Add to your monitoring/analytics
$hits = 0;
$misses = 0;

// In cache get method
if ($cachedData !== false) {
    $hits++;
} else {
    $misses++;
}

$hitRate = ($hits / ($hits + $misses)) * 100;
// Aim for 80%+
```

---

## Cache Warming

**Pre-populate cache for common queries**

```php
// Warm cache for frequently accessed users
public function warmUserCache($userIds) {
    foreach ($userIds as $userId) {
        $user = $this->userModel->findUserById($userId);
        $this->cache->set("user:id:{$userId}", $user, 3600);
    }
}
```

**When to warm**:
- Application startup
- After database migrations
- Scheduled background jobs

---

## Best Practices

### ✅ DO

1. **Use descriptive cache keys** - `user:id:123` not `u123`
2. **Set appropriate TTLs** - Balance freshness vs performance
3. **Invalidate on updates** - Keep cache consistent
4. **Handle cache misses** - Always have fallback to database
5. **Monitor hit rates** - Track cache effectiveness

### ❌ DON'T

1. **Don't cache everything** - Cache queries that are slow/frequent
2. **Don't use very long TTLs** - Risk stale data
3. **Don't forget invalidation** - Update cache when data changes
4. **Don't cache user-specific data globally** - Use unique keys per user
5. **Don't rely on cache** - Always fallback to database

---

## Migration to Redis (Future)

### When to Upgrade?

Upgrade to Redis when:
- **Load balancing** multiple servers (file cache not shared)
- **High traffic** (>10,000 requests/min)
- **Need advanced features** (pub/sub, sorted sets)

### Redis Migration

**1. Install Redis**:
```bash
sudo apt install redis-server
sudo systemctl start redis
```

**2. Create RedisCache class** (drop-in replacement):
```php
class RedisCache {
    private $redis;
    
    public function __construct() {
        $this->redis = new Redis();
        $this->redis->connect('127.0.0.1', 6379);
    }
    
    public function get($key) {
        $value = $this->redis->get($key);
        return $value !== false ? unserialize($value) : false;
    }
    
    public function set($key, $value, $ttl = 3600) {
        return $this->redis->setex($key, $ttl, serialize($value));
    }
    
    // Same interface as file cache...
}
```

**3. Update bootstrap.php**:
```php
// Replace Cache with RedisCache
// No code changes needed in controllers!
```

### Redis Benefits

- ✅ **Shared across servers** - Essential for load balancing
- ✅ **10-100x faster** than file cache
- ✅ **Built-in TTL** management
- ✅ **Atomic operations** - No race conditions
- ✅ **Pub/sub** for real-time features
- ✅ **Persistence** options

---

## Troubleshooting

### Issue: Low cache hit rate (<50%)

**Causes**:
- TTL too short
- Cache keys inconsistent
- High cache churn

**Solutions**:
- Increase TTL for stable data
- Standardize key naming
- Monitor which keys are accessed

### Issue: Stale data

**Causes**:
- Missing invalidation on updates
- TTL too long

**Solutions**:
- Add cache invalidation to update methods
- Reduce TTL
- Implement write-through pattern

### Issue: Disk space growing

**Causes**:
- Expired cache files not cleaned
- Too many unique cache keys

**Solutions**:
```php
// Clean expired files (run periodically)
function cleanExpiredCache() {
    $cacheDir = APPROOT . '/cache';
    $files = glob($cacheDir . '/*.cache');
    
    foreach ($files as $file) {
        $content = file_get_contents($file);
        $data = unserialize($content);
        
        if (time() >= $data['expires']) {
            unlink($file);
        }
    }
}
```

---

## Summary

Your file-based cache provides excellent performance for a single-server setup:

✅ **Simple** - No external dependencies  
✅ **Fast** - 40-100x faster than database  
✅ **Reliable** - Automatic expiration  
✅ **Production-ready** - Used in Users controller  

**Upgrade to Redis** when scaling horizontally with load balancing.
