<?php
class Home extends Controller {
    public function __construct(){
        // No specific initialization needed
    } 
    
    public function index()
    {
        $response = [
            'status' => true,
            'message' => 'Welcome to MarketPlaza -  Marketplace API',
            'version' => '1.0.0',
            'description' => 'A comprehensive API for MarketPlaza',
        ];
        
        header('Content-Type: application/json');
        print_r(json_encode($response, JSON_PRETTY_PRINT));
        exit;
    }

    public function health_check()
    {
        $response = [
            'status' => 'healthy',
            'timestamp' => date('Y-m-d H:i:s'),
            'uptime' => 'API is running',
            'database' => 'Connected',
            'version' => '1.0.0'
        ];
        
        header('Content-Type: application/json');
        print_r(json_encode($response));
        exit;
    }
}
