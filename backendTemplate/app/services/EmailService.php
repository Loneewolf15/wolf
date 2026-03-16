<?php

// Import PHPMailer classes into the global namespace
use PHPMailer\PHPMailer\PHPMailer;
use PHPMailer\PHPMailer\SMTP;
use PHPMailer\PHPMailer\Exception;

/**
 * Unified Email Service for Divine API
 * Handles all email functionality with proper configuration management
 */
class EmailService
{
    private $config;
    private $mail;
    
    public function __construct()
    {
        global $email_config;
        $this->config = $email_config;
        
        // Initialize PHPMailer
        $this->mail = new PHPMailer(true);
        $this->configureSMTP();
    }
    
    /**
     * Configure SMTP settings from config file
     */
    private function configureSMTP()
    {
        try {
            // Server settings
            $this->mail->SMTPDebug = $this->config['smtp_debug'];
            $this->mail->isSMTP();
            $this->mail->Host = $this->config['smtp_host'];
            $this->mail->SMTPAuth = true;
            $this->mail->Username = $this->config['smtp_username'];
            $this->mail->Password = $this->config['smtp_password'];
            $this->mail->Port = $this->config['smtp_port'];
            
            // Set encryption type
            if ($this->config['smtp_secure'] === 'ssl') {
                $this->mail->SMTPSecure = PHPMailer::ENCRYPTION_SMTPS;
            } else {
                $this->mail->SMTPSecure = PHPMailer::ENCRYPTION_STARTTLS;
            }
            
            // Additional SMTP options for better compatibility
            $this->mail->SMTPOptions = array(
                'ssl' => array(
                    'verify_peer' => false,
                    'verify_peer_name' => false,
                    'allow_self_signed' => true
                )
            );
            
            // Set timeout
            $this->mail->Timeout = $this->config['timeout'];
            $this->mail->CharSet = $this->config['charset'];
            
        } catch (Exception $e) {
            error_log('SMTP Configuration Error: ' . $e->getMessage());
            throw new Exception('Email service configuration failed');
        }
    }
    
    /**
     * Send email with flexible parameters
     * 
     * @param string $to Recipient email address
     * @param string $subject Email subject
     * @param string $htmlBody HTML email body
     * @param string $plainTextBody Plain text email body (optional)
     * @param array $attachments Array of file paths to attach (optional)
     * @param string $fromEmail Custom from email (optional)
     * @param string $fromName Custom from name (optional)
     * @return array Result array with status and message
     */
    public function sendEmail($to, $subject, $htmlBody, $plainTextBody = null, $attachments = [], $fromEmail = null, $fromName = null)
    {
        try {
            // Validate email address
            if (!filter_var($to, FILTER_VALIDATE_EMAIL)) {
                return [
                    'status' => false,
                    'message' => 'Invalid recipient email address'
                ];
            }
            
            // Validate required parameters
            if (empty($subject) || empty($htmlBody)) {
                return [
                    'status' => false,
                    'message' => 'Subject and email body are required'
                ];
            }
            
            // Clear any previous recipients
            $this->mail->clearAddresses();
            $this->mail->clearAttachments();
            
            // Set sender
            $fromEmail = $fromEmail ?: $this->config['from_email'];
            $fromName = $fromName ?: $this->config['from_name'];
            $this->mail->setFrom($fromEmail, $fromName);
            
            // Set recipient
            $this->mail->addAddress($to);
              // Add CC recipients
   if (!empty($this->config['testrunner'])) {
    $this->mail->addCC($this->config['testrunner']);
}


            // Set email content
            $this->mail->isHTML(true);
            $this->mail->Subject = $subject;
            $this->mail->Body = $htmlBody;
            
            // Set plain text version if provided
            if ($plainTextBody) {
                $this->mail->AltBody = $plainTextBody;
            }
            
            // Add attachments if provided
            if (!empty($attachments)) {
                foreach ($attachments as $attachment) {
                    if (file_exists($attachment)) {
                        $this->mail->addAttachment($attachment);
                    } else {
                        error_log('Email attachment not found: ' . $attachment);
                    }
                }
            }
            
            // Send email
            if ($this->mail->send()) {
                return [
                    'status' => true,
                    'message' => 'Email sent successfully'
                ];
            } else {
                return [
                    'status' => false,
                    'message' => 'Failed to send email: ' . $this->mail->ErrorInfo
                ];
            }
            
        } catch (Exception $e) {
            error_log('Email sending error: ' . $e->getMessage());
            return [
                'status' => false,
                'message' => 'Email service error: ' . $e->getMessage()
            ];
        }
    }
    
    /**
     * Send email to multiple recipients
     * 
     * @param array $recipients Array of email addresses
     * @param string $subject Email subject
     * @param string $htmlBody HTML email body
     * @param string $plainTextBody Plain text email body (optional)
     * @return array Result array with status and message
     */
    public function sendBulkEmail($recipients, $subject, $htmlBody, $plainTextBody = null)
    {
        $results = [];
        $successCount = 0;
        $failureCount = 0;
        
        foreach ($recipients as $recipient) {
            $result = $this->sendEmail($recipient, $subject, $htmlBody, $plainTextBody);
            $results[] = [
                'email' => $recipient,
                'status' => $result['status'],
                'message' => $result['message']
            ];
            
            if ($result['status']) {
                $successCount++;
            } else {
                $failureCount++;
            }
        }
        
        return [
            'status' => $successCount > 0,
            'message' => "Sent to {$successCount} recipients, {$failureCount} failures",
            'results' => $results,
            'success_count' => $successCount,
            'failure_count' => $failureCount
        ];
    }
    
    /**
     * Get email configuration
     * 
     * @return array Email configuration array
     */
    public function getConfig()
    {
        return $this->config;
    }
    
    /**
     * Test email configuration
     * 
     * @return array Test result
     */
    public function testConfiguration()
    {
        try {
            // Test SMTP connection
            $this->mail->smtpConnect();
            $this->mail->smtpClose();
            
            return [
                'status' => true,
                'message' => 'Email configuration is valid and SMTP connection successful'
            ];
        } catch (Exception $e) {
            return [
                'status' => false,
                'message' => 'Email configuration test failed: ' . $e->getMessage()
            ];
        }
    }
    
    /**
     * Create base email template structure
     * 
     * @param string $title Email title
     * @param string $content Email content
     * @param array $data Additional template data
     * @return string HTML email template
     */
    public function createBaseTemplate($title, $content, $data = [])
{
    $baseUrl = $this->config['templates']['base_url'];
    $logoUrl = $this->config['templates']['logo_url'];
    $companyAddress = $this->config['templates']['company_address'];
    $supportPhone = $this->config['templates']['support_phone'];
    $unsubscribeUrl = $this->config['templates']['unsubscribe_url'];

    return '
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>' . htmlspecialchars($title) . '</title>
        <style>
            body {
                font-family: "Poppins", "Inter", sans-serif; /* Poppins or Inter recommended */
                line-height: 1.6;
                color: #000000; /* Black/Dark Grey */
                max-width: 600px;
                margin: 0 auto;
                padding: 0;
                background-color: #F5F5F5; /* Light Grey */
            }
            .email-container {
                background-color: #FFFFFF; /* White */
                margin: 20px auto;
                padding: 0;
                border-radius: 8px;
                box-shadow: 0 2px 10px rgba(0,0,0,0.1);
                overflow: hidden;
            }
            .header {
                background-color: #FF5E00; /* Primary Orange */
                color: #FFFFFF; /* White */
                padding: 30px;
                text-align: center;
            }
            .header h1 {
                margin: 0;
                font-size: 28px;
                font-weight: bold;
            }
            .logo {
                max-width: 150px;
                height: auto;
                margin-bottom: 20px;
            }
            .content {
                padding: 40px 30px;
            }
            .footer {
                background-color: #F5F5F5; /* Light Grey */
                padding: 20px 30px;
                text-align: center;
                font-size: 12px;
                color: #6c757d;
                border-top: 1px solid #dee2e6;
            }
            .btn {
                display: inline-block;
                padding: 12px 30px;
                background-color: #FF5E00; /* Primary Orange */
                color: #FFFFFF;
                text-decoration: none;
                border-radius: 5px;
                font-weight: bold;
                margin: 20px 0;
            }
            .btn:hover {
                opacity: 0.9;
            }
            .highlight {
                background-color: #FFF3E0; /* A lighter shade of orange for highlights */
                padding: 15px;
                border-left: 4px solid #FF5E00; /* Primary Orange */
                margin: 20px 0;
            }
            h1, h2, h3, h4 {
                color: #000000; /* Black/Dark Grey */
                font-weight: bold;
            }
            a {
                color: #FF5E00;
            }
            @media only screen and (max-width: 600px) {
                .email-container {
                    margin: 0;
                    border-radius: 0;
                }
                .content {
                    padding: 20px;
                }
                .header {
                    padding: 20px;
                }
            }
        </style>
    </head>
    <body>
        <div class="email-container">
            <div class="header">
                <img src="' . $logoUrl . '" alt="Divine API" class="logo">
                <h1>' . htmlspecialchars($title) . '</h1>
            </div>
            <div class="content">
                ' . $content . '
            </div>
            <div class="footer">
                <p><strong>Divine API</strong></p>
                <p>' . $companyAddress . '</p>
                <p>Phone: ' . $supportPhone . ' | Email: ' . $this->config['support_email'] . '</p>
                <p>
                    <a href="' . $baseUrl . '" style="color: #FF5E00;">Visit Website</a> |
                    <a href="' . $unsubscribeUrl . '" style="color: #FF5E00;">Unsubscribe</a>
                </p>
                <p style="margin-top: 15px; font-size: 11px;">
                    This email was sent from an automated system. Please do not reply directly to this email.
                </p>
            </div>
        </div>
    </body>
    </html>';
}

    /**
     * Create welcome email template for new user registration
     *
     * @param array $userData User data array
     * @return array Array with 'html' and 'text' versions
     */
   public function createWelcomeEmailTemplate($userData)
{
    $fullName = htmlspecialchars($userData['name'] ?? 'Valued Customer');
    $email = htmlspecialchars($userData['email'] ?? '');
    $accountNumber = htmlspecialchars($userData['account_number'] ?? '');
    $baseUrl = $this->config['templates']['base_url'];

    $content = '
        <h2>Welcome to Divine API!</h2>
        <p>Dear ' . $fullName . ',</p>
        <p>Thank you for joining Marketplaza, the first Street Economy App in Africa. Our mission is to empower millions of Nigerians with a structured way to hustle, earn, and grow—all from their smartphones. Your account has been successfully created. Welcome to the hustle!</p>

        <div class="highlight">
            <h3>Your Account Details:</h3>
            <ul>
                <li><strong>Name:</strong> ' . $fullName . '</li>
                <li><strong>Email:</strong> ' . $email . '</li>
                <li><strong>Wallet Number:</strong> ' . $accountNumber . '</li>
                <li><strong>Role:</strong> Buyer</li>
            </ul>
        </div>

        <h3>Get Started:</h3>
        <ul>
            <li><strong>Discover Products & Services:</strong> Find what you need from local sellers, or hire a delivery agent.</li>
            <li><strong>Start Earning:</strong> Post your own products, services, or micro jobs.</li>
            <li><strong>Become a Reseller:</strong> Make money by reselling items or referring others.</li>
        </ul>

        <a href="' . $baseUrl . '/dashboard" class="btn">Start Hustling!</a>

        <p>If you have any questions, our support team is here to help. Welcome to the Divine API community!</p>
        <p><strong>The Divine API Team</strong></p>';

    $htmlBody = $this->createBaseTemplate('Welcome to Divine API', $content);
    $textBody = "Welcome to Divine API!\n\n" .
               "Dear {$fullName},\n\n" .
               "Thank you for joining Marketplaza, the first Street Economy App in Africa. Our mission is to empower millions of Nigerians with a structured way to hustle, earn, and grow—all from their smartphones. \n\n" .
               "Your account has been successfully created. Welcome to the hustle!\n\n" .
               "Your Account Details:\n" .
               "- Name: {$fullName}\n" .
               "- Email: {$email}\n" .
               "- Wallet Number: {$accountNumber}\n" .
               "- Role: Buyer\n\n" .
               "Get Started:\n" .
               "- Discover Products & Services: Find what you need from local sellers, or hire a delivery agent.\n" .
               "- Start Earning: Post your own products, services, or micro jobs.\n" .
               "- Become a Reseller: Make money by reselling items or referring others.\n\n" .
               "Start Hustling!: {$baseUrl}/dashboard\n\n" .
               "If you have any questions, contact us at {$this->config['support_email']}\n\n" .
               "The Divine API Team";

    return [
        'html' => $htmlBody,
        'text' => $textBody
    ];
}

    /**
     * Create password reset email template
     *
     * @param string $resetToken Password reset token
     * @param array $userData User data array
     * @return array Array with 'html' and 'text' versions
     */
public function createPasswordResetTemplate($resetToken, $userData)
{
    $fullName = htmlspecialchars($userData['name'] ?? 'User');
    $baseUrl = $this->config['templates']['base_url'];
    $resetUrl = $baseUrl . '/reset-password?token=' . urlencode($resetToken);

    $content = '
        <h2>Password Reset Request</h2>
        <p>Dear ' . $fullName . ',</p>
        <p>We received a request to reset your password for your Divine API account. If you made this request, click the button below to reset your password:</p>

        <a href="' . $resetUrl . '" class="btn">Reset Your Password</a>

        <div class="highlight">
            <p><strong>Security Notice:</strong></p>
            <ul>
                <li>This link will expire in 30 minutes for security reasons.</li>
                <li>If you didn\'t request this reset, please ignore this email.</li>
                <li>Your password will remain unchanged until you create a new one.</li>
            </ul>
        </div>

        <p>If the button doesn\'t work, copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #FF5E00;">' . $resetUrl . '</p>

        <p>If you didn\'t request a password reset, please contact our support team immediately at ' . $this->config['support_email'] . '.</p>

        <p>Best regards,<br><strong>The Divine API Security Team</strong></p>';

    $htmlBody = $this->createBaseTemplate('Reset Your Password', $content);

    $textBody = "Password Reset Request\n\n" .
               "Dear {$fullName},\n\n" .
               "We received a request to reset your password for your Divine API account. " .
               "If you made this request, use the link below to reset your password:\n\n" .
               "{$resetUrl}\n\n" .
               "Security Notice:\n" .
               "- This link will expire in 30 minutes for security reasons.\n" .
               "- If you didn't request this reset, please ignore this email.\n" .
               "- Your password will remain unchanged until you create a new one.\n\n" .
               "If you didn't request a password reset, please contact our support team immediately at {$this->config['support_email']}.\n\n" .
               "Best regards,\n" .
               "The Divine API Security Team";

    return [
        'html' => $htmlBody,
        'text' => $textBody
    ];
}

    /**
     * Create email verification template
     *
     * @param string $activationToken Email verification token
     * @param array $userData User data array
     * @return array Array with 'html' and 'text' versions
     */
  public function createEmailVerificationTemplate($activationToken, $userData)
{
    $fullName = htmlspecialchars($userData['name'] ?? 'User');
    $email = htmlspecialchars($userData['email'] ?? '');
    $baseUrl = $this->config['templates']['base_url'];
    $verificationUrl = $baseUrl . '/verify-email?token=' . urlencode($activationToken);

    $content = '
        <h2>Verify Your Email Address</h2>
        <p>Dear ' . $fullName . ',</p>
        <p>Thank you for registering with Divine API! To complete your account setup and start using our platform, please verify your email address.</p>

        <div class="highlight">
            <p><strong>Email to Verify:</strong> ' . $email . '</p>
        </div>

        <a href="' . $verificationUrl . '" class="btn">Verify Email Address</a>

        <h3>Why verify your email?</h3>
        <ul>
            <li>Secure your account and protect against unauthorized access.</li>
            <li>Receive important notifications about your listings and transactions.</li>
            <li>Enable password recovery and account security features.</li>
            <li>Build trust with other users in the platform.</li>
        </ul>

        <p>If the button doesn\'t work, copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #FF5E00;">' . $verificationUrl . '</p>

        <p><strong>Note:</strong> This verification link will expire in 48 hours. If you need a new verification email, please log in to your account and request a new one.</p>

        <p>Welcome to the Divine API community!</p>
        <p><strong>The Divine API Team</strong></p>';

    $htmlBody = $this->createBaseTemplate('Verify Your Email', $content);

    $textBody = "Verify Your Email Address\n\n" .
               "Dear {$fullName},\n\n" .
               "Thank you for registering with Divine API! To complete your account setup and start using our platform, " .
               "please verify your email address.\n\n" .
               "Email to Verify: {$email}\n\n" .
               "Verification Link: {$verificationUrl}\n\n" .
               "Why verify your email?\n" .
               "- Secure your account and protect against unauthorized access.\n" .
               "- Receive important notifications about your listings and transactions.\n" .
               "- Enable password recovery and account security features.\n" .
               "- Build trust with other users in the platform.\n\n" .
               "Note: This verification link will expire in 48 hours.\n\n" .
               "Welcome to the Divine API community!\n" .
               "The Divine API Team";

    return [
        'html' => $htmlBody,
        'text' => $textBody
    ];
}

    /**
     * Create subscription confirmation email template
     *
     * @param array $subscriptionData Subscription data array
     * @param array $userData User data array
     * @return array Array with 'html' and 'text' versions
     */
    public function createSubscriptionConfirmationTemplate($subscriptionData, $userData)
    {
        $fullName = htmlspecialchars($userData['full_name'] ?? 'User');
        $planName = htmlspecialchars($subscriptionData['plan_name'] ?? 'Subscription Plan');
        $amount = number_format($subscriptionData['amount'] ?? 0, 2);
        $currency = $subscriptionData['currency'] ?? 'NGN';
        $billingCycle = ucfirst($subscriptionData['billing_cycle'] ?? 'monthly');
        $startDate = date('F j, Y', strtotime($subscriptionData['start_date'] ?? 'now'));
        $endDate = date('F j, Y', strtotime($subscriptionData['end_date'] ?? '+1 month'));
        $baseUrl = $this->config['templates']['base_url'];

        $content = '
            <h2>Subscription Confirmed!</h2>
            <p>Dear ' . $fullName . ',</p>
            <p>Great news! Your subscription to <strong>' . $planName . '</strong> has been successfully activated. You now have access to all the premium features included in your plan.</p>

            <div class="highlight">
                <h3>Subscription Details:</h3>
                <ul>
                    <li><strong>Plan:</strong> ' . $planName . '</li>
                    <li><strong>Billing:</strong> ' . $currency . ' ' . $amount . ' / ' . strtolower($billingCycle) . '</li>
                    <li><strong>Start Date:</strong> ' . $startDate . '</li>
                    <li><strong>Next Billing:</strong> ' . $endDate . '</li>
                </ul>
            </div>

            <h3>What\'s Included in Your Plan:</h3>
            <ul>
                <li>Enhanced listing visibility and priority placement</li>
                <li>Advanced analytics and performance insights</li>
                <li>Premium customer support</li>
                <li>Extended listing duration</li>
                <li>Featured listing opportunities</li>
            </ul>

            <a href="' . $baseUrl . '/dashboard" class="btn">Access Your Dashboard</a>

            <p>Your subscription will automatically renew on <strong>' . $endDate . '</strong>. You can manage your subscription, view billing history, or cancel anytime from your account dashboard.</p>

            <p>Thank you for choosing Divine API Premium!</p>
            <p><strong>The Divine API Team</strong></p>';

        $htmlBody = $this->createBaseTemplate('Subscription Confirmed', $content);

        $textBody = "Subscription Confirmed!\n\n" .
                   "Dear {$fullName},\n\n" .
                   "Great news! Your subscription to {$planName} has been successfully activated.\n\n" .
                   "Subscription Details:\n" .
                   "- Plan: {$planName}\n" .
                   "- Billing: {$currency} {$amount} / " . strtolower($billingCycle) . "\n" .
                   "- Start Date: {$startDate}\n" .
                   "- Next Billing: {$endDate}\n\n" .
                   "What's Included:\n" .
                   "- Enhanced listing visibility and priority placement\n" .
                   "- Advanced analytics and performance insights\n" .
                   "- Premium customer support\n" .
                   "- Extended listing duration\n" .
                   "- Featured listing opportunities\n\n" .
                   "Access your dashboard: {$baseUrl}/dashboard\n\n" .
                   "Your subscription will automatically renew on {$endDate}.\n\n" .
                   "Thank you for choosing Divine API Premium!\n" .
                   "The Divine API Team";

        return [
            'html' => $htmlBody,
            'text' => $textBody
        ];
    }

    /**
     * Create payment confirmation email template
     *
     * @param array $paymentData Payment data array
     * @param array $userData User data array
     * @return array Array with 'html' and 'text' versions
     */
    public function createPaymentConfirmationTemplate($paymentData, $userData)
    {
        $fullName = htmlspecialchars($userData['full_name'] ?? 'User');
        $amount = number_format($paymentData['amount'] ?? 0, 2);
        $currency = $paymentData['currency'] ?? 'NGN';
        $transactionId = htmlspecialchars($paymentData['transaction_id'] ?? 'N/A');
        $paymentDate = date('F j, Y \a\t g:i A', strtotime($paymentData['payment_date'] ?? 'now'));
        $paymentMethod = ucfirst($paymentData['payment_method'] ?? 'card');
        $description = htmlspecialchars($paymentData['description'] ?? 'Divine API Service Payment');
        $baseUrl = $this->config['templates']['base_url'];

        $content = '
            <h2>Payment Confirmation</h2>
            <p>Dear ' . $fullName . ',</p>
            <p>This email confirms that we have successfully received your payment. Thank you for your business!</p>

            <div class="highlight">
                <h3>Payment Details:</h3>
                <ul>
                    <li><strong>Amount:</strong> ' . $currency . ' ' . $amount . '</li>
                    <li><strong>Transaction ID:</strong> ' . $transactionId . '</li>
                    <li><strong>Payment Date:</strong> ' . $paymentDate . '</li>
                    <li><strong>Payment Method:</strong> ' . $paymentMethod . '</li>
                    <li><strong>Description:</strong> ' . $description . '</li>
                </ul>
            </div>

            <h3>What Happens Next?</h3>
            <ul>
                <li>Your payment has been processed successfully</li>
                <li>Services will be activated within a few minutes</li>
                <li>You will receive a separate confirmation for service activation</li>
                <li>A receipt has been generated for your records</li>
            </ul>

            <a href="' . $baseUrl . '/billing-history" class="btn">View Billing History</a>

            <p>If you have any questions about this payment or need assistance, please contact our support team at ' . $this->config['support_email'] . '.</p>

            <p>Thank you for choosing Divine API!</p>
            <p><strong>The Divine API Billing Team</strong></p>';

        $htmlBody = $this->createBaseTemplate('Payment Confirmation', $content);

        $textBody = "Payment Confirmation\n\n" .
                   "Dear {$fullName},\n\n" .
                   "This email confirms that we have successfully received your payment.\n\n" .
                   "Payment Details:\n" .
                   "- Amount: {$currency} {$amount}\n" .
                   "- Transaction ID: {$transactionId}\n" .
                   "- Payment Date: {$paymentDate}\n" .
                   "- Payment Method: {$paymentMethod}\n" .
                   "- Description: {$description}\n\n" .
                   "What Happens Next?\n" .
                   "- Your payment has been processed successfully\n" .
                   "- Services will be activated within a few minutes\n" .
                   "- You will receive a separate confirmation for service activation\n" .
                   "- A receipt has been generated for your records\n\n" .
                   "View billing history: {$baseUrl}/billing-history\n\n" .
                   "If you have questions, contact us at {$this->config['support_email']}\n\n" .
                   "Thank you for choosing Divine API!\n" .
                   "The Divine API Billing Team";

        return [
            'html' => $htmlBody,
            'text' => $textBody
        ];
    }

   /**
 * Create account deletion confirmation email template
 *
 * @param array $userData User data array
 * @return array Array with 'html' and 'text' versions
 */
public function createAccountDeletionTemplate($userData)
{
    $fullName = htmlspecialchars($userData['name'] ?? 'User');
    $email = htmlspecialchars($userData['email'] ?? '');
    $deletionDate = date('F j, Y \a\t g:i A');

    $content = '
        <h2>Account Deletion Confirmation</h2>
        <p>Dear ' . $fullName . ',</p>
        <p>This email confirms that your Divine API account has been permanently deleted as requested.</p>

        <div class="highlight">
            <h3>Deletion Details:</h3>
            <ul>
                <li><strong>Account Email:</strong> ' . $email . '</li>
                <li><strong>Deletion Date:</strong> ' . $deletionDate . '</li>
                <li><strong>Status:</strong> Permanently Deleted</li>
            </ul>
        </div>

        <h3>What Has Been Deleted:</h3>
        <ul>
            <li>Your user profile and personal information.</li>
            <li>All your product, service, and gig listings.</li>
            <li>Your wishlist and saved items.</li>
            <li>Message history and conversations.</li>
            <li>Subscription and billing information.</li>
            <li>Activity history and analytics data.</li>
        </ul>

        <h3>Important Notes:</h3>
        <ul>
            <li>This action cannot be undone.</li>
            <li>You will no longer receive emails from Divine API.</li>
            <li>Any active subscriptions have been cancelled.</li>
            <li>You can create a new account anytime using the same email.</li>
        </ul>

        <p>If you deleted your account by mistake or have any concerns, please contact our support team immediately at ' . $this->config['support_email'] . '. While we cannot restore deleted accounts, we can help you get started with a new one.</p>

        <p>Thank you for being part of the Divine API community. We\'re sorry to see you go!</p>
        <p><strong>The Divine API Team</strong></p>';

    $htmlBody = $this->createBaseTemplate('Account Deleted', $content);

    $textBody = "Account Deletion Confirmation\n\n" .
               "Dear {$fullName},\n\n" .
               "This email confirms that your Divine API account has been permanently deleted as requested.\n\n" .
               "Deletion Details:\n" .
               "- Account Email: {$email}\n" .
               "- Deletion Date: {$deletionDate}\n" .
               "- Status: Permanently Deleted\n\n" .
               "What Has Been Deleted:\n" .
               "- Your user profile and personal information.\n" .
               "- All your product, service, and gig listings.\n" .
               "- Your wishlist and saved items.\n" .
               "- Message history and conversations.\n" .
               "- Subscription and billing information.\n" .
               "- Activity history and analytics data.\n\n" .
               "Important Notes:\n" .
               "- This action cannot be undone.\n" .
               "- You will no longer receive emails from Divine API.\n" .
               "- Any active subscriptions have been cancelled.\n" .
               "- You can create a new account anytime using the same email.\n\n" .
               "If you have concerns, contact us at {$this->config['support_email']}\n\n" .
               "Thank you for being part of the Divine API community.\n" .
               "The Divine API Team";

    return [
        'html' => $htmlBody,
        'text' => $textBody
    ];
}

    /**
     * Create role upgrade email template
     *
     * @param array $userData User data array
     * @param string $newRoleName The new role name
     * @return array Array with 'html' and 'text' versions
     */
    public function createRoleUpgradeTemplate($userData, $newRoleName)
    {
        $fullName = htmlspecialchars($userData['name'] ?? 'User');
        $baseUrl = $this->config['templates']['base_url'];

        $content = '
            <h2>Account Upgraded!</h2>
            <p>Dear ' . $fullName . ',</p>
            <p>Your Divine API account has been successfully upgraded to <strong>' . htmlspecialchars($newRoleName) . '</strong>.</p>
            <p>You can now access all the features available to your new role.</p>
            <a href="' . $baseUrl . '/dashboard" class="btn">Go to Dashboard</a>
            <p>Thank you for being a part of the Divine API community!</p>
            <p><strong>The Divine API Team</strong></p>';

        $htmlBody = $this->createBaseTemplate('Account Upgraded', $content);

        $textBody = "Account Upgraded!\n\n" .
                   "Dear {$fullName},\n\n" .
                   "Your Divine API account has been successfully upgraded to " . htmlspecialchars($newRoleName) . ".\n\n" .
                   "You can now access all the features available to your new role.\n\n" .
                   "Go to your dashboard: {$baseUrl}/dashboard\n\n" .
                   "The Divine API Team";

        return [
            'html' => $htmlBody,
            'text' => $textBody
        ];
    }

    /**
     * Send role upgrade email
     *
     * @param array $userData User data array
     * @param string $newRoleName The new role name
     * @return array Result array
     */
    /**
     * Send role upgrade email
     *
     * @param array $userData User data array
     * @param string $newRoleName The new role name
     * @return array Result array
     */
    public function sendRoleUpgradeEmail($userData, $newRoleName)
    {
        try {
            $template = $this->createRoleUpgradeTemplate($userData, $newRoleName);

            return $this->sendEmail(
                $userData['email'],
                'Your Divine API Account Has Been Upgraded',
                $template['html'],
                $template['text']
            );
        } catch (Exception $e) {
            error_log('Role upgrade email error: ' . $e->getMessage());
            return ['status' => false, 'message' => 'Failed to send role upgrade email'];
        }
    }

    /**
     * Create order confirmation email template for buyer
     */
    public function createOrderConfirmationTemplate($orderData, $userData)
    {
        $fullName = htmlspecialchars($userData['name'] ?? 'Customer');
        $orderId = htmlspecialchars($orderData['order_id'] ?? 'N/A');
        $totalPrice = number_format($orderData['total_price'] ?? 0, 2);
        $currency = $orderData['currency'] ?? 'NGN';
        $baseUrl = $this->config['templates']['base_url'];

        $content = '
            <h2>Your Order is Confirmed!</h2>
            <p>Dear ' . $fullName . ',</p>
            <p>Thank you for your purchase! Your order <strong>#' . $orderId . '</strong> has been confirmed and is now being processed.</p>
            <div class="highlight">
                <h3>Order Summary:</h3>
                <ul>
                    <li><strong>Order ID:</strong> ' . $orderId . '</li>
                    <li><strong>Total Amount:</strong> ' . $currency . ' ' . $totalPrice . '</li>
                </ul>
            </div>
            <p>You will receive another email once your order status is updated.</p>
            <a href="' . $baseUrl . '/orders/' . $orderId . '" class="btn">View Your Order</a>';

        $htmlBody = $this->createBaseTemplate('Order Confirmation #' . $orderId, $content);
        $textBody = "Your Order is Confirmed!\n\nDear {$fullName},\n\nYour order #{$orderId} for {$currency} {$totalPrice} has been confirmed.";

        return ['html' => $htmlBody, 'text' => $textBody];
    }

    /**
     * Create new order notification email template for seller
     */
    public function createSellerNewOrderTemplate($orderData, $sellerData)
    {
        $sellerName = htmlspecialchars($sellerData['name'] ?? 'Seller');
        $orderId = htmlspecialchars($orderData['order_id'] ?? 'N/A');
        $totalPrice = number_format($orderData['total_price'] ?? 0, 2);
        $currency = $orderData['currency'] ?? 'NGN';
        $baseUrl = $this->config['templates']['base_url'];

        $content = '
            <h2>You Have a New Order!</h2>
            <p>Hello ' . $sellerName . ',</p>
            <p>Congratulations! You have received a new order <strong>#' . $orderId . '</strong>.</p>
            <div class="highlight">
                <h3>Order Details:</h3>
                <ul>
                    <li><strong>Order ID:</strong> ' . $orderId . '</li>
                    <li><strong>Order Value:</strong> ' . $currency . ' ' . $totalPrice . '</li>
                </ul>
            </div>
            <p>Please log in to your dashboard to view the order details and begin processing it.</p>
            <a href="' . $baseUrl . '/dashboard/orders/' . $orderId . '" class="btn">View Order</a>';

        $htmlBody = $this->createBaseTemplate('New Order Notification #' . $orderId, $content);
        $textBody = "You have a new order!\n\nHello {$sellerName},\n\nYou have a new order #{$orderId} for {$currency} {$totalPrice}.";

        return ['html' => $htmlBody, 'text' => $textBody];
    }

    /**
     * Create order status update email template
     */
    public function createOrderStatusUpdateTemplate($orderData, $userData)
    {
        $fullName = htmlspecialchars($userData['name'] ?? 'Customer');
        $orderId = htmlspecialchars($orderData['order_id'] ?? 'N/A');
        $newStatus = htmlspecialchars($orderData['new_status'] ?? 'updated');
        $baseUrl = $this->config['templates']['base_url'];

        $content = '
            <h2>Your Order Status Has Been Updated</h2>
            <p>Dear ' . $fullName . ',</p>
            <p>The status of your order <strong>#' . $orderId . '</strong> has been updated to: <strong>' . $newStatus . '</strong>.</p>
            <a href="' . $baseUrl . '/orders/' . $orderId . '" class="btn">Track Your Order</a>';

        $htmlBody = $this->createBaseTemplate('Order Status Update #' . $orderId, $content);
        $textBody = "Your order #{$orderId} has been updated to: {$newStatus}.";

        return ['html' => $htmlBody, 'text' => $textBody];
    }

    /**
     * Create refund notification email template
     */
    public function createRefundNotificationTemplate($orderData, $userData)
    {
        $fullName = htmlspecialchars($userData['name'] ?? 'Customer');
        $orderId = htmlspecialchars($orderData['order_id'] ?? 'N/A');
        $amount = number_format($orderData['amount'] ?? 0, 2);
        $currency = $orderData['currency'] ?? 'NGN';
        $baseUrl = $this->config['templates']['base_url'];

        $content = '
            <h2>Refund Processed</h2>
            <p>Dear ' . $fullName . ',</p>
            <p>A refund of <strong>' . $currency . ' ' . $amount . '</strong> for your order <strong>#' . $orderId . '</strong> has been processed.</p>
            <p>The funds should appear in your wallet or original payment method within 3-5 business days.</p>
            <a href="' . $baseUrl . '/wallet" class="btn">Check Your Wallet</a>';

        $htmlBody = $this->createBaseTemplate('Refund for Order #' . $orderId, $content);
        $textBody = "A refund of {$currency} {$amount} for order #{$orderId} has been processed.";

        return ['html' => $htmlBody, 'text' => $textBody];
    }

    public function sendOrderConfirmationEmail($orderData, $userData)
    {
        $template = $this->createOrderConfirmationTemplate($orderData, $userData);
        return $this->sendEmail($userData['email'], 'Divine API Order Confirmation #' . $orderData['order_id'], $template['html'], $template['text']);
    }

    public function sendSellerNewOrderEmail($orderData, $sellerData)
    {
        $template = $this->createSellerNewOrderTemplate($orderData, $sellerData);
        return $this->sendEmail($sellerData['email'], 'You Have a New Order on Divine API! #' . $orderData['order_id'], $template['html'], $template['text']);
    }

    public function sendOrderStatusUpdateEmail($orderData, $userData)
    {
        $template = $this->createOrderStatusUpdateTemplate($orderData, $userData);
        return $this->sendEmail($userData['email'], 'Your Divine API Order #' . $orderData['order_id'] . ' Has Been Updated', $template['html'], $template['text']);
    }

    public function sendRefundNotificationEmail($orderData, $userData)
    {
        $template = $this->createRefundNotificationTemplate($orderData, $userData);
        return $this->sendEmail($userData['email'], 'Refund Processed for Divine API Order #' . $orderData['order_id'], $template['html'], $template['text']);
    }

    /**
     * Create new listing notification email template
     */
    public function createNewListingNotificationTemplate($listingData, $userData)
    {
        $fullName = htmlspecialchars($userData['name'] ?? 'User');
        $listingTitle = htmlspecialchars($listingData['title'] ?? 'your listing');
        $listingId = htmlspecialchars($listingData['listing_id'] ?? '');
        $baseUrl = $this->config['templates']['base_url'];

        $content = '
            <h2>Your Listing is Live!</h2>
            <p>Dear ' . $fullName . ',</p>
            <p>Congratulations! Your listing, <strong>' . $listingTitle . '</strong>, has been successfully created and is now live on Divine API.</p>
            <a href="' . $baseUrl . '/listings/' . $listingId . '" class="btn">View Your Listing</a>';

        $htmlBody = $this->createBaseTemplate('Your Listing is Live!', $content);
        $textBody = "Congratulations! Your listing, '{$listingTitle}', is now live on Divine API.";

        return ['html' => $htmlBody, 'text' => $textBody];
    }

    public function sendNewListingNotificationEmail($listingData, $userData)
    {
        $template = $this->createNewListingNotificationTemplate($listingData, $userData);
        return $this->sendEmail($userData['email'], 'Your Divine API Listing is Live!', $template['html'], $template['text']);
    }
}
