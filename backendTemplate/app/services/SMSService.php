<?php

/**
 * SMS Service
 * 
 * Send SMS via Twilio or Africa's Talking
 */
class SMSService
{
    private $config;
    private $provider;
    private $db;

    public function __construct()
    {
        $this->loadConfig();
        $this->provider = $this->config['provider'];
        $this->db = new Database();
    }

    /**
     * Load SMS configuration
     */
    private function loadConfig(): void
    {
        $configPath = APPROOT . '/config/sms_config.php';
        $this->config = file_exists($configPath) ? require $configPath : [];
    }

    /**
     * Send SMS
     */
    public function send(string $to, string $message): array
    {
        try {
            // Check rate limit
            if (!$this->checkRateLimit($to)) {
                return [
                    'success' => false,
                    'error' => 'SMS rate limit exceeded'
                ];
            }

            // Send via provider
            $result = match ($this->provider) {
                'twilio' => $this->sendViaTwilio($to, $message),
                'africas_talking' => $this->sendViaAfricasTalking($to, $message),
                default => ['success' => false, 'error' => 'Invalid SMS provider']
            };

            // Log SMS
            $this->logSMS($to, $message, $result['success']);

            return $result;
        } catch (Exception $e) {
            error_log('SMS error: ' . $e->getMessage());
            return [
                'success' => false,
                'error' => $e->getMessage()
            ];
        }
    }

    /**
     * Send OTP code
     */
    public function sendOTP(string $phone, string $purpose = 'verification'): array
    {
        // Generate OTP
        $code = $this->generateOTP();

        // Store OTP in database
        $this->storeOTP($phone, $code, $purpose);

        // Send SMS
        $message = $this->getTemplate('otp', [
            'app_name' => 'Divine API',
            'code' => $code,
            'minutes' => $this->config['otp']['expiry_minutes']
        ]);

        return $this->send($phone, $message);
    }

    /**
     * Verify OTP code
     */
    public function verifyOTP(string $phone, string $code, string $purpose = 'verification'): bool
    {
        $this->db->query("SELECT * FROM otp_codes 
            WHERE phone = :phone 
            AND code = :code 
            AND purpose = :purpose 
            AND expires_at > NOW() 
            AND verified = 0
            LIMIT 1");

        $this->db->bind(':phone', $phone);
        $this->db->bind(':code', $code);
        $this->db->bind(':purpose', $purpose);

        $otp = $this->db->single();

        if ($otp) {
            // Check attempts
            if ($otp->attempts >= $this->config['otp']['max_attempts']) {
                return false;
            }

            // Mark as verified
            $this->db->query("UPDATE otp_codes SET verified = 1 WHERE id = :id");
            $this->db->bind(':id', $otp->id);
            $this->db->execute();

            return true;
        }

        // Increment attempts
        if ($otp) {
            $this->db->query("UPDATE otp_codes SET attempts = attempts + 1 WHERE id = :id");
            $this->db->bind(':id', $otp->id);
            $this->db->execute();
        }

        return false;
    }

    /**
     * Send via Twilio
     */
    private function sendViaTwilio(string $to, string $message): array
    {
        $credentials = $this->config['providers']['twilio'];

        $url = "https://api.twilio.com/2010-04-01/Accounts/{$credentials['account_sid']}/Messages.json";

        $data = [
            'From' => $credentials['from_number'],
            'To' => $to,
            'Body' => $message
        ];

        $ch = curl_init($url);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_POST, true);
        curl_setopt($ch, CURLOPT_POSTFIELDS, http_build_query($data));
        curl_setopt($ch, CURLOPT_USERPWD, $credentials['account_sid'] . ':' . $credentials['auth_token']);

        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        if ($httpCode === 201) {
            $result = json_decode($response, true);
            return [
                'success' => true,
                'message_id' => $result['sid'] ?? null
            ];
        }

        return [
            'success' => false,
            'error' => 'Twilio API error: ' . $httpCode
        ];
    }

    /**
     * Send via Africa's Talking
     */
    private function sendViaAfricasTalking(string $to, string $message): array
    {
        $credentials = $this->config['providers']['africas_talking'];

        $url = 'https://api.africastalking.com/version1/messaging';

        $data = [
            'username' => $credentials['username'],
            'to' => $to,
            'message' => $message,
            'from' => $credentials['from']
        ];

        $ch = curl_init($url);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_POST, true);
        curl_setopt($ch, CURLOPT_POSTFIELDS, http_build_query($data));
        curl_setopt($ch, CURLOPT_HTTPHEADER, [
            'apiKey: ' . $credentials['api_key'],
            'Accept: application/json'
        ]);

        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        if ($httpCode === 201) {
            return ['success' => true];
        }

        return [
            'success' => false,
            'error' => 'Africa\'s Talking API error: ' . $httpCode
        ];
    }

    /**
     * Generate OTP code
     */
    private function generateOTP(): string
    {
        $length = $this->config['otp']['length'];
        $min = pow(10, $length - 1);
        $max = pow(10, $length) - 1;

        return (string)random_int($min, $max);
    }

    /**
     * Store OTP in database
     */
    private function storeOTP(string $phone, string $code, string $purpose): void
    {
        $expiryMinutes = $this->config['otp']['expiry_minutes'];

        $this->db->query("INSERT INTO otp_codes (phone, code, purpose, expires_at) 
            VALUES (:phone, :code, :purpose, DATE_ADD(NOW(), INTERVAL :minutes MINUTE))");

        $this->db->bind(':phone', $phone);
        $this->db->bind(':code', $code);
        $this->db->bind(':purpose', $purpose);
        $this->db->bind(':minutes', $expiryMinutes);

        $this->db->execute();
    }

    /**
     * Get message template
     */
    private function getTemplate(string $name, array $vars = []): string
    {
        $template = $this->config['templates'][$name] ?? '{code}';

        foreach ($vars as $key => $value) {
            $template = str_replace('{' . $key . '}', $value, $template);
        }

        return $template;
    }

    /**
     * Check rate limit
     */
    private function checkRateLimit(string $phone): bool
    {
        // Check hourly limit
        $this->db->query("SELECT COUNT(*) as count FROM sms_logs 
            WHERE phone = :phone 
            AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)");
        $this->db->bind(':phone', $phone);
        $hourly = $this->db->single();

        if ($hourly->count >= $this->config['rate_limits']['per_user_per_hour']) {
            return false;
        }

        // Check daily limit
        $this->db->query("SELECT COUNT(*) as count FROM sms_logs 
            WHERE phone = :phone 
            AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)");
        $this->db->bind(':phone', $phone);
        $daily = $this->db->single();

        return $daily->count < $this->config['rate_limits']['per_user_per_day'];
    }

    /**
     * Log SMS
     */
    private function logSMS(string $to, string $message, bool $success): void
    {
        $this->db->query("INSERT INTO sms_logs (phone, message, success, provider) 
            VALUES (:phone, :message, :success, :provider)");

        $this->db->bind(':phone', $to);
        $this->db->bind(':message', $message);
        $this->db->bind(':success', $success ? 1 : 0);
        $this->db->bind(':provider', $this->provider);

        $this->db->execute();
    }
}
