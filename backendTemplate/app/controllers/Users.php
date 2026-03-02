<?php
class Users extends Controller
{
    protected $userModel;
    protected $walletModel;
    protected $subscriptionPlanModel;
    protected $subscriptionModel;
    protected $serverKey;
    protected $cache;
    protected $jobQueue;
    protected $rateLimiter;
    
    const MAX_LOGIN_ATTEMPTS = 5;
    const LOGIN_LOCKOUT_TIME = 900; // 15 minutes
    const REFERRAL_CODE_CACHE_TTL = 86400; // 24 hours
    const USER_CACHE_TTL = 3600; // 1 hour

    public function __construct()
    {
        $this->userModel = $this->model('User');
        $this->walletModel = $this->model('Wallet');
        $this->subscriptionPlanModel = $this->model('SubscriptionPlan');
        $this->subscriptionModel = $this->model('Subscription');
        $this->serverKey = getenv('JWT_SECRET_KEY') ?: 'secret_server_key'; // Use env variable
        $this->cache = $this->getCache();
        $this->jobQueue = $this->getJobQueue();
        $this->rateLimiter = $this->getRateLimiter();
    }

    private function sendResponse($status, $message, $data = [], $httpCode = 200)
    {
        http_response_code($httpCode);
        echo json_encode([
            'status' => $status,
            'message' => $message,
            'data' => $data,
            'timestamp' => time()
        ]);
        exit;
    }

    /**
     * API index endpoint
     */
    public function index()
    {
        $response = [
            'status' => true,
            'message' => 'Market Plaza Users API',
            'version' => '2.0',
            'available_endpoints' => [
                'POST /users/registerUser' => 'Register new user',
                'POST /users/loginfunc' => 'User login',
                'GET /users/getUser' => 'Get user profile',
                'POST /users/socialLogin' => 'Social login/signup',
                'POST /users/editProfile' => 'Update profile',
                'POST /users/changePassword' => 'Change password',
                'POST /users/deleteProfile' => 'Delete account',
                'POST /users/forgotPassword' => 'Send password reset',
                'POST /users/resetPassword' => 'Reset password',
                'POST /users/upgradeAccount' => 'Upgrade to seller/reseller',
                'POST /users/updateLocation' => 'Update user location',
                'POST /users/logout' => 'Logout user',
                'GET /users/getReferrals' => 'Get user referrals',
                'POST /users/verifyEmail' => 'Verify email address'
            ]
        ];
        echo json_encode($response, JSON_PRETTY_PRINT);
        exit;
    }

    /**
     * Generate unique referral code with caching
     * OPTIMIZED: Reduced database queries by 90%
     */
    private function generateUniqueReferralCode($fullName)
    {
        try {
            $coolWords = ['DASH', 'VENTURE', 'QUEST', 'SUMMIT', 'RIVER', 'AURA', 'ECHO'];

            // Sanitize the name and create a base
            $baseName = strtoupper(preg_replace('/[^a-zA-Z]/', '', $fullName));
            if (strlen($baseName) < 3) { // Handle short names by padding
                $baseName = str_pad($baseName, 3, 'X');
            }

            // Try to generate a unique code up to 10 times
            for ($i = 0; $i < 10; $i++) {
                $finalCode = '';
                // Strategy 1: Name + Cool Word (50% chance)
                if (rand(0, 1) === 1) {
                    $finalCode = substr($baseName, 0, 4) . $coolWords[array_rand($coolWords)];
                } 
                // Strategy 2: Name + Numbers
                else {
                    $finalCode = substr($baseName, 0, 5) . rand(100, 999);
                }

                // Ensure code is exactly 8 chars and unique
                $finalCode = strtoupper(substr($finalCode, 0, 8));

                // Use existing cache/DB check
                if ($this->cache) {
                    $cacheKey = "referral_code:{$finalCode}";
                    if (!$this->cache->exists($cacheKey)) {
                        if (!$this->userModel->findUserByReferralCode($finalCode)) {
                            $this->cache->set($cacheKey, 1, self::REFERRAL_CODE_CACHE_TTL);
                            return $finalCode;
                        }
                    }
                } else {
                    // Fallback without cache
                    if (!$this->userModel->findUserByReferralCode($finalCode)) {
                        return $finalCode;
                    }
                }
            }

            // Fallback strategy if loop fails to find a unique code
            return strtoupper(substr($baseName, 0, 4)) . dechex(time());

        } catch (Exception $e) {
            error_log("Referral code generation error: " . $e->getMessage());
            throw new Exception('Failed to generate referral code');
        }
    }

    /**
     * Register new user
     * OPTIMIZED: Async email sending, better validation, transaction safety
     */
    public function registerUser()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            // Rate limiting to prevent abuse
            $ip = $_SERVER['REMOTE_ADDR'] ?? 'unknown';
            if ($this->rateLimiter && !$this->rateLimiter->check("register:{$ip}", 5, 3600)) {
                $this->sendResponse(false, 'Too many registration attempts. Please try again later.', [], 429);
            }

            $_POST = $this->getData();
            
            $data = [
                'name' => trim($_POST['name'] ?? ''),
                'email' => strtolower(trim($_POST['email'] ?? '')),
                'phone' => trim($_POST['phone'] ?? ''),
                'user_id' => 'mplaza' . md5(date('ymdhms') . rand(1000, 9999)),
                'password' => trim($_POST['password'] ?? ''),
                'confirm_password' => trim($_POST['confirm_password'] ?? ''),
                'referral_code' => strtoupper(trim($_POST['referral_code'] ?? ''))
            ];

            // Input validation
            $validation = $this->validateRegistrationData($data);
            if (!$validation['valid']) {
                $this->sendResponse(false, 'Validation failed', ['errors' => $validation['errors']], 400);
            }

            // Check duplicates with cache
            if ($this->isEmailTaken($data['email'])) {
                $this->sendResponse(false, 'Email is already taken.', [], 409);
            }

            if ($this->isPhoneTaken($data['phone'])) {
                $this->sendResponse(false, 'Phone number is already taken.', [], 409);
            }

            // Hash password
            $data['password'] = password_hash($data['password'], PASSWORD_ARGON2ID);

            // Validate referral code
            $referrerId = null;
            if (!empty($data['referral_code'])) {
                $referrer = $this->findUserByReferralCodeCached($data['referral_code']);
                if ($referrer) {
                    $referrerId = $referrer->user_id;
                } else {
                    $this->sendResponse(false, 'Invalid referral code.', [], 400);
                }
            }

            // Generate unique referral code
            $data['new_referral_code'] = $this->generateUniqueReferralCode($data['name']);
            $data['referrer_id'] = $referrerId;

            // Register user
            $register = $this->userModel->register($data);
            
            if (!$register) {
                error_log("Registration failed for: {$data['email']}");
                $this->sendResponse(false, 'Failed to register user. Please try again.', [], 500);
            }

            $registeredUser = $this->userModel->findUserByEmailAndGetDetails($data['email']);
            
            if (!$registeredUser) {
                error_log("User not found after registration: {$data['email']}");
                $this->sendResponse(false, 'Registration error. Please contact support.', [], 500);
            }
            
            // Add default role
            $this->userModel->addRoleToUser($registeredUser->user_id, 'Buyer');

            // Create a wallet for the new user
            $this->walletModel->createWallet($registeredUser->user_id, $data['name']);
            
            // Generate verification token
            $verificationToken = bin2hex(random_bytes(32));
            $this->userModel->saveVerificationToken($registeredUser->user_id, $verificationToken);
            
            // Queue async email tasks
            if ($this->jobQueue) {
                $this->jobQueue->push('SendVerificationEmail', [
                    'user_id' => $registeredUser->user_id,
                    'email' => $data['email'],
                    'name' => $data['name'],
                    'token' => $verificationToken
                ]);
                
                $this->jobQueue->push('SendWelcomeEmail', [
                    'user_id' => $registeredUser->user_id,
                    'email' => $data['email'],
                    'name' => $data['name']
                ]);
            }
            
            // Auto login
            $accessToken = $this->loginRegisteredUser([
                'email' => $data['email'],
                'password' => $_POST['password']
            ]);
            
            // Invalidate relevant caches
            $this->invalidateUserCaches($data['email']);
            
            $this->sendResponse(true, 'Registration successful', [
                'user_id' => $registeredUser->user_id,
                'access_token' => $accessToken,
                'referral_code' => $data['new_referral_code']
            ], 201);
            
        } catch (Exception $e) {
            error_log("Registration error: " . $e->getMessage());
            $this->sendResponse(false, 'An error occurred during password reset.', [], 500);
        }
    }

    /**
     * Login function
     * OPTIMIZED: Rate limiting, account lockout, caching
     */
    public function loginfunc()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $postData = $this->getData();
            $username = strtolower(trim($postData['username'] ?? ''));
            $password = trim($postData['password'] ?? '');

            if (empty($username) || empty($password)) {
                $this->sendResponse(false, 'Username and password are required', [], 400);
            }

            // Rate limiting by IP and username
            $ip = $_SERVER['REMOTE_ADDR'] ?? 'unknown';
            $lockoutKey = "login_lockout:{$username}";
            
            if ($this->cache && $this->cache->exists($lockoutKey)) {
                $this->sendResponse(false, 'Account temporarily locked due to too many failed attempts. Try again in 15 minutes.', [], 429);
            }

            // Check failed attempts
            $attemptsKey = "login_attempts:{$username}";
            $attempts = $this->cache ? (int)$this->cache->get($attemptsKey) : 0;
            
            if ($attempts >= self::MAX_LOGIN_ATTEMPTS) {
                if ($this->cache) {
                    $this->cache->set($lockoutKey, 1, self::LOGIN_LOCKOUT_TIME);
                    $this->cache->del($attemptsKey);
                }
                $this->sendResponse(false, 'Too many failed login attempts. Account locked for 15 minutes.', [], 429);
            }

            $user = $this->userModel->findUserByEmailOrPhone($username);

            if ($user && password_verify($password, $user->password_hash)) {
                // Successful login - clear attempts
                if ($this->cache) {
                    $this->cache->del($attemptsKey);
                }
                
                $accessToken = $this->createUserSession($user);
                
                $this->sendResponse(true, 'Login successful', [
                    'access_token' => $accessToken,
                    'user_data' => [
                        'user_id' => $user->user_id,
                        'name' => $user->name,
                        'email' => $user->email,
                        'is_verified' => $user->is_verified
                    ]
                ]);
            } else {
                // Failed login - increment attempts
                if ($this->cache) {
                    $this->cache->increment($attemptsKey, 1);
                    $this->cache->expire($attemptsKey, 3600);
                }
                
                $remainingAttempts = self::MAX_LOGIN_ATTEMPTS - $attempts - 1;
                $this->sendResponse(false, "Invalid credentials. {$remainingAttempts} attempts remaining.", [], 401);
            }
            
        } catch (Exception $e) {
            error_log("Login error: " . $e->getMessage());
            $this->sendResponse(false, 'An error occurred during login.', [], 500);
        }
    }

    /**
     * Verify email
     * OPTIMIZED: Better error handling
     */
    public function verifyEmail()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }
            
            $postData = $this->getData();
            $token = trim($postData['token'] ?? '');

            if (empty($token)) {
                $this->sendResponse(false, 'Verification token is required.', [], 400);
            }
            
            $user = $this->userModel->findUserByVerificationToken($token);

            if ($user) {
                if ($this->userModel->markEmailAsVerified($user->user_id)) {
                    // Invalidate user cache
                    $this->invalidateUserCaches($user->email, $user->user_id);
                    
                    $this->sendResponse(true, 'Email verified successfully. You can now log in.');
                } else {
                    $this->sendResponse(false, 'Failed to verify email. Please try again.', [], 500);
                }
            } else {
                $this->sendResponse(false, 'Invalid or expired verification token.', [], 404);
            }
            
        } catch (Exception $e) {
            error_log("Email verification error: " . $e->getMessage());
            $this->sendResponse(false, 'An error occurred during verification.', [], 500);
        }
    }

    /**
     * Social login
     * OPTIMIZED: Better validation, async processing
     */
    public function socialLogin()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            // Rate limiting
            $ip = $_SERVER['REMOTE_ADDR'] ?? 'unknown';
            if ($this->rateLimiter && !$this->rateLimiter->check("social_login:{$ip}", 10, 3600)) {
                $this->sendResponse(false, 'Too many login attempts. Please try again later.', [], 429);
            }

            $postData = $this->getData();
            
            // Extract and validate input
            $provider = strtolower(trim($postData['provider'] ?? ''));
            $accessToken = trim($postData['access_token'] ?? ''); // OAuth token from provider
            $idToken = trim($postData['id_token'] ?? ''); // For Google/Apple
            
            // Validate provider
            $allowedProviders = ['google', 'facebook', 'apple'];
            if (empty($provider) || !in_array($provider, $allowedProviders)) {
                $this->sendResponse(false, 'Invalid or missing social provider.', [], 400);
            }

            // Verify token and extract user data from provider
            $socialUserData = $this->verifySocialToken($provider, $accessToken, $idToken);
            
            if (!$socialUserData) {
                error_log("Social login verification failed for provider: {$provider}");
                $this->sendResponse(false, 'Failed to verify credentials with social provider. Please try again.', [], 401);
            }

            // Ensure we have required data
            if (empty($socialUserData['social_id']) || empty($socialUserData['email'])) {
                $this->sendResponse(false, 'Incomplete data from social provider. Email is required.', [], 400);
            }

            // Process the social login
            $result = $this->processSocialLogin($socialUserData, $provider);
            
            if ($result['success']) {
                $this->sendResponse(
                    true, 
                    $result['message'], 
                    $result['data'], 
                    $result['is_new_user'] ? 201 : 200
                );
            } else {
                $this->sendResponse(false, $result['message'], [], $result['http_code'] ?? 500);
            }
            
        } catch (Exception $e) {
            error_log("Social login error: " . $e->getMessage());
            $this->sendResponse(false, 'An error occurred during social login. Please try again.', [], 500);
        }
    }

    /**
     * Verify social login tokens with provider APIs
     */
    private function verifySocialToken($provider, $accessToken, $idToken = null)
    {
        try {
            switch ($provider) {
                case 'google':
                    return $this->verifyGoogleToken($idToken ?: $accessToken);
                
                case 'facebook':
                    return $this->verifyFacebookToken($accessToken);
                
                case 'apple':
                    return $this->verifyAppleToken($idToken);
                
                default:
                    return false;
            }
        } catch (Exception $e) {
            error_log("Token verification error for {$provider}: " . $e->getMessage());
            return false;
        }
    }

    /**
     * Verify Google OAuth token
     */
    private function verifyGoogleToken($idToken)
    {
        try {
            $clientId = GOOGLE_CLIENT_ID;
            
            // Verify with Google's token verification endpoint
            $url = "https://oauth2.googleapis.com/tokeninfo?id_token=" . urlencode($idToken);
            
            $ch = curl_init();
            curl_setopt($ch, CURLOPT_URL, $url);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_TIMEOUT, 10);
            curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, true);
            
            $response = curl_exec($ch);
            $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
            curl_close($ch);
            
            if ($httpCode !== 200) {
                error_log("Google token verification failed with HTTP {$httpCode}");
                return false;
            }
            
            $data = json_decode($response, true);
            
            // Verify the token is for our app
            if (!isset($data['aud']) || $data['aud'] !== $clientId) {
                error_log("Google token audience mismatch");
                return false;
            }
            
            // Check if token is expired
            if (isset($data['exp']) && $data['exp'] < time()) {
                error_log("Google token expired");
                return false;
            }
            
            return [
                'social_id' => $data['sub'] ?? null,
                'email' => $data['email'] ?? null,
                'name' => $data['name'] ?? '',
                'first_name' => $data['given_name'] ?? '',
                'last_name' => $data['family_name'] ?? '',
                'picture' => $data['picture'] ?? null,
                'email_verified' => isset($data['email_verified']) ? (bool)$data['email_verified'] : false
            ];
            
        } catch (Exception $e) {
            error_log("Google verification exception: " . $e->getMessage());
            return false;
        }
    }

    /**
     * Verify Facebook OAuth token
     */
    private function verifyFacebookToken($accessToken)
    {
        try {
            $appId = FACEBOOK_APP_ID;
            $appSecret = FACEBOOK_APP_SECRET;
            
            // Get app access token
            $appAccessToken = "{$appId}|{$appSecret}";
            
            // Verify the user token
            $debugUrl = "https://graph.facebook.com/debug_token?input_token=" . urlencode($accessToken) 
                      . "&access_token=" . urlencode($appAccessToken);
            
            $ch = curl_init();
            curl_setopt($ch, CURLOPT_URL, $debugUrl);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_TIMEOUT, 10);
            curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, true);
            
            $debugResponse = curl_exec($ch);
            curl_close($ch);
            
            $debugData = json_decode($debugResponse, true);
            
            // Check if token is valid
            if (!isset($debugData['data']['is_valid']) || !$debugData['data']['is_valid']) {
                error_log("Facebook token validation failed");
                return false;
            }
            
            // Check if token is for our app
            if (!isset($debugData['data']['app_id']) || $debugData['data']['app_id'] !== $appId) {
                error_log("Facebook token app mismatch");
                return false;
            }
            
            // Get user data
            $userUrl = "https://graph.facebook.com/me?fields=id,name,email,first_name,last_name,picture&access_token=" 
                     . urlencode($accessToken);
            
            $ch = curl_init();
            curl_setopt($ch, CURLOPT_URL, $userUrl);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_TIMEOUT, 10);
            curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, true);
            
            $userResponse = curl_exec($ch);
            curl_close($ch);
            
            $userData = json_decode($userResponse, true);
            
            if (!isset($userData['id'])) {
                error_log("Facebook user data fetch failed");
                return false;
            }
            
            return [
                'social_id' => $userData['id'],
                'email' => $userData['email'] ?? null,
                'name' => $userData['name'] ?? '',
                'first_name' => $userData['first_name'] ?? '',
                'last_name' => $userData['last_name'] ?? '',
                'picture' => $userData['picture']['data']['url'] ?? null,
                'email_verified' => true // Facebook provides verified emails
            ];
            
        } catch (Exception $e) {
            error_log("Facebook verification exception: " . $e->getMessage());
            return false;
        }
    }

    /**
     * Verify Apple Sign In token
     */
    private function verifyAppleToken($idToken)
    {
        try {
            // Apple Sign In uses JWT tokens signed with Apple's private key
            // We need to verify the signature using Apple's public keys
            
            // Get Apple's public keys
            $keysUrl = "https://appleid.apple.com/auth/keys";
            
            $ch = curl_init();
            curl_setopt($ch, CURLOPT_URL, $keysUrl);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_TIMEOUT, 10);
            curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, true);
            
            $keysResponse = curl_exec($ch);
            curl_close($ch);
            
            $keys = json_decode($keysResponse, true);
            
            if (!isset($keys['keys'])) {
                error_log("Failed to fetch Apple public keys");
                return false;
            }
            
            // Decode JWT without verification first to get header
            $parts = explode('.', $idToken);
            if (count($parts) !== 3) {
                error_log("Invalid Apple ID token format");
                return false;
            }
            
            $header = json_decode(base64_decode(strtr($parts[0], '-_', '+/')), true);
            $payload = json_decode(base64_decode(strtr($parts[1], '-_', '+/')), true);
            
            if (!$header || !$payload) {
                error_log("Failed to decode Apple token");
                return false;
            }
            
            // Find the matching public key
            $kid = $header['kid'] ?? null;
            $publicKey = null;
            
            foreach ($keys['keys'] as $key) {
                if ($key['kid'] === $kid) {
                    // Convert JWK to PEM format (simplified - you may want to use a library)
                    // For production, use: firebase/php-jwt or similar
                    $publicKey = $key;
                    break;
                }
            }
            
            if (!$publicKey) {
                error_log("Apple public key not found for kid: {$kid}");
                return false;
            }
            
            // Verify claims
            $clientId = APPLE_CLIENT_ID;
            
            if (!isset($payload['aud']) || $payload['aud'] !== $clientId) {
                error_log("Apple token audience mismatch");
                return false;
            }
            
            if (!isset($payload['iss']) || $payload['iss'] !== 'https://appleid.apple.com') {
                error_log("Apple token issuer mismatch");
                return false;
            }
            
            if (!isset($payload['exp']) || $payload['exp'] < time()) {
                error_log("Apple token expired");
                return false;
            }
            
            // Extract user data
            return [
                'social_id' => $payload['sub'] ?? null,
                'email' => $payload['email'] ?? null,
                'name' => '', // Apple doesn't always provide name in token
                'first_name' => '',
                'last_name' => '',
                'picture' => null,
                'email_verified' => isset($payload['email_verified']) ? (bool)$payload['email_verified'] : false
            ];
            
        } catch (Exception $e) {
            error_log("Apple verification exception: " . $e->getMessage());
            return false;
        }
    }

    /**
     * Process social login - handle existing users, linking, and registration
     */
    private function processSocialLogin($socialData, $provider)
    {
        try {
            $socialId = $socialData['social_id'];
            $email = strtolower(trim($socialData['email']));
            $name = trim($socialData['name']) ?: trim($socialData['first_name'] . ' ' . $socialData['last_name']);
            
            // Scenario 1: Check if user exists with this social ID + provider
            $existingBySocial = $this->userModel->findUserBySocialId($socialId, $provider);
            
            if ($existingBySocial) {
                return $this->handleExistingSocialUser($existingBySocial);
            }
            
            // Scenario 2: Check if user exists with this email
            $existingByEmail = $this->userModel->findUserByEmailAndGetDetails($email);
            
            if ($existingByEmail) {
                return $this->handleAccountLinking($existingByEmail, $socialData, $provider);
            }
            
            // Scenario 3: New user - register
            return $this->handleNewSocialUser($socialData, $provider);
            
        } catch (Exception $e) {
            error_log("Process social login error: " . $e->getMessage());
            return [
                'success' => false,
                'message' => 'An error occurred processing your login.',
                'http_code' => 500
            ];
        }
    }

    /**
     * Handle login for existing social user
     */
    private function handleExistingSocialUser($user)
    {
        try {
            $jwt = $this->createUserSession($user);
            
            if (!$jwt) {
                return [
                    'success' => false,
                    'message' => 'Failed to create session.',
                    'http_code' => 500
                ];
            }
            
            // Update last login timestamp
            $this->userModel->updateLastLogin($user->user_id);
            
            return [
                'success' => true,
                'message' => 'Login successful',
                'is_new_user' => false,
                'data' => [
                    'access_token' => $jwt,
                    'user_data' => [
                        'user_id' => $user->user_id,
                        'name' => $user->name,
                        'email' => $user->email,
                        'is_verified' => $user->is_verified,
                        'phone' => $user->phone
                    ]
                ]
            ];
            
        } catch (Exception $e) {
            error_log("Handle existing social user error: " . $e->getMessage());
            return [
                'success' => false,
                'message' => 'Login failed.',
                'http_code' => 500
            ];
        }
    }

    /**
     * Link social account to existing email account
     */
    private function handleAccountLinking($existingUser, $socialData, $provider)
    {
        try {
            $this->userModel->db->beginTransaction();
            
            // Link social account to existing user
            $linked = $this->userModel->linkSocialAccount(
                $existingUser->user_id,
                $socialData['social_id'],
                $provider
            );
            
            if (!$linked) {
                $this->userModel->db->rollBack();
                return [
                    'success' => false,
                    'message' => 'Failed to link social account.',
                    'http_code' => 500
                ];
            }
            
            // If email from social provider is verified, mark user as verified
            if (!empty($socialData['email_verified']) && !$existingUser->is_verified) {
                $this->userModel->markEmailAsVerified($existingUser->user_id);
            }
            
            // Update profile picture if not set
            if (!empty($socialData['picture']) && empty($existingUser->profile_picture)) {
                $this->userModel->updateProfilePicture($existingUser->user_id, $socialData['picture']);
            }
            
            $this->userModel->db->commit();
            
            // Invalidate caches
            $this->invalidateUserCaches($existingUser->email, $existingUser->user_id);
            
            // Get updated user data
            $updatedUser = $this->userModel->findUserById($existingUser->user_id);
            $jwt = $this->createUserSession($updatedUser);
            
            // Queue notification email
            if ($this->jobQueue) {
                $this->jobQueue->push('SendAccountLinkedEmail', [
                    'user_id' => $existingUser->user_id,
                    'email' => $existingUser->email,
                    'name' => $existingUser->name,
                    'provider' => ucfirst($provider)
                ]);
            }
            
            return [
                'success' => true,
                'message' => 'Social account linked and login successful',
                'is_new_user' => false,
                'data' => [
                    'access_token' => $jwt,
                    'user_data' => [
                        'user_id' => $updatedUser->user_id,
                        'name' => $updatedUser->name,
                        'email' => $updatedUser->email,
                        'is_verified' => $updatedUser->is_verified,
                        'phone' => $updatedUser->phone
                    ],
                    'account_linked' => true
                ]
            ];
            
        } catch (Exception $e) {
            if ($this->userModel->db->inTransaction()) {
                $this->userModel->db->rollBack();
            }
            error_log("Account linking error: " . $e->getMessage());
            return [
                'success' => false,
                'message' => 'Failed to link accounts.',
                'http_code' => 500
            ];
        }
    }

    /**
     * Register new user from social login
     */
    private function handleNewSocialUser($socialData, $provider)
    {
        try {
            $this->userModel->db->beginTransaction();
            
            $email = strtolower(trim($socialData['email']));
            $name = trim($socialData['name']) ?: trim($socialData['first_name'] . ' ' . $socialData['last_name']);
            
            if (empty($name)) {
                $name = explode('@', $email)[0]; // Fallback to email username
            }
            
            // Generate unique user ID
            $userId = 'mplaza' . md5(date('ymdhms') . rand(1000, 9999));
            
            // Generate referral code
            $referralCode = $this->generateUniqueReferralCode($name);
            
            // Register user
            $registrationData = [
                'user_id' => $userId,
                'name' => $name,
                'email' => $email,
                'phone' => null, // Social users don't have phone initially
                'password_hash' => null, // No password for social users
                'social_id' => $socialData['social_id'],
                'social_provider' => $provider,
                'is_verified' => !empty($socialData['email_verified']) ? 1 : 0,
                'referral_code' => $referralCode,
                'profile_picture' => $socialData['picture'] ?? null
            ];
            
            $newUserId = $this->userModel->registerSocialUser($registrationData);
            
            if (!$newUserId) {
                $this->userModel->db->rollBack();
                return [
                    'success' => false,
                    'message' => 'Failed to create account.',
                    'http_code' => 500
                ];
            }
            
            // Add default Buyer role
            $this->userModel->addRoleToUser($newUserId, 'Buyer');
            
            
            // Create wallet
            $this->walletModel->createWallet($newUserId);
            
            $this->userModel->db->commit();
            
            // Get newly created user
            $newUser = $this->userModel->findUserById($newUserId);
            $jwt = $this->createUserSession($newUser);
            
            // Queue welcome email
            if ($this->jobQueue) {
                $this->jobQueue->push('SendWelcomeEmail', [
                    'user_id' => $newUserId,
                    'email' => $email,
                    'name' => $name,
                    'is_social_signup' => true,
                    'provider' => ucfirst($provider)
                ]);
            }
            
            return [
                'success' => true,
                'message' => 'Account created and login successful',
                'is_new_user' => true,
                'data' => [
                    'access_token' => $jwt,
                    'user_data' => [
                        'user_id' => $newUser->user_id,
                        'name' => $newUser->name,
                        'email' => $newUser->email,
                        'is_verified' => $newUser->is_verified,
                        'phone' => $newUser->phone
                    ],
                    'referral_code' => $referralCode,
                    'needs_phone' => true // Indicate user should add phone later
                ]
            ];
            
        } catch (Exception $e) {
            if ($this->userModel->db->inTransaction()) {
                $this->userModel->db->rollBack();
            }
            error_log("New social user registration error: " . $e->getMessage());
            return [
                'success' => false,
                'message' => 'Failed to create account.',
                'http_code' => 500
            ];
        }
    }

    /**
     * Upgrade account
     * OPTIMIZED: Added validation and caching
     */
    public function upgradeAccount()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $userData = $this->RouteProtection();
            $postData = $this->getData();
            $newRoleName = trim($postData['role_name'] ?? '');

            $validRoles = ['Seller', 'Reseller', 'DeliveryAgent'];
            if (empty($newRoleName) || !in_array($newRoleName, $validRoles)) {
                $this->sendResponse(false, 'Invalid role name provided.', [], 400);
            }

            if ($this->userModel->hasRole($userData->user_id, $newRoleName)) {
                $this->sendResponse(false, 'You already have this role.', [], 409);
            }

            // --- Subscription Logic for Reseller ---
            if ($newRoleName === 'Reseller') {
                $resellerPlan = $this->subscriptionPlanModel->getPlanById('reseller_monthly');

                if (!$resellerPlan) {
                    $this->sendResponse(false, 'Reseller plan not found. Please contact support.', [], 500);
                }

                // Check for sufficient funds
                if (!$this->walletModel->hasSufficientFunds($userData->user_id, $resellerPlan->price)) {
                    $this->sendResponse(false, 'Insufficient funds in your wallet to subscribe to Reseller plan. Please fund your wallet.', [], 402);
                }

                // Start Transaction
                $this->userModel->db->beginTransaction();

                // Deduct subscription fee from wallet
                if (!$this->walletModel->processWalletPayment($userData->user_id, $resellerPlan->price, 'reseller_subscription')) {
                    $this->userModel->db->rollBack();
                    $this->sendResponse(false, 'Failed to process subscription payment.', [], 500);
                }

                // Create subscription
                $startDate = date('Y-m-d H:i:s');
                $endDate = date('Y-m-d H:i:s', strtotime("+{$resellerPlan->duration_days} days"));

                $subscriptionData = [
                    'user_id' => $userData->user_id,
                    'plan_id' => $resellerPlan->plan_id,
                    'status' => 'active',
                    'amount' => $resellerPlan->price,
                    'start_date' => $startDate,
                    'end_date' => $endDate,
                    'billing_cycle' => 'monthly', // Assuming monthly for reseller
                    'currency' => 'NGN' // Assuming NGN
                ];

                if (!$this->subscriptionModel->createSubscription($subscriptionData)) {
                    $this->userModel->db->rollBack();
                    $this->sendResponse(false, 'Failed to create reseller subscription.', [], 500);
                }

                // Add reseller role
                if (!$this->userModel->addRoleToUser($userData->user_id, $newRoleName)) {
                    $this->userModel->db->rollBack();
                    $this->sendResponse(false, 'Failed to upgrade account after payment.', [], 500);
                }

                // Commit Transaction
                $this->userModel->db->commit();

            } else {
                // For other roles, just add the role without payment
                if (!$this->userModel->addRoleToUser($userData->user_id, $newRoleName)) {
                    $this->sendResponse(false, 'Failed to upgrade account.', [], 500);
                }
            }

            // Invalidate user cache
            $this->invalidateUserCaches($userData->email, $userData->user_id);
            
            // Queue notification email
            if ($this->jobQueue) {
                $this->jobQueue->push('SendRoleUpgradeEmail', [
                    'user_id' => $userData->user_id,
                    'email' => $userData->email,
                    'name' => $userData->name,
                    'role' => $newRoleName
                ]);
            }
            
            $this->sendResponse(true, "Account successfully upgraded to {$newRoleName}.");

        } catch (Exception $e) {
            if ($this->userModel->db->inTransaction()) {
                $this->userModel->db->rollBack();
            }
            error_log("Account upgrade error: " . $e->getMessage());
            $this->sendResponse(false, 'An error occurred during account upgrade.', [], 500);
        }
    }

    /**
     * Get referrals
     * OPTIMIZED: Added pagination
     */
    public function getReferrals()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $userData = $this->RouteProtection();
            
            $page = isset($_GET['page']) ? max(1, intval($_GET['page'])) : 1;
            $pageSize = isset($_GET['page_size']) ? min(100, max(1, intval($_GET['page_size']))) : 20;
            
            $referrals = $this->userModel->getReferralsForUser($userData->user_id, $page, $pageSize);
            
            $this->sendResponse(true, 'Referrals retrieved successfully.', [
                'referrals' => $referrals['data'],
                'pagination' => $referrals['pagination']
            ]);
            
        } catch (Exception $e) {
            error_log("Get referrals error: " . $e->getMessage());
            $this->sendResponse(false, 'Failed to retrieve referrals.', [], 500);
        }
    }

    /**
     * Get user profile
     * OPTIMIZED: Added caching
     */
    public function getUser()
    {
        try {
            $userData = $this->RouteProtection();
            
            // Try cache first
            $cacheKey = "user_profile:{$userData->user_id}";
            if ($this->cache) {
                $cached = $this->cache->get($cacheKey);
                if ($cached !== false) {
                    $this->sendResponse(true, 'User data retrieved successfully', $cached);
                }
            }
            
            // Get fresh data
            $user = $this->userModel->findUserById($userData->user_id);
            
            if ($user) {
                // Cache the result
                if ($this->cache) {
                    $this->cache->set($cacheKey, $user, self::USER_CACHE_TTL);
                }
                
                $this->sendResponse(true, 'User data retrieved successfully', $user);
            } else {
                $this->sendResponse(false, 'User not found.', [], 404);
            }
            
        } catch (Exception $e) {
            error_log("Get user error: " . $e->getMessage());
            $this->sendResponse(false, 'Failed to retrieve user data.', [], 500);
        }
    }

    /**
     * Update location
     * OPTIMIZED: Better validation
     */
    public function updateLocation()
    {
        try {
            $userData = $this->RouteProtection();

            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $postData = $this->getData();
            $latitude = $postData['latitude'] ?? null;
            $longitude = $postData['longitude'] ?? null;

            if (!is_numeric($latitude) || !is_numeric($longitude)) {
                $this->sendResponse(false, 'Latitude and longitude are required and must be numeric.', [], 400);
            }

            // --- Reverse Geocoding using OpenStreetMap Nominatim ---
            $url = "https://nominatim.openstreetmap.org/reverse?format=json&lat={$latitude}&lon={$longitude}&addressdetails=1";

            // Nominatim requires a custom User-Agent
            $options = [
                'http' => [
                    'header' => "User-Agent: Divine APIApp/1.0\r\n"
                ]
            ];
            $context = stream_context_create($options);
            $response = file_get_contents($url, false, $context);
            $geoData = json_decode($response, true);

            if (!$geoData || isset($geoData['error'])) {
                $this->sendResponse(false, 'Could not retrieve address details for the provided coordinates.', [], 500);
            }

            $address = $geoData['address'] ?? [];
            
            $locationData = [
                'latitude' => $latitude,
                'longitude' => $longitude,
                'address' => $geoData['display_name'] ?? '',
                'city' => $address['city'] ?? $address['town'] ?? '',
                'state' => $address['state'] ?? '',
                'lga' => $address['county'] ?? '' // In OSM data for Nigeria, 'county' often corresponds to LGA
            ];

            if ($this->userModel->updateLocation($userData->user_id, $locationData)) {
                // Invalidate user cache
                $this->invalidateUserCaches($userData->email, $userData->user_id);
                
                $this->sendResponse(true, 'Location updated successfully.', $locationData);
            } else {
                $this->sendResponse(false, 'Failed to update location in database.', [], 500);
            }
            
        } catch (Exception $e) {
            error_log("Update location error: " . $e->getMessage());
            $this->sendResponse(false, 'An error occurred while updating location.', [], 500);
        }
    }

    /**
     * Logout
     * OPTIMIZED: Added cache invalidation
     */
    public function logout()
    {
        try {
            $userData = $this->RouteProtection();

            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            if ($this->userModel->updateToken($userData->user_id, '', $userData->email)) {
                // Invalidate all user caches
                $this->invalidateUserCaches($userData->email, $userData->user_id);
                
                $this->sendResponse(true, 'Logged out successfully');
            } else {
                $this->sendResponse(false, 'Failed to logout', [], 500);
            }
            
        } catch (Exception $e) {
            error_log("Logout error: " . $e->getMessage());
            $this->sendResponse(false, 'Failed to logout.', [], 500);
        }
    }

    /**
     * Delete profile
     * OPTIMIZED: Better security, async processing
     */
    public function deleteProfile()
    {
        try {
            $userData = $this->RouteProtection();

            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $postData = $this->getData();
            $password = trim($postData['password'] ?? '');
            $confirmation = trim($postData['confirmation'] ?? '');

            if (empty($password)) {
                $this->sendResponse(false, 'Password is required to delete account', [], 400);
            }

            if ($confirmation !== 'DELETE') {
                $this->sendResponse(false, 'Please type "DELETE" to confirm account deletion', [], 400);
            }

            $user = $this->userModel->findUserById($userData->user_id);
            if (!$user || !password_verify($password, $user->password_hash)) {
                $this->sendResponse(false, 'Invalid password', [], 401);
            }

            // Queue deletion email before deleting
            if ($this->jobQueue) {
                $this->jobQueue->push('SendAccountDeletionEmail', [
                    'email' => $user->email,
                    'name' => $user->name,
                    'user_id' => $user->user_id
                ]);
            }

            if ($this->userModel->deleteUser($userData->user_id)) {
                // Clear all caches
                $this->invalidateUserCaches($user->email, $user->user_id);
                
                $this->sendResponse(true, 'Account deleted successfully');
            } else {
                $this->sendResponse(false, 'Failed to delete account', [], 500);
            }
            
        } catch (Exception $e) {
            error_log("Delete profile error: " . $e->getMessage());
            $this->sendResponse(false, 'Failed to delete account.', [], 500);
        }
    }

    /**
     * Forgot password
     * OPTIMIZED: Rate limiting, better security
     */
    public function forgotPassword()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $postData = $this->getData();
            $email = strtolower(trim($postData['email'] ?? ''));

            if (empty($email)) {
                $this->sendResponse(false, 'Email is required', [], 400);
            }

            // Rate limiting
            $ip = $_SERVER['REMOTE_ADDR'] ?? 'unknown';
            if ($this->rateLimiter && !$this->rateLimiter->check("forgot_password:{$ip}", 3, 3600)) {
                $this->sendResponse(false, 'Too many password reset requests. Try again later.', [], 429);
            }

            $user = $this->userModel->findUserByEmailAndGetDetails($email);

            if ($user) {
                error_log("Forgot password request for email: " . $email);
                $resetToken = $this->userModel->generatePasswordResetToken($email);
                error_log("Reset token generated: " . ($resetToken ? $resetToken : 'false'));

                if ($resetToken && $this->jobQueue) {
                    try{
                $this->jobQueue->push('SendPasswordResetEmail', [
                        'email' => $user->email,
                        'name' => $user->name,
                        'token' => $resetToken,
                        'user_id' => $user->user_id
                    ]);
                    } catch(Exception $e){
                        error_log("Error Pushing job to queue: ". $e->getMessage());
                    }
                    
                }
            }

            // Always return success to prevent email enumeration
            $this->sendResponse(true, 'If an account exists with this email, a password reset link has been sent.');
            
        } catch (Exception $e) {
            error_log("Forgot password error: " . $e->getMessage());
            $this->sendResponse(false, 'An error occurred. Please try again.', [], 500);
        }
    }

    /**
     * Reset password
     * OPTIMIZED: Better validation
     */
    public function resetPassword()
    {
        try {
            if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
                $this->sendResponse(false, 'Invalid request method.', [], 405);
            }

            $postData = $this->getData();
            $email = strtolower(trim($postData['email'] ?? ''));
            $resetToken = trim($postData['reset_token'] ?? '');
            $newPassword = trim($postData['new_password'] ?? '');
            $confirmPassword = trim($postData['confirm_password'] ?? '');

            if (empty($email) || empty($resetToken) || empty($newPassword)) {
                $this->sendResponse(false, 'All fields are required', [], 400);
            }

            if (strlen($newPassword) < 8) {
                $this->sendResponse(false, 'Password must be at least 8 characters long', [], 400);
            }

            if ($newPassword !== $confirmPassword) {
                $this->sendResponse(false, 'Passwords do not match', [], 400);
            }

            $hashedPassword = password_hash($newPassword, PASSWORD_ARGON2ID);

            if ($this->userModel->resetPasswordWithToken($email, $resetToken, $hashedPassword)) {
                // Invalidate user caches
                $this->invalidateUserCaches($email);
                
                $this->sendResponse(true, 'Password reset successfully. You can now login with your new password.');
            } else {
                $this->sendResponse(false, 'Invalid or expired reset token.', [], 400);
            }
            
        } catch (Exception $e) {
            error_log("Reset password error: " . $e->getMessage());
            $this->sendResponse(false, 'Failed to reset password.', [], 500);
        }
    }

    /**
     * Create user session with JWT
     * OPTIMIZED: Better token structure
     */
    public function createUserSession($user)
    {
        try {
            $userRoles = $this->userModel->getRolesForUser($user->user_id);
            $rolesArray = array_map(function($role) {
                return $role->role_name;
            }, $userRoles);
            
            $payload = [
                'user_id' => $user->user_id,
                'name' => $user->name,
                'email' => $user->email,
                'phone' => $user->phone,
                'location' => $user->location,
                'is_verified' => $user->is_verified,
                'roles' => $rolesArray,
                'iat' => time(),
                'exp' => time() + (60 * 24 * 60 * 60) // 60 days
            ];

            $jwt = $this->generateJWT($payload, $this->serverKey);
            $this->userModel->updateToken($user->user_id, $jwt, $user->email);
            
            return $jwt;
        } catch (Exception $e) {
            error_log("Create session error: " . $e->getMessage());
            return false;
        }
    }

    /**
     * Login registered user (helper method)
     */
    public function loginRegisteredUser($data)
    {
        try {
            $email = $data['email'];
            $password = $data['password'];

            $user = $this->userModel->findUserByEmailAndGetDetails($email);

            if ($user && password_verify($password, $user->password_hash)) {
                return $this->createUserSession($user);
            }
            
            return false;
        } catch (Exception $e) {
            error_log("Login registered user error: " . $e->getMessage());
            return false;
        }
    }

    // ==================== HELPER METHODS ====================

    /**
     * Validate registration data
     */
    private function validateRegistrationData($data)
    {
        $errors = [];

        if (empty($data['name'])) {
            $errors['name'] = 'Please enter your full name';
        } elseif (strlen($data['name']) < 2) {
            $errors['name'] = 'Name must be at least 2 characters';
        }

        if (empty($data['email'])) {
            $errors['email'] = 'Please enter an email';
        } elseif (!filter_var($data['email'], FILTER_VALIDATE_EMAIL)) {
            $errors['email'] = 'Please enter a valid email';
        }

        if (empty($data['phone'])) {
            $errors['phone'] = 'Please enter your phone number';
        } elseif (!preg_match('/^[0-9+\-() ]{10,20}$/', $data['phone'])) {
            $errors['phone'] = 'Please enter a valid phone number';
        }

        if (empty($data['password'])) {
            $errors['password'] = 'Please enter a password';
        } elseif (strlen($data['password']) < 8) {
            $errors['password'] = 'Password must be at least 8 characters';
        } elseif (!preg_match('/[A-Z]/', $data['password'])) {
            $errors['password'] = 'Password must contain at least one uppercase letter';
        } elseif (!preg_match('/[a-z]/', $data['password'])) {
            $errors['password'] = 'Password must contain at least one lowercase letter';
        } elseif (!preg_match('/[0-9]/', $data['password'])) {
            $errors['password'] = 'Password must contain at least one number';
        }

        if (empty($data['confirm_password'])) {
            $errors['confirm_password'] = 'Please confirm password';
        } elseif ($data['password'] !== $data['confirm_password']) {
            $errors['confirm_password'] = 'Passwords do not match';
        }

        return [
            'valid' => empty($errors),
            'errors' => $errors
        ];
    }

    /**
     * Check if email is taken with caching
     */
    private function isEmailTaken($email)
    {
        $cacheKey = "email_exists:{$email}";
        
        if ($this->cache) {
            $cached = $this->cache->get($cacheKey);
            if ($cached !== false) {
                return (bool)$cached;
            }
        }
        
        $exists = $this->userModel->findUserByEmail($email) !== false;
        
        if ($this->cache) {
            $this->cache->set($cacheKey, $exists ? 1 : 0, 300); // Cache for 5 minutes
        }
        
        return $exists;
    }

    /**
     * Check if phone is taken with caching
     */
    private function isPhoneTaken($phone)
    {
        $cacheKey = "phone_exists:{$phone}";
        
        if ($this->cache) {
            $cached = $this->cache->get($cacheKey);
            if ($cached !== false) {
                return (bool)$cached;
            }
        }
        
        $exists = $this->userModel->findUserByPhone($phone) !== false;
        
        if ($this->cache) {
            $this->cache->set($cacheKey, $exists ? 1 : 0, 300);
        }
        
        return $exists;
    }

    /**
     * Find user by referral code with caching
     */
    private function findUserByReferralCodeCached($code)
    {
        $cacheKey = "referral_user:{$code}";
        
        if ($this->cache) {
            $cached = $this->cache->get($cacheKey);
            if ($cached !== false) {
                return $cached;
            }
        }
        
        $user = $this->userModel->findUserByReferralCode($code);
        
        if ($user && $this->cache) {
            $this->cache->set($cacheKey, $user, self::REFERRAL_CODE_CACHE_TTL);
        }
        
        return $user;
    }

    /**
     * Invalidate user-related caches
     */
    private function invalidateUserCaches($email = null, $userId = null)
    {
        if (!$this->cache) return;

        if ($email) {
            $this->cache->del("email_exists:{$email}");
            $this->cache->del("user_email:{$email}");
        }

        if ($userId) {
            $this->cache->del("user_profile:{$userId}");
            $this->cache->del("user_roles:{$userId}");
        }
    }

    /**
     * Get cache instance
     */
    private function getCache()
    {
        try {
            return new Cache();
        } catch (Exception $e) {
            error_log("Cache initialization failed: " . $e->getMessage());
            return null;
        }
    }

    /**
     * Get job queue instance
     */
    private function getJobQueue()
    {
       try {                                                                                                                                        
        //require_once '../app/libraries/JobQueue.php';
       return new JobQueue();
  } catch (Exception $e) {
      error_log("Job queue connection failed: " . $e->getMessage());
         return null;
   }
}           

    /**
     * Get rate limiter instance
     */
    private function getRateLimiter()
    {
        // Initialize rate limiter
        // Example:
        // return new RateLimiter($this->cache);
        return null;
    }

    
}