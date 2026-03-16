<?php
// app/commands/HandleSubscriptionExpiry.php

// Bootstrap the application
require_once dirname(__DIR__) . '/bootstrap.php';

class HandleSubscriptionExpiryCommand
{
    private $subscriptionModel;
    private $walletModel;
    private $emailService;
    private $db;

    const RENEWAL_ATTEMPTS_LIMIT = 7;

    public function __construct()
    {
        $this->db = new Database();
        $this->subscriptionModel = new Subscription();
        $this->walletModel = new Wallet();
        $this->emailService = new EmailService();
    }

    public function execute()
    {
        echo "Starting subscription expiry and renewal process...\n";

        // Get subscriptions expiring within the next day
        $expiringSubscriptions = $this->subscriptionModel->getExpiringSubscriptions(1);

        if (empty($expiringSubscriptions)) {
            echo "No subscriptions expiring soon.\n";
            return;
        }

        foreach ($expiringSubscriptions as $subscription) {
            echo "Processing subscription ID: {$subscription->subscription_id} for user {$subscription->user_id}...\n";

            // Attempt to renew
            if ($this->attemptRenewal($subscription)) {
                echo "Subscription {$subscription->subscription_id} renewed successfully.\n";
            } else {
                echo "Failed to renew subscription {$subscription->subscription_id}.\n";
                // Check if max attempts reached
                if ($subscription->renewal_attempts >= self::RENEWAL_ATTEMPTS_LIMIT - 1) { // -1 because we increment before checking
                    $this->subscriptionModel->updateSubscriptionStatus($subscription->subscription_id, 'expired');
                    echo "Subscription {$subscription->subscription_id} marked as expired after max attempts.\n";
                    // Send final expiry email
                    $this->emailService->sendSubscriptionExpiredEmail($subscription, (array)$subscription); // Assuming user data is in subscription object
                } else {
                    // Send renewal failure email
                    $this->emailService->sendSubscriptionRenewalFailedEmail($subscription, (array)$subscription); // Assuming user data is in subscription object
                }
            }
        }
        echo "Subscription expiry and renewal process complete.\n";
    }

    private function attemptRenewal($subscription)
    {
        try {
            $this->db->beginTransaction();

            // Increment renewal attempts
            $this->subscriptionModel->incrementRenewalAttempts($subscription->subscription_id);

            // Check if max attempts reached (already handled in execute, but good to re-check)
            if ($subscription->renewal_attempts >= self::RENEWAL_ATTEMPTS_LIMIT) {
                $this->db->rollBack();
                return false;
            }

            // Process payment
            if (!$this->walletModel->processWalletPayment($subscription->user_id, $subscription->amount, 'subscription_renewal', $subscription->subscription_id)) {
                $this->db->rollBack();
                return false;
            }

            // Renew subscription
            if (!$this->subscriptionModel->renewSubscription($subscription->subscription_id)) {
                $this->db->rollBack();
                return false;
            }

            $this->db->commit();
            return true;

        } catch (Exception $e) {
            if ($this->db->inTransaction()) {
                $this->db->rollBack();
            }
            error_log("Renewal attempt failed for subscription {$subscription->subscription_id}: " . $e->getMessage());
            return false;
        }
    }
}

$command = new HandleSubscriptionExpiryCommand();
$command->execute();