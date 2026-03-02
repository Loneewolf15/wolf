<?php
class Wishlist {
    private $db;
    
    public function __construct(){
        $this->db = new Database;
    }

    // Get all wishlisted listings for a user
    public function getAllWishlisted($user_id) {
        try {
            $this->db->query("SELECT w.*, l.title, l.price, l.property_type, l.city, l.state, l.image_1, l.status 
                             FROM wishlist w 
                             INNER JOIN listings l ON w.listing_id = l.listing_id 
                             WHERE w.user_id = :user_id 
                             ORDER BY w.created_at DESC");
            
            $this->db->bind(':user_id', $user_id);
            
            $rows = $this->db->resultSet();
            
            if (!empty($rows)) {
                return $rows;
            } else {
                return [];
            }
        } catch (PDOException $e) {
            return "Error: " . $e->getMessage();
        }
    }

    // Add listing to wishlist
    public function addToWishlist($user_id, $listing_id) {
        try {
            // First check if already in wishlist
            if ($this->isInWishlist($user_id, $listing_id)) {
                return "already_exists";
            }

            // Check if listing exists
            $this->db->query("SELECT listing_id FROM listings WHERE listing_id = :listing_id");
            $this->db->bind(':listing_id', $listing_id);
            $listing = $this->db->single();
            
            if (!$listing) {
                return "listing_not_found";
            }

            // Add to wishlist
            $this->db->query('INSERT INTO wishlist (user_id, listing_id, created_at) VALUES (:user_id, :listing_id, NOW())');
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':listing_id', $listing_id);
            
            if($this->db->execute()){
                return true;
            } else {
                return false;
            }
        } catch (PDOException $e) {
            return "Error: " . $e->getMessage();
        }
    }

    // Remove listing from wishlist
    public function removeFromWishlist($user_id, $listing_id) {
        try {
            $this->db->query('DELETE FROM wishlist WHERE user_id = :user_id AND listing_id = :listing_id');
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':listing_id', $listing_id);
            
            if($this->db->execute()){
                return true;
            } else {
                return false;
            }
        } catch (PDOException $e) {
            return "Error: " . $e->getMessage();
        }
    }

    // Check if listing is in user's wishlist
    public function isInWishlist($user_id, $listing_id) {
        try {
            $this->db->query("SELECT id FROM wishlist WHERE user_id = :user_id AND listing_id = :listing_id");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':listing_id', $listing_id);
            
            $row = $this->db->single();
            
            return $row ? true : false;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get wishlist count for a user
    public function getWishlistCount($user_id) {
        try {
            $this->db->query("SELECT COUNT(*) as count FROM wishlist WHERE user_id = :user_id");
            
            $this->db->bind(':user_id', $user_id);
            
            $row = $this->db->single();
            
            return $row ? $row->count : 0;
        } catch (PDOException $e) {
            return 0;
        }
    }

    // Clear all wishlist items for a user
    public function clearWishlist($user_id) {
        try {
            $this->db->query('DELETE FROM wishlist WHERE user_id = :user_id');
            
            $this->db->bind(':user_id', $user_id);
            
            if($this->db->execute()){
                return true;
            } else {
                return false;
            }
        } catch (PDOException $e) {
            return "Error: " . $e->getMessage();
        }
    }

    // Get wishlist with pagination
    public function getWishlistPaginated($user_id, $limit = 20, $offset = 0) {
        try {
            $this->db->query("SELECT w.*, l.title, l.price, l.property_type, l.city, l.state, l.image_1, l.status, l.bedrooms, l.bathrooms 
                             FROM wishlist w 
                             INNER JOIN listings l ON w.listing_id = l.listing_id 
                             WHERE w.user_id = :user_id 
                             ORDER BY w.created_at DESC 
                             LIMIT :limit OFFSET :offset");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            $this->db->bind(':offset', $offset, PDO::PARAM_INT);
            
            $rows = $this->db->resultSet();
            
            if (!empty($rows)) {
                return $rows;
            } else {
                return [];
            }
        } catch (PDOException $e) {
            return "Error: " . $e->getMessage();
        }
    }

    // Get recently added wishlist items
    public function getRecentWishlist($user_id, $limit = 5) {
        try {
            $this->db->query("SELECT w.*, l.title, l.price, l.property_type, l.city, l.state, l.image_1 
                             FROM wishlist w 
                             INNER JOIN listings l ON w.listing_id = l.listing_id 
                             WHERE w.user_id = :user_id 
                             ORDER BY w.created_at DESC 
                             LIMIT :limit");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            
            $rows = $this->db->resultSet();
            
            if (!empty($rows)) {
                return $rows;
            } else {
                return [];
            }
        } catch (PDOException $e) {
            return "Error: " . $e->getMessage();
        }
    }

    // Get wishlist items by property type
    public function getWishlistByPropertyType($user_id, $property_type) {
        try {
            $this->db->query("SELECT w.*, l.title, l.price, l.property_type, l.city, l.state, l.image_1, l.status 
                             FROM wishlist w 
                             INNER JOIN listings l ON w.listing_id = l.listing_id 
                             WHERE w.user_id = :user_id AND l.property_type = :property_type 
                             ORDER BY w.created_at DESC");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':property_type', $property_type);
            
            $rows = $this->db->resultSet();
            
            if (!empty($rows)) {
                return $rows;
            } else {
                return [];
            }
        } catch (PDOException $e) {
            return "Error: " . $e->getMessage();
        }
    }

    // Remove multiple items from wishlist
    public function removeMultipleFromWishlist($user_id, $listing_ids) {
        try {
            if (empty($listing_ids) || !is_array($listing_ids)) {
                return false;
            }

            $placeholders = str_repeat('?,', count($listing_ids) - 1) . '?';
            $this->db->query("DELETE FROM wishlist WHERE user_id = ? AND listing_id IN ($placeholders)");
            
            // Bind user_id first, then all listing_ids
            $params = array_merge([$user_id], $listing_ids);
            
            // Manual parameter binding for this complex query
            $stmt = $this->db->stmt;
            for ($i = 0; $i < count($params); $i++) {
                $stmt->bindValue($i + 1, $params[$i]);
            }
            
            if($stmt->execute()){
                return true;
            } else {
                return false;
            }
        } catch (PDOException $e) {
            return "Error: " . $e->getMessage();
        }
    }
}
