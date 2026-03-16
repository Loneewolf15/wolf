<?php
class Listings extends Controller
{
    protected $listingModel;
    protected $userModel;
    protected $jobQueue;
    protected $cache;

    public function __construct()
    {
        $this->listingModel = $this->model('Listing');
        $this->userModel = $this->model('User');
        $this->serverKey = 'secret_server_key';
        $this->jobQueue = $this->getJobQueue();
        $this->cache = $this->getCache();
    }
        private function sendResponse($status, $message, $data = [], $httpCode = 200) {
        http_response_code($httpCode);
        echo json_encode(['status' => $status, 'message' => $message, 'data' => $data]);
        exit;
    }

    // Endpoint to create a new listing
public function createListing()
{
    try {
        if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
            $this->sendResponse(false, 'Invalid request method.', [], 405);
        }

        $userData = $this->RouteProtection();

        $userRoles = $this->userModel->getRolesForUser($userData->user_id);
        $isSeller = in_array('Seller', array_column($userRoles, 'role_name'));
        if (!$isSeller) {
            $this->sendResponse(false, 'Access Denied: You must be a seller to create a listing.', [], 403);
        }

        $postData = $this->getData();
        $data = [
            'user_id' => $userData->user_id,
            'title' => trim($postData['title'] ?? ''),
            'description' => trim($postData['description'] ?? ''),
            'price' => floatval($postData['price'] ?? 0),
            'category_id' => trim($postData['category_id'] ?? ''),
            'status' => 'active',
            'is_physical' => boolval($postData['is_physical'] ?? true),
            'is_resellable' => boolval($postData['is_resellable'] ?? false),
            'reseller_commission_percent' => floatval($postData['reseller_commission_percent'] ?? 0),
            'location' => $userData->location,
            'media_files' => $_FILES['media_files'] ?? []
        ];

        if (empty($data['title'])) {
            $this->sendResponse(false, 'Please enter a title for the listing.', [], 400);
        }
        if (empty($data['description'])) {
            $this->sendResponse(false, 'Please provide a description.', [], 400);
        }
        if ($data['price'] <= 0) {
            $this->sendResponse(false, 'Please enter a valid price.', [], 400);
        }
        if (empty($data['category_id'])) {
            $this->sendResponse(false, 'Please select a category.', [], 400);
        }
        if (empty($data['media_files'])) {
            $this->sendResponse(false, 'Please upload at least one image or video.', [], 400);
        }
        if ($data['is_resellable'] && $data['reseller_commission_percent'] <= 0) {
            $this->sendResponse(false, 'Please enter a valid reseller commission percentage.', [], 400);
        }

        $listingId = $this->generateUniqueId('listing_');
        $listingData = array_merge($data, ['listing_id' => $listingId]);

        // Step 1: Call handleMediaUploads to process the files
        $uploadedMedia = $this->handleMediaUploads($data['media_files'], $listingId);

        // Step 2: Check if any files were uploaded successfully
        if (empty($uploadedMedia)) {
            // Log the error and return a failure response
            error_log('Failed to upload any media files.');
            $this->sendResponse(false, 'Failed to upload media files. Check file size and type.', [], 500);
        }

        // Step 3: Pass both the listing data and the uploaded media URLs to the model
        if ($this->listingModel->createListingAndMedia($listingData, $uploadedMedia)) {
            // Queue a notification email
            if ($this->jobQueue) {
                $this->jobQueue->push('SendNewListingNotification', [
                    'user_id' => $userData->user_id,
                    'listing_id' => $listingId,
                    'title' => $data['title']
                ]);
            }
            $this->sendResponse(true, 'Listing created successfully.', ['listing_id' => $listingId], 201);
        } else {
            // If the model fails, delete the uploaded files to clean up
            $this->deleteUploadedFiles($uploadedMedia);
            $this->sendResponse(false, 'Failed to create listing.', [], 500);
        }
    } catch (UnexpectedValueException $e) {
        $this->sendResponse(false, 'Unauthorized: ' . $e->getMessage(), [], 401);
    } catch (Exception $e) {
        error_log("Create Listing Error: " . $e->getMessage() . " on line " . $e->getLine() . " in " . $e->getFile());
        $this->sendResponse(false, 'An unexpected error occurred while creating the listing.', [], 500);
    }
}
 // Helper function to generate unique ID
    private function generateUniqueId($prefix)
    {
        return $prefix . date('Ymdhms') . bin2hex(random_bytes(20));
    }

    // Helper function to handle file uploads
private function handleMediaUploads($files, $listingId)
{
    $uploadedUrls = [];
    $uploadDir = 'uploads/listings/' . $listingId . '/';

    if (!is_dir($uploadDir)) {
        if (!mkdir($uploadDir, 0755, true)) {
            return [];
        }
    }
    
    $normalizedFiles = $this->reArrayFiles($files);

    $imageCounter = 1;
    $videoCounter = 1;

    foreach ($normalizedFiles as $file) {
        if ($file['error'] === UPLOAD_ERR_OK) {
            $allowedTypes = ['image/jpFeg', 'image/jpg', 'image/png', 'video/mp4', 'video/quicktime'];
            $maxSize = 20 * 1024 * 1024; // 20MB

            if (!in_array($file['type'], $allowedTypes) || $file['size'] > $maxSize) {
                continue;
            }

            $fileExtension = pathinfo($file['name'], PATHINFO_EXTENSION);
            
            // Determine file type and create the unique name
            if (strpos($file['type'], 'image') !== false) {
                $fileName = "product_img{$imageCounter}." . $fileExtension;
                $imageCounter++;
            } else {
                $fileName = "product_vid{$videoCounter}." . $fileExtension;
                $videoCounter++;
            }
            
            $filePath = $uploadDir . $fileName;

            if (move_uploaded_file($file['tmp_name'], $filePath)) {
                $uploadedUrls[] = [
                    'url' => 'assets/uploads/listings/' . $listingId . '/' . $fileName,
                    'type' => (strpos($file['type'], 'image') !== false) ? 'image' : 'video'
                ];
            }
        }
    }
    return $uploadedUrls;
}
// Helper function to re-organize the $_FILES array for easier looping
private function reArrayFiles(&$file_post)
{
    // Check if the input is a single file upload
    if (!is_array($file_post['name'])) {
        // Normalize the single file into an array with one element
        $file_ary = array();
        $file_ary[0] = $file_post;
        return $file_ary;
    }
    
    // Original logic for multiple files
    $file_ary = array();
    $file_count = count($file_post['name']);
    $file_keys = array_keys($file_post);

    for ($i = 0; $i < $file_count; $i++) {
        foreach ($file_keys as $key) {
            $file_ary[$i][$key] = $file_post[$key][$i];
        }
    }
    return $file_ary;
}

    // Helper function to delete files on failure
    private function deleteUploadedFiles($files)
    {
        foreach ($files as $file) {
            // if (file_exists($file['url'])) {
            //     unlink($file['url']);
            $fullPath = APPROOT . '/../public/' . $file['url'];
            if(file_exists($fullPath)) {
                unlink($fullPath);
            
            }
        }
    }


    // Endpoint to get all listings
    public function getAllListings()
    {
        // Get pagination parameters
        $page = isset($_GET['page']) ? max(1, intval($_GET['page'])) : 1;
        $pageSize = isset($_GET['page_size']) ? min(100, max(1, intval($_GET['page_size']))) : 20;

        // Get user data for personalization
        $userData = $this->getOptionalUser();
        
        $personalizationData = [
            'user_id' => null,
            'state' => null,
            'lga' => null,
            'search_history' => []
        ];

        if ($userData) {
            $personalizationData['user_id'] = $userData->user_id;
            $personalizationData['state'] = $userData->state;
            $personalizationData['lga'] = $userData->local_government_area;
            $personalizationData['search_history'] = $this->listingModel->getRecentSearchTerms($userData->user_id);
        }

        $result = $this->listingModel->getAllListings($page, $pageSize, $personalizationData);

        if ($result) {
            $this->sendResponse(true, 'Listings retrieved successfully.', [
                'listings' => $result['data'],
                'pagination' => $result['pagination']
            ]);
        } else {
            $this->sendResponse(false, 'No listings found.', ['listings' => []]);
        }
    }

    // Endpoint to edit an existing listing
    public function editListing()
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

    // Data for validation and insertion, including new reseller fields
    $data = [
        'user_id' => $userData->user_id,
        'listing_id' => trim($postData['listing_id'] ?? ''),
        'title' => trim($postData['title'] ?? ''),
        'description' => trim($postData['description'] ?? ''),
        'price' => floatval($postData['price'] ?? 0),
        'category_id' => trim($postData['category_id'] ?? ''),
        'is_physical' => boolval($postData['is_physical'] ?? false),
        'is_resellable' => boolval($postData['is_resellable'] ?? false),
        'reseller_commission_percent' => floatval($postData['reseller_commission_percent'] ?? 0),
        'media_files' => $_FILES['media_files'] ?? []
    ];

    // Basic validation
    if (empty($data['listing_id'])) {
        $this->sendResponse(false, 'Listing ID is required.', [], 400);
    }
    if (!$this->listingModel->isOwner($data['listing_id'], $data['user_id'])) {
        $this->sendResponse(false, 'Access Denied: You do not own this listing.', [], 403);
    }
    if (empty($data['title'])) {
        $this->sendResponse(false, 'Title is required.', [], 400);
    }
    if (empty($data['description'])) {
        $this->sendResponse(false, 'Description is required.', [], 400);
    }
    if ($data['price'] <= 0) {
        $this->sendResponse(false, 'A valid price is required.', [], 400);
    }
    if (empty($data['category_id'])) {
        $this->sendResponse(false, 'Category is required.', [], 400);
    }

    // Validation for reseller commission
    if ($data['is_resellable'] && $data['reseller_commission_percent'] <= 0) {
        $this->sendResponse(false, 'Reseller commission percentage is required for resellable items.', [], 400);
    }

    // Handle media updates
    if (!empty($data['media_files']['name'][0])) {
        if (!$this->listingModel->deleteListingMedia($data['listing_id'])) {
            $this->sendResponse(false, 'Failed to clear old listing media.', [], 500);
        }
        $mediaUpdates = $this->handleMediaUploads($data['media_files'], $data['listing_id']);
        if (!$this->listingModel->saveListingMedia($data['listing_id'], $mediaUpdates)) {
            $this->sendResponse(false, 'Failed to save new listing media.', [], 500);
        }
    }
    
    // Prepare data for the model
    $listingUpdateData = [
        'listing_id' => $data['listing_id'],
        'title' => $data['title'],
        'description' => $data['description'],
        'price' => $data['price'],
        'category_id' => $data['category_id'],
        'is_physical' => $data['is_physical'],
        'is_resellable' => $data['is_resellable'],
        'reseller_commission_percent' => $data['reseller_commission_percent']
    ];
    
    if ($this->listingModel->updateListing($listingUpdateData)) {
        if ($this->cache) {
            $this->cache->del("listing_details_{$data['listing_id']}");
        }
        $this->sendResponse(true, 'Listing updated successfully.', ['listing_id' => $data['listing_id']]);
    } else {
        $this->sendResponse(false, 'Failed to update listing.', [], 500);
    }
}


    //Endpoint to get all categories data
    public function getAllCategories()
    {
        if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
            $this->sendResponse(false, 'Invalid request method.', [], 405);
        }

        $categories = $this->listingModel->getAllCategories();
        
        if (empty($categories)) {
            $this->sendResponse(true, 'No categories found.', ['categories' => []]);
        }

        $hierarchicalCategories = $this->buildCategoryTree($categories);
        
        $this->sendResponse(true, 'Categories retrieved successfully.', ['categories' => $hierarchicalCategories]);
    }

    private function buildCategoryTree($categories, $parentId = NULL)
    {
        $tree = [];
        foreach ($categories as $category) {
            if ($category->parent_id === $parentId) {
                $children = $this->buildCategoryTree($categories, $category->category_id);
                if ($children) {
                    $category->children = $children;
                }
                $tree[] = $category;
            }
        }
        return $tree;
    }

    // Endpoint to get all listings for a specific user
public function getUserListings($userId)
{
    try {
        $userData = $this->RouteProtection();
    } catch (UnexpectedValueException $e) {
        $this->sendResponse(false, 'Unauthorized: ' . $e->getMessage());
    }

    // Check if the requested userId matches the authenticated user's ID
    if ($userData->user_id !== $userId) {
        $this->sendResponse(false, 'Access Denied: You can only view your own listings.');
    }

    // After passing all security checks, proceed to retrieve the listings
    if (empty($userId)) {
        $this->sendResponse(false, 'User ID is required.');
    }

    // Get pagination parameters
    $page = isset($_GET['page']) ? max(1, intval($_GET['page'])) : 1;
    $pageSize = isset($_GET['page_size']) ? min(100, max(1, intval($_GET['page_size']))) : 20;

    $result = $this->listingModel->getUserListings($userId, $page, $pageSize);

    if ($result) {
        $this->sendResponse(true, 'User listings retrieved successfully.', [
            'listings' => $result['data'],
            'pagination' => $result['pagination']
        ]);
    } else {
        $this->sendResponse(false, 'No listings found for this user.', ['listings' => []]);
    }
}



// Deletes a listing
    public function deleteListing($listingId)
    {
        if ($_SERVER['REQUEST_METHOD'] !== 'DELETE') {
            $this->sendResponse(false, 'Invalid request method.', [], 405);
        }

        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $this->sendResponse(false, 'Unauthorized.', [], 401);
        }

        if (empty($listingId)) {
            $this->sendResponse(false, 'Listing ID is required.', [], 400);
        }

        if (!$this->listingModel->isOwner($listingId, $userData->user_id)) {
            $this->sendResponse(false, 'Access Denied: You do not own this listing.', [], 403);
        }
        
        // Optional: Get media file paths to delete from file system
        $mediaFilePaths = $this->listingModel->getMediaFilePaths($listingId);

        if ($this->listingModel->deleteListing($listingId)) {
            if ($this->cache) {
                $this->cache->del("listing_details_{$listingId}");
            }
         //   Delete files from the server
            foreach ($mediaFilePaths as $path) {
                if (file_exists($path)) {
                    unlink($path);
                }
            }
            $this->sendResponse(true, 'Listing deleted successfully.');
        } else {
            $this->sendResponse(false, 'Failed to delete listing.', [], 500);
        }
    }

public function searchListings()
{
    if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
        $this->sendResponse(false, 'Invalid request method.', [], 405);
    }

    $queryParams = $_GET;

    // Sanitize and validate query parameters
    $searchTerm = trim($queryParams['q'] ?? '');

    // Log search query if user is authenticated
    $userData = $this->getOptionalUser();
    if ($userData && !empty($searchTerm)) {
        $this->listingModel->logSearchHistory($userData->user_id, $searchTerm);
    }

    $categoryId = trim($queryParams['category_id'] ?? '');
    $priceMin = floatval($queryParams['price_min'] ?? 0);
    $priceMax = floatval($queryParams['price_max'] ?? 0);
    $location = trim($queryParams['location'] ?? '');
    $sortBy = trim($queryParams['sort_by'] ?? 'newest'); // default sort
    $rating = intval($queryParams['rating'] ?? 0);

    // Validate sort_by value to prevent SQL injection
    $allowedSorts = ['newest', 'lowest_price', 'highest_price', 'popularity'];
    if (!in_array($sortBy, $allowedSorts)) {
        $sortBy = 'newest';
    }

    // Get pagination parameters
    $page = isset($_GET['page']) ? max(1, intval($_GET['page'])) : 1;
    $pageSize = isset($_GET['page_size']) ? min(100, max(1, intval($_GET['page_size']))) : 20;

    $searchData = [
        'searchTerm' => $searchTerm,
        'categoryId' => $categoryId,
        'priceMin' => $priceMin,
        'priceMax' => $priceMax,
        'location' => $location,
        'sortBy' => $sortBy,
        'rating' => $rating,
        'page' => $page,
        'pageSize' => $pageSize
    ];
    
    $result = $this->listingModel->searchListings($searchData);

    if ($result) {
        $this->sendResponse(true, 'Search results retrieved successfully.', [
            'results' => $result['data'],
            'pagination' => $result['pagination']
        ]);
    } else {
        $this->sendResponse(true, 'No listings found matching your criteria.', ['results' => []]);
    }
}

public function getListingsByCategory($categoryId)
{
    if (empty($categoryId)) {
        $this->sendResponse(false, 'Category ID is required.', [], 400);
    }
    
    // Get pagination parameters
    $page = isset($_GET['page']) ? max(1, intval($_GET['page'])) : 1;
    $pageSize = isset($_GET['page_size']) ? min(100, max(1, intval($_GET['page_size']))) : 20;

    $result = $this->listingModel->getListingsByCategory($categoryId, $page, $pageSize);

    if ($result) {
        $this->sendResponse(true, 'Listings retrieved successfully.', [
            'listings' => $result['data'],
            'pagination' => $result['pagination']
        ]);
    } else {
        $this->sendResponse(true, 'No listings found for this category.', ['listings' => []]);
    }
}

public function updateListingStatus($listingId)
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
    $newStatus = trim($postData['status'] ?? '');

    if (empty($listingId) || empty($newStatus)) {
        $this->sendResponse(false, 'Listing ID and new status are required.', [], 400);
    }
    
    if (!$this->listingModel->isOwner($listingId, $userData->user_id)) {
        $this->sendResponse(false, 'Access Denied: You do not own this listing.', [], 403);
    }

    if ($this->listingModel->updateListingStatus($listingId, $newStatus)) {
        if ($this->cache) {
            $this->cache->del("listing_details_{$listingId}");
        }
        $this->sendResponse(true, 'Listing status updated successfully.');
    } else {
        $this->sendResponse(false, 'Failed to update listing status.', [], 500);
    }
}

    public function getTopRatedListings()
{
    // Get pagination parameters
    $page = isset($_GET['page']) ? max(1, intval($_GET['page'])) : 1;
    $pageSize = isset($_GET['page_size']) ? min(100, max(1, intval($_GET['page_size']))) : 20;

    $result = $this->listingModel->getTopRatedListings($page, $pageSize);

    if ($result) {
        $this->sendResponse(true, 'Top-rated listings retrieved successfully.', [
            'listings' => $result['data'],
            'pagination' => $result['pagination']
        ]);
    } else {
        $this->sendResponse(true, 'No top-rated listings found.', ['listings' => []]);
    }
}

    private function getCache()
    {
        try {
            return new Cache();
        } catch (Exception $e) {
            error_log("Cache initialization failed: " . $e->getMessage());
            return null;
        }
    }

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

