<?php
class Orders extends Controller
{
    protected $orderModel;
    protected $listingModel;
    protected $userModel;
    protected $walletModel;
    protected $cache;
    protected $jobQueue;
    protected $validator;
    
    const MAX_QUANTITY = 1000;
    const DEFAULT_PAGE_SIZE = 20;

    public function __construct()
    {
        $this->orderModel = $this->model('Order');
        $this->listingModel = $this->model('Listing');
        $this->userModel = $this->model('User');
        $this->walletModel = $this->model('Wallet');
        
        // Initialize cache and queue (Redis, RabbitMQ, etc.)
        $this->cache = $this->getCache();
        $this->jobQueue = $this->getJobQueue();
        $this->validator = new OrderValidator();
    }

    private function sendResponse($status, $message, $data = [], $httpCode = 200) {
        http_response_code($httpCode);
        echo json_encode([
            'status' => $status, 
            'message' => $message, 
            'data' => $data,
            'timestamp' => time()
        ]);
        exit;
    }
    
    /**
     * Endpoint: POST /orders/checkout
     * Creates order with optimized transaction handling and async processing
     */
    public function checkout()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $userData = $this->RouteProtection();
            $postData = $this->getData();
            
            // Validate input
            $validation = $this->validator->validateCheckout($postData);
            if (!$validation['valid']) {
                $this->sendResponse(false, $validation['message'], [], 400);
            }
            
            $listingId = trim($postData['listing_id']);
            $quantity = intval($postData['quantity']);
            $paymentMethod = trim($postData['payment_method']);
            $deliveryAddress = trim($postData['delivery_address'] ?? $userData->location);
            $resellerId = trim($postData['reseller_id'] ?? null);

            // Quantity validation
            if ($quantity < 1 || $quantity > self::MAX_QUANTITY) {
                $this->sendResponse(false, "Quantity must be between 1 and " . self::MAX_QUANTITY, [], 400);
            }

            // Get listing with caching
            $listing = $this->getListingWithCache($listingId);
            if (!$listing) {
                $this->sendResponse(false, 'Listing not found.', [], 404);
            }

            // Business logic validations
            if ($listing->user_id === $userData->user_id) {
                $this->sendResponse(false, 'You cannot buy your own product.', [], 400);
            }

            // Check stock availability
            if (isset($listing->stock) && $listing->stock < $quantity) {
                $this->sendResponse(false, 'Insufficient stock available.', [], 400);
            }

            // Calculate amounts
            $totalPrice = $listing->price * $quantity;
            $platformFee = $totalPrice * 0.01;
            $sellerPayoutAmount = $totalPrice - $platformFee;

            // Validate payment method and funds
            if ($paymentMethod === 'wallet') {
                if (!$this->walletModel->hasSufficientFunds($userData->user_id, $totalPrice)) {
                    $this->sendResponse(false, 'Insufficient funds in your wallet.', [], 400);
                }
            } elseif ($paymentMethod === 'card') {
                // External payment gateway integration point
                // $paymentGateway = new VPayGateway();
                // $paymentResult = $paymentGateway->charge($totalPrice, $postData['card_token']);
                // if (!$paymentResult->success) { ... }
            } else {
                $this->sendResponse(false, 'Invalid payment method.', [], 400);
            }

            // Generate order ID outside transaction
            $orderId = $this->generateUniqueId('order');
            
            $orderData = [
                'order_id' => $orderId,
                'buyer_id' => $userData->user_id,
                'seller_id' => $listing->user_id,
                'listing_id' => $listingId,
                'reseller_id' => $resellerId,
                'quantity' => $quantity,
                'total_price' => $totalPrice,
                'delivery_address' => $deliveryAddress,
                'payment_method' => $paymentMethod
            ];

            // Create order with escrow (optimized transaction)
            $result = $this->orderModel->createOrderWithEscrow($orderData, $totalPrice);
            
            if (!$result || !$result['success']) {
                $this->sendResponse(false, 'Failed to create order.', [], 500);
            }

            // Process payment
            if ($paymentMethod === 'wallet') {
                $paymentSuccess = $this->walletModel->processWalletPayment(
                    $userData->user_id, 
                    $totalPrice, 
                    $orderId
                );
                
                if (!$paymentSuccess) {
                    // Rollback order if payment fails
                    $this->orderModel->updateOrderStatus($orderId, 'payment_failed');
                    $this->sendResponse(false, 'Payment processing failed.', [], 500);
                }
            }

            // Queue async tasks (emails, notifications, analytics)
            $this->queuePostOrderTasks($orderId, $userData->user_id, $listing->user_id);

            // Invalidate relevant caches
            $this->invalidateOrderCaches($userData->user_id, $listing->user_id);

            $this->sendResponse(true, 'Order placed and payment secured in escrow.', [
                'order_id' => $orderId,
                'total_price' => $totalPrice,
                'estimated_delivery' => $this->calculateEstimatedDelivery()
            ], 201);
            
        } catch (DatabaseException $e) {
            error_log("Checkout DB error [user:{$userData->user_id}]: " . $e->getMessage());
            $this->sendResponse(false, 'Database error occurred.', [], 500);
        } catch (Exception $e) {
            error_log("Checkout error [user:{$userData->user_id}]: " . $e->getMessage());
            $this->sendResponse(false, 'An error occurred while processing your order.', [], 500);
        }
    }
    
    /**
     * Endpoint: POST /orders/confirm-delivery/{orderId}
     * Optimized with transaction safety and async commission processing
     */
    public function confirmDelivery($orderId)
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $userData = $this->RouteProtection();
            
            if (empty($orderId)) {
                $this->sendResponse(false, 'Order ID is required.', [], 400);
            }

            $order = $this->orderModel->getOrderDetails($orderId, false);
            if (!$order) {
                $this->sendResponse(false, 'Order not found.', [], 404);
            }

            if ($order->buyer_id !== $userData->user_id) {
                $this->sendResponse(false, 'Access Denied: You are not the buyer of this order.', [], 403);
            }

            if ($order->order_status === 'delivered') {
                $this->sendResponse(false, 'Order is already marked as delivered.', [], 409);
            }

            if (!in_array($order->order_status, ['shipping', 'processing'])) {
                $this->sendResponse(false, 'Order cannot be confirmed in current status.', [], 400);
            }

            // Queue the payout processing
            if ($this->jobQueue) {
                $this->jobQueue->push('ProcessPayout', ['order_id' => $orderId]);
            } else {
                // Fallback to synchronous processing if queue is not available
                $payoutService = new PayoutService();
                $payoutService->processPayout($orderId);
            }

            // Invalidate caches
            $this->invalidateOrderCaches($order->buyer_id, $order->seller_id);
            if ($this->cache) {
                $this->cache->delete("order_details:{$orderId}");
            }

            $this->sendResponse(true, 'Order confirmation received. Funds are being processed.', [
                'order_id' => $orderId,
                'status' => 'delivered'
            ]);
            
        } catch (Exception $e) {
            error_log("Confirm delivery error [order:{$orderId}]: " . $e->getMessage());
            $this->sendResponse(false, 'Failed to confirm delivery.', [], 500);
        }
    }
    
    /**
     * Endpoint: GET /orders/seller
     * Retrieves seller orders with pagination
     */
    public function getSellerOrders()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $userData = $this->RouteProtection();
            
            // Get pagination parameters
            $page = isset($_GET['page']) ? max(1, intval($_GET['page'])) : 1;
            $pageSize = isset($_GET['page_size']) ? min(100, max(1, intval($_GET['page_size']))) : self::DEFAULT_PAGE_SIZE;
            $status = isset($_GET['status']) ? trim($_GET['status']) : null;

            // Get orders with pagination
            $result = $this->orderModel->getOrdersBySeller($userData->user_id, $page, $pageSize, $status);
            
            $this->sendResponse(true, 'Orders retrieved successfully.', [
                'orders' => $result['data'],
                'pagination' => $result['pagination']
            ]);
            
        } catch (Exception $e) {
            error_log("Get seller orders error [user:{$userData->user_id}]: " . $e->getMessage());
            $this->sendResponse(false, 'Failed to retrieve orders.', [], 500);
        }
    }

    /**
     * Endpoint: POST /orders/status/{orderId}
     * Allows seller to update order status with validation
     */
    public function updateOrderStatus($orderId)
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }
            
            $userData = $this->RouteProtection();
            $postData = $this->getData();
            $newStatus = trim($postData['status'] ?? '');
            
            if (empty($orderId) || empty($newStatus)) {
                $this->sendResponse(false, 'Order ID and new status are required.', [], 400);
            }
            
            // Get order details (no cache for status updates)
            $order = $this->orderModel->getOrderDetails($orderId, false);
            if (!$order || $order->seller_id !== $userData->user_id) {
                $this->sendResponse(false, 'Access Denied: You are not authorized to update this order.', [], 403);
            }
            
            // Validate status transition
            $allowedTransitions = [
                'pending' => ['processing', 'canceled'],
                'processing' => ['shipping', 'canceled'],
                'shipping' => ['delivered'],
            ];
            
            if (!isset($allowedTransitions[$order->order_status]) || 
                !in_array($newStatus, $allowedTransitions[$order->order_status])) {
                $this->sendResponse(false, "Cannot transition from {$order->order_status} to {$newStatus}.", [], 400);
            }
            
            // Handle cancellation with refund
            if ($newStatus === 'canceled') {
                $this->processCancellation($orderId, $order);
            }
            
            if ($this->orderModel->updateOrderStatus($orderId, $newStatus)) {
                // Queue notification
                $this->jobQueue->push('SendOrderStatusNotification', [
                    'order_id' => $orderId,
                    'buyer_id' => $order->buyer_id,
                    'new_status' => $newStatus
                ]);
                
                // Invalidate caches
                $this->cache->delete("order_details:{$orderId}");
                $this->invalidateOrderCaches($order->buyer_id, $order->seller_id);
                
                $this->sendResponse(true, "Order status updated to {$newStatus}.", [
                    'order_id' => $orderId,
                    'status' => $newStatus
                ]);
            } else {
                $this->sendResponse(false, 'Failed to update order status.', [], 500);
            }
            
        } catch (Exception $e) {
            error_log("Update order status error [order:{$orderId}]: " . $e->getMessage());
            $this->sendResponse(false, 'Failed to update order status.', [], 500);
        }
    }

    /**
     * Endpoint: GET /orders/buyer
     * Retrieves buyer orders with pagination and filtering
     */
    public function getBuyerOrders()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $userData = $this->RouteProtection();
            
            // Get pagination and filter parameters
            $page = isset($_GET['page']) ? max(1, intval($_GET['page'])) : 1;
            $pageSize = isset($_GET['page_size']) ? min(100, max(1, intval($_GET['page_size']))) : self::DEFAULT_PAGE_SIZE;
            $status = isset($_GET['status']) ? trim($_GET['status']) : null;

            $result = $this->orderModel->getOrdersByBuyer($userData->user_id, $page, $pageSize, $status);
            
            $this->sendResponse(true, 'Buyer order history retrieved successfully.', [
                'orders' => $result['data'],
                'pagination' => $result['pagination']
            ]);
            
        } catch (Exception $e) {
            error_log("Get buyer orders error [user:{$userData->user_id}]: " . $e->getMessage());
            $this->sendResponse(false, 'Failed to retrieve order history.', [], 500);
        }
    }

    /**
     * Endpoint: GET /orders/{orderId}
     * Retrieves single order with caching
     */
    public function getSingleOrder($orderId)
    {
        try {
            if (empty($orderId)) {
                $this->sendResponse(false, 'Order ID is required.', [], 400);
            }

            $userData = $this->RouteProtection();

            // Use cache for read operations
            $order = $this->orderModel->getOrderDetails($orderId, true);
            
            if (!$order) {
                $this->sendResponse(false, 'Order not found.', [], 404);
            }

            // Authorization check
            if ($order->buyer_id !== $userData->user_id && $order->seller_id !== $userData->user_id) {
                $this->sendResponse(false, 'Access Denied: You are not a party to this transaction.', [], 403);
            }

            $this->sendResponse(true, 'Order details retrieved successfully.', ['order' => $order]);
            
        } catch (Exception $e) {
            error_log("Get single order error [order:{$orderId}]: " . $e->getMessage());
            $this->sendResponse(false, 'Failed to retrieve order details.', [], 500);
        }
    }

    // ==================== HELPER METHODS ====================

    /**
     * Get listing with caching
     */
    private function getListingWithCache($listingId)
    {
        $cacheKey = "listing:{$listingId}";
        
        if ($this->cache) {
            $cached = $this->cache->get($cacheKey);
            if ($cached !== false) {
                return $cached;
            }
        }
        
        $listing = $this->listingModel->getSingleListingWithCategoryDetails($listingId);
        
        if ($listing && $this->cache) {
            $this->cache->set($cacheKey, $listing, 3600); // Cache for 1 hour
        }
        
        return $listing;
    }

    /**
     * Queue post-order tasks for async processing
     */
    private function queuePostOrderTasks($orderId, $buyerId, $sellerId)
    {
        if (!$this->jobQueue) return;

        // Send order confirmation emails
        $this->jobQueue->push('SendOrderConfirmationEmail', [
            'order_id' => $orderId,
            'buyer_id' => $buyerId
        ]);

        $this->jobQueue->push('SendNewOrderNotificationToSeller', [
            'order_id' => $orderId,
            'seller_id' => $sellerId
        ]);

        // Track analytics
        $this->jobQueue->push('TrackOrderAnalytics', [
            'order_id' => $orderId,
            'event' => 'order_created'
        ]);
    }

    /**
     * Process order cancellation with refund
     */
    private function processCancellation($orderId, $order)
    {
        // Refund to buyer if paid via wallet
        if ($order->payment_method === 'wallet') {
            $this->walletModel->creditWallet(
                $order->buyer_id, 
                $order->total_price, 
                'order_refund', 
                $orderId
            );
        }
        
        // Update escrow status
        $this->orderModel->updateEscrowTransactionStatus($orderId, 'refunded');
        
        // Queue refund notification
        if ($this->jobQueue) {
            $this->jobQueue->push('SendRefundNotification', [
                'order_id' => $orderId,
                'buyer_id' => $order->buyer_id,
                'amount' => $order->total_price
            ]);
        }
    }

    /**
     * Invalidate order-related caches
     */
    private function invalidateOrderCaches($buyerId, $sellerId)
    {
        if (!$this->cache) return;

        $this->cache->delete("buyer_orders:{$buyerId}");
        $this->cache->delete("seller_orders:{$sellerId}");
    }

    /**
     * Calculate estimated delivery date
     */
    private function calculateEstimatedDelivery()
    {
        // Simple calculation - can be made more sophisticated
        return date('Y-m-d', strtotime('+7 days'));
    }

    /**
     * Generate unique ID with timestamp
     */
    private function generateUniqueId($prefix)
    {
        return $prefix . '_' . bin2hex(random_bytes(8)) . '_' . time();
    }

    /**
     * Get cache instance (Redis, Memcached, etc.)
     */
    private function getCache()
    {
        // Initialize your cache system here
        // return new Redis();
        return null; // Fallback if no cache configured
    }

    /**
     * Get job queue instance
     */
    private function getJobQueue()
    {
        try {
            return new JobQueue();
        } catch (Exception $e) {
            error_log("Job queue connection failed: " . $e->getMessage());
            return null;
        }
    }
}

/**
 * Validation helper class
 */
class OrderValidator
{
    public function validateCheckout($data)
    {
        $required = ['listing_id', 'quantity', 'payment_method'];
        
        foreach ($required as $field) {
            if (empty($data[$field])) {
                return [
                    'valid' => false,
                    'message' => "Missing required field: {$field}"
                ];
            }
        }
        
        if (!in_array($data['payment_method'], ['wallet', 'card'])) {
            return [
                'valid' => false,
                'message' => 'Invalid payment method'
            ];
        }
        
        if (!is_numeric($data['quantity']) || $data['quantity'] < 1) {
            return [
                'valid' => false,
                'message' => 'Invalid quantity'
            ];
        }
        
        return ['valid' => true];
    }
}