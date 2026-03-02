<?php
/*
 *Base Controller
 *Loads the modals and views
*/

use PHPMailer\PHPMailer\PHPMailer;
use PHPMailer\PHPMailer\SMTP;
use PHPMailer\PHPMailer\Exception;

class Controller
{
    protected $auth_header;
    protected $userModel;
    protected $serverKey;
    protected $rateLimiter;

    public function __construct()
    {
        $this->userModel = $this->model('User');
        $this->serverKey  = 'secret_server_key';
        // Initialize rate limiter if available
        if (class_exists('RateLimiter')) {
            $this->rateLimiter = new RateLimiter();
        }
    }


    //Load model
    public function model($model)
    {
        // Instantiate model
        return new $model();
    }

    // Lets us load view from controllers
    public function view($url, $data = [])
    {
        // Check for view file
        if (file_exists(APPROOT . '/views/' . $url . '.php')) {
            // Require view file
            require_once APPROOT . '/views/' . $url . '.php';
        } else {
            // No view exists
            //die('View does not exist');
            // Redirect to error view instead of dying
            $data['error_message'] = 'View does not exist: ' . $url;
            $data['status_code'] = 404;
            $this->view('error/index', $data);
        }
    }

    public function getData()
    {
        $raw = file_get_contents('php://input');
        $data = json_decode($raw, true);
        //print_r($raw);

        if (json_encode($data) === 'null') {
            return $data =  $_POST;
        } else {
            return $data;
        }
        exit;
    }

    public function generateJWT($token, $serverKey)
    {

        return    $JWT_token = JWT::encode($token, $serverKey);
    }




    public function getAuthorizationHeader()
    {
        $headers =  null;
        if (isset($_SERVER['Authorization'])) {

            $headers = trim($_SERVER['Authorization']);
        } else if (isset($_SERVER['HTTP_ATHORIZATION'])) {
            $headers = trim($_SERVER['HTTP_ATHORIZATION']);
        } elseif (function_exists('apache_request_headers')) {
            $requestHeaders = apache_request_headers();
            $request_headers = array_combine(array_map('ucwords', array_keys($requestHeaders)), array_values($requestHeaders));
            if (isset($requestHeaders['Authorization'])) {
                $headers = trim($requestHeaders['Authorization']);
            }
        }


        return $headers;
    }

    public function bearer()
    {


        $this->auth_header  = $this->getAuthorizationHeader();


        if (
            $this->auth_header
            &&
            preg_match('#Bearer\s(\S+)#', $this->auth_header, $matches)
        ) {

            return $bearer = $matches['1'];
        }
    }




    public function myJsonID($bearer, $serverKey)
    {
        $myJsonID = JWT::decode($bearer, $serverKey);
        if ($myJsonID === 401) {
            return false;
        } else {

            return $myJsonID;
        }
    }
    public function passresetemailer($data, $sub)
    {

        $template_file = '/home/gkzxhmso/api.vplaza.com.ng/app/controllers/emailtemplates/passwordotp.php';

        $swap_arr = array(

            "CODE" => $data['otp'],

        );

        if (file_exists($template_file)) {
            $message = file_get_contents($template_file);

            foreach ($swap_arr as $key => $value) {
                if (strlen($key) > 2 && trim($key) != "" && !empty($value)) {
                    $message = str_replace("{" . $key . "}", $value, $message);
                } else {
                    $res = [
                        "status" => "false",
                        "message" => "Unable to replace placeholder: {$key}",
                    ];
                    return json_encode($res);
                    exit;
                }
            }
        } else {
            $res = [
                "status" => "false",
                "message" => "The file {$template_file} is not found",
            ];
            return json_encode($res);
            exit;
        }

        $data['r_email'] = $data['email'];
        $success = $this->sendHtmlEmailWithAttachment($data['r_email'], $sub, $message);


        // print_r(json_encode($data));
        // exit;


        if ($success) {
            //   $res = [
            //     "status" => "true",
            //     "message" => "Email sent successfully",
            //   ];
            //   return json_encode($res);
            return true;
        } else {
            return false;
        }
    }

    public function generateSixDigitValue()
    {
        $random = mt_rand(0, 999); // Generate a random number between 0 and 999
        $timeString = date('s'); // Get current seconds (or you can use 'u' for microseconds)

        // Concatenate and then take the last 6 digits to ensure it's always 6 digits
        $combined = $random . $timeString;
        return substr(str_pad($combined, 6, '0', STR_PAD_LEFT), -6);
    }

    // Send Email - Refactored to use EmailService
    public function sendEmail($to, $subject, $htmlBody, $plainTextBody = null, $attachments = [], $fromEmail = null, $fromName = null)
    {
        try {

            $emailService = new EmailService();

            return $emailService->sendEmail($to, $subject, $htmlBody, $plainTextBody, $attachments, $fromEmail, $fromName);
        } catch (Exception $e) {
            error_log('Email sending error in Controller: ' . $e->getMessage());
            return [
                'status' => false,
                'message' => 'Failed to send email: ' . $e->getMessage()
            ];
        }
    }

    // Legacy sendEmail method for backward compatibility
    // private function sendEmailLegacy($data)
    // {
    //     try {

    //         $emailService = new EmailService();

    //         $subject = 'New Contact Form Submission - ' . htmlspecialchars($data['name'] ?? 'Unknown');
    //         $htmlBody = $this->createEmailTemplate($data);
    //         $plainTextBody = $this->createPlainTextTemplate($data);

    //         // Use support email as recipient for contact forms
    //         $config = $emailService->getConfig();
    //         $result = $emailService->sendEmail(
    //             $config['support_email'],
    //             $subject,
    //             $htmlBody,
    //             $plainTextBody
    //         );

    //         return $result['status'];
    //     } catch (Exception $e) {
    //         error_log('Legacy email sending failed: ' . $e->getMessage());
    //         return false;
    //     }
    // }

    // Send Welcome Email
    public function sendWelcomeEmail($userData)
    {
        try {

            $emailService = new EmailService();

            $template = $emailService->createWelcomeEmailTemplate($userData);

            return $emailService->sendEmail(
                $userData['email'],
                'Welcome to Divine API',
                $template['html'],
                $template['text'],
                [], // No attachments
                "noreply@divineapi.com",
                "Divine API Team"
            );
        } catch (Exception $e) {
            error_log('Welcome email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send welcome email'];
        }
    }

    // Send Password Reset Email
    public function sendPasswordResetEmail($resetToken, $userData)
    {
        try {

            $emailService = new EmailService();

            $template = $emailService->createPasswordResetTemplate($resetToken, $userData);

            return $emailService->sendEmail(
                $userData['email'],
                'Reset Your Divine API Password',
                $template['html'],
                $template['text']
            );
        } catch (Exception $e) {
            error_log('Password reset email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send password reset email'];
        }
    }

    // Send Email Verification
    public function sendEmailVerification($activationToken, $userData)
    {
        try {

            $emailService = new EmailService();

            $template = $emailService->createEmailVerificationTemplate($activationToken, $userData);

            return $emailService->sendEmail(
                $userData['email'],
                'Verify Your Divine API Account',
                $template['html'],
                $template['text']
            );
        } catch (Exception $e) {
            error_log('Email verification error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send verification email'];
        }
    }

    // Send Subscription Confirmation Email
    public function sendSubscriptionConfirmationEmail($subscriptionData, $userData)
    {
        try {

            $emailService = new EmailService();

            $template = $emailService->createSubscriptionConfirmationTemplate($subscriptionData, $userData);

            return $emailService->sendEmail(
                $userData['email'],
                'Subscription Confirmed - Divine API',
                $template['html'],
                $template['text']
            );
        } catch (Exception $e) {
            error_log('Subscription confirmation email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send subscription confirmation email'];
        }
    }

    // Send Payment Confirmation Email
    public function sendPaymentConfirmationEmail($paymentData, $userData)
    {
        try {

            $emailService = new EmailService();

            $template = $emailService->createPaymentConfirmationTemplate($paymentData, $userData);

            return $emailService->sendEmail(
                $userData['email'],
                'Payment Confirmation - Divine API',
                $template['html'],
                $template['text']
            );
        } catch (Exception $e) {
            error_log('Payment confirmation email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send payment confirmation email'];
        }
    }

    // Send Account Deletion Confirmation Email
    public function sendAccountDeletionEmail($userData)
    {
        try {
            // 

            $emailService = new EmailService();

            $template = $emailService->createAccountDeletionTemplate($userData);

            return $emailService->sendEmail(
                $userData['email'],
                'Account Deletion Confirmation - Divine API',
                $template['html'],
                $template['text']
            );
        } catch (Exception $e) {
            error_log('Account deletion email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send account deletion email'];
        }
    }

    // Legacy method for backward compatibility - DEPRECATED
    private function sendHtmlEmailWithAttachment($to, $subject, $message, $attachmentPath = null)
    {
        try {

            $emailService = new EmailService();

            $attachments = [];
            if ($attachmentPath && file_exists($attachmentPath)) {
                $attachments[] = $attachmentPath;
            }

            $result = $emailService->sendEmail($to, $subject, $message, null, $attachments);

            return json_encode([
                'status' => $result['status'],
                'message' => $result['message']
            ]);
        } catch (Exception $e) {
            error_log('Legacy HTML email error: ' . $e->getMessage());
            return json_encode([
                'status' => false,
                'message' => 'Failed to send email: ' . $e->getMessage()
            ]);
        }
    }



    public function sendRoleUpgradeEmail($data)
    {
        try {

            $emailService = new EmailService();
            return $emailService->sendRoleUpgradeEmail($data, $data['role']);
        } catch (Exception $e) {
            error_log('Role upgrade email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send role upgrade email'];
        }
    }

    public function sendOrderConfirmationEmail($orderData, $userData)
    {
        try {

            $emailService = new EmailService();
            return $emailService->sendOrderConfirmationEmail($orderData, $userData);
        } catch (Exception $e) {
            error_log('Order confirmation email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send order confirmation email'];
        }
    }

    public function sendSellerNewOrderEmail($orderData, $sellerData)
    {
        try {

            $emailService = new EmailService();
            return $emailService->sendSellerNewOrderEmail($orderData, $sellerData);
        } catch (Exception $e) {
            error_log('Seller new order email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send seller new order email'];
        }
    }

    public function sendOrderStatusUpdateEmail($orderData, $userData)
    {
        try {

            $emailService = new EmailService();
            return $emailService->sendOrderStatusUpdateEmail($orderData, $userData);
        } catch (Exception $e) {
            error_log('Order status update email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send order status update email'];
        }
    }

    public function sendRefundNotificationEmail($orderData, $userData)
    {
        try {

            $emailService = new EmailService();
            return $emailService->sendRefundNotificationEmail($orderData, $userData);
        } catch (Exception $e) {
            error_log('Refund notification email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send refund notification email'];
        }
    }

    public function sendNewListingNotificationEmail($listingData, $userData)
    {
        try {

            $emailService = new EmailService();
            return $emailService->sendNewListingNotificationEmail($listingData, $userData);
        } catch (Exception $e) {
            error_log('New listing notification email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send new listing notification email'];
        }
    }


    public function serverKey()
    {
        return 'secret_server_key';
    }



    //JWT::decode($bearer,'secret_server_key'.date("H"))
    public function RouteProtection()
    {

        $headers = $this->getAuthorizationHeader();

        if (!isset($headers)) {
            throw new UnexpectedValueException('Authorization header missing');
        }

        $jwt = str_replace('Bearer ', '', $headers);

        // Decode JWT token
        $decoded = $this->myJsonID($jwt, $this->serverKey);

        if (!$decoded) {
            throw new UnexpectedValueException('Invalid token format');
        }

        // SECURITY FIX: Validate token against database
        if (!isset($decoded->user_id)) {
            throw new UnexpectedValueException('Invalid token payload');
        }

        $user_id = $decoded->user_id;

        // Get user from database
        $userModel = $this->model('User');
        $user = $userModel->findUserById($user_id);

        // Check if user exists
        if (!$user) {
            throw new UnexpectedValueException('User not found');
        }

        // CRITICAL SECURITY CHECK: Validate token against database
        if (empty($user->access_token) || $user->access_token !== $jwt) {
            throw new UnexpectedValueException('Invalid or expired token');
        }

        // Check if user account is active
        // if ($user->activation != 1) {
        //     throw new UnexpectedValueException('Account not activated');
        // }

        return $user;
    }

    //return $bearer;
    public function getuserbyid()
    {
        $bearer = $this->bearer();

        if ($bearer) {
            $userId = $this->myJsonID($bearer, $this->serverKey);
            //  print_r($userId->user_id) ;
            //  exit;
            if (!isset($userId)) {
                $response = array(

                    'status' => 'false',
                    'message' => 'Oops Something Went Wrong x get!!',

                );
                print_r(json_encode($response));
                exit;
            }
            $vb = $this->userModel->getuserbyid($userId->user_id);

            if (empty($userId->user_id)) {
                //   print_r(json_encode($vb));
                $response = array(
                    'status' => 'false',
                    'message' => 'No user with this userID!'
                );
                print_r(json_encode($response));
            } else {
                // $response = [
                //   'status' => 'true',
                //   'fullname'=> $vb->full_name,
                //   'email'=> $vb->email,
                //   'tagname'=> $vb->user_tag,
                //   'user_id'=> $vb->veluxite_id,
                // //   'data' => $vb,
                // ];
                //  print_r(json_encode(($vb)));
                return $vb;
            }
        }
    }

    public function getOptionalUser()
    {
        try {
            return $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            return null;
        }
    }

    /**
     * Apply rate limiting to current endpoint
     */
    protected function applyRateLimit($endpoint, $identifier = null)
    {
        if ($this->rateLimiter) {
            return $this->rateLimiter->apply($endpoint, $identifier);
        }
        return true;
    }

    /**
     * Check rate limit without blocking request
     */
    protected function checkRateLimit($endpoint, $identifier = null)
    {
        if ($this->rateLimiter) {
            return $this->rateLimiter->checkLimit($endpoint, $identifier);
        }
        return ['allowed' => true];
    }

    /**
     * Get rate limit status for endpoint
     */
    protected function getRateLimitStatus($endpoint, $identifier = null)
    {
        if ($this->rateLimiter) {
            return $this->rateLimiter->getStatus($endpoint, $identifier);
        }
        return null;
    }
}
