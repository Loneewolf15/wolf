<?php
class Social {
    private $db;
    
    public function __construct(){
        $this->db = new Database;
    }

    // Follow a user
    public function followUser($follower_id, $following_id) {
        try {
            // Can't follow yourself
            if ($follower_id === $following_id) {
                return false;
            }
            
            $this->db->query('INSERT INTO user_follows (follower_id, following_id, created_at) 
                             VALUES (:follower_id, :following_id, NOW())');
            
            $this->db->bind(':follower_id', $follower_id);
            $this->db->bind(':following_id', $following_id);
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Unfollow a user
    public function unfollowUser($follower_id, $following_id) {
        try {
            $this->db->query('DELETE FROM user_follows 
                             WHERE follower_id = :follower_id AND following_id = :following_id');
            
            $this->db->bind(':follower_id', $follower_id);
            $this->db->bind(':following_id', $following_id);
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Check if user is following another user
    public function isFollowing($follower_id, $following_id) {
        try {
            $this->db->query("SELECT 1 FROM user_follows 
                             WHERE follower_id = :follower_id AND following_id = :following_id");
            
            $this->db->bind(':follower_id', $follower_id);
            $this->db->bind(':following_id', $following_id);
            
            return $this->db->single() ? true : false;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get user's followers
    public function getFollowers($user_id, $limit = 20, $offset = 0) {
        try {
            $this->db->query("SELECT u.user_id, u.full_name, u.profile_image, u.user_type, u.city, u.state, uf.created_at as followed_since
                             FROM user_follows uf
                             INNER JOIN initkey_rid u ON uf.follower_id = u.user_id
                             WHERE uf.following_id = :user_id
                             ORDER BY uf.created_at DESC
                             LIMIT :limit OFFSET :offset");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            $this->db->bind(':offset', $offset, PDO::PARAM_INT);
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return [];
        }
    }

    // Get users that a user is following
    public function getFollowing($user_id, $limit = 20, $offset = 0) {
        try {
            $this->db->query("SELECT u.user_id, u.full_name, u.profile_image, u.user_type, u.city, u.state, uf.created_at as followed_since
                             FROM user_follows uf
                             INNER JOIN initkey_rid u ON uf.following_id = u.user_id
                             WHERE uf.follower_id = :user_id
                             ORDER BY uf.created_at DESC
                             LIMIT :limit OFFSET :offset");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            $this->db->bind(':offset', $offset, PDO::PARAM_INT);
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return [];
        }
    }

    // Get follow counts
    public function getFollowCounts($user_id) {
        try {
            $counts = [];
            
            // Followers count
            $this->db->query("SELECT COUNT(*) as followers_count FROM user_follows WHERE following_id = :user_id");
            $this->db->bind(':user_id', $user_id);
            $result = $this->db->single();
            $counts['followers'] = $result ? $result->followers_count : 0;
            
            // Following count
            $this->db->query("SELECT COUNT(*) as following_count FROM user_follows WHERE follower_id = :user_id");
            $this->db->bind(':user_id', $user_id);
            $result = $this->db->single();
            $counts['following'] = $result ? $result->following_count : 0;
            
            return $counts;
        } catch (PDOException $e) {
            return ['followers' => 0, 'following' => 0];
        }
    }

    // Add a review for a user
    public function addUserReview($reviewer_id, $reviewed_user_id, $rating, $review_text = null, $listing_id = null, $review_type = 'general') {
        try {
            // Can't review yourself
            if ($reviewer_id === $reviewed_user_id) {
                return false;
            }
            
            $review_id = "review_" . md5($reviewer_id . $reviewed_user_id . time() . rand(1000, 9999));
            
            $this->db->query('INSERT INTO user_reviews (
                review_id, 
                reviewer_id, 
                reviewed_user_id, 
                related_listing_id, 
                rating, 
                review_text, 
                review_type, 
                created_at
            ) VALUES (
                :review_id, 
                :reviewer_id, 
                :reviewed_user_id, 
                :related_listing_id, 
                :rating, 
                :review_text, 
                :review_type, 
                NOW()
            )');

            $this->db->bind(':review_id', $review_id);
            $this->db->bind(':reviewer_id', $reviewer_id);
            $this->db->bind(':reviewed_user_id', $reviewed_user_id);
            $this->db->bind(':related_listing_id', $listing_id);
            $this->db->bind(':rating', $rating);
            $this->db->bind(':review_text', $review_text);
            $this->db->bind(':review_type', $review_type);
            
            if ($this->db->execute()) {
                return $review_id;
            }
            
            return false;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get reviews for a user
    public function getUserReviews($user_id, $limit = 20, $offset = 0) {
        try {
            $this->db->query("SELECT ur.*, 
                             u.full_name as reviewer_name, 
                             u.profile_image as reviewer_image,
                             l.title as listing_title
                             FROM user_reviews ur
                             INNER JOIN initkey_rid u ON ur.reviewer_id = u.user_id
                             LEFT JOIN listings l ON ur.related_listing_id = l.listing_id
                             WHERE ur.reviewed_user_id = :user_id
                             ORDER BY ur.created_at DESC
                             LIMIT :limit OFFSET :offset");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            $this->db->bind(':offset', $offset, PDO::PARAM_INT);
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return [];
        }
    }

    // Get user's average rating and review count
    public function getUserRatingStats($user_id) {
        try {
            $this->db->query("SELECT 
                             AVG(rating) as average_rating, 
                             COUNT(*) as total_reviews,
                             COUNT(CASE WHEN rating = 5 THEN 1 END) as five_star,
                             COUNT(CASE WHEN rating = 4 THEN 1 END) as four_star,
                             COUNT(CASE WHEN rating = 3 THEN 1 END) as three_star,
                             COUNT(CASE WHEN rating = 2 THEN 1 END) as two_star,
                             COUNT(CASE WHEN rating = 1 THEN 1 END) as one_star
                             FROM user_reviews 
                             WHERE reviewed_user_id = :user_id");
            
            $this->db->bind(':user_id', $user_id);
            $result = $this->db->single();
            
            if ($result) {
                return [
                    'average_rating' => round($result->average_rating, 2),
                    'total_reviews' => $result->total_reviews,
                    'rating_breakdown' => [
                        '5' => $result->five_star,
                        '4' => $result->four_star,
                        '3' => $result->three_star,
                        '2' => $result->two_star,
                        '1' => $result->one_star
                    ]
                ];
            }
            
            return [
                'average_rating' => 0,
                'total_reviews' => 0,
                'rating_breakdown' => ['5' => 0, '4' => 0, '3' => 0, '2' => 0, '1' => 0]
            ];
        } catch (PDOException $e) {
            return [
                'average_rating' => 0,
                'total_reviews' => 0,
                'rating_breakdown' => ['5' => 0, '4' => 0, '3' => 0, '2' => 0, '1' => 0]
            ];
        }
    }

    // Get user's public profile
    public function getUserProfile($user_id, $viewer_id = null) {
        try {
            $this->db->query("SELECT user_id, full_name, profile_image, user_type, company_name, 
                             bio, city, state, country, created_at
                             FROM initkey_rid 
                             WHERE user_id = :user_id AND activation = 1");
            
            $this->db->bind(':user_id', $user_id);
            $profile = $this->db->single();
            
            if (!$profile) {
                return false;
            }
            
            // Add follow counts
            $followCounts = $this->getFollowCounts($user_id);
            $profile->followers_count = $followCounts['followers'];
            $profile->following_count = $followCounts['following'];
            
            // Add rating stats
            $ratingStats = $this->getUserRatingStats($user_id);
            $profile->average_rating = $ratingStats['average_rating'];
            $profile->total_reviews = $ratingStats['total_reviews'];
            
            // Check if viewer is following this user
            if ($viewer_id) {
                $profile->is_following = $this->isFollowing($viewer_id, $user_id);
            } else {
                $profile->is_following = false;
            }
            
            // Get listing counts if user is agent/realtor
            if ($profile->user_type === 'agent' || $profile->user_type === 'realtor') {
                $this->db->query("SELECT COUNT(*) as listing_count FROM listings WHERE agent_id = :user_id AND status = 'active'");
                $this->db->bind(':user_id', $user_id);
                $result = $this->db->single();
                $profile->active_listings = $result ? $result->listing_count : 0;
            }
            
            return $profile;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get suggested users to follow (agents/realtors in same area)
    public function getSuggestedFollows($user_id, $limit = 10) {
        try {
            // Get user's location
            $this->db->query("SELECT city, state FROM initkey_rid WHERE user_id = :user_id");
            $this->db->bind(':user_id', $user_id);
            $userLocation = $this->db->single();
            
            if (!$userLocation) {
                return [];
            }
            
            $this->db->query("SELECT u.user_id, u.full_name, u.profile_image, u.user_type, u.company_name, u.city, u.state,
                             COUNT(l.listing_id) as listing_count,
                             AVG(ur.rating) as average_rating
                             FROM initkey_rid u
                             LEFT JOIN listings l ON u.user_id = l.agent_id AND l.status = 'active'
                             LEFT JOIN user_reviews ur ON u.user_id = ur.reviewed_user_id
                             WHERE u.user_id != :user_id
                             AND u.activation = 1
                             AND u.user_type IN ('agent', 'realtor')
                             AND (u.city = :city OR u.state = :state)
                             AND u.user_id NOT IN (
                                 SELECT following_id FROM user_follows WHERE follower_id = :user_id
                             )
                             GROUP BY u.user_id
                             ORDER BY listing_count DESC, average_rating DESC
                             LIMIT :limit");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':city', $userLocation->city);
            $this->db->bind(':state', $userLocation->state);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return [];
        }
    }
}
