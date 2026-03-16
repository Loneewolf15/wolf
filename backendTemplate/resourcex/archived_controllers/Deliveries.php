<?php
class Deliveries extends Controller
{
    protected $deliveryModel;
    protected $userModel;
// //// leave out the postman json and focus on the openAPI spec, we are not using REST means of fetching endpoints, we are using the PHP default method.. for example,    │
// │    /deliveries/{delivery_id}/status, should be deliveries/delivery_id, which would then return all data to the frontend and he would perform necessary requirements   │
// fromthence                                                                                                                                                        │
    public function __construct()
    {
        $this->deliveryModel = $this->model('Delivery');
        $this->userModel = $this->model('User');
    }

    private function sendResponse($status, $message, $data = [], $httpCode = 200) {
        
        http_response_code($httpCode);
        echo json_encode(['status' => $status, 'message' => $message, 'data' => $data]);
        exit;
    }
    // Retrieves a list of available agents near the seller's location.
    public function getNearbyAgents()
    {
        if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
            $this->sendResponse(false, 'Invalid request method.', [], 405);
        }

        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $this->sendResponse(false, 'Unauthorized.', [], 401);
        }
        
        // Use the seller's stored location for local-friendly matching
        $sellerLGA = $userData->local_government_area; 

        if (empty($sellerLGA)) {
            $this->sendResponse(false, 'Location data missing. Please update your profile with your Local Government Area.', [], 400);
        }

        $agents = $this->deliveryModel->findNearbyAgents($sellerLGA);
        
        if ($agents) {
            $this->sendResponse(true, 'Nearby agents retrieved successfully.', ['agents' => $agents]);
        } else {
            $this->sendResponse(true, 'No available delivery agents found in your area.', ['agents' => []]);
        }
    }
}