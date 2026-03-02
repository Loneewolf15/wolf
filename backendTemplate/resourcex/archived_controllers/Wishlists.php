<?php
class Wishlists extends Controller
{
    public function __construct()
    {
        $this->wishlistModel = $this->model('Wishlist');
        $this->listingModel = $this->model('Listing');
        $this->userModel = $this->model('User');
        $this->serverKey = 'secret_server_key';
    }

    // Get all wishlisted listings for authenticated user
    public function getAllWishlisted()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            try {
                // Get pagination parameters
                $page = isset($_GET['page']) ? (int)$_GET['page'] : 1;
                $limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 20;
                $offset = ($page - 1) * $limit;

                // Check if pagination is requested
                if (isset($_GET['paginated']) && $_GET['paginated'] == 'true') {
                    $wishlistItems = $this->wishlistModel->getWishlistPaginated($userData->user_id, $limit, $offset);
                    $totalCount = $this->wishlistModel->getWishlistCount($userData->user_id);
                    $totalPages = ceil($totalCount / $limit);

                    $response = [
                        'status' => true,
                        'message' => 'Wishlist retrieved successfully',
                        'data' => $wishlistItems,
                        'pagination' => [
                            'current_page' => $page,
                            'total_pages' => $totalPages,
                            'total_count' => $totalCount,
                            'limit' => $limit
                        ]
                    ];
                } else {
                    // Get all wishlist items
                    $wishlistItems = $this->wishlistModel->getAllWishlisted($userData->user_id);
                    $totalCount = count($wishlistItems);

                    $response = [
                        'status' => true,
                        'message' => 'Wishlist retrieved successfully',
                        'data' => $wishlistItems,
                        'count' => $totalCount
                    ];
                }

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error retrieving wishlist: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Add listing to wishlist
    public function addToWishlist()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $postData = $this->getData();
            
            $listing_id = trim($postData['listing_id'] ?? '');

            if (empty($listing_id)) {
                $response = [
                    'status' => false,
                    'message' => 'Listing ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                $result = $this->wishlistModel->addToWishlist($userData->user_id, $listing_id);

                if ($result === true) {
                    $response = [
                        'status' => true,
                        'message' => 'Listing added to wishlist successfully'
                    ];
                } elseif ($result === "already_exists") {
                    $response = [
                        'status' => false,
                        'message' => 'Listing is already in your wishlist'
                    ];
                } elseif ($result === "listing_not_found") {
                    $response = [
                        'status' => false,
                        'message' => 'Listing not found'
                    ];
                } else {
                    $response = [
                        'status' => false,
                        'message' => 'Failed to add listing to wishlist'
                    ];
                }

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error adding to wishlist: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Remove listing from wishlist
    public function removeFromWishlist()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'POST' || $_SERVER['REQUEST_METHOD'] == 'DELETE') {
            $postData = $this->getData();
            
            $listing_id = trim($postData['listing_id'] ?? '');

            if (empty($listing_id)) {
                $response = [
                    'status' => false,
                    'message' => 'Listing ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                if ($this->wishlistModel->removeFromWishlist($userData->user_id, $listing_id)) {
                    $response = [
                        'status' => true,
                        'message' => 'Listing removed from wishlist successfully'
                    ];
                } else {
                    $response = [
                        'status' => false,
                        'message' => 'Failed to remove listing from wishlist or listing not in wishlist'
                    ];
                }

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error removing from wishlist: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Check if listing is in wishlist
    public function checkWishlist()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'GET' || $_SERVER['REQUEST_METHOD'] == 'POST') {
            $listing_id = '';
            
            if ($_SERVER['REQUEST_METHOD'] == 'GET') {
                $listing_id = isset($_GET['listing_id']) ? trim($_GET['listing_id']) : '';
            } else {
                $postData = $this->getData();
                $listing_id = trim($postData['listing_id'] ?? '');
            }

            if (empty($listing_id)) {
                $response = [
                    'status' => false,
                    'message' => 'Listing ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                $isInWishlist = $this->wishlistModel->isInWishlist($userData->user_id, $listing_id);

                $response = [
                    'status' => true,
                    'message' => 'Wishlist status checked successfully',
                    'in_wishlist' => $isInWishlist
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error checking wishlist status: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Clear entire wishlist
    public function clearWishlist()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'POST' || $_SERVER['REQUEST_METHOD'] == 'DELETE') {
            try {
                if ($this->wishlistModel->clearWishlist($userData->user_id)) {
                    $response = [
                        'status' => true,
                        'message' => 'Wishlist cleared successfully'
                    ];
                } else {
                    $response = [
                        'status' => false,
                        'message' => 'Failed to clear wishlist'
                    ];
                }

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error clearing wishlist: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Get wishlist count
    public function getWishlistCount()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            try {
                $count = $this->wishlistModel->getWishlistCount($userData->user_id);

                $response = [
                    'status' => true,
                    'message' => 'Wishlist count retrieved successfully',
                    'count' => $count
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error getting wishlist count: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Get recent wishlist items
    public function getRecentWishlist()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            try {
                $limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 5;
                $recentItems = $this->wishlistModel->getRecentWishlist($userData->user_id, $limit);

                $response = [
                    'status' => true,
                    'message' => 'Recent wishlist items retrieved successfully',
                    'data' => $recentItems,
                    'count' => count($recentItems)
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error getting recent wishlist: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Remove multiple items from wishlist
    public function removeMultiple()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $postData = $this->getData();

            $listing_ids = $postData['listing_ids'] ?? [];

            if (empty($listing_ids) || !is_array($listing_ids)) {
                $response = [
                    'status' => false,
                    'message' => 'Listing IDs array is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                if ($this->wishlistModel->removeMultipleFromWishlist($userData->user_id, $listing_ids)) {
                    $response = [
                        'status' => true,
                        'message' => 'Selected listings removed from wishlist successfully',
                        'removed_count' => count($listing_ids)
                    ];
                } else {
                    $response = [
                        'status' => false,
                        'message' => 'Failed to remove selected listings from wishlist'
                    ];
                }

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error removing multiple items: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }
}
