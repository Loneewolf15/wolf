<?php
class PlanManager {
    private $db;
    private $subscriptionModel;
    private $planModel;
    private $paymentModel;
    
    public function __construct(){
        $this->db = new Database;
        $this->subscriptionModel = new Subscription();
        $this->planModel = new SubscriptionPlan();
        $this->paymentModel = new Payment();
    }

    // Upgrade user to a new plan
    public function upgradePlan($user_id, $new_plan_id, $billing_cycle = 'monthly', $payment_method = 'paystack') {
        try {
            // Get current subscription
            $currentSubscription = $this->subscriptionModel->getUserSubscription($user_id);
            $newPlan = $this->planModel->getPlanById($new_plan_id);
            
            if (!$newPlan) {
                return ['success' => false, 'message' => 'Invalid plan selected'];
            }
            
            // Calculate pricing
            $amount = $billing_cycle === 'yearly' ? $newPlan->price_yearly : $newPlan->price_monthly;
            $prorated_amount = 0;
            
            // Handle prorated billing if upgrading from existing plan
            if ($currentSubscription && $currentSubscription->status === 'active') {
                $days_remaining = $this->calculateDaysRemaining($currentSubscription->end_date);
                
                if ($days_remaining > 0) {
                    $proration = $this->planModel->calculateProratedAmount(
                        $currentSubscription->plan_id, 
                        $new_plan_id, 
                        $days_remaining
                    );
                    
                    if ($proration) {
                        $prorated_amount = $proration['net_amount'];
                        $amount = max(0, $prorated_amount); // Ensure non-negative amount
                    }
                }
            }
            
            // Create new subscription
            $start_date = date('Y-m-d H:i:s');
            $end_date = $billing_cycle === 'yearly' 
                ? date('Y-m-d H:i:s', strtotime('+1 year'))
                : date('Y-m-d H:i:s', strtotime('+1 month'));
            
            $subscriptionData = [
                'user_id' => $user_id,
                'plan_id' => $new_plan_id,
                'status' => $amount > 0 ? 'pending' : 'active',
                'billing_cycle' => $billing_cycle,
                'amount' => $amount,
                'start_date' => $start_date,
                'end_date' => $end_date,
                'next_billing_date' => $amount > 0 ? $end_date : null,
                'auto_renew' => 1
            ];
            
            $new_subscription_id = $this->subscriptionModel->createSubscription($subscriptionData);
            
            if (!$new_subscription_id) {
                return ['success' => false, 'message' => 'Failed to create subscription'];
            }
            
            // Cancel old subscription if exists
            if ($currentSubscription) {
                $this->subscriptionModel->cancelSubscription(
                    $currentSubscription->subscription_id, 
                    'Plan upgrade', 
                    true
                );
            }
            
            // Initialize plan usage
            $this->subscriptionModel->initializePlanUsage($user_id, $new_subscription_id, $new_plan_id);
            
            // Create payment transaction if amount > 0
            $payment_reference = null;
            if ($amount > 0) {
                $payment_reference = $this->paymentModel->generatePaymentReference();
                
                $transactionData = [
                    'subscription_id' => $new_subscription_id,
                    'user_id' => $user_id,
                    'payment_method' => $payment_method,
                    'payment_reference' => $payment_reference,
                    'amount' => $amount,
                    'status' => 'pending'
                ];
                
                $transaction_id = $this->paymentModel->createTransaction($transactionData);
                
                if (!$transaction_id) {
                    return ['success' => false, 'message' => 'Failed to create payment transaction'];
                }
            } else {
                // Free plan or credit available - activate immediately
                $this->subscriptionModel->updateSubscriptionStatus($new_subscription_id, 'active');
            }
            
            return [
                'success' => true,
                'message' => 'Plan upgrade initiated successfully',
                'subscription_id' => $new_subscription_id,
                'payment_reference' => $payment_reference,
                'amount' => $amount,
                'prorated_amount' => $prorated_amount,
                'requires_payment' => $amount > 0
            ];
            
        } catch (Exception $e) {
            return ['success' => false, 'message' => 'Upgrade failed: ' . $e->getMessage()];
        }
    }

    // Downgrade user to a lower plan
    public function downgradePlan($user_id, $new_plan_id, $immediate = false) {
        try {
            $currentSubscription = $this->subscriptionModel->getUserSubscription($user_id);
            $newPlan = $this->planModel->getPlanById($new_plan_id);
            
            if (!$currentSubscription || !$newPlan) {
                return ['success' => false, 'message' => 'Invalid subscription or plan'];
            }
            
            // Check if it's actually a downgrade
            $currentPlan = $this->planModel->getPlanById($currentSubscription->plan_id);
            if ($newPlan->price_monthly >= $currentPlan->price_monthly) {
                return ['success' => false, 'message' => 'This is not a downgrade'];
            }
            
            // Handle immediate vs end-of-period downgrade
            if ($immediate) {
                // Immediate downgrade - calculate refund
                $days_remaining = $this->calculateDaysRemaining($currentSubscription->end_date);
                $refund_amount = 0;
                
                if ($days_remaining > 0) {
                    $proration = $this->planModel->calculateProratedAmount(
                        $currentSubscription->plan_id, 
                        $new_plan_id, 
                        $days_remaining
                    );
                    
                    if ($proration && $proration['net_amount'] < 0) {
                        $refund_amount = abs($proration['net_amount']);
                    }
                }
                
                // Create new subscription starting now
                $start_date = date('Y-m-d H:i:s');
                $end_date = date('Y-m-d H:i:s', strtotime($currentSubscription->end_date));
                
                $subscriptionData = [
                    'user_id' => $user_id,
                    'plan_id' => $new_plan_id,
                    'status' => 'active',
                    'billing_cycle' => $currentSubscription->billing_cycle,
                    'amount' => $newPlan->price_monthly,
                    'start_date' => $start_date,
                    'end_date' => $end_date,
                    'next_billing_date' => $end_date,
                    'auto_renew' => $currentSubscription->auto_renew
                ];
                
                $new_subscription_id = $this->subscriptionModel->createSubscription($subscriptionData);
                
                if ($new_subscription_id) {
                    // Cancel current subscription
                    $this->subscriptionModel->cancelSubscription(
                        $currentSubscription->subscription_id, 
                        'Immediate downgrade', 
                        true
                    );
                    
                    // Initialize new plan usage
                    $this->subscriptionModel->initializePlanUsage($user_id, $new_subscription_id, $new_plan_id);
                    
                    // Handle listings that exceed new plan limits
                    $this->handleListingLimits($user_id, $new_plan_id);
                    
                    return [
                        'success' => true,
                        'message' => 'Plan downgraded immediately',
                        'subscription_id' => $new_subscription_id,
                        'refund_amount' => $refund_amount,
                        'effective_date' => $start_date
                    ];
                }
            } else {
                // Schedule downgrade for end of current billing period
                $scheduled_change = [
                    'user_id' => $user_id,
                    'current_subscription_id' => $currentSubscription->subscription_id,
                    'new_plan_id' => $new_plan_id,
                    'effective_date' => $currentSubscription->end_date,
                    'change_type' => 'downgrade'
                ];
                
                // Store scheduled change (you might want to create a separate table for this)
                $this->scheduleSubscriptionChange($scheduled_change);
                
                return [
                    'success' => true,
                    'message' => 'Downgrade scheduled for end of billing period',
                    'effective_date' => $currentSubscription->end_date,
                    'current_plan_expires' => $currentSubscription->end_date
                ];
            }
            
            return ['success' => false, 'message' => 'Failed to process downgrade'];
            
        } catch (Exception $e) {
            return ['success' => false, 'message' => 'Downgrade failed: ' . $e->getMessage()];
        }
    }

    // Cancel subscription
    public function cancelSubscription($user_id, $reason = null, $immediate = false) {
        try {
            $currentSubscription = $this->subscriptionModel->getUserSubscription($user_id);
            
            if (!$currentSubscription) {
                return ['success' => false, 'message' => 'No active subscription found'];
            }
            
            $success = $this->subscriptionModel->cancelSubscription(
                $currentSubscription->subscription_id, 
                $reason, 
                $immediate
            );
            
            if ($success) {
                $effective_date = $immediate ? date('Y-m-d H:i:s') : $currentSubscription->end_date;
                
                // If immediate cancellation, downgrade to free plan
                if ($immediate) {
                    $this->downgradePlan($user_id, 'starter', true);
                }
                
                return [
                    'success' => true,
                    'message' => $immediate ? 'Subscription cancelled immediately' : 'Subscription will be cancelled at end of billing period',
                    'effective_date' => $effective_date,
                    'access_until' => $immediate ? date('Y-m-d H:i:s') : $currentSubscription->end_date
                ];
            }
            
            return ['success' => false, 'message' => 'Failed to cancel subscription'];
            
        } catch (Exception $e) {
            return ['success' => false, 'message' => 'Cancellation failed: ' . $e->getMessage()];
        }
    }

    // Handle listings that exceed new plan limits
    private function handleListingLimits($user_id, $new_plan_id) {
        try {
            $newPlan = $this->planModel->getPlanById($new_plan_id);
            
            if (!$newPlan || $newPlan->max_listings == -1) {
                return; // Unlimited listings
            }
            
            // Get user's active listings
            $this->db->query("SELECT listing_id FROM listings 
                             WHERE agent_id = :user_id AND status = 'active' 
                             ORDER BY created_at DESC");
            $this->db->bind(':user_id', $user_id);
            $listings = $this->db->resultSet();
            
            // If user has more listings than allowed, deactivate oldest ones
            if (count($listings) > $newPlan->max_listings) {
                $excess_listings = array_slice($listings, $newPlan->max_listings);
                
                foreach ($excess_listings as $listing) {
                    $this->db->query("UPDATE listings SET status = 'inactive', 
                                     updated_at = NOW() 
                                     WHERE listing_id = :listing_id");
                    $this->db->bind(':listing_id', $listing->listing_id);
                    $this->db->execute();
                }
            }
            
        } catch (Exception $e) {
            // Log error but don't fail the downgrade
            error_log("Failed to handle listing limits: " . $e->getMessage());
        }
    }

    // Schedule subscription change
    private function scheduleSubscriptionChange($changeData) {
        try {
            // This could be stored in a separate table or as part of subscription data
            // For now, we'll add it to the subscription record as JSON
            $this->db->query("UPDATE user_subscriptions SET 
                             updated_at = NOW()
                             WHERE subscription_id = :subscription_id");
            $this->db->bind(':subscription_id', $changeData['current_subscription_id']);
            
            return $this->db->execute();
        } catch (Exception $e) {
            return false;
        }
    }

    // Calculate days remaining in subscription
    private function calculateDaysRemaining($end_date) {
        $end = new DateTime($end_date);
        $now = new DateTime();
        
        if ($end <= $now) {
            return 0;
        }
        
        $diff = $now->diff($end);
        return $diff->days;
    }

    // Process scheduled subscription changes
    public function processScheduledChanges() {
        try {
            // Get subscriptions that have ended and need processing
            $this->db->query("SELECT * FROM user_subscriptions 
                             WHERE status = 'active' 
                             AND end_date <= NOW() 
                             AND auto_renew = 0");
            
            $expiredSubscriptions = $this->db->resultSet();
            
            foreach ($expiredSubscriptions as $subscription) {
                // Check if there's a scheduled downgrade
                // For now, just expire the subscription
                $this->subscriptionModel->updateSubscriptionStatus($subscription->subscription_id, 'expired');
                
                // Downgrade to free plan
                $this->downgradePlan($subscription->user_id, 'starter', true);
            }
            
            return count($expiredSubscriptions);
        } catch (Exception $e) {
            return false;
        }
    }

    // Get plan upgrade recommendations
    public function getUpgradeRecommendations($user_id) {
        try {
            $currentSubscription = $this->subscriptionModel->getUserSubscription($user_id);
            $usage = $this->subscriptionModel->getUserPlanUsage($user_id);
            
            if (!$currentSubscription) {
                return [];
            }
            
            $recommendations = [];
            
            // Check if user is hitting limits
            if (isset($usage['listings_created']) && 
                $usage['listings_created']['percentage_used'] >= 80) {
                $recommendations[] = [
                    'reason' => 'listing_limit',
                    'message' => 'You\'re approaching your listing limit. Consider upgrading for more listings.',
                    'suggested_plans' => $this->getSuggestedPlans($currentSubscription->plan_id, 'more_listings')
                ];
            }
            
            if (isset($usage['featured_used']) && 
                $usage['featured_used']['percentage_used'] >= 80) {
                $recommendations[] = [
                    'reason' => 'featured_limit',
                    'message' => 'You\'re running low on featured listings. Upgrade for more featured placements.',
                    'suggested_plans' => $this->getSuggestedPlans($currentSubscription->plan_id, 'more_featured')
                ];
            }
            
            return $recommendations;
        } catch (Exception $e) {
            return [];
        }
    }

    // Get suggested plans based on current plan and needs
    private function getSuggestedPlans($current_plan_id, $need) {
        try {
            $currentPlan = $this->planModel->getPlanById($current_plan_id);
            
            if (!$currentPlan) {
                return [];
            }
            
            // Get plans with higher limits
            $this->db->query("SELECT * FROM subscription_plans 
                             WHERE price_monthly > :current_price 
                             AND is_active = 1 
                             ORDER BY price_monthly ASC 
                             LIMIT 3");
            $this->db->bind(':current_price', $currentPlan->price_monthly);
            
            return $this->db->resultSet();
        } catch (Exception $e) {
            return [];
        }
    }
}
