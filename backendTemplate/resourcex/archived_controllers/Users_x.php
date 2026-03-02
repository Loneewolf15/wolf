<?php
class Users extends Controller
{
  protected $userModel;
  protected $serverKey;
  
  public function __construct()
  {
    $this->userModel = $this->model('User');
    $this->serverKey = 'secret_server_key';
  }

   private function sendResponse($status, $message, $data = [])
        {
          echo json_encode([
            'status' => $status,
            'message' => $message,
            'data' => $data
          ]);
          exit;
        }
        
  public function index(){
    $response = [
      'status' => true,
      'message' => 'Market Plaza Users API',
      'available_endpoints' => [
        'POST /users/registerUser' => 'Register new user',
        'POST /users/loginfunc' => 'User login',
        'GET /users/getUser' => 'Get user profile',
        'POST /users/editProfile' => 'Update profile',
        'POST /users/changePassword' => 'Change password',
        'POST /users/deleteProfile' => 'Delete account',
        'POST /users/forgotPassword' => 'Send password reset',
        'POST /users/resetPassword' => 'Reset password',
        'POST /users/upgradeAccount' => 'Upgrade to seller/reseller',
        'POST /users/logout' => 'Logout user'
      ]
    ];
    print_r(json_encode($response, JSON_PRETTY_PRINT));
    exit;
  }

  public function getUser()
  {
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

      $response = [
          'status' => true,
          'message' => 'User data retrieved successfully',
          'data' => $userData
      ];
      print_r(json_encode($response));
      exit;
  }

  public function registerUserx()
  {
    // Check if POST
    if ($_SERVER['REQUEST_METHOD'] == 'POST') {
      // Sanitize POST
      $_POST = $this->getData();
      $str = date('ymdhms');
      $userid = "user".md5($str . rand(1000, 9999));
      
      $data = [
        'email' => trim($_POST['email'] ?? ''),
        'user_id' => $userid,
        'password' => trim($_POST['password'] ?? ''),
        'confirm_password' => trim($_POST['confirm_password'] ?? ''),
        'full_name' => trim($_POST['full_name'] ?? ''),
        'phone' => trim($_POST['phone'] ?? ''),
        'user_type' => 'buyer', // Everyone starts as buyer
        'company_name' => trim($_POST['company_name'] ?? ''),
        'license_number' => trim($_POST['license_number'] ?? ''),
        'bio' => trim($_POST['bio'] ?? ''),
        'profile_image' => trim($_POST['profile_image'] ?? ''),
        'address' => trim($_POST['address'] ?? ''),
        'city' => trim($_POST['city'] ?? ''),
        'state' => trim($_POST['state'] ?? ''),
        'country' => trim($_POST['country'] ?? 'Nigeria'),
        // Error fields
        'email_err' => '',
        'password_err' => '',
        'confirm_password_err' => '',
        'full_name_err' => '',
        'phone_err' => ''
      ];

      // Validate full name
      if (empty($data['full_name'])) {
        $data['full_name_err'] = 'Please enter your full name';
      }

      // Validate email
      if (empty($data['email'])) {
        $data['email_err'] = 'Please enter an email';
      } elseif (!filter_var($data['email'], FILTER_VALIDATE_EMAIL)) {
        $data['email_err'] = 'Please enter a valid email';
      } else {
        // Check Email
        if ($this->userModel->findUserByEmail($data['email'])) {
          $data['email_err'] = 'Email is already taken.';
        }
      }

      // Validate phone
      if (empty($data['phone'])) {
        $data['phone_err'] = 'Please enter your phone number';
      }

      // User type is automatically set to buyer - no validation needed

      // Validate password
      if (empty($data['password'])) {
        $data['password_err'] = 'Please enter a password.';
      } elseif (strlen($data['password']) < 6) {
        $data['password_err'] = 'Password must have atleast 6 characters.';
      }

      // Validate confirm password
      if (empty($data['confirm_password'])) {
        $data['confirm_password_err'] = 'Please confirm password.';
      } else {
        if ($data['password'] != $data['confirm_password']) {
          $data['confirm_password_err'] = 'Password do not match.';
        }
      }

      // Make sure errors are empty
      if (empty($data['email_err']) && empty($data['password_err']) && empty($data['confirm_password_err']) && 
          empty($data['full_name_err']) && empty($data['phone_err'])) {
        
        // SUCCESS - Proceed to insert
        $data['password'] = password_hash($data['password'], PASSWORD_ARGON2ID);
      
        if ($this->userModel->register($data)) {
          // Send welcome email using new email service
          $emailResult = $this->sendWelcomeEmail($data);
          if (!$emailResult['status']) {
            error_log('Failed to send welcome email to: ' . $data['email'] . ' - ' . $emailResult['message']);
          }
          
          // Auto login the user
          $loginData = [
            'email' => $data['email'],
            'password' => $_POST['password'] // Use original password for login
          ];
          
          $log = $this->loginRegisteredUser($loginData);
          if ($log) {
            $response = [
              'status' => true,
              'message' => 'Registration successful',
              'user_id' => $data['user_id'],
              'data' => $log
            ];
            print_r(json_encode($response));
            exit;
          } else {
            $response = [
              'status' => true,
              'message' => 'Registration successful, please login',
              'user_id' => $data['user_id']
            ];
            print_r(json_encode($response));
            exit;
          }
        
        } else {
          $response = [
            'status' => false,
            'message' => 'Failed to register user'
          ];
          print_r(json_encode($response));
        }
      } else {
        // error response
        $response = [
          'status' => false,
          'message' => 'Validation failed',
          'errors' => array_filter([
            'email' => $data['email_err'],
            'password' => $data['password_err'],
            'confirm_password' => $data['confirm_password_err'],
            'full_name' => $data['full_name_err'],
            'phone' => $data['phone_err']
          ])
        ];
        print_r(json_encode($response));
      }
    } else {
      $response = [
        'status' => false,
        'message' => 'Invalid request method'
      ];
      print_r(json_encode($response));
    }
  }

  public function registerUser()
{
  // Check if POST
  if ($_SERVER['REQUEST_METHOD'] == 'POST') {
    // Sanitize POST
    $_POST = $this->getData();
    
    // Data fields to be used for validation and registration
    $data = [
      'name' => trim($_POST['name'] ?? ''),
      'email' => trim($_POST['email'] ?? ''),
      'phone' => trim($_POST['phone'] ?? ''),
      'password' => trim($_POST['password'] ?? ''),
      'confirm_password' => trim($_POST['confirm_password'] ?? ''),
      // Location is now handled after login, so it's not collected here
      // Error fields
      'name_err' => '',
      'email_err' => '',
      'phone_err' => '',
      'password_err' => '',
      'confirm_password_err' => ''
    ];

    // Validate name
    if (empty($data['name'])) {
      $data['name_err'] = 'Please enter your full name';
    }

    // Validate email
    if (empty($data['email'])) {
      $data['email_err'] = 'Please enter an email';
    } elseif (!filter_var($data['email'], FILTER_VALIDATE_EMAIL)) {
      $data['email_err'] = 'Please enter a valid email';
    } else {
      // Check if email already exists
      if ($this->userModel->findUserByEmail($data['email'])) {
        $data['email_err'] = 'Email is already taken.';
      }
    }

    // Validate phone
    if (empty($data['phone'])) {
      $data['phone_err'] = 'Please enter your phone number';
    } else {
      // Check if phone already exists
      if ($this->userModel->findUserByPhone($data['phone'])) {
        $data['phone_err'] = 'Phone number is already taken.';
      }
    }
    
    // Validate password
    if (empty($data['password'])) {
      $data['password_err'] = 'Please enter a password.';
    } elseif (strlen($data['password']) < 6) {
      $data['password_err'] = 'Password must have at least 6 characters.';
    }

    // Validate confirm password
    if (empty($data['confirm_password'])) {
      $data['confirm_password_err'] = 'Please confirm password.';
    } else {
      if ($data['password'] != $data['confirm_password']) {
        $data['confirm_password_err'] = 'Password do not match.';
      }
    }

    // Make sure errors are empty
    if (empty($data['name_err']) && empty($data['email_err']) && empty($data['phone_err']) && 
        empty($data['password_err']) && empty($data['confirm_password_err'])) {
      
      // SUCCESS - Proceed to insert
      $data['password'] = password_hash($data['password'], PASSWORD_ARGON2ID);
      
      if ($this->userModel->register($data)) {
        // Find the newly registered user to get their user_id
        $registeredUser = $this->userModel->findUserByEmail($data['email']);
        
        // Assign the default 'Buyer' role to the new user
        $this->userModel->addRoleToUser($registeredUser->user_id, 'Buyer');
        
        // Send welcome email
        $emailResult = $this->sendWelcomeEmail($data);
        if (!$emailResult['status']) {
          error_log('Failed to send welcome email to: ' . $data['email'] . ' - ' . $emailResult['message']);
        }
        
        // Auto login the user
        $loginData = [
          'email' => $data['email'],
          'password' => $_POST['password'] // Use original password for login
        ];
        
        $log = $this->loginRegisteredUser($loginData);
        if ($log) {
          $this->sendResponse(true, 'Registration successful', ['user_id' => $registeredUser->user_id, 'data' => $log]);
        } else {
          $this->sendResponse(true, 'Registration successful, please login', ['user_id' => $registeredUser->user_id]);
        }
      
      } else {
        $this->sendResponse(false, 'Failed to register user');
      }
    } else {
      // Error response
      $errors = array_filter([
        'name' => $data['name_err'],
        'email' => $data['email_err'],
        'phone' => $data['phone_err'],
        'password' => $data['password_err'],
        'confirm_password' => $data['confirm_password_err']
      ]);
      $this->sendResponse(false, 'Validation failed', ['errors' => $errors]);
    }
  } else {
    $this->sendResponse(false, 'Invalid request method');
  }
}


  public function loginRegisteredUser($data)
  {
    if ($_SERVER['REQUEST_METHOD'] == 'POST') {
      $email = $data['email'];
      $password = $data['password'];

      $row = $this->userModel->loginUser($email);
      
      if ($row) {
        $hashedPassword = $row->password;
        if (password_verify($password, $hashedPassword)) {
          // Users can login even if not verified
          
          $this->createUserSession($row);
          return $row;
        } else {
          return false;
        }
      } else {
        return false;
      }
    }
    return false;
  }

  public function createUserSessionr($user)
  {
    $payload = [
      'user_id' => $user->user_id,
      'email' => $user->email,
      'full_name' => $user->full_name,
      'user_type' => $user->user_type,
      'company_name' => $user->company_name,
      'license_number' => $user->license_number,
      'bio' => $user->bio,
      'profile_image' => $user->profile_image,
      'address' => $user->address,
      'city' => $user->city,
      'state' => $user->state,
      'country' => $user->country,
      'activation' => $user->activation,
      'iat' => time(),
      'exp' => time() + (24 * 60 * 60) // 24 hours
    ];

    $jwt = $this->generateJWT($payload, $this->serverKey);
    
    // Update user's bearer token in database
    $this->userModel->updateToken($user->user_id, $jwt, $user->email);

    return $jwt;
  }

  public function createUserSession($user)
{
    // Retrieve all roles for the user from the database
    $userRoles = $this->userModel->getRolesForUser($user->user_id);
    
    // Create an array of role names
    $rolesArray = [];
    if ($userRoles) {
        foreach ($userRoles as $role) {
            $rolesArray[] = $role->role_name;
        }
    }

    $payload = [
        'user_id' => $user->user_id,
        'name' => $user->name,
        'email' => $user->email,
        'phone' => $user->phone,
        'location' => $user->location,
        'is_verified' => $user->is_verified,
        'roles' => $rolesArray, // Include the array of roles
        'iat' => time(),
        'exp' => time() + (24 * 60 * 60) // 24 hours
    ];

    $jwt = $this->generateJWT($payload, $this->serverKey);
    
    // Call the model method to save the token to the database
    $this->userModel->updateAccessToken($user->user_id, $jwt);

    return $jwt;
}

  public function loginfunc()
  {
    if ($_SERVER['REQUEST_METHOD'] == 'POST') {
      $postData = $this->getData();

      $email = trim($postData['email'] ?? '');
      $password = trim($postData['password'] ?? '');

      if (empty($email) || empty($password)) {
        $response = [
          'status' => false,
          'message' => 'Email and password are required'
        ];
        print_r(json_encode($response));
        exit;
      }

      $row = $this->userModel->loginUser($email);

      if ($row) {
        $hashedPassword = $row->password;
        if (password_verify($password, $hashedPassword)) {
          // Users can login even if not verified, but with limited functionality

          $access_token = $this->createUserSession($row);

          $response = [
            'status' => 'true',
            'message' => 'Login successful',
            'access_token' => $access_token,
            'user_data' => [
              'user_id' => $row->user_id,
              'email' => $row->email,
              'full_name' => $row->full_name,
              'user_type' => $row->user_type,
              'activation' => $row->activation
            ]
          ];
          print_r(json_encode($response));
          exit;
        } else {
          $response = [
            'status' => false,
            'message' => 'Invalid email or password'
          ];
          print_r(json_encode($response));
          exit;
        }
      } else {
        $response = [
          'status' => false,
          'message' => 'Invalid email or password'
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

  // Forgot password - send reset token
  public function forgotPassword()
  {
    if ($_SERVER['REQUEST_METHOD'] == 'POST') {
      $postData = $this->getData();

      $email = trim($postData['email'] ?? '');

      if (empty($email)) {
        $response = [
          'status' => false,
          'message' => 'Email is required'
        ];
        print_r(json_encode($response));
        exit;
      }

      if (!filter_var($email, FILTER_VALIDATE_EMAIL)) {
        $response = [
          'status' => false,
          'message' => 'Please enter a valid email address'
        ];
        print_r(json_encode($response));
        exit;
      }

      // Check if user exists
      $user = $this->userModel->findUserByEmail($email);
      if (!$user) {
        $response = [
          'status' => false,
          'message' => 'No account found with this email address'
        ];
        print_r(json_encode($response));
        exit;
      }

      // Generate reset token
      $resetToken = $this->userModel->generatePasswordResetToken($email);

      if ($resetToken) {
        // Send reset email using new email service
        $userDataArray = [
          'full_name' => $user->full_name ?? 'User',
          'email' => $user->email,
          'user_id' => $user->user_id
        ];

        $emailResult = $this->sendPasswordResetEmail($resetToken, $userDataArray);

        if ($emailResult['status']) {
          $response = [
            'status' => true,
            'message' => 'Password reset code sent to your email. Please check your inbox.'
          ];
        } else {
          $response = [
            'status' => false,
            'message' => 'Failed to send reset email. Please try again.'
          ];
        }
      } else {
        $response = [
          'status' => false,
          'message' => 'Failed to generate reset token. Please try again.'
        ];
      }

      print_r(json_encode($response));
      exit;

    } else {
      $response = [
        'status' => false,
        'message' => 'Invalid request method'
      ];
      print_r(json_encode($response));
      exit;
    }
  }

  // Reset password with token
  public function resetPassword()
  {
    if ($_SERVER['REQUEST_METHOD'] == 'POST') {
      $postData = $this->getData();

      $email = trim($postData['email'] ?? '');
      $resetToken = trim($postData['reset_token'] ?? '');
      $newPassword = trim($postData['new_password'] ?? '');
      $confirmPassword = trim($postData['confirm_password'] ?? '');

      // Validation
      if (empty($email)) {
        $response = [
          'status' => false,
          'message' => 'Email is required'
        ];
        print_r(json_encode($response));
        exit;
      }

      if (empty($resetToken)) {
        $response = [
          'status' => false,
          'message' => 'Reset token is required'
        ];
        print_r(json_encode($response));
        exit;
      }

      if (empty($newPassword)) {
        $response = [
          'status' => false,
          'message' => 'New password is required'
        ];
        print_r(json_encode($response));
        exit;
      }

      if (strlen($newPassword) < 6) {
        $response = [
          'status' => false,
          'message' => 'Password must be at least 6 characters long'
        ];
        print_r(json_encode($response));
        exit;
      }

      if ($newPassword !== $confirmPassword) {
        $response = [
          'status' => false,
          'message' => 'Passwords do not match'
        ];
        print_r(json_encode($response));
        exit;
      }

      // Hash the new password
      $hashedPassword = password_hash($newPassword, PASSWORD_ARGON2ID);

      // Reset password with token
      if ($this->userModel->resetPasswordWithToken($email, $resetToken, $hashedPassword)) {
        $response = [
          'status' => true,
          'message' => 'Password reset successfully. You can now login with your new password.'
        ];
      } else {
        $response = [
          'status' => false,
          'message' => 'Invalid or expired reset token. Please request a new password reset.'
        ];
      }

      print_r(json_encode($response));
      exit;

    } else {
      $response = [
        'status' => false,
        'message' => 'Invalid request method'
      ];
      print_r(json_encode($response));
      exit;
    }
  }

  // Upgrade user type (buyer to agent/realtor) - with verification check
  public function upgradeAccount()
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

      // Check if user is verified first
      if ($userData->activation != 1) {
        $response = [
          'status' => false,
          'message' => 'Account must be verified before upgrading to agent or realtor. Please contact support to verify your account.'
        ];
        print_r(json_encode($response));
        exit;
      }

      $newUserType = trim($postData['user_type'] ?? '');
      $companyName = trim($postData['company_name'] ?? '');
      $licenseNumber = trim($postData['license_number'] ?? '');

      // Validation
      $validTypes = ['agent', 'realtor'];
      if (empty($newUserType) || !in_array($newUserType, $validTypes)) {
        $response = [
          'status' => false,
          'message' => 'Please select a valid account type (agent or realtor)'
        ];
        print_r(json_encode($response));
        exit;
      }

      if ($userData->user_type === $newUserType) {
        $response = [
          'status' => false,
          'message' => 'You are already a ' . $newUserType
        ];
        print_r(json_encode($response));
        exit;
      }

      // Additional validation for realtor
      if ($newUserType === 'realtor') {
        if (empty($licenseNumber)) {
          $response = [
            'status' => false,
            'message' => 'License number is required for realtor account'
          ];
          print_r(json_encode($response));
          exit;
        }
      }

      // Additional data for upgrade
      $additionalData = [
        'company_name' => $companyName,
        'license_number' => $licenseNumber
      ];

      if ($this->userModel->upgradeUserType($userData->user_id, $newUserType, $additionalData)) {
        $response = [
          'status' => true,
          'message' => 'Account upgraded successfully to ' . $newUserType . '. You can now create listings!'
        ];
      } else {
        $response = [
          'status' => false,
          'message' => 'Failed to upgrade account. Please try again.'
        ];
      }

      print_r(json_encode($response));
      exit;

    } else {
      $response = [
        'status' => false,
        'message' => 'Invalid request method'
      ];
      print_r(json_encode($response));
      exit;
    }
  }

  // Change password
  public function changePassword()
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

      $currentPassword = trim($postData['current_password'] ?? '');
      $newPassword = trim($postData['new_password'] ?? '');
      $confirmPassword = trim($postData['confirm_password'] ?? '');

      // Validation
      if (empty($currentPassword)) {
        $response = [
          'status' => false,
          'message' => 'Current password is required'
        ];
        print_r(json_encode($response));
        exit;
      }

      if (empty($newPassword)) {
        $response = [
          'status' => false,
          'message' => 'New password is required'
        ];
        print_r(json_encode($response));
        exit;
      }

      if (strlen($newPassword) < 6) {
        $response = [
          'status' => false,
          'message' => 'New password must be at least 6 characters'
        ];
        print_r(json_encode($response));
        exit;
      }

      if ($newPassword !== $confirmPassword) {
        $response = [
          'status' => false,
          'message' => 'New passwords do not match'
        ];
        print_r(json_encode($response));
        exit;
      }

      // Verify current password
      $loginData = $this->userModel->loginUser($userData->email);
      if (!password_verify($currentPassword, $loginData->password)) {
        $response = [
          'status' => false,
          'message' => 'Current password is incorrect'
        ];
        print_r(json_encode($response));
        exit;
      }

      // Update password
      $hashedPassword = password_hash($newPassword, PASSWORD_ARGON2ID);
      if ($this->userModel->updateUserPassword($userData->user_id, $hashedPassword)) {
        $response = [
          'status' => true,
          'message' => 'Password changed successfully'
        ];
      } else {
        $response = [
          'status' => false,
          'message' => 'Failed to change password'
        ];
      }

      print_r(json_encode($response));
      exit;

    } else {
      $response = [
        'status' => false,
        'message' => 'Invalid request method'
      ];
      print_r(json_encode($response));
      exit;
    }
  }

  // Logout user
  public function logout()
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
      // Clear the bearer token
      if ($this->userModel->updateToken($userData->user_id, '', $userData->email)) {
        $response = [
          'status' => true,
          'message' => 'Logged out successfully'
        ];
      } else {
        $response = [
          'status' => false,
          'message' => 'Failed to logout'
        ];
      }

      print_r(json_encode($response));
      exit;

    } else {
      $response = [
        'status' => false,
        'message' => 'Invalid request method'
      ];
      print_r(json_encode($response));
      exit;
    }
  }

  // Edit user profile
  public function editProfile()
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

      // Validate required fields
      $full_name = trim($postData['full_name'] ?? '');
      $phone = trim($postData['phone'] ?? '');
      $city = trim($postData['city'] ?? '');
      $state = trim($postData['state'] ?? '');
      $country = trim($postData['country'] ?? 'Nigeria');

      // Optional fields
      $company_name = trim($postData['company_name'] ?? '');
      $license_number = trim($postData['license_number'] ?? '');
      $bio = trim($postData['bio'] ?? '');
      $address = trim($postData['address'] ?? '');

      // Validation
      if (empty($full_name) || empty($phone) || empty($city) || empty($state)) {
        $response = [
          'status' => false,
          'message' => 'Full name, phone, city, and state are required'
        ];
        print_r(json_encode($response));
        exit;
      }

      // Phone validation
      if (!preg_match('/^\+?[0-9\-\s\(\)]{10,20}$/', $phone)) {
        $response = [
          'status' => false,
          'message' => 'Invalid phone number format'
        ];
        print_r(json_encode($response));
        exit;
      }

      // Prepare update data
      $updateData = [
        'user_id' => $userData->user_id,
        'full_name' => $full_name,
        'phone' => $phone,
        'city' => $city,
        'state' => $state,
        'country' => $country,
        'company_name' => $company_name,
        'license_number' => $license_number,
        'bio' => $bio,
        'address' => $address
      ];

      // Handle profile image upload if provided
      if (isset($_FILES['profile_image']) && $_FILES['profile_image']['error'] === UPLOAD_ERR_OK) {
        $uploadResult = $this->handleProfileImageUpload($_FILES['profile_image'], $userData->user_id);
        if ($uploadResult['success']) {
          $updateData['profile_image'] = $uploadResult['file_path'];
        } else {
          $response = [
            'status' => false,
            'message' => 'Profile image upload failed: ' . $uploadResult['message']
          ];
          print_r(json_encode($response));
          exit;
        }
      }

      // Update profile
      if ($this->userModel->updateProfile($updateData)) {
        // Get updated user data
        $updatedUser = $this->userModel->findUserById($userData->user_id);

        $response = [
          'status' => true,
          'message' => 'Profile updated successfully',
          'data' => [
            'user_id' => $updatedUser->user_id,
            'full_name' => $updatedUser->full_name,
            'email' => $updatedUser->email,
            'phone' => $updatedUser->phone,
            'user_type' => $updatedUser->user_type,
            'company_name' => $updatedUser->company_name,
            'license_number' => $updatedUser->license_number,
            'bio' => $updatedUser->bio,
            'profile_image' => $updatedUser->profile_image,
            'address' => $updatedUser->address,
            'city' => $updatedUser->city,
            'state' => $updatedUser->state,
            'country' => $updatedUser->country
          ]
        ];
      } else {
        $response = [
          'status' => false,
          'message' => 'Failed to update profile'
        ];
      }

      print_r(json_encode($response));
      exit;

    } else {
      $response = [
        'status' => false,
        'message' => 'Invalid request method'
      ];
      print_r(json_encode($response));
      exit;
    }
  }

  // Delete user profile
  public function deleteProfile()
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

      // Require password confirmation for security
      $password = trim($postData['password'] ?? '');
      $confirmation = trim($postData['confirmation'] ?? '');

      if (empty($password)) {
        $response = [
          'status' => false,
          'message' => 'Password is required to delete account'
        ];
        print_r(json_encode($response));
        exit;
      }

      if ($confirmation !== 'DELETE') {
        $response = [
          'status' => false,
          'message' => 'Please type "DELETE" to confirm account deletion'
        ];
        print_r(json_encode($response));
        exit;
      }

      // Verify password
      $user = $this->userModel->findUserById($userData->user_id);
      if (!$user || !password_verify($password, $user->password)) {
        $response = [
          'status' => false,
          'message' => 'Invalid password'
        ];
        print_r(json_encode($response));
        exit;
      }

      // Check if user has active subscriptions
      $subscriptionModel = $this->model('Subscription');
      $activeSubscription = $subscriptionModel->getUserSubscription($userData->user_id);

      if ($activeSubscription && $activeSubscription->status === 'active') {
        $response = [
          'status' => false,
          'message' => 'Cannot delete account with active subscription. Please cancel your subscription first.',
          'has_active_subscription' => true
        ];
        print_r(json_encode($response));
        exit;
      }

      // Send account deletion confirmation email before deleting
      $userDataArray = [
        'full_name' => $user->full_name,
        'email' => $user->email,
        'user_id' => $user->user_id
      ];

      $emailResult = $this->sendAccountDeletionEmail($userDataArray);
      if (!$emailResult['status']) {
        error_log('Failed to send account deletion email to: ' . $user->email . ' - ' . $emailResult['message']);
      }

      // Delete user account (this will cascade delete related data due to foreign keys)
      if ($this->userModel->deleteUser($userData->user_id)) {
        $response = [
          'status' => true,
          'message' => 'Account deleted successfully'
        ];
      } else {
        $response = [
          'status' => false,
          'message' => 'Failed to delete account'
        ];
      }

      print_r(json_encode($response));
      exit;

    } else {
      $response = [
        'status' => false,
        'message' => 'Invalid request method'
      ];
      print_r(json_encode($response));
      exit;
    }
  }

  // Handle profile image upload
  private function handleProfileImageUpload($file, $user_id)
  {
    try {
      // Validate file type
      $allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
      if (!in_array($file['type'], $allowedTypes)) {
        return [
          'success' => false,
          'message' => 'Invalid file type. Only JPEG, PNG, GIF, and WebP are allowed.'
        ];
      }

      // Validate file size (max 5MB)
      $maxSize = 5 * 1024 * 1024; // 5MB
      if ($file['size'] > $maxSize) {
        return [
          'success' => false,
          'message' => 'File size too large. Maximum size is 5MB.'
        ];
      }

      // Create upload directory if it doesn't exist
      $uploadDir = 'uploads/profiles/';
      if (!is_dir($uploadDir)) {
        mkdir($uploadDir, 0755, true);
      }

      // Generate unique filename
      $extension = pathinfo($file['name'], PATHINFO_EXTENSION);
      $filename = 'profile_' . $user_id . '_' . time() . '.' . $extension;
      $filepath = $uploadDir . $filename;

      // Move uploaded file
      if (move_uploaded_file($file['tmp_name'], $filepath)) {
        return [
          'success' => true,
          'file_path' => $filepath,
          'filename' => $filename
        ];
      } else {
        return [
          'success' => false,
          'message' => 'Failed to upload file'
        ];
      }

    } catch (Exception $e) {
      return [
        'success' => false,
        'message' => 'Upload error: ' . $e->getMessage()
      ];
    }
  }
}
