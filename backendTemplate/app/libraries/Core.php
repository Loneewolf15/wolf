<?php
/* 
   *  APP CORE CLASS
   *  Creates URL & Loads Core Controller
   *  URL Format - /controller/method/param1/param2
   */
class Core
{
  // Set Defaults
  protected $currentController = 'Home'; // Default controller
  protected $currentMethod = 'index'; // Default method
  protected $params = []; // Set initial empty params array
  protected $apiVersion = null; // API version (e.g., 'v1', 'v2')

  public function __construct()
  {
    $url = $this->getUrl();

    // Check for API versioning (e.g., /v1/users/login)
    if (isset($url[0]) && preg_match('/^v\d+$/', $url[0])) {
      $this->apiVersion = $url[0];
      unset($url[0]);
      $url = array_values($url); // Re-index array
    }

    // Determine controller path based on version
    $controllerPath = '../app/controllers/';
    if ($this->apiVersion) {
      $controllerPath .= $this->apiVersion . '/';
    }

    // Look for versioned controller first, then fall back to root
    if (isset($url[0]) && file_exists($controllerPath . ucwords($url[0]) . '.php')) {
      // If versioned controller exists, use it
      $this->currentController = ucwords($url[0]);
      unset($url[0]);
      $url = array_values($url);
    } elseif (isset($url[0]) && !$this->apiVersion && file_exists('../app/controllers/' . ucwords($url[0]) . '.php')) {
      // Fall back to unversioned controller
      $this->currentController = ucwords($url[0]);
      unset($url[0]);
      $url = array_values($url);
    }

    // Build full controller path
    $fullControllerPath = $controllerPath . $this->currentController . '.php';

    // Check if controller file exists
    if (file_exists($fullControllerPath)) {
      require_once($fullControllerPath);
    } else {
      // Fallback to root controllers if versioned not found
      if ($this->apiVersion && file_exists('../app/controllers/' . $this->currentController . '.php')) {
        require_once('../app/controllers/' . $this->currentController . '.php');
      } else {
        // Controller not found - return 404
        http_response_code(404);
        echo json_encode([
          'status' => false,
          'message' => 'Endpoint not found',
          'requested_path' => $_GET['url'] ?? ''
        ]);
        exit;
      }
    }

    // Instantiate the current controller
    $this->currentController = new $this->currentController;

    // Check if second part of url is set (method)
    if (isset($url[0])) {
      // Check if method/function exists in current controller class
      if (method_exists($this->currentController, $url[0])) {
        // Set current method if it exsists
        $this->currentMethod = $url[0];
        // Unset 0 index
        unset($url[0]);
        $url = array_values($url);
      }
    }

    // Get params - Any values left over in url are params
    $this->params = $url ? array_values($url) : [];

    // Add API version header
    if ($this->apiVersion) {
      header("API-Version: {$this->apiVersion}");
    }

    // Call a callback with an array of parameters
    call_user_func_array([$this->currentController, $this->currentMethod], $this->params);
  }

  // Construct URL From $_GET['url']
  public function getUrl()
  {
    if (isset($_GET['url'])) {
      $url = rtrim($_GET['url'], '/');
      $url = filter_var($url, FILTER_SANITIZE_URL);
      $url = explode('/', $url);
      return $url;
    }
  }
}
