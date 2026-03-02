<?php
class Reviews extends Controller
{
    protected $reviewModel;
    protected $userModel;
    protected $orderModel;

    public function __construct()
    {
        $this->reviewModel = $this->model('Review');
        $this->userModel = $this->model('User');
        $this->orderModel = $this->model('Order');
    }

    private function sendResponse($status, $message, $data = [], $httpCode = 200) {
        http_response_code($httpCode);
        echo json_encode(['status' => $status, 'message' => $message, 'data' => $data]);
        exit;
    }

    public function reportReview($reviewId)
{
    if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
        $this->sendResponse(false, 'Invalid request method.', [], 405);
    }

    try {
        $userData = $this->RouteProtection();
    } catch (UnexpectedValueException $e) {
        $this->sendResponse(false, 'Unauthorized.', [], 401);
    }
    
    $postData = $this->getData();
    $reportReason = trim($postData['reason'] ?? '');

    if (empty($reviewId)) {
        $this->sendResponse(false, 'Review ID is required.', [], 400);
    }
    if (empty($reportReason)) {
        $this->sendResponse(false, 'A reason for the report is required.', [], 400);
    }

    if ($this->reviewModel->reportReview($reviewId, $reportReason)) {
        $this->sendResponse(true, 'Review reported for moderation. Thank you for your feedback!');
    } else {
        $this->sendResponse(false, 'Failed to report review.', [], 500);
    }
}

public function getSellerReviews($sellerId)
{
    if (empty($sellerId)) {
        $this->sendResponse(false, 'Seller ID is required.', [], 400);
    }
    
    $reviews = $this->reviewModel->getReviewsBySeller($sellerId);
    
    if ($reviews) {
        $this->sendResponse(true, 'Seller reviews retrieved successfully.', ['reviews' => $reviews]);
    } else {
        $this->sendResponse(true, 'No reviews found for this seller.', ['reviews' => []]);
    }
}
    public function getSellerAverageRating($sellerId)
{
    if (empty($sellerId)) {
        $this->sendResponse(false, 'Seller ID is required.', [], 400);
    }

    $averageRating = $this->reviewModel->getSellerAverageRating($sellerId);
    
    $this->sendResponse(true, 'Seller rating retrieved successfully.', ['average_rating' => $averageRating]);
}

    // Endpoint: DELETE /reviews/delete/{reviewId}
    public function deleteReview($reviewId)
    {
        if ($_SERVER['REQUEST_METHOD'] !== 'DELETE') {
            $this->sendResponse(false, 'Invalid request method.', [], 405);
        }

        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $this->sendResponse(false, 'Unauthorized.', [], 401);
        }

        if (empty($reviewId)) {
            $this->sendResponse(false, 'Review ID is required.', [], 400);
        }

        if (!$this->reviewModel->isReviewOwner($reviewId, $userData->user_id)) {
            $this->sendResponse(false, 'Access Denied: You do not own this review.', [], 403);
        }

        if ($this->reviewModel->deleteReview($reviewId)) {
            $this->sendResponse(true, 'Review deleted successfully.');
        } else {
            $this->sendResponse(false, 'Failed to delete review.', [], 500);
        }
    }
    // Endpoint: GET /reviews/listing/{listingId}
    public function getListingReviews($listingId)
    {
        if (empty($listingId)) {
            $this->sendResponse(false, 'Listing ID is required.', [], 400);
        }
        
        $reviews = $this->reviewModel->getReviewsByListing($listingId);
        if ($reviews) {
            $this->sendResponse(true, 'Reviews retrieved successfully.', ['reviews' => $reviews]);
        } else {
            $this->sendResponse(true, 'No reviews found for this listing.', ['reviews' => []]);
        }
    }

    // Endpoint: POST /reviews/create
    public function addOrEditReview()
    {
        if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
            $this->sendResponse(false, 'Invalid request method.', [], 405);
        }

        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $this->sendResponse(false, 'Unauthorized.', [], 401);
        }

        $postData = $this->getData();
        $data = [
            'listing_id' => trim($postData['listing_id'] ?? ''),
            'reviewer_id' => $userData->user_id,
            'rating' => intval($postData['rating'] ?? 0),
            'comment' => trim($postData['comment'] ?? ''),
        ];

        // 1. Critical Validation: Check if the user has a completed order
        if (!$this->reviewModel->hasCompletedOrder($data['listing_id'], $data['reviewer_id'])) {
            $this->sendResponse(false, 'You can only review products you have purchased.', [], 403);
        }

        // 2. Input Validation
        if (empty($data['listing_id']) || $data['rating'] < 1 || $data['rating'] > 5) {
            $this->sendResponse(false, 'Listing ID and a valid rating (1-5) are required.', [], 400);
        }
        
        // 3. Check for existing review
        $existingReview = $this->reviewModel->getReviewByUser($data['listing_id'], $data['reviewer_id']);
        if ($existingReview) {
            if ($this->reviewModel->updateReview($data)) {
                $this->sendResponse(true, 'Review updated successfully.');
            } else {
                $this->sendResponse(false, 'Failed to update review.', [], 500);
            }
        } else {
            $data['review_id'] = $this->generateUniqueId('review');
            if ($this->reviewModel->addReview($data)) {
                $this->sendResponse(true, 'Review added successfully.', [], 201);
            } else {
                $this->sendResponse(false, 'Failed to add review.', [], 500);
            }
        }
    }

    private function generateUniqueId($prefix) {
        return $prefix . bin2hex(random_bytes(8));
    }
}