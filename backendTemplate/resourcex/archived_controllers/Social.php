<?php
class Social extends Controller
{
    public function __construct()
    {
        $this->socialModel = $this->model('Social');
        $this->userModel = $this->model('User');
        $this->dashboardModel = $this->model('Dashboard');
        $this->serverKey = 'secret_server_key';
    }

    // Get user's public profile
    public function getUserProfile($user_id = null)
    {
        // Get viewer info if authenticated
        $viewer_id = null;
        try {
            $userData = $this->RouteProtection();
            $viewer_id = $userData->user_id;
        } catch (UnexpectedValueException $e) {
            // Not authenticated - continue as guest
        }

        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            if (!$user_id) {
                $user_id = $_GET['user_id'] ?? '';
            }

            if (empty($user_id)) {
                $response = [
                    'status' => false,
                    'message' => 'User ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                $profile = $this->socialModel->getUserProfile($user_id, $viewer_id);

                if (!$profile) {
                    $response = [
                        'status' => false,
                        'message' => 'User not found or not verified'
                    ];
                    print_r(json_encode($response));
                    exit;
                }

                $response = [
                    'status' => true,
                    'message' => 'User profile retrieved successfully',
                    'data' => $profile
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error retrieving user profile: ' . $e->getMessage()
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

    // Follow a user
    public function followUser()
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
            $following_id = trim($postData['user_id'] ?? '');

            if (empty($following_id)) {
                $response = [
                    'status' => false,
                    'message' => 'User ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            if ($userData->user_id === $following_id) {
                $response = [
                    'status' => false,
                    'message' => 'You cannot follow yourself'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                if ($this->socialModel->followUser($userData->user_id, $following_id)) {
                    $response = [
                        'status' => true,
                        'message' => 'User followed successfully'
                    ];
                } else {
                    $response = [
                        'status' => false,
                        'message' => 'Failed to follow user or already following'
                    ];
                }

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error following user: ' . $e->getMessage()
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

    // Unfollow a user
    public function unfollowUser()
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
            $following_id = trim($postData['user_id'] ?? '');

            if (empty($following_id)) {
                $response = [
                    'status' => false,
                    'message' => 'User ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                if ($this->socialModel->unfollowUser($userData->user_id, $following_id)) {
                    $response = [
                        'status' => true,
                        'message' => 'User unfollowed successfully'
                    ];
                } else {
                    $response = [
                        'status' => false,
                        'message' => 'Failed to unfollow user'
                    ];
                }

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error unfollowing user: ' . $e->getMessage()
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

    // Get user's followers
    public function getFollowers($user_id = null)
    {
        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            if (!$user_id) {
                $user_id = $_GET['user_id'] ?? '';
            }

            if (empty($user_id)) {
                $response = [
                    'status' => false,
                    'message' => 'User ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                $page = isset($_GET['page']) ? (int)$_GET['page'] : 1;
                $limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 20;
                $offset = ($page - 1) * $limit;

                $followers = $this->socialModel->getFollowers($user_id, $limit, $offset);

                $response = [
                    'status' => true,
                    'message' => 'Followers retrieved successfully',
                    'data' => $followers,
                    'pagination' => [
                        'current_page' => $page,
                        'limit' => $limit
                    ]
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error retrieving followers: ' . $e->getMessage()
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

    // Get users that a user is following
    public function getFollowing($user_id = null)
    {
        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            if (!$user_id) {
                $user_id = $_GET['user_id'] ?? '';
            }

            if (empty($user_id)) {
                $response = [
                    'status' => false,
                    'message' => 'User ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                $page = isset($_GET['page']) ? (int)$_GET['page'] : 1;
                $limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 20;
                $offset = ($page - 1) * $limit;

                $following = $this->socialModel->getFollowing($user_id, $limit, $offset);

                $response = [
                    'status' => true,
                    'message' => 'Following list retrieved successfully',
                    'data' => $following,
                    'pagination' => [
                        'current_page' => $page,
                        'limit' => $limit
                    ]
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error retrieving following list: ' . $e->getMessage()
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

    // Add a review for a user
    public function addUserReview()
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
            
            $reviewed_user_id = trim($postData['user_id'] ?? '');
            $rating = (int)($postData['rating'] ?? 0);
            $review_text = trim($postData['review_text'] ?? '');
            $listing_id = trim($postData['listing_id'] ?? '');
            $review_type = trim($postData['review_type'] ?? 'general');

            // Validation
            if (empty($reviewed_user_id)) {
                $response = [
                    'status' => false,
                    'message' => 'User ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            if ($rating < 1 || $rating > 5) {
                $response = [
                    'status' => false,
                    'message' => 'Rating must be between 1 and 5'
                ];
                print_r(json_encode($response));
                exit;
            }

            if ($userData->user_id === $reviewed_user_id) {
                $response = [
                    'status' => false,
                    'message' => 'You cannot review yourself'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                $review_id = $this->socialModel->addUserReview(
                    $userData->user_id,
                    $reviewed_user_id,
                    $rating,
                    $review_text ?: null,
                    $listing_id ?: null,
                    $review_type
                );

                if ($review_id) {
                    $response = [
                        'status' => true,
                        'message' => 'Review added successfully',
                        'review_id' => $review_id
                    ];
                } else {
                    $response = [
                        'status' => false,
                        'message' => 'Failed to add review'
                    ];
                }

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error adding review: ' . $e->getMessage()
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

    // Get reviews for a user
    public function getUserReviews($user_id = null)
    {
        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            if (!$user_id) {
                $user_id = $_GET['user_id'] ?? '';
            }

            if (empty($user_id)) {
                $response = [
                    'status' => false,
                    'message' => 'User ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                $page = isset($_GET['page']) ? (int)$_GET['page'] : 1;
                $limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 20;
                $offset = ($page - 1) * $limit;

                $reviews = $this->socialModel->getUserReviews($user_id, $limit, $offset);
                $ratingStats = $this->socialModel->getUserRatingStats($user_id);

                $response = [
                    'status' => true,
                    'message' => 'User reviews retrieved successfully',
                    'data' => [
                        'reviews' => $reviews,
                        'rating_stats' => $ratingStats
                    ],
                    'pagination' => [
                        'current_page' => $page,
                        'limit' => $limit
                    ]
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error retrieving reviews: ' . $e->getMessage()
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

    // Get suggested users to follow
    public function getSuggestedFollows()
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
                $limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 10;
                $suggestions = $this->socialModel->getSuggestedFollows($userData->user_id, $limit);

                $response = [
                    'status' => true,
                    'message' => 'Suggested follows retrieved successfully',
                    'data' => $suggestions
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error retrieving suggestions: ' . $e->getMessage()
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
