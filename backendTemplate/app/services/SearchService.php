<?php

/**
 * Search Service
 * 
 * Universal search across entities using MySQL Full-Text Search
 */
class SearchService
{
    private $db;
    private $logger;

    // Searchable entities configuration
    private $searchableEntities = [
        'users' => [
            'table' => 'users',
            'fields' => ['name', 'email', 'bio'],
            'select' => 'user_id as id, name, email, bio, profile_image, created_at',
            'type' => 'user'
        ],
        // Add more entities as needed
        // 'listings' => [
        //     'table' => 'listings',
        //     'fields' => ['title', 'description', 'tags'],
        //     'select' => 'id, title, description, price, created_at',
        //     'type' => 'listing'
        // ]
    ];

    public function __construct()
    {
        $this->db = new Database();
        $this->logger = new Logger();
    }

    /**
     * Universal search across all entities
     */
    public function search(string $query, array $options = []): array
    {
        $startTime = microtime(true);

        // Options
        $entityTypes = $options['types'] ?? array_keys($this->searchableEntities);
        $limit = min((int)($options['limit'] ?? 20), 100);
        $offset = (int)($options['offset'] ?? 0);
        $mode = $options['mode'] ?? 'NATURAL LANGUAGE'; // or 'BOOLEAN'

        $results = [];
        $totalResults = 0;

        foreach ($entityTypes as $entityType) {
            if (!isset($this->searchableEntities[$entityType])) {
                continue;
            }

            $entityResults = $this->searchEntity($entityType, $query, $mode, $limit, $offset);

            if (!empty($entityResults)) {
                $results[$entityType] = $entityResults;
                $totalResults += count($entityResults);
            }
        }

        $executionTime = microtime(true) - $startTime;

        // Log search
        $this->logSearch($query, implode(',', $entityTypes), $totalResults, $executionTime);

        // Store suggestion
        $this->storeSuggestion($query);

        return [
            'query' => $query,
            'results' => $results,
            'total' => $totalResults,
            'execution_time' => round($executionTime * 1000, 2) . 'ms'
        ];
    }

    /**
     * Search specific entity type
     */
    private function searchEntity(string $entityType, string $query, string $mode, int $limit, int $offset): array
    {
        $config = $this->searchableEntities[$entityType];

        $fields = implode(', ', $config['fields']);

        $sql = "SELECT {$config['select']},
                MATCH(" . $fields . ") AGAINST (:query IN {$mode} MODE) as relevance
                FROM {$config['table']}
                WHERE MATCH(" . $fields . ") AGAINST (:query IN {$mode} MODE)
                ORDER BY relevance DESC
                LIMIT :limit OFFSET :offset";

        $this->db->query($sql);
        $this->db->bind(':query', $query);
        $this->db->bind(':limit', $limit, PDO::PARAM_INT);
        $this->db->bind(':offset', $offset, PDO::PARAM_INT);

        $results = $this->db->resultSet();

        // Add entity type to each result
        foreach ($results as $result) {
            $result->entity_type = $config['type'];
        }

        return $results;
    }

    /**
     * Search users specifically
     */
    public function searchUsers(string $query, int $limit = 20, int $offset = 0): array
    {
        $this->db->query("SELECT user_id as id, name, email, bio, profile_image, created_at,
            MATCH(name, email, bio) AGAINST (:query IN NATURAL LANGUAGE MODE) as relevance
            FROM users
            WHERE MATCH(name, email, bio) AGAINST (:query IN NATURAL LANGUAGE MODE)
            ORDER BY relevance DESC
            LIMIT :limit OFFSET :offset");

        $this->db->bind(':query', $query);
        $this->db->bind(':limit', $limit, PDO::PARAM_INT);
        $this->db->bind(':offset', $offset, PDO::PARAM_INT);

        return $this->db->resultSet();
    }

    /**
     * Boolean search (advanced)
     * Supports: +word, -word, "exact phrase", word*
     */
    public function booleanSearch(string $entityType, string $query, int $limit = 20): array
    {
        if (!isset($this->searchableEntities[$entityType])) {
            return [];
        }

        return $this->searchEntity($entityType, $query, 'BOOLEAN', $limit, 0);
    }

    /**
     * Get autocomplete suggestions
     */
    public function getSuggestions(string $query, int $limit = 10): array
    {
        $this->db->query("SELECT suggestion, search_count 
            FROM search_suggestions 
            WHERE suggestion LIKE :query 
            ORDER BY search_count DESC, suggestion ASC 
            LIMIT :limit");

        $this->db->bind(':query', $query . '%');
        $this->db->bind(':limit', $limit, PDO::PARAM_INT);

        $results = $this->db->resultSet();

        return array_map(function ($row) {
            return $row->suggestion;
        }, $results);
    }

    /**
     * Get popular searches
     */
    public function getPopularSearches(int $limit = 10): array
    {
        $this->db->query("SELECT suggestion, search_count 
            FROM search_suggestions 
            ORDER BY search_count DESC 
            LIMIT :limit");

        $this->db->bind(':limit', $limit, PDO::PARAM_INT);

        return $this->db->resultSet();
    }

    /**
     * Get recent searches for user
     */
    public function getUserSearchHistory(int $userId, int $limit = 10): array
    {
        $this->db->query("SELECT DISTINCT query, created_at 
            FROM search_logs 
            WHERE user_id = :user_id 
            ORDER BY created_at DESC 
            LIMIT :limit");

        $this->db->bind(':user_id', $userId);
        $this->db->bind(':limit', $limit, PDO::PARAM_INT);

        return $this->db->resultSet();
    }

    /**
     * Log search query
     */
    private function logSearch(string $query, string $entityType, int $resultsCount, float $executionTime): void
    {
        $userId = null; // Get from session/auth if available

        $this->db->query("INSERT INTO search_logs 
            (user_id, query, entity_type, results_count, execution_time_ms) 
            VALUES (:user_id, :query, :entity_type, :results_count, :execution_time)");

        $this->db->bind(':user_id', $userId);
        $this->db->bind(':query', $query);
        $this->db->bind(':entity_type', $entityType);
        $this->db->bind(':results_count', $resultsCount);
        $this->db->bind(':execution_time', round($executionTime * 1000, 2));

        $this->db->execute();
    }

    /**
     * Store search suggestion
     */
    private function storeSuggestion(string $query): void
    {
        // Only store if query is meaningful (3+ chars)
        if (strlen($query) < 3) {
            return;
        }

        $query = strtolower(trim($query));

        $this->db->query("INSERT INTO search_suggestions (suggestion, search_count) 
            VALUES (:suggestion, 1) 
            ON DUPLICATE KEY UPDATE search_count = search_count + 1");

        $this->db->bind(':suggestion', $query);
        $this->db->execute();
    }

    /**
     * Get search analytics
     */
    public function getSearchAnalytics(string $dateFrom = null, string $dateTo = null): array
    {
        $sql = "SELECT 
            COUNT(*) as total_searches,
            COUNT(DISTINCT user_id) as unique_users,
            AVG(results_count) as avg_results,
            AVG(execution_time_ms) as avg_time_ms
            FROM search_logs
            WHERE 1=1";

        $bindings = [];

        if ($dateFrom) {
            $sql .= " AND created_at >= :date_from";
            $bindings[':date_from'] = $dateFrom;
        }

        if ($dateTo) {
            $sql .= " AND created_at <= :date_to";
            $bindings[':date_to'] = $dateTo;
        }

        $this->db->query($sql);

        foreach ($bindings as $key => $value) {
            $this->db->bind($key, $value);
        }

        return (array)$this->db->single();
    }
}
