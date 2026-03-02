# Search Service - Documentation

## Overview

Powerful search system using MySQL Full-Text Search with autocomplete, suggestions, and analytics.

---

## Features

✅ **Universal Search**: Search across all entities  
✅ **Full-Text Search**: MySQL FULLTEXT indexes  
✅ **Relevance Scoring**: Results ranked by relevance  
✅ **Autocomplete**: Real-time suggestions  
✅ **Boolean Search**: Advanced operators (+, -, "", *)  
✅ **Search History**: Track user searches  
✅ **Popular Searches**: Trending queries  
✅ **Analytics**: Search metrics and stats  

---

## Quick Start

### 1. Apply Database Migration

```bash
mysql -u root -p your_database < database/search_indexes.sql
```

This creates:
- FULLTEXT indexes on `users` table
- `search_logs` table
- `search_suggestions` table

### 2. Use Search Endpoints

```bash
# Universal search
GET /v1/search?q=john

# Search users
GET /v1/search/users?q=john

# Get autocomplete suggestions
GET /v1/search/suggestions?q=jo

# Popular searches
GET /v1/search/popular

# User search history
GET /v1/search/history
```

---

## API Endpoints

### 1. Universal Search

**`GET /v1/search`**

Search across all entities (users, listings, etc.)

**Query Parameters:**
- `q` - Search query (required, min 2 chars)
- `types` - Entity types (optional, e.g., "users,listings")
- `mode` - Search mode: `natural` or `boolean` (optional)
- `limit` - Results per entity (optional, default: 20, max: 100)
- `offset` - Pagination offset (optional)

**Example:**
```bash
curl "http://api.yourdomain.com/v1/search?q=john%20developer&limit=10"
```

**Response:**
```json
{
    "status": true,
    "data": {
        "query": "john developer",
        "results": {
            "users": [
                {
                    "id": 123,
                    "name": "John Smith",
                    "email": "john@example.com",
                    "bio": "Software developer",
                    "relevance": 2.5,
                    "entity_type": "user"
                }
            ]
        },
        "total": 1,
        "execution_time": "15.23ms"
    }
}
```

### 2. Search Users

**`GET /v1/search/users`**

Search specifically in users table.

**Example:**
```bash
curl "http://api.yourdomain.com/v1/search/users?q=john&limit=5"
```

### 3. Autocomplete Suggestions

**`GET /v1/search/suggestions`**

Get search suggestions as user types.

**Example:**
```bash
curl "http://api.yourdomain.com/v1/search/suggestions?q=jo"
```

**Response:**
```json
{
    "status": true,
    "data": {
        "query": "jo",
        "suggestions": [
            "john",
            "john developer",
            "jobs"
        ]
    }
}
```

### 4. Popular Searches

**`GET /v1/search/popular`**

Get trending/popular search queries.

**Example:**
```bash
curl "http://api.yourdomain.com/v1/search/popular?limit=10"
```

### 5. Search History

**`GET /v1/search/history`**

Get user's recent searches (requires authentication).

**Example:**
```bash
curl "http://api.yourdomain.com/v1/search/history" \
  -H "Authorization: Bearer JWT_TOKEN"
```

### 6. Search Analytics (Admin)

**`GET /v1/admin/search/analytics`**

Get search statistics and metrics.

**Query Parameters:**
- `date_from` - Start date (optional)
- `date_to` - End date (optional)

**Example:**
```bash
curl "http://api.yourdomain.com/v1/admin/search/analytics?date_from=2026-01-01" \
  -H "Authorization: Bearer ADMIN_JWT"
```

---

## Search Modes

### Natural Language Mode (Default)

Best for everyday searches. MySQL ranks results by relevance.

```bash
GET /v1/search?q=software developer&mode=natural
```

### Boolean Mode (Advanced)

Supports special operators:

| Operator | Meaning | Example |
|----------|---------|---------|
| `+` | Must include | `+developer` |
| `-` | Must exclude | `-junior` |
| `""` | Exact phrase | `"full stack"` |
| `*` | Wildcard | `dev*` (developer, development) |

**Example:**
```bash
# Must have "developer", exclude "junior", prefer "senior"
GET /v1/search?q=+developer -junior senior&mode=boolean
```

---

## Adding New Searchable Entities

To make more tables searchable:

### 1. Add FULLTEXT Index to Table

```sql
ALTER TABLE listings 
ADD FULLTEXT INDEX ft_listings_search (title, description, tags);
```

### 2. Add to SearchService

```php
// In SearchService.php, add to $searchableEntities array
private $searchableEntities = [
    'users' => [...],
    
    'listings' => [
        'table' => 'listings',
        'fields' => ['title', 'description', 'tags'],
        'select' => 'id, title, description, price, created_at',
        'type' => 'listing'
    ]
];
```

### 3. Create Specific Endpoint (Optional)

```php
// In Search controller
public function listings()
{
    $query = $this->getData('q');
    $results = $this->searchService->searchEntity('listings', $query, 'NATURAL LANGUAGE', 20, 0);
    return $this->sendResponse(true, 'Listings search', $results);
}
```

---

## Usage Examples

### Example 1: Search Users from Frontend

```javascript
// Autocomplete as user types
const searchInput = document.getElementById('search');

searchInput.addEventListener('input', async (e) => {
    const query = e.target.value;
    
    if (query.length >= 2) {
        const response = await fetch(`/v1/search/suggestions?q=${query}`);
        const data = await response.json();
        
        // Show suggestions dropdown
        showSuggestions(data.data.suggestions);
    }
});

// Full search on submit
async function search(query) {
    const response = await fetch(`/v1/search?q=${query}&types=users&limit=20`);
    const data = await response.json();
    
    displayResults(data.data.results.users);
}
```

### Example 2: Admin Search Dashboard

```php
// Get search analytics
$analytics = $searchService->getSearchAnalytics('2026-01-01', '2026-01-31');

echo "Total Searches: {$analytics['total_searches']}\n";
echo "Unique Users: {$analytics['unique_users']}\n";
echo "Avg Results: " . round($analytics['avg_results'], 2) . "\n";
echo "Avg Time: {$analytics['avg_time_ms']}ms\n";
```

### Example 3: Integrate Search in Existing Features

```php
class Users extends Controller
{
    public function find()
    {
        $searchService = new SearchService();
        $query = $this->getData('query');
        
        $users = $searchService->searchUsers($query, 10);
        
        return $this->sendResponse(true, 'Users found', $users);
    }
}
```

---

## Performance Optimization

### 1. FULLTEXT Index Best Practices

- **Minimum word length**: MySQL default is 4 chars (can be changed)
- **Stop words**: Common words like "the", "and" are ignored
- **Rebuild indexes**: `OPTIMIZE TABLE users;`

### 2. Query Optimization

```php
// ✅ Good - Specific entity search
$searchService->searchUsers('john', 10);

// ❌ Avoid - Universal search with too many results
$searchService->search('a', ['limit' => 1000]); // Slow!
```

### 3. Caching Popular Searches

```php
$cache = new Cache();
$cacheKey = 'popular_searches';

if ($cache->exists($cacheKey)) {
    $popular = $cache->get($cacheKey);
} else {
    $popular = $searchService->getPopularSearches(10);
    $cache->set($cacheKey, $popular, 3600); // 1 hour
}
```

---

## Monitoring

### Check Search Performance

```sql
-- Slow searches
SELECT query, AVG(execution_time_ms) as avg_time, COUNT(*) as count
FROM search_logs
GROUP BY query
HAVING avg_time > 100
ORDER BY avg_time DESC;
```

### Most Searched Terms

```sql
SELECT suggestion, search_count
FROM search_suggestions
ORDER BY search_count DESC
LIMIT 20;
```

### Zero Result Searches

```sql
SELECT query, COUNT(*) as count
FROM search_logs
WHERE results_count = 0
GROUP BY query
ORDER BY count DESC
LIMIT 20;
```

---

## Troubleshooting

### No Results Returned

**Check FULLTEXT index:**
```sql
SHOW INDEX FROM users WHERE Key_name LIKE 'ft_%';
```

**Rebuild index:**
```sql
ALTER TABLE users DROP INDEX ft_users_search;
ALTER TABLE users ADD FULLTEXT INDEX ft_users_search (name, email, bio);
```

### Query Too Short

MySQL FULLTEXT requires minimum word length (default: 4 chars).

**Fix:** Lower minimum in MySQL config
```ini
[mysqld]
ft_min_word_len=2
```

Then rebuild indexes: `OPTIMIZE TABLE users;`

---

## Summary

**Search Service provides:**
- ✅ Fast full-text search with MySQL
- ✅ Autocomplete & suggestions
- ✅ Search history & analytics
- ✅ Boolean operators
- ✅ Relevance scoring
- ✅ Multiple entity search

**Perfect for:**
- User directories
- Product catalogs
- Content platforms
- Knowledge bases

Your API now has powerful search! 🔍
