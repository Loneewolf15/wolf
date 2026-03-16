<?php

/**
 * Search Controller
 * 
 * Universal search endpoints
 */
class Search extends Controller
{
    private $searchService;

    public function __construct()
    {
        $this->searchService = new SearchService();
    }

    /**
     * GET /v1/search - Universal search
     * 
     * Query params:
     * - q: Search query (required)
     * - types: Comma-separated entity types (optional, default: all)
     * - mode: 'natural' or 'boolean' (optional, default: natural)
     * - limit: Results per entity (optional, default: 20, max: 100)
     * - offset: Pagination offset (optional, default: 0)
     */
    public function index()
    {
        $query = $this->getData('q');

        if (empty($query)) {
            return $this->sendResponse(false, 'Search query is required', [], 400);
        }

        // Minimum query length
        if (strlen($query) < 2) {
            return $this->sendResponse(false, 'Query must be at least 2 characters', [], 400);
        }

        $options = [
            'types' => $this->getData('types') ? explode(',', $this->getData('types')) : null,
            'mode' => $this->getData('mode') === 'boolean' ? 'BOOLEAN' : 'NATURAL LANGUAGE',
            'limit' => min((int)($this->getData('limit') ?? 20), 100),
            'offset' => (int)($this->getData('offset') ?? 0)
        ];

        $results = $this->searchService->search($query, $options);

        return $this->sendResponse(true, 'Search completed', $results);
    }

    /**
     * GET /v1/search/users - Search users specifically
     */
    public function users()
    {
        $query = $this->getData('q');

        if (empty($query)) {
            return $this->sendResponse(false, 'Search query is required', [], 400);
        }

        $limit = min((int)($this->getData('limit') ?? 20), 100);
        $offset = (int)($this->getData('offset') ?? 0);

        $results = $this->searchService->searchUsers($query, $limit, $offset);

        return $this->sendResponse(true, 'User search completed', [
            'query' => $query,
            'total' => count($results),
            'users' => $results
        ]);
    }

    /**
     * GET /v1/search/suggestions - Get autocomplete suggestions
     */
    public function suggestions()
    {
        $query = $this->getData('q');

        if (empty($query)) {
            return $this->sendResponse(false, 'Query is required', [], 400);
        }

        $limit = min((int)($this->getData('limit') ?? 10), 20);

        $suggestions = $this->searchService->getSuggestions($query, $limit);

        return $this->sendResponse(true, 'Suggestions retrieved', [
            'query' => $query,
            'suggestions' => $suggestions
        ]);
    }

    /**
     * GET /v1/search/popular - Get popular searches
     */
    public function popular()
    {
        $limit = min((int)($this->getData('limit') ?? 10), 50);

        $popular = $this->searchService->getPopularSearches($limit);

        return $this->sendResponse(true, 'Popular searches retrieved', [
            'searches' => $popular
        ]);
    }

    /**
     * GET /v1/search/history - Get user's search history
     */
    public function history()
    {
        $user = $this->RouteProtection();

        $limit = min((int)($this->getData('limit') ?? 10), 50);

        $history = $this->searchService->getUserSearchHistory($user->user_id, $limit);

        return $this->sendResponse(true, 'Search history retrieved', [
            'history' => $history
        ]);
    }

    /**
     * GET /v1/admin/search/analytics - Search analytics (admin)
     */
    public function analytics()
    {
        $user = $this->RouteProtection();

        // TODO: Check if admin

        $dateFrom = $this->getData('date_from');
        $dateTo = $this->getData('date_to');

        $analytics = $this->searchService->getSearchAnalytics($dateFrom, $dateTo);

        return $this->sendResponse(true, 'Search analytics retrieved', $analytics);
    }
}
