<?php
class Reseller
{
    private $db;

    public function __construct()
    {
        $this->db = new Database;
    }

    /**
     * Retrieves all active listings that are marked as resellable.
     * @return array An array of listing objects or an empty array.
     */
    public function getAllResellableListings()
    {
        $this->db->query('SELECT
            l.listing_id,
            l.title,
            l.description,
            l.price,
            l.location,
            l.created_at,
            l.reseller_commission_percent,
            u.name AS seller_name,
            u.is_verified,
            u.profile_pic_url AS seller_profile_pic,
            c.name AS category_name,
            GROUP_CONCAT(lm.media_url) AS media_urls,
            COALESCE(AVG(r.rating), 0) AS average_rating
            FROM Listings l
            JOIN initkey_rid u ON l.user_id = u.user_id
            JOIN Categories c ON l.category_id = c.category_id
            LEFT JOIN ListingMedia lm ON l.listing_id = lm.listing_id
            LEFT JOIN Reviews r ON l.listing_id = r.listing_id
            WHERE l.status = "active" AND l.is_resellable = 1
            GROUP BY l.listing_id
            ORDER BY l.created_at DESC');

        $rows = $this->db->resultSet();
        return $rows;
    }

    /**
     * Retrieves analytics data for a specific reseller.
     * @param int $userId The ID of the reseller.
     * @return array An associative array of analytics data.
     */
    public function getResellerAnalyticsData($userId)
    {
        // Total active listings being resold by this user
        $this->db->query('SELECT COUNT(*) AS total_active_reselling_listings
                            FROM Listings
                            WHERE user_id = :user_id AND is_resellable = 1 AND status = "active"');
        $this->db->bind(':user_id', $userId);
        $activeListings = $this->db->single()->total_active_reselling_listings;

        // Total sales and estimated commission from listings resold by this user
        // This assumes that the user_id on the listing is the reseller's ID
        // and that reseller_commission_percent is applied to the listing price.
        $this->db->query('SELECT
                            COUNT(o.order_id) AS total_sales_made,
                            SUM(o.total_amount) AS total_sales_value,
                            SUM(o.total_amount * (l.reseller_commission_percent / 100)) AS total_estimated_commission
                            FROM Orders o
                            JOIN Listings l ON o.listing_id = l.listing_id
                            WHERE l.user_id = :user_id AND o.status = "completed"'); // Assuming completed orders count as sales
        $this->db->bind(':user_id', $userId);
        $salesData = $this->db->single();

        return [
            'total_active_reselling_listings' => $activeListings,
            'total_sales_made' => $salesData->total_sales_made ?? 0,
            'total_sales_value' => $salesData->total_sales_value ?? 0,
            'total_estimated_commission' => $salesData->total_estimated_commission ?? 0
        ];
    }
    
}