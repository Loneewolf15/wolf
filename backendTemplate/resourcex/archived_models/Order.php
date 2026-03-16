<?php
class Order
{
    private $db;
    private $cache;
    private $jobQueue;
    
    const CACHE_TTL = 3600; // 1 hour
    const DEFAULT_PAGE_SIZE = 20;
    const MAX_PAGE_SIZE = 100;

    public function __construct($cache = null, $jobQueue = null)
    {
        $this->db = new Database;
        $this->cache = $cache; // Redis or similar
        $this->jobQueue = $jobQueue; // Queue system for async processing
    }
    
    public function getUnconfirmedOrders($days, $page = 1, $pageSize = self::DEFAULT_PAGE_SIZE)
    {
        try {
            // Validate and sanitize pagination params
            $page = max(1, (int)$page);
            $pageSize = min(max(1, (int)$pageSize), self::MAX_PAGE_SIZE);
            $offset = ($page - 1) * $pageSize;
            
            $dateThreshold = date('Y-m-d H:i:s', strtotime("-{$days} days"));

            // Get total count for pagination metadata
            $this->db->query('SELECT COUNT(*) as total
                FROM Orders o
                WHERE o.order_status IN ("shipping", "processing")
                AND o.delivery_date IS NOT NULL
                AND o.delivery_date <= :date_threshold');
            $this->db->bind(':date_threshold', $dateThreshold);
            $totalCount = $this->db->single()->total;

            // Get paginated results
            $this->db->query('SELECT
                o.order_id, o.buyer_id, o.seller_id, o.total_price,
                l.reseller_commission_percent, u.referrer_id
                FROM Orders o
                JOIN Listings l ON o.listing_id = l.listing_id
                JOIN initkey_rid u ON o.buyer_id = u.user_id
                WHERE o.order_status IN ("shipping", "processing")
                AND o.delivery_date IS NOT NULL
                AND o.delivery_date <= :date_threshold
                ORDER BY o.delivery_date ASC
                LIMIT :limit OFFSET :offset');
            
            $this->db->bind(':date_threshold', $dateThreshold);
            $this->db->bind(':limit', $pageSize);
            $this->db->bind(':offset', $offset);
            
            $results = $this->db->resultSet();
            
            return [
                'data' => $results,
                'pagination' => [
                    'current_page' => $page,
                    'page_size' => $pageSize,
                    'total_records' => $totalCount,
                    'total_pages' => ceil($totalCount / $pageSize),
                    'has_next' => $page < ceil($totalCount / $pageSize),
                    'has_prev' => $page > 1
                ]
            ];
        } catch (PDOException $e) {
            error_log("Get unconfirmed orders failed [days:{$days}, page:{$page}]: " . $e->getMessage());
            throw new DatabaseException('Failed to retrieve unconfirmed orders', 0, $e);
        }
    }

    /**
     * Get unconfirmed orders with pagination and batch processing support
     * @param int $days Number of days threshold
     * @param int $page Current page (1-indexed)
     * @param int $pageSize Number of results per page
     * @return array ['data' => [], 'pagination' => []]
     */
    public function getUnconfirmedDeliveryByBuyer($days, $page = 1, $pageSize = self::DEFAULT_PAGE_SIZE)
    {
        try {
            // Validate and sanitize pagination params
            $page = max(1, (int)$page);
            $pageSize = min(max(1, (int)$pageSize), self::MAX_PAGE_SIZE);
            $offset = ($page - 1) * $pageSize;
            
            $dateThreshold = date('Y-m-d H:i:s', strtotime("-{$days} days"));

            // Get total count for pagination metadata
            $this->db->query('SELECT COUNT(*) as total
                FROM Orders o
                WHERE o.order_status = "delivered"
                AND o.status_updated_at <= :date_threshold');
            $this->db->bind(':date_threshold', $dateThreshold);
            $totalCount = $this->db->single()->total;

            // Get paginated results
            $this->db->query('SELECT
                o.order_id, o.buyer_id, o.seller_id, o.total_price,
                l.reseller_commission_percent, u.referrer_id
                FROM Orders o
                JOIN Listings l ON o.listing_id = l.listing_id
                JOIN initkey_rid u ON o.buyer_id = u.user_id
                WHERE o.order_status = "delivered"
                AND o.status_updated_at <= :date_threshold
                ORDER BY o.status_updated_at ASC
                LIMIT :limit OFFSET :offset');
            
            $this->db->bind(':date_threshold', $dateThreshold);
            $this->db->bind(':limit', $pageSize);
            $this->db->bind(':offset', $offset);
            
            $results = $this->db->resultSet();
            
            return [
                'data' => $results,
                'pagination' => [
                    'current_page' => $page,
                    'page_size' => $pageSize,
                    'total_records' => $totalCount,
                    'total_pages' => ceil($totalCount / $pageSize),
                    'has_next' => $page < ceil($totalCount / $pageSize),
                    'has_prev' => $page > 1
                ]
            ];
        } catch (PDOException $e) {
            error_log("Get unconfirmed orders failed [days:{$days}, page:{$page}]: " . $e->getMessage());
            throw new DatabaseException('Failed to retrieve unconfirmed orders', 0, $e);
        }
    }

    /**
     * Create order with escrow - optimized transaction handling
     */
    public function createOrderWithEscrow($orderData, $totalAmount)
    {
        try {
            // Generate IDs BEFORE starting transaction to minimize lock time
            $transactionId = $this->generateUniqueId('escrow');
            if (!$transactionId) {
                throw new Exception('Failed to generate transaction ID');
            }
            
            $feeAmount = $totalAmount * 0.01;
            
            // Now start transaction with all data ready
            $this->db->beginTransaction();
            
            // 1. Create the order
            $this->db->query('INSERT INTO Orders 
                (order_id, buyer_id, seller_id, listing_id, reseller_id, quantity, total_price) 
                VALUES (:order_id, :buyer_id, :seller_id, :listing_id, :reseller_id, :quantity, :total_price)');
            $this->db->bind(':order_id', $orderData['order_id']);
            $this->db->bind(':buyer_id', $orderData['buyer_id']);
            $this->db->bind(':seller_id', $orderData['seller_id']);
            $this->db->bind(':listing_id', $orderData['listing_id']);
            $this->db->bind(':reseller_id', $orderData['reseller_id']);
            $this->db->bind(':quantity', $orderData['quantity']);
            $this->db->bind(':total_price', $orderData['total_price']);
            $this->db->execute();

            // 2. Create the escrow transaction
            $this->db->query('INSERT INTO EscrowTransactions 
                (transaction_id, order_id, amount, fee_amount) 
                VALUES (:transaction_id, :order_id, :amount, :fee_amount)');
            $this->db->bind(':transaction_id', $transactionId);
            $this->db->bind(':order_id', $orderData['order_id']);
            $this->db->bind(':amount', $totalAmount);
            $this->db->bind(':fee_amount', $feeAmount);
            $this->db->execute();

            $this->db->commit();
            
            // Queue async tasks AFTER successful commit
            if ($this->jobQueue) {
                $this->jobQueue->push('SendOrderConfirmationEmail', [
                    'order_id' => $orderData['order_id'],
                    'buyer_id' => $orderData['buyer_id']
                ]);
                
                $this->jobQueue->push('ProcessCommissions', [
                    'order_id' => $orderData['order_id']
                ]);
            }
            
            return [
                'success' => true,
                'order_id' => $orderData['order_id'],
                'transaction_id' => $transactionId
            ];
        } catch (Exception $e) {
            if ($this->db->inTransaction()) {
                $this->db->rollBack();
            }
            error_log("Order creation failed [order_id:{$orderData['order_id']}]: " . $e->getMessage());
            throw new OrderCreationException('Failed to create order with escrow', 0, $e);
        }
    }

    /**
     * Get orders by seller with pagination
     */
    public function getOrdersBySeller($sellerId, $page = 1, $pageSize = self::DEFAULT_PAGE_SIZE)
    {
        try {
            $page = max(1, (int)$page);
            $pageSize = min(max(1, (int)$pageSize), self::MAX_PAGE_SIZE);
            $offset = ($page - 1) * $pageSize;
            
            // Get total count
            $this->db->query('SELECT COUNT(*) as total FROM Orders WHERE seller_id = :seller_id');
            $this->db->bind(':seller_id', $sellerId);
            $totalCount = $this->db->single()->total;
            
            // Get paginated results
            $this->db->query('SELECT
                o.order_id, o.quantity, o.total_price, o.order_status, o.created_at,
                b.name AS buyer_name, b.profile_pic_url AS buyer_profile_pic,
                l.title AS listing_title, l.location AS listing_location, l.price AS listing_price
                FROM Orders o
                JOIN initkey_rid b ON o.buyer_id = b.user_id
                JOIN Listings l ON o.listing_id = l.listing_id
                WHERE o.seller_id = :seller_id
                ORDER BY o.created_at DESC
                LIMIT :limit OFFSET :offset');

            $this->db->bind(':seller_id', $sellerId);
            $this->db->bind(':limit', $pageSize);
            $this->db->bind(':offset', $offset);
            $results = $this->db->resultSet();
            
            return [
                'data' => $results,
                'pagination' => [
                    'current_page' => $page,
                    'page_size' => $pageSize,
                    'total_records' => $totalCount,
                    'total_pages' => ceil($totalCount / $pageSize),
                    'has_next' => $page < ceil($totalCount / $pageSize),
                    'has_prev' => $page > 1
                ]
            ];
        } catch (PDOException $e) {
            error_log("Get orders by seller failed [seller_id:{$sellerId}, page:{$page}]: " . $e->getMessage());
            throw new DatabaseException('Failed to retrieve seller orders', 0, $e);
        }
    }
    
    /**
     * Get orders by buyer with pagination
     */
    public function getOrdersByBuyer($buyerId, $page = 1, $pageSize = self::DEFAULT_PAGE_SIZE)
    {
        try {
            $page = max(1, (int)$page);
            $pageSize = min(max(1, (int)$pageSize), self::MAX_PAGE_SIZE);
            $offset = ($page - 1) * $pageSize;
            
            // Get total count
            $this->db->query('SELECT COUNT(*) as total FROM Orders WHERE buyer_id = :buyer_id');
            $this->db->bind(':buyer_id', $buyerId);
            $totalCount = $this->db->single()->total;
            
            // Get paginated results
            $this->db->query('SELECT
                o.order_id, o.quantity, o.total_price, o.order_status, o.created_at,
                s.name AS seller_name, s.profile_pic_url AS seller_profile_pic,
                l.title AS listing_title, l.location AS listing_location, l.price AS listing_price
                FROM Orders o
                JOIN initkey_rid s ON o.seller_id = s.user_id
                JOIN Listings l ON o.listing_id = l.listing_id
                WHERE o.buyer_id = :buyer_id
                ORDER BY o.created_at DESC
                LIMIT :limit OFFSET :offset');

            $this->db->bind(':buyer_id', $buyerId);
            $this->db->bind(':limit', $pageSize);
            $this->db->bind(':offset', $offset);
            $results = $this->db->resultSet();
            
            return [
                'data' => $results,
                'pagination' => [
                    'current_page' => $page,
                    'page_size' => $pageSize,
                    'total_records' => $totalCount,
                    'total_pages' => ceil($totalCount / $pageSize),
                    'has_next' => $page < ceil($totalCount / $pageSize),
                    'has_prev' => $page > 1
                ]
            ];
        } catch (PDOException $e) {
            error_log("Get orders by buyer failed [buyer_id:{$buyerId}, page:{$page}]: " . $e->getMessage());
            throw new DatabaseException('Failed to retrieve buyer orders', 0, $e);
        }
    }

    /**
     * Get order details with optional caching
     */
    public function getOrderDetails($orderId, $useCache = true)
    {
        try {
            // Try cache first if enabled
            if ($useCache && $this->cache) {
                $cacheKey = "order_details:{$orderId}";
                $cached = $this->cache->get($cacheKey);
                if ($cached !== false) {
                    return $cached;
                }
            }
            
            $this->db->query('SELECT
                o.order_id, o.buyer_id, o.seller_id, o.listing_id, o.quantity, 
                o.total_price, o.order_status, o.created_at,
                l.title AS listing_title, l.description AS listing_description, 
                l.reseller_commission_percent,
                b.name AS buyer_name, b.email AS buyer_email,
                s.name AS seller_name, s.email AS seller_email,
                e.amount AS escrow_amount_held, e.status AS escrow_status
                FROM Orders o
                JOIN Listings l ON o.listing_id = l.listing_id
                JOIN initkey_rid b ON o.buyer_id = b.user_id
                JOIN initkey_rid s ON o.seller_id = s.user_id
                JOIN EscrowTransactions e ON o.order_id = e.order_id
                WHERE o.order_id = :order_id
                LIMIT 1');

            $this->db->bind(':order_id', $orderId);
            $result = $this->db->single();
            
            // Cache the result
            if ($result && $useCache && $this->cache) {
                $this->cache->set($cacheKey, $result, self::CACHE_TTL);
            }
            
            return $result;
        } catch (PDOException $e) {
            error_log("Get order details failed [order_id:{$orderId}]: " . $e->getMessage());
            throw new DatabaseException('Failed to retrieve order details', 0, $e);
        }
    }

    /**
     * Update order status and invalidate cache
     */
    public function updateOrderStatus($orderId, $status)
    {
        try {
            $this->db->query('UPDATE Orders SET order_status = :status, status_updated_at = NOW() WHERE order_id = :order_id');
            $this->db->bind(':status', $status);
            $this->db->bind(':order_id', $orderId);
            $result = $this->db->execute();
            
            // Invalidate cache after update
            if ($this->cache) {
                $this->cache->delete("order_details:{$orderId}");
            }
            
            return $result;
        } catch (PDOException $e) {
            error_log("Update order status failed [order_id:{$orderId}, status:{$status}]: " . $e->getMessage());
            throw new DatabaseException('Failed to update order status', 0, $e);
        }
    }
    
    /**
     * Update escrow transaction status
     */
    public function updateEscrowTransactionStatus($orderId, $status)
    {
        try {
            $this->db->query('UPDATE EscrowTransactions SET status = :status WHERE order_id = :order_id');
            $this->db->bind(':status', $status);
            $this->db->bind(':order_id', $orderId);
            $result = $this->db->execute();
            
            // Invalidate cache after update
            if ($this->cache) {
                $this->cache->delete("order_details:{$orderId}");
            }
            
            return $result;
        } catch (PDOException $e) {
            error_log("Update escrow status failed [order_id:{$orderId}, status:{$status}]: " . $e->getMessage());
            throw new DatabaseException('Failed to update escrow status', 0, $e);
        }
    }

    /**
     * Get referral commission rate with caching
     */
    public function getReferralCommissionRate()
    {
        try {
            $cacheKey = 'referral_commission_rate';
            
            // Check cache first
            if ($this->cache) {
                $cached = $this->cache->get($cacheKey);
                if ($cached !== false) {
                    return $cached;
                }
            }
            
            // Query database if not cached
            $this->db->query('SELECT referral_commission_percent 
                FROM ReferralSettings 
                ORDER BY created_at DESC 
                LIMIT 1');
            $result = $this->db->single();
            $rate = $result->referral_commission_percent ?? 0;
            
            // Cache for longer since this rarely changes
            if ($this->cache) {
                $this->cache->set($cacheKey, $rate, self::CACHE_TTL * 24); // 24 hours
            }
            
            return $rate;
        } catch (PDOException $e) {
            error_log("Get referral commission rate failed: " . $e->getMessage());
            return 0; // Safe default
        }
    }

    /**
     * Generate unique ID - moved outside transactions
     */
    private function generateUniqueId($prefix)
    {
        try {
            return $prefix . '_' . bin2hex(random_bytes(8)) . '_' . time();
        } catch (Exception $e) {
            error_log("Generate unique ID failed [prefix:{$prefix}]: " . $e->getMessage());
            return false;
        }
    }
    
    /**
     * Batch process unconfirmed orders for cron jobs
     */
    public function processUnconfirmedOrdersBatch($days, $batchSize = 100, $callback = null)
    {
        try {
            $page = 1;
            $processedCount = 0;
            
            do {
                $result = $this->getUnconfirmedOrders($days, $page, $batchSize);
                $orders = $result['data'];
                
                foreach ($orders as $order) {
                    if ($callback && is_callable($callback)) {
                        $callback($order);
                    }
                    $processedCount++;
                }
                
                $page++;
                
                // Continue while there are more pages
            } while ($result['pagination']['has_next']);
            
            return $processedCount;
        } catch (Exception $e) {
            error_log("Batch process unconfirmed orders failed: " . $e->getMessage());
            throw $e;
        }
    }
}

// Custom exception classes for better error handling
class DatabaseException extends Exception {}
class OrderCreationException extends Exception {}