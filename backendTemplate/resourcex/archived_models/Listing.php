<?php

class Listing
{
    private $db;

    public function __construct()
    {
        $this->db = new Database;
    }


    /**
     * Creates a new listing entry in the Listings table.
     * @param array $data An array containing listing details.
     * @return bool True on success, false on failure.
     */
  /**
 * Creates a new listing entry in the Listings table and its media.
 * @param array $listingData An array containing listing details.
 * @param array $mediaUrls An array of media objects with url and type.
 * @return bool True on success, false on failure.
 */
public function createListingAndMedia($listingData, $mediaUrls)
{
    // Start a transaction
    $this->db->beginTransaction();
    
    try {
        // 1. Insert the main listing data
        $this->db->query('INSERT INTO Listings (
            listing_id,
            user_id,
            title,
            description,
            price,
            category_id,
            is_resellable,
            reseller_commission_percent,
            is_physical,
            location,
            status
        ) VALUES (
            :listing_id,
            :user_id,
            :title,
            :description,
            :price,
            :category_id,
            :is_resellable,
            :reseller_commission_percent,
            :is_physical,
            :location,
            :status
        )');
        
        $this->db->bind(':listing_id', $listingData['listing_id']);
        $this->db->bind(':user_id', $listingData['user_id']);
        $this->db->bind(':title', $listingData['title']);
        $this->db->bind(':description', $listingData['description']);
        $this->db->bind(':price', $listingData['price']);
        $this->db->bind(':category_id', $listingData['category_id']);
        $this->db->bind(':is_resellable', $listingData['is_resellable']);
        $this->db->bind(':reseller_commission_percent', $listingData['reseller_commission_percent']);
        $this->db->bind(':is_physical', $listingData['is_physical']);
        $this->db->bind(':location', $listingData['location']);
        $this->db->bind(':status', $listingData['status']);
        
        $this->db->execute();
        
        // 2. Insert the media file URLs and check for failure
        if (!$this->saveListingMedia($listingData['listing_id'], $mediaUrls)) {
            // Throw an exception to trigger the catch block and rollback
            throw new Exception('Failed to save listing media.');
        }

        // Commit the transaction if both steps succeeded
        $this->db->commit();
        return true;
        
    } catch (PDOException $e) {
        // Rollback on any failure
        $this->db->rollBack();
        error_log('Failed to create listing and media: ' . $e->getMessage());
        return false;
    }
}

/**
 * Saves the media file URLs associated with a listing.
 * @param string $listingId The unique ID of the listing.
 * @param array $mediaUrls An array of media objects with url and type.
 * @return bool True on success, false on failure.
 */
public function saveListingMedia($listingId, $mediaUrls)
{
    // Fix: Ensure $mediaUrls is an array before processing
    if (empty($mediaUrls) || !is_array($mediaUrls)) {
        return true; // Nothing to save or invalid data
    }
    
    $sql = 'INSERT INTO ListingMedia (
        listing_id,
        media_url,
        media_type
    ) VALUES (
        :listing_id,
        :media_url,
        :media_type
    )';

    $this->db->query($sql);
    
    try {
        foreach ($mediaUrls as $media) {
            // Fix: Check if $media is a valid array/object before trying to access its offsets
            if (!is_array($media) || !isset($media['url']) || !isset($media['type'])) {
                continue; // Skip invalid entries
            }
            $this->db->bind(':listing_id', $listingId);
            $this->db->bind(':media_url', $media['url']);
            $this->db->bind(':media_type', $media['type']);
            $this->db->execute();
        }
        return true;
    } catch (PDOException $e) {
        error_log('Failed to save listing media: ' . $e->getMessage());
        return false;
    }
}

      // Check if the user is the owner of the listing
    public function isOwner($listingId, $userId)
    {
        $this->db->query('SELECT COUNT(*) AS count FROM Listings WHERE listing_id = :listing_id AND user_id = :user_id');
        $this->db->bind(':listing_id', $listingId);
        $this->db->bind(':user_id', $userId);
        $result = $this->db->single();
        return $result->count > 0;
    }

    // Get all listings
  public function getAllListings($page = 1, $pageSize = 20, $personalizationData = null)
{
    $offset = ($page - 1) * $pageSize;

    // Get total count for pagination
    $this->db->query('SELECT COUNT(*) as total FROM Listings WHERE status = "active"');
    $totalRows = $this->db->single()->total;

    $selectClause = 'SELECT
        l.listing_id, l.title, l.description, l.price, l.location, l.created_at, l.is_resellable,
        u.name AS seller_name, u.is_verified, u.profile_pic_url AS seller_profile_pic,
        c.name AS item_name,
        p.name AS category_name,
        g.name AS group_name,
        GROUP_CONCAT(lm.media_url) AS media_urls,
        COALESCE(AVG(r.rating), 0) AS average_rating';

    $fromClause = ' FROM Listings l
        JOIN initkey_rid u ON l.user_id = u.user_id
        LEFT JOIN Categories c ON l.category_id = c.category_id
        LEFT JOIN Categories p ON c.parent_id = p.category_id
        LEFT JOIN Categories g ON p.parent_id = g.category_id
        LEFT JOIN ListingMedia lm ON l.listing_id = lm.listing_id
        LEFT JOIN Reviews r ON l.listing_id = r.listing_id';
    
    $whereClause = ' WHERE l.status = "active"';
    $groupByClause = ' GROUP BY l.listing_id';
    $orderByClause = ' ORDER BY ';

    $bindings = [];

    if ($personalizationData && $personalizationData['user_id']) {
        $orderByClause .= 'CASE ';
        // Prioritize listings matching search history
        if (!empty($personalizationData['search_history'])) {
            $searchTerms = implode(' ', $personalizationData['search_history']);
            $selectClause .= ', MATCH(l.title, l.description) AGAINST (:search_history IN BOOLEAN MODE) as relevance';
            $bindings[':search_history'] = $searchTerms;
            $orderByClause .= 'WHEN MATCH(l.title, l.description) AGAINST (:search_history IN BOOLEAN MODE) THEN 1 ';
        }
        // Prioritize listings in the same LGA
        if ($personalizationData['lga']) {
            $orderByClause .= 'WHEN u.local_government_area = :lga THEN 2 ';
            $bindings[':lga'] = $personalizationData['lga'];
        }
        // Prioritize listings in the same state
        if ($personalizationData['state']) {
            $orderByClause .= 'WHEN u.state = :state THEN 3 ';
            $bindings[':state'] = $personalizationData['state'];
        }
        $orderByClause .= 'ELSE 4 END, ';
    }

    $orderByClause .= 'l.created_at DESC';

    $limitClause = ' LIMIT :limit OFFSET :offset';

    $this->db->query($selectClause . $fromClause . $whereClause . $groupByClause . $orderByClause . $limitClause);
    
    foreach ($bindings as $key => $value) {
        $this->db->bind($key, $value);
    }
    $this->db->bind(':limit', $pageSize, PDO::PARAM_INT);
    $this->db->bind(':offset', $offset, PDO::PARAM_INT);

    $rows = $this->db->resultSet();

    return [
        'data' => $rows,
        'pagination' => [
            'total_items' => $totalRows,
            'total_pages' => ceil($totalRows / $pageSize),
            'current_page' => $page,
            'page_size' => $pageSize
        ]
    ];
}

  // Update an existing listing
public function updateListing($data)
{
    // If the listing is not resellable, set the commission to 0
    if (!$data['is_resellable']) {
        $data['reseller_commission_percent'] = 0;
    }
    
    $this->db->query('UPDATE Listings SET
        title = :title,
        description = :description,
        price = :price,
        category_id = :category_id,
        is_resellable = :is_resellable,
        reseller_commission_percent = :reseller_commission_percent,
        is_physical = :is_physical,
        updated_at = NOW()
        WHERE listing_id = :listing_id
    ');

    $this->db->bind(':title', $data['title']);
    $this->db->bind(':description', $data['description']);
    $this->db->bind(':price', $data['price']);
    $this->db->bind(':category_id', $data['category_id']);
    $this->db->bind(':is_resellable', $data['is_resellable']);
    $this->db->bind(':reseller_commission_percent', $data['reseller_commission_percent']);
    $this->db->bind(':is_physical', $data['is_physical']);
    $this->db->bind(':listing_id', $data['listing_id']);

    return $this->db->execute();
}
public function getAllCategories()
    {
        $this->db->query('SELECT category_id, name, parent_id, icon_url FROM Categories ORDER BY name ASC');
        $rows = $this->db->resultSet();
        return $rows;
    }
    
public function getRelatedListings($categoryId, $excludeListingId)
{
    $this->db->query('SELECT
        l.listing_id, l.title, l.description, l.price, l.location, l.created_at,
        u.name AS seller_name, u.profile_pic_url AS seller_profile_pic,
        GROUP_CONCAT(lm.media_url) AS media_urls
        FROM Listings l
        JOIN initkey_rid u ON l.user_id = u.user_id
        LEFT JOIN ListingMedia lm ON l.listing_id = lm.listing_id
        WHERE l.status = "active" AND l.category_id = :category_id AND l.listing_id != :exclude_listing_id
        GROUP BY l.listing_id
        ORDER BY l.created_at DESC
        LIMIT 12'); // Limit the number of related listings for performance

    $this->db->bind(':category_id', $categoryId);
    $this->db->bind(':exclude_listing_id', $excludeListingId);

    $rows = $this->db->resultSet();
    return $rows;
}    

    // Get all listings for specific user
public function getUserListings($userId, $page = 1, $pageSize = 20)
{
    $offset = ($page - 1) * $pageSize;

    // Get total count for pagination
    $this->db->query('SELECT COUNT(*) as total FROM Listings WHERE user_id = :user_id');
    $this->db->bind(':user_id', $userId);
    $totalRows = $this->db->single()->total;

    $this->db->query('SELECT
        l.listing_id, l.title, l.description, l.price, l.location, l.created_at, l.status,
        u.name AS seller_name, u.is_verified, u.profile_pic_url AS seller_profile_pic,
        c.name AS category_name,
        GROUP_CONCAT(lm.media_url) AS media_urls
        FROM Listings l
        JOIN initkey_rid u ON l.user_id = u.user_id
        JOIN Categories c ON l.category_id = c.category_id
        LEFT JOIN ListingMedia lm ON l.listing_id = lm.listing_id
        WHERE l.user_id = :user_id
        GROUP BY l.listing_id
        ORDER BY l.created_at DESC
        LIMIT :limit OFFSET :offset');

    $this->db->bind(':user_id', $userId);
    $this->db->bind(':limit', $pageSize, PDO::PARAM_INT);
    $this->db->bind(':offset', $offset, PDO::PARAM_INT);

    $rows = $this->db->resultSet();

    return [
        'data' => $rows,
        'pagination' => [
            'total_items' => $totalRows,
            'total_pages' => ceil($totalRows / $pageSize),
            'current_page' => $page,
            'page_size' => $pageSize
        ]
    ];
}
    // Deletes a listing from the database
    public function deleteListing($listingId)
    {
        $this->db->query('DELETE FROM Listings WHERE listing_id = :listing_id');
        $this->db->bind(':listing_id', $listingId);
        return $this->db->execute();
    }
    // Deletes all media associated with a listing
    public function deleteListingMedia($listingId)
    {
        $this->db->query('DELETE FROM ListingMedia WHERE listing_id = :listing_id');
        $this->db->bind(':listing_id', $listingId);
        return $this->db->execute();
    }
 // Gets the file paths of all media for a listing
    public function getMediaFilePaths($listingId)
    {
        $this->db->query('SELECT media_url FROM ListingMedia WHERE listing_id = :listing_id');
        $this->db->bind(':listing_id', $listingId);
        $rows = $this->db->resultSet();
        return array_column($rows, 'media_url');
    }

   public function getSingleListing($listingId)
{
    $this->db->query('SELECT
        l.listing_id, l.title, l.description, l.price, l.location, l.created_at,
        u.name AS seller_name, u.is_verified, u.profile_pic_url AS seller_profile_pic,
        c.name AS category_name,
        GROUP_CONCAT(lm.media_url) AS media_urls
        FROM Listings l
        JOIN initkey_rid u ON l.user_id = u.user_id
        JOIN Categories c ON l.category_id = c.category_id
        LEFT JOIN ListingMedia lm ON l.listing_id = lm.listing_id
        WHERE l.listing_id = :listing_id
        GROUP BY l.listing_id
        LIMIT 1');

    $this->db->bind(':listing_id', $listingId);
    $row = $this->db->single();

    return $row;
}
public function searchListings($data)
{
    $page = $data['page'];
    $pageSize = $data['pageSize'];
    $offset = ($page - 1) * $pageSize;

    $baseSql = 'FROM Listings l
            JOIN initkey_rid u ON l.user_id = u.user_id
            JOIN Categories c ON l.category_id = c.category_id
            LEFT JOIN ListingMedia lm ON l.listing_id = lm.listing_id
            LEFT JOIN Reviews r ON l.listing_id = r.listing_id
            WHERE l.status = "active"';

    $bindings = [];
    $whereClause = '';

    if (!empty($data['searchTerm'])) {
        $whereClause .= ' AND (l.title LIKE :searchTerm OR l.description LIKE :searchTerm)';
        $bindings[':searchTerm'] = '%' . $data['searchTerm'] . '%';
    }
    if (!empty($data['categoryId'])) {
        $whereClause .= ' AND l.category_id = :categoryId';
        $bindings[':categoryId'] = $data['categoryId'];
    }
    if ($data['priceMin'] > 0) {
        $whereClause .= ' AND l.price >= :priceMin';
        $bindings[':priceMin'] = $data['priceMin'];
    }
    if ($data['priceMax'] > 0) {
        $whereClause .= ' AND l.price <= :priceMax';
        $bindings[':priceMax'] = $data['priceMax'];
    }
    if (!empty($data['location'])) {
        $whereClause .= ' AND l.location LIKE :location';
        $bindings[':location'] = '%' . $data['location'] . '%';
    }

    // Count query
    $countSql = 'SELECT COUNT(DISTINCT l.listing_id) as total ' . $baseSql . $whereClause;
    $this->db->query($countSql);
    foreach ($bindings as $key => $value) {
        $this->db->bind($key, $value);
    }
    $totalRows = $this->db->single()->total;

    // Main query
    $sql = 'SELECT
                l.listing_id, l.title, l.description, l.price, l.location, l.created_at, l.is_resellable,
                u.name AS seller_name, u.is_verified, u.profile_pic_url AS seller_profile_pic,
                c.name AS category_name,
                GROUP_CONCAT(lm.media_url) AS media_urls,
                COALESCE(AVG(r.rating), 0) AS average_rating ' . $baseSql . $whereClause;
    
    $sql .= ' GROUP BY l.listing_id';

    if ($data['rating'] > 0) {
        $sql .= ' HAVING COALESCE(AVG(r.rating), 0) >= :rating';
        $bindings[':rating'] = $data['rating'];
    }

    $orderBy = '';
    switch ($data['sortBy']) {
        case 'lowest_price': $orderBy = 'l.price ASC'; break;
        case 'highest_price': $orderBy = 'l.price DESC'; break;
        case 'popularity': $orderBy = 'average_rating DESC'; break;
        case 'newest': default: $orderBy = 'l.created_at DESC'; break;
    }
    $sql .= ' ORDER BY ' . $orderBy;
    $sql .= ' LIMIT :limit OFFSET :offset';

    $this->db->query($sql);
    foreach ($bindings as $key => $value) {
        $this->db->bind($key, $value);
    }
    $this->db->bind(':limit', $pageSize, PDO::PARAM_INT);
    $this->db->bind(':offset', $offset, PDO::PARAM_INT);
    
    $rows = $this->db->resultSet();

    return [
        'data' => $rows,
        'pagination' => [
            'total_items' => $totalRows,
            'total_pages' => ceil($totalRows / $pageSize),
            'current_page' => $page,
            'page_size' => $pageSize
        ]
    ];
}

public function getListingsByCategory($categoryId, $page = 1, $pageSize = 20)
{
    $offset = ($page - 1) * $pageSize;

    // Get total count for pagination
    $this->db->query('SELECT COUNT(*) as total FROM Listings WHERE status = "active" AND category_id = :category_id');
    $this->db->bind(':category_id', $categoryId);
    $totalRows = $this->db->single()->total;

    $this->db->query('SELECT
        l.listing_id, l.title, l.description, l.price, l.location, l.created_at,
        u.name AS seller_name, u.is_verified, u.profile_pic_url AS seller_profile_pic,
        c.name AS category_name,
        GROUP_CONCAT(lm.media_url) AS media_urls
        FROM Listings l
        JOIN initkey_rid u ON l.user_id = u.user_id
        JOIN Categories c ON l.category_id = c.category_id
        LEFT JOIN ListingMedia lm ON l.listing_id = lm.listing_id
        WHERE l.status = "active" AND c.category_id = :category_id
        GROUP BY l.listing_id
        ORDER BY l.created_at DESC
        LIMIT :limit OFFSET :offset');

    $this->db->bind(':category_id', $categoryId);
    $this->db->bind(':limit', $pageSize, PDO::PARAM_INT);
    $this->db->bind(':offset', $offset, PDO::PARAM_INT);

    $rows = $this->db->resultSet();

    return [
        'data' => $rows,
        'pagination' => [
            'total_items' => $totalRows,
            'total_pages' => ceil($totalRows / $pageSize),
            'current_page' => $page,
            'page_size' => $pageSize
        ]
    ];
}

public function updateListingStatus($listingId, $newStatus)
{
    $allowedStatuses = ['active', 'sold', 'draft'];
    if (!in_array($newStatus, $allowedStatuses)) {
        return false;
    }

    $this->db->query('UPDATE Listings SET status = :status WHERE listing_id = :listing_id');
    $this->db->bind(':status', $newStatus);
    $this->db->bind(':listing_id', $listingId);
    
    return $this->db->execute();
}

public function getTopRatedListings($page = 1, $pageSize = 20)
{
    $offset = ($page - 1) * $pageSize;

    // Get total count for pagination
    $this->db->query('SELECT COUNT(DISTINCT l.listing_id) as total 
        FROM Listings l
        JOIN Reviews r ON l.listing_id = r.listing_id
        WHERE l.status = "active"');
    $totalRows = $this->db->single()->total;

    $this->db->query('SELECT
        l.listing_id, l.title, l.description, l.price, l.location, l.created_at,
        u.name AS seller_name, u.is_verified, u.profile_pic_url AS seller_profile_pic,
        c.name AS category_name,
        GROUP_CONCAT(lm.media_url) AS media_urls,
        COALESCE(AVG(r.rating), 0) AS average_rating
        FROM Listings l
        JOIN initkey_rid u ON l.user_id = u.user_id
        JOIN Categories c ON l.category_id = c.category_id
        LEFT JOIN ListingMedia lm ON l.listing_id = lm.listing_id
        LEFT JOIN Reviews r ON l.listing_id = r.listing_id
        WHERE l.status = "active"
        GROUP BY l.listing_id
        HAVING AVG(r.rating) IS NOT NULL
        ORDER BY average_rating DESC
        LIMIT :limit OFFSET :offset');

    $this->db->bind(':limit', $pageSize, PDO::PARAM_INT);
    $this->db->bind(':offset', $offset, PDO::PARAM_INT);

    $rows = $this->db->resultSet();

    return [
        'data' => $rows,
        'pagination' => [
            'total_items' => $totalRows,
            'total_pages' => ceil($totalRows / $pageSize),
            'current_page' => $page,
            'page_size' => $pageSize
        ]
    ];
}

    public function logSearchHistory($userId, $searchQuery)
    {
        try {
            $this->db->query('INSERT INTO UserSearchHistory (user_id, search_query) VALUES (:user_id, :search_query)');
            $this->db->bind(':user_id', $userId);
            $this->db->bind(':search_query', $searchQuery);
            return $this->db->execute();
        } catch (Exception $e) {
            error_log('logSearchHistory error: ' . $e->getMessage());
            return false;
        }
    }

    public function getRecentSearchTerms($userId, $limit = 5)
    {
        try {
            $this->db->query('SELECT DISTINCT search_query FROM UserSearchHistory WHERE user_id = :user_id ORDER BY search_timestamp DESC LIMIT :limit');
            $this->db->bind(':user_id', $userId);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            $results = $this->db->resultSet();
            return array_column($results, 'search_query');
        } catch (Exception $e) {
            error_log('getRecentSearchTerms error: ' . $e->getMessage());
            return [];
        }
    }

}