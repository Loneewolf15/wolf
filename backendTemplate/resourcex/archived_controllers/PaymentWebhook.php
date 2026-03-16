<?php
class PaymentWebhook extends Controller
{
    public function __construct()
    {
        $this->paymentModel = $this->model('Payment');
        $this->subscriptionModel = $this->model('Subscription');
        $this->paystackService = new PaystackService();
        $this->userModel = $this->model('User');
    }

    // Handle Paystack webhook
    public function handlePaystackWebhook()
    {
        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            try {
                // Get the payload
                $payload = file_get_contents('php://input');
                $signature = $_SERVER['HTTP_X_PAYSTACK_SIGNATURE'] ?? '';
                
                // Verify webhook signature
                if (!$this->paystackService->verifyWebhookSignature($payload, $signature)) {
                    http_response_code(400);
                    echo json_encode(['error' => 'Invalid signature']);
                    exit;
                }
                
                $event = json_decode($payload, true);
                
                if (!$event || !isset($event['event'])) {
                    http_response_code(400);
                    echo json_encode(['error' => 'Invalid payload']);
                    exit;
                }
                
                // Validate event type
                if (!$this->paystackService->validateWebhookEvent($event['event'])) {
                    http_response_code(200);
                    echo json_encode(['message' => 'Event not handled']);
                    exit;
                }
                
                // Process the event
                $result = $this->processWebhookEvent($event);
                
                if ($result) {
                    http_response_code(200);
                    echo json_encode(['message' => 'Event processed successfully']);
                } else {
                    http_response_code(500);
                    echo json_encode(['error' => 'Event processing failed']);
                }
                
            } catch (Exception $e) {
                error_log('Webhook processing error: ' . $e->getMessage());
                http_response_code(500);
                echo json_encode(['error' => 'Internal server error']);
            }
        } else {
            http_response_code(405);
            echo json_encode(['error' => 'Method not allowed']);
        }
        exit;
    }

    // Process webhook event
    private function processWebhookEvent($event)
    {
        try {
            $eventType = $event['event'];
            $data = $event['data'];
            
            switch ($eventType) {
                case 'charge.success':
                    return $this->handleChargeSuccess($data);
                    
                case 'charge.failed':
                    return $this->handleChargeFailed($data);
                    
                case 'subscription.create':
                    return $this->handleSubscriptionCreate($data);
                    
                case 'subscription.disable':
                    return $this->handleSubscriptionDisable($data);
                    
                case 'invoice.create':
                    return $this->handleInvoiceCreate($data);
                    
                case 'invoice.payment_failed':
                    return $this->handleInvoicePaymentFailed($data);
                    
                default:
                    error_log('Unhandled webhook event: ' . $eventType);
                    return true; // Return true to acknowledge receipt
            }
            
        } catch (Exception $e) {
            error_log('Error processing webhook event: ' . $e->getMessage());
            return false;
        }
    }

    // Handle successful charge
    private function handleChargeSuccess($data)
    {
        try {
            $reference = $data['reference'];
            $amount = $data['amount'] / 100; // Convert from kobo to naira
            
            // Find transaction by reference
            $transaction = $this->paymentModel->getTransactionByReference($reference);
            
            if (!$transaction) {
                error_log('Transaction not found for reference: ' . $reference);
                return false;
            }
            
            // Update transaction status
            $this->paymentModel->updateTransactionStatus(
                $transaction->transaction_id,
                'successful',
                $data
            );
            
            // Activate subscription
            $this->subscriptionModel->updateSubscriptionStatus(
                $transaction->subscription_id,
                'active'
            );
            
            // Create invoice record
            $this->createInvoiceFromTransaction($transaction, $data);
            
            // Send confirmation email
            $this->sendPaymentConfirmationEmail($transaction->user_id, $transaction, $data);
            
            return true;
            
        } catch (Exception $e) {
            error_log('Error handling charge success: ' . $e->getMessage());
            return false;
        }
    }

    // Handle failed charge
    private function handleChargeFailed($data)
    {
        try {
            $reference = $data['reference'];
            $failureReason = $data['gateway_response'] ?? 'Payment failed';
            
            // Find transaction by reference
            $transaction = $this->paymentModel->getTransactionByReference($reference);
            
            if (!$transaction) {
                error_log('Transaction not found for reference: ' . $reference);
                return false;
            }
            
            // Update transaction status
            $this->paymentModel->updateTransactionStatus(
                $transaction->transaction_id,
                'failed',
                $data,
                $failureReason
            );
            
            // Update subscription status to pending or expired
            $this->subscriptionModel->updateSubscriptionStatus(
                $transaction->subscription_id,
                'pending'
            );
            
            // Send failure notification email
            $this->sendPaymentFailureEmail($transaction->user_id, $transaction, $failureReason);
            
            return true;
            
        } catch (Exception $e) {
            error_log('Error handling charge failure: ' . $e->getMessage());
            return false;
        }
    }

    // Handle subscription creation
    private function handleSubscriptionCreate($data)
    {
        try {
            // This is handled when we create subscriptions manually
            // Log for monitoring purposes
            error_log('Subscription created on Paystack: ' . json_encode($data));
            
            return true;
            
        } catch (Exception $e) {
            error_log('Error handling subscription create: ' . $e->getMessage());
            return false;
        }
    }

    // Handle subscription disable
    private function handleSubscriptionDisable($data)
    {
        try {
            $subscription_code = $data['subscription_code'];
            
            // Find subscription by Paystack code (you'd need to store this)
            // For now, log the event
            error_log('Subscription disabled on Paystack: ' . $subscription_code);
            
            return true;
            
        } catch (Exception $e) {
            error_log('Error handling subscription disable: ' . $e->getMessage());
            return false;
        }
    }

    // Handle invoice creation
    private function handleInvoiceCreate($data)
    {
        try {
            // Log invoice creation
            error_log('Invoice created on Paystack: ' . json_encode($data));
            
            return true;
            
        } catch (Exception $e) {
            error_log('Error handling invoice create: ' . $e->getMessage());
            return false;
        }
    }

    // Handle invoice payment failure
    private function handleInvoicePaymentFailed($data)
    {
        try {
            // Handle failed invoice payment
            error_log('Invoice payment failed: ' . json_encode($data));
            
            // You might want to:
            // 1. Notify the user
            // 2. Suspend the subscription
            // 3. Set a grace period
            
            return true;
            
        } catch (Exception $e) {
            error_log('Error handling invoice payment failure: ' . $e->getMessage());
            return false;
        }
    }

    // Create invoice from successful transaction
    private function createInvoiceFromTransaction($transaction, $paymentData)
    {
        try {
            // Get subscription details
            $subscription = $this->subscriptionModel->getUserSubscription($transaction->user_id);
            
            if (!$subscription) {
                return false;
            }
            
            $invoiceData = [
                'subscription_id' => $transaction->subscription_id,
                'user_id' => $transaction->user_id,
                'plan_id' => $subscription->plan_id,
                'billing_period_start' => $subscription->start_date,
                'billing_period_end' => $subscription->end_date,
                'amount' => $transaction->amount,
                'total_amount' => $transaction->amount,
                'status' => 'paid',
                'due_date' => date('Y-m-d'),
                'invoice_data' => [
                    'payment_reference' => $transaction->payment_reference,
                    'payment_method' => $transaction->payment_method,
                    'gateway_data' => $paymentData
                ]
            ];
            
            return $this->paymentModel->createInvoice($invoiceData);
            
        } catch (Exception $e) {
            error_log('Error creating invoice: ' . $e->getMessage());
            return false;
        }
    }

    // Send payment confirmation email
    private function sendPaymentConfirmationEmail($user_id, $transaction, $paymentData)
    {
        try {
            // Get user details
            $user = $this->userModel->findUserById($user_id);

            if (!$user) {
                return false;
            }

            // Prepare payment data for email template
            $paymentDataArray = [
                'amount' => $transaction->amount,
                'currency' => 'NGN',
                'transaction_id' => $transaction->payment_reference,
                'payment_date' => date('Y-m-d H:i:s'),
                'payment_method' => 'card',
                'description' => 'Selledge Subscription Payment'
            ];

            $userDataArray = [
                'full_name' => $user->full_name,
                'email' => $user->email,
                'user_id' => $user->user_id
            ];

            // Send payment confirmation email using new email service
            $emailResult = $this->sendPaymentConfirmationEmail($paymentDataArray, $userDataArray);

            return $emailResult['status'];

        } catch (Exception $e) {
            error_log('Error sending confirmation email: ' . $e->getMessage());
            return false;
        }
    }

    // Send payment failure email
    private function sendPaymentFailureEmail($user_id, $transaction, $reason)
    {
        try {
            // Get user details
            $user = $this->userModel->findUserById($user_id);
            
            if (!$user) {
                return false;
            }
            
            // Email content
            $subject = 'Payment Failed - Selledge Subscription';
            $message = "
                <h2>Payment Failed</h2>
                <p>Dear {$user->full_name},</p>
                <p>We were unable to process your payment for your Selledge subscription.</p>
                <p><strong>Transaction Details:</strong></p>
                <ul>
                    <li>Amount: ₦" . number_format($transaction->amount, 2) . "</li>
                    <li>Reference: {$transaction->payment_reference}</li>
                    <li>Reason: {$reason}</li>
                </ul>
                <p>Please try again or contact our support team for assistance.</p>
                <p>Best regards,<br>The Selledge Team</p>
            ";
            
            // Send email (implement your email service)
            return $this->sendEmail($user->email, $subject, $message);
            
        } catch (Exception $e) {
            error_log('Error sending failure email: ' . $e->getMessage());
            return false;
        }
    }

    // Send email (placeholder - implement with your email service)
    private function sendEmail($to, $subject, $message)
    {
        try {
            // Implement your email sending logic here
            // This could use PHPMailer, SendGrid, Mailgun, etc.
            
            // For now, just log the email
            error_log("Email to {$to}: {$subject}");
            
            return true;
            
        } catch (Exception $e) {
            error_log('Error sending email: ' . $e->getMessage());
            return false;
        }
    }
}
