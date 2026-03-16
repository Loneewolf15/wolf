<?php

if (php_sapi_name() !== 'cli') {
    header("Access-Control-Allow-Origin: *");
    header("Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS");
    header("Access-Control-Allow-Headers: Content-Type, Authorization, X-Requested-With");

    // Handle preflight OPTIONS requests
    if ($_SERVER['REQUEST_METHOD'] == 'OPTIONS') {
        http_response_code(200);
        exit();
    }
}

function getData()
{
    if (php_sapi_name() === 'cli') {
        return [];
    }
    $raw = file_get_contents('php://input');
    $data = json_decode($raw, true);

    if (json_encode($data) === 'null' || empty($data)) {
        return $_POST;
    } else {
        return $data;
    }
}

// Get request data
$val = getData();

if (php_sapi_name() !== 'cli') {
    // Check requestID for POST requests only (GET requests don't need requestID)
    $requestMethod = $_SERVER['REQUEST_METHOD'];

    if ($requestMethod === 'POST' || $requestMethod === 'PUT' || $requestMethod === 'DELETE') {
        // For POST/PUT/DELETE requests, require requestID
        if (!isset($val['requestID']) || $val['requestID'] !== 'pid2025') {
            $response = [
                'status' => false,
                'message' => 'Invalid Access - requestID required for ' . $requestMethod . ' requests'
            ];
            echo json_encode($response);
            exit;
        }
        $dbname = 'market_plaza_' . $val['requestID'];
    } else {
        // For GET requests, use default database or check query parameter
        $requestID = $_GET['requestID'] ?? 'pid2025';
        if ($requestID === 'pid2025') {
            $dbname = 'market_plaza_' . $requestID;
        } else {
            $dbname = 'market_plaza_pid2025'; // Default database
        }
    }
} else {
    // For CLI, use a default database
    $dbname = 'market_plaza_pid2025';
}


// DB Params
define('DB_HOST', 'localhost');
define('DB_USER', 'root');
define('DB_PASS', '');
define('DB_NAME', $dbname);

// App Root
define('APPROOT', dirname(dirname(__FILE__)));
define('PUBLIC_PATH', APPROOT . '/public/assets');
// URL Root
define('URLROOT', 'http://localhost/backendTemplate/');

// Site Name
define('SITENAME', 'DivineAPI');

define('RESELLER_UPGRADE_FEE', 500); // Upgrade fee for reseller role

// Continue processing the request as needed

// Social Login Credentials (Replace with your actual keys)
define('GOOGLE_CLIENT_ID', 'YOUR_GOOGLE_CLIENT_ID');
define('FACEBOOK_APP_ID', 'YOUR_FACEBOOK_APP_ID');
define('FACEBOOK_APP_SECRET', 'YOUR_FACEBOOK_APP_SECRET');
define('APPLE_CLIENT_ID', 'YOUR_APPLE_CLIENT_ID');
