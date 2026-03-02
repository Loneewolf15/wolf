<?php
class Review
{
    private $db;

    public function __construct()
    {
        $this->db = new Database;
    }

    /**
     * Verifies if a user has a completed order for a given listing.
     * This is a critical business rule for allowing reviews.
     * @param string $listingId The listing ID.
     * @param string $reviewerId The user ID.
     * @return bool True if a completed order exists, false otherwise.
     */
    public function hasCompletedOrder($listingId, $reviewerId)
    {
        $this->db->query('SELECT COUNT(*) AS count FROM Orders WHERE listing_id = :listing_id AND buyer_id = :buyer_id AND order_status = "delivered"');
        $this->db->bind(':listing_id', $listingId);
        $this->db->bind(':buyer_id', $reviewerId);
        $result = $this->db->single();
        return $result->count > 0;
    }

    // Add a new review
     public function addReview($data)
    {
        $this->db->query('INSERT INTO Reviews (review_id, listing_id, reviewer_id, rating, comment) VALUES (:review_id, :listing_id, :reviewer_id, :rating, :comment)');
        $this->db->bind(':review_id', $data['review_id']);
        $this->db->bind(':listing_id', $data['listing_id']);
        $this->db->bind(':reviewer_id', $data['reviewer_id']);
        $this->db->bind(':rating', $data['rating']);
        $this->db->bind(':comment', $data['comment']);
        
        if ($this->db->execute()) {
            // New: Update the average rating in the Listings table
            $this->updateListingAverageRating($data['listing_id']);
            return true;
        } else {
            return false;
        }
    }

    // Update an existing review
       public function updateReview($data)
    {
        $this->db->query('UPDATE Reviews SET rating = :rating, comment = :comment WHERE listing_id = :listing_id AND reviewer_id = :reviewer_id');
        $this->db->bind(':rating', $data['rating']);
        $this->db->bind(':comment', $data['comment']);
        $this->db->bind(':listing_id', $data['listing_id']);
        $this->db->bind(':reviewer_id', $data['reviewer_id']);
        
        if ($this->db->execute()) {
            // New: Update the average rating in the Listings table
            $this->updateListingAverageRating($data['listing_id']);
            return true;
        } else {
            return false;
        }
    }
    

    // Get a specific review by a user for a listing
    public function getReviewByUser($listingId, $reviewerId)
    {
        $this->db->query('SELECT * FROM Reviews WHERE listing_id = :listing_id AND reviewer_id = :reviewer_id');
        $this->db->bind(':listing_id', $listingId);
        $this->db->bind(':reviewer_id', $reviewerId);
        $row = $this->db->single();
        return $row;
    }

    // Get all reviews for a given listing
    public function getReviewsByListing($listingId)
    {
        $this->db->query('SELECT
            r.rating, r.comment, r.created_at,
            u.name AS reviewer_name, u.profile_pic_url AS reviewer_profile_pic
            FROM Reviews r
            JOIN initkey_rid u ON r.reviewer_id = u.user_id
            WHERE r.listing_id = :listing_id
            ORDER BY r.created_at DESC');
        $this->db->bind(':listing_id', $listingId);
        $rows = $this->db->resultSet();
        return $rows;
    }

     public function isReviewOwner($reviewId, $userId)
    {
        $this->db->query('SELECT COUNT(*) AS count FROM Reviews WHERE review_id = :review_id AND reviewer_id = :user_id');
        $this->db->bind(':review_id', $reviewId);
        $this->db->bind(':user_id', $userId);
        $result = $this->db->single();
        return $result->count > 0;
    }

    /**
 * Recalculates and updates the average rating for a listing.
 * @param string $listingId The unique ID of the listing.
 * @return bool True on success, false on failure.
 */
public function updateListingAverageRating($listingId)
{
    $this->db->query('SELECT COALESCE(AVG(rating), 0) AS average FROM Reviews WHERE listing_id = :listing_id AND is_deleted = FALSE');
    $this->db->bind(':listing_id', $listingId);
    $result = $this->db->single();
    $averageRating = floatval($result->average);

    $this->db->query('UPDATE Listings SET average_rating = :average_rating WHERE listing_id = :listing_id');
    $this->db->bind(':average_rating', $averageRating);
    $this->db->bind(':listing_id', $listingId);
    
    return $this->db->execute();
}

public function getReviewsBySeller($sellerId)
{
    $this->db->query('SELECT
        r.review_id, r.rating, r.comment, r.created_at,
        u.name AS reviewer_name, u.profile_pic_url AS reviewer_profile_pic,
        l.listing_id, l.title AS listing_title
        FROM Reviews r
        JOIN initkey_rid u ON r.reviewer_id = u.user_id
        JOIN Listings l ON r.listing_id = l.listing_id
        WHERE l.user_id = :seller_id AND r.is_deleted = FALSE
        ORDER BY r.created_at DESC');

    $this->db->bind(':seller_id', $sellerId);
    $rows = $this->db->resultSet();
    return $rows;
}
    public function reportReview($reviewId, $reportReason)
{
    $this->db->query('UPDATE Reviews SET is_flagged = TRUE, report_reason = :report_reason WHERE review_id = :review_id');
    $this->db->bind(':report_reason', $reportReason);
    $this->db->bind(':review_id', $reviewId);
    
    return $this->db->execute();
}
    
    /**
     * Deletes a review from the database.
     * @param string $reviewId The review ID.
     * @return bool True on success, false on failure.
     */
     public function deleteReview($reviewId)
    {
        // First, get the listing ID before deleting the review
        $this->db->query('SELECT listing_id FROM Reviews WHERE review_id = :review_id');
        $this->db->bind(':review_id', $reviewId);
        $result = $this->db->single();
        $listingId = $result->listing_id ?? null;

        // Soft delete the review
        $this->db->query('UPDATE Reviews SET is_deleted = TRUE WHERE review_id = :review_id');
        $this->db->bind(':review_id', $reviewId);
        
        if ($this->db->execute() && $listingId) {
            // New: Update the average rating in the Listings table
            $this->updateListingAverageRating($listingId);
            return true;
        } else {
            return false;
        }
    }
    /**
     * Calculates the average rating for a given seller across all their listings.
     * @param string $sellerId The seller's user ID.
     * @return float The average rating, or 0 if no reviews are found.
     */
    public function getSellerAverageRating($sellerId)
    {
        $this->db->query('SELECT COALESCE(AVG(r.rating), 0) AS average_rating
            FROM Reviews r
            JOIN Listings l ON r.listing_id = l.listing_id
            WHERE l.user_id = :seller_id');
        
        $this->db->bind(':seller_id', $sellerId);
        $result = $this->db->single();
        
        return floatval($result->average_rating);
    }
}