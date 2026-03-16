<?php


//  use QRcode;
  class Scans extends Controller {
    public function __construct(){
        $this->userModel = $this->model('User');
        $this->scanModel = $this->model('Scan');      
        $this->serverKey  = 'secret_server_key';
      }

     
      
    
    ///proccess user coookie
    public function cookValid($liveToken){
 

        
        if(!($this->userModel->cookieChecker($liveToken))){
   
         redirect('users/login/8');
        
        }else{
   
           return true;
   
        }
   
    }
        


    public function edit($tokenX){
      $this->cookValid($tokenX);
    //echo $_SERVER['REQUEST_METHOD'];

     if($_SERVER['REQUEST_METHOD'] == 'POST'){  ///post start
      
    //Sanitize POST data
    $_POST = filter_input_array(INPUT_POST, FILTER_SANITIZE_STRING);
    // Process f
        $data = [
          'siteName' => trim($_POST['siteName']),
          'state' => trim($_POST['state']),
          'city' => trim($_POST['city']),
          'email' => trim($_POST['email']),
          'country' => trim($_POST['country']),
          'webAdd' => trim($_POST['webAdd']),
          'phone' => trim($_POST['phone']),          
          'officeAdd' => trim($_POST['officeAdd']),
          'pobox' => trim($_POST['pobox']),
          'zip' => trim($_POST['zip']),
          'slogan' => $_FILES['slogan'],
          'currentYear' => trim($_POST['currentYear']),
          'jobSession' => trim($_POST['jobSession']),
          'integrationKey' => trim($_POST['integrationKey']),
          'wallet1' => trim($_POST['wallet1']),
          'wallet2' => trim($_POST['wallet2']),
          'nisCost' => trim($_POST['nisCost']),
          'surconCost' => trim($_POST['surconCost']),   
          'logo' => $_FILES['logo'],
          'siteName_err' => '',
          'officeAdd_err' => '',
          'email_err' => '',
          'phone_err' => '',
          'state_err' => '',
          'city_err' => '',
          'country_err' => '',
          'webAdd_err' =>'',
             
        ];


        // Validate Name
        if(empty($data['siteName'])){
          $data['siteName_err'] = 'Please Enter Site Name';
        }
    
        // Validate Middle Name
        if(empty($data['officeAdd'])){
          $data['officeAdd_err'] = 'Please Enter Office Address';
        }
      
        // Validate Middle Name
        if(empty($data['email'])){
          $data['email_err'] = 'Please Enter Email Address';
        }
      
        // Validate phone
        if(empty($data['phone'])){
          $data['phone_err'] = 'Please Enter Phone Number';
        }

    
        // Validate Address
        if(empty($data['webAdd'])){
          $data['webAdd_err'] = 'Please Enter Web Address';
        }

         // Validate Address
         if(empty($data['country'])){
          $data['country_err'] = 'Please Enter Country';
        }


         // Validate Address
         if(empty($data['city'])){
          $data['city_err'] = 'Please Enter City Name';
        }

         // Validate state_err
         if(empty($data['state'])){
          $data['state_err'] = 'Please Enter State Name ';
        }

        
    
        //Make sure Errors are Empty
        if(empty($data['state_err']) && empty($data['city_err']) && empty($data['country_err'])  && empty($data['email_err']) && empty($data['webAdd_err']) && empty($data['phone_err']) && empty($data['siteName_err']) && empty($data['officeAdd_err'])){
          $this->settingsModel->save($data);
//echo "helo";
//exit;
//$id
         //

        }else{

     

          $propertyTypeData = $this->jobsModel->getAllPropertyType();
          $getAllSizeType = $this->jobsModel->getAllSizeType();
          $loginData = $this->userModel->findLoginByToken($tokenX);
          $userData = $this->userModel->findUserByEmail_det($loginData->email);
          $menuList = $this->userModel->fetchUserModule($loginData->roleID);
          $submoduleList = $this->userModel->getAllsubmodule();
          $getAppSettings = $this->settingsModel->getAppSettings();
     
          print_r($data);
          $data = [
          'submoduleList'=> $submoduleList,
          'menuList'=> $menuList,
          'loginData' => $loginData,
          'userData' => $userData,
          'wel'=> $wel,
          'propertyTypeData'=> $propertyTypeData,
          'getAllSizeType'=>$getAllSizeType,
          'getAppSettings'=>$getAppSettings,
          'datax'=>$data,

          ];
            
 //return error messages;
//Load view with errors
$this->view('settings/show', $data);

         
        }

        //////end

  }

      $wel = "general settings ";
   
     $propertyTypeData = $this->jobsModel->getAllPropertyType();
     $getAllSizeType = $this->jobsModel->getAllSizeType();
     $loginData = $this->userModel->findLoginByToken($tokenX);
     $userData = $this->userModel->findUserByEmail_det($loginData->email);
     $menuList = $this->userModel->fetchUserModule($loginData->roleID);
     $submoduleList = $this->userModel->getAllsubmodule();
     $getAppSettings = $this->settingsModel->getAppSettings();

     
     $data = [
     'submoduleList'=> $submoduleList,
     'menuList'=> $menuList,
     'loginData' => $loginData,
     'userData' => $userData,
     'wel'=> $wel,
     'propertyTypeData'=> $propertyTypeData,
     'getAllSizeType'=>$getAllSizeType,
     'getAppSettings'=>$getAppSettings,

     ];
       

     
         $this->view('settings/show', $data);

    }
    public  function show($tokenX){
        $this->cookValid($tokenX);

         $wel = "general settings ";
      
        $propertyTypeData = $this->jobsModel->getAllPropertyType();
        $getAllSizeType = $this->jobsModel->getAllSizeType();
        $loginData = $this->userModel->findLoginByToken($tokenX);
        $userData = $this->userModel->findUserByEmail_det($loginData->email);
        $menuList = $this->userModel->fetchUserModule($loginData->roleID);
        $submoduleList = $this->userModel->getAllsubmodule();
        $getAppSettings = $this->settingsModel->getAppSettings();

        
        $data = [
        'submoduleList'=> $submoduleList,
        'menuList'=> $menuList,
        'loginData' => $loginData,
        'userData' => $userData,
        'wel'=> $wel,
        'propertyTypeData'=> $propertyTypeData,
        'getAllSizeType'=>$getAllSizeType,
        'getAppSettings'=>$getAppSettings,

        ];
          

        
            $this->view('settings/show', $data);


        }



        public function getQrCode() 
        {
          if ($_SERVER['REQUEST_METHOD'] == 'POST'){  ///post start
      
            //Sanitize POST data
            $loginData = $this->getData();
            
            $data =[
              "qrcode" => $loginData['qrcode'],
            ];
          }
        }


        public function generateQrcode()
        {
          
   try {
    $userData = $this->RouteProtection();
   //print_r($userData);
   
} catch (UnexpectedValueException $e) {
    // Handle the exception
    $res = [
      'status' => 401,
      'message' =>  $e->getMessage(),
    ];
    print_r(json_encode($res));
    exit;
}
  
try {
 
  $name = $userData->full_name;
  $tag = $userData->user_tag;
  $image = $userData->image;
  //print_r($email);
  $oQRC = new QRCode; // Create vCard Object
  $oQRC->fullName($name)// Add Full Name
  ->nickname($tag)// Add Nickname
  ->email($image)// Add Email Address
  ->lang('en-US')// Add Language
  ->finish(); // End vCard
  $oQRC->display(); // Display
  $oQRC = json_encode($oQRC);

} catch (Exception $oExcept) {
  echo '<p><b>Exception launched!</b><br /><br />' .
      'Message: ' . $oExcept->getMessage() . '<br />' .
      'File: ' . $oExcept->getFile() . '<br />' .
      'Line: ' . $oExcept->getLine() . '<br />' .
      'Trace: <p/><pre>' . $oExcept->getTraceAsString() . '</pre>';
}

        }
        public function generatesubQrcode()
        {
          
   try {
    $userData = $this->RouteProtection();
  
} catch (UnexpectedValueException $e) {
    // Handle the exception
    $res = [
      'status' => 401,
      'message' =>  $e->getMessage(),
    ];
    //
}
  
try {
 
 
  $regData = $this->userModel->getUserByid($userData->registrer_id); 
//   print_r(json_encode($regData));
//     exit;
  $tag = $userData->user_tag;
  $email = $regData->email;
  $name = $regData->full_name;

  //print_r($email);
  $oQRC = new QRCode; // Create vCard Object
  $oQRC->fullName($name)// Add Full Name
  ->nickname($tag)// Add Nickname
  ->email($email)// Add Email Address
  
  ->lang('en-US')// Add Language
  ->finish(); // End vCard
  $oQRC->display(); // Display
  $oQRC = json_encode($oQRC);
} catch (Exception $oExcept) {
  print_r(json_encode( '<p><b>Exception launched!</b><br /><br />' .
      'Message: ' . $oExcept->getMessage() . '<br />' .
      'File: ' . $oExcept->getFile() . '<br />' .
      'Line: ' . $oExcept->getLine() . '<br />' .
      'Trace: <p/><pre>' . $oExcept->getTraceAsString() . '</pre>'));
}

        }
public function getIPAddress() {
    // Check for shared Internet/ISP IP
    if (isset($_SERVER['HTTP_CLIENT_IP']) && validateIPAddress($_SERVER['HTTP_CLIENT_IP'])) {
        return $_SERVER['HTTP_CLIENT_IP'];
    }

    // Check for IP addresses from proxies
    if (isset($_SERVER['HTTP_X_FORWARDED_FOR'])) {
        // Extract the IP addresses
        $ipAddresses = explode(',', $_SERVER['HTTP_X_FORWARDED_FOR']);


        // Loop through the IP addresses
        foreach ($ipAddresses as $ip) {
            $ip = trim($ip);
            if (filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV4 | FILTER_FLAG_IPV6) ) {
                print_r( " Your IP address is: " . $ip);
            }
        }
    }

    // Return the remote IP address
//     $PP =  $_SERVER['REMOTE_ADDR'];
//     $ipAddress = $PP;
// print_r( "Your IP address is: " . $ipAddress); 
}




public function checkloginDevice(){
  try {
    $userData = $this->RouteProtection();
} catch (UnexpectedValueException $e) {
    $res = [
      'status' => 401,
      'message' =>  $e->getMessage(),
    ];
    print_r(json_encode($res));
    exit;
}
  $user_agent = $_SERVER['HTTP_USER_AGENT'];
$ip_address = $_SERVER['REMOTE_ADDR'];
$is_mobile = preg_match('/(iPhone|iPod|iPad|Android|webOS|BlackBerry|Windows Phone)/i', $user_agent);

// Check if the user agent string contains certain keywords to identify the device
if (strpos($user_agent, 'iPhone') !== false) {
    print_r("Accessed from an iPhone.") ;
} elseif (strpos($user_agent, 'Android') !== false) {
    print_r( "Accessed from an Android device.");
} elseif (strpos($user_agent, 'Windows Phone') !== false) {
    print_r( "Accessed from a Windows Phone.");
} else if ($is_mobile) {
  print_r("User is using a mobile device") ;
} else {
  print_r( "User is using a desktop device");
}

// Log the IP address for the request
// Note: the user's IP address can be spoofed or hidden behind a proxy, so this is not always reliable
print_r( "IP address: $ip_address");

}
    }
