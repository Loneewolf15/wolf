<?php
class Resellers extends Controller
{
    protected $resellerModel;
    protected $userModel;
    protected $listingModel;
    protected $subscriptionMiddleware;

    public function __construct()
    {
        $this->resellerModel = $this->model('Reseller');
        $this->userModel = $this->model('User');
        $this->listingModel = $this->model('Listing');
        $this->subscriptionMiddleware = new SubscriptionMiddleware();
    }

    private function sendResponse($status, $message, $data = [], $httpCode = 200) {
        http_response_code($httpCode);
        echo json_encode(['status' => $status, 'message' => $message, 'data' => $data]);
        exit;
    }

    // Endpoint: GET /reseller/listings/all
    // Retrieves all listings that are marked as resellable
    public function getAllResellableListings()
    {
        if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
            $this->sendResponse(false, 'Invalid request method.', [], 405);
        }

        $userId = $_SESSION['user_id'] ?? null; // Assuming user ID is in session
        if (!$userId) {
            $this->sendResponse(false, 'Unauthorized: User not logged in.', [], 401);
        }

        // Assuming 'Reseller Plan' is the name of the subscription plan for resellers
        $check = $this->subscriptionMiddleware->checkSubscription($userId, 'Reseller Monthly');
        if (!$check['status']) {
            $this->sendResponse(false, $check['message'], [], 403); 
            return;// Forbidden
        }

        $listings = $this->resellerModel->getAllResellableListings();
        
        if ($listings) {
            $this->sendResponse(true, 'Resellable listings retrieved successfully.', ['listings' => $listings]);
        } else {
            $this->sendResponse(true, 'No resellable listings found.', ['listings' => []]);
        }
    }
    
}