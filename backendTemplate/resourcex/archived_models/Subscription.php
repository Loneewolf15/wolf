<?php
class Subscription {
    private $db;
    
    public function __construct(){
        $this->db = new Database;
    }

    // Get user's current subscription
    public function getUserSubscription($user_id) {
        try {
            $this->db->query("SELECT us.*, sp.plan_name, sp.plan_type, sp.max_listings, sp.listing_duration_days,
                             sp.max_featured_per_week, sp.max_featured_per_month, sp.analytics_access,
                             sp.support_level, sp.plan_features, sp.plan_restrictions
                             FROM UserSubscriptions us
                             INNER JOIN subscription_plans sp ON us.plan_id = sp.plan_id
                             WHERE us.user_id = :user_id AND us.status IN ('active', 'pending')
                             ORDER BY us.created_at DESC LIMIT 1");
            
            $this->db->bind(':user_id', $user_id);
            $subscription = $this->db->single();
            
            if ($subscription) {
                // Decode JSON fields
                $subscription->plan_features = $subscription->plan_features ? json_decode($subscription->plan_features, true) : [];
                $subscription->plan_restrictions = $subscription->plan_restrictions ? json_decode($subscription->plan_restrictions, true) : [];
            }
            
            return $subscription;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Create new subscription
    public function createSubscription($subscriptionData) {
        try {
            $subscription_id = "sub_" . md5($subscriptionData['user_id'] . time() . rand(1000, 9999));
            
            $this->db->query('INSERT INTO UserSubscriptions (
                subscription_id, user_id, plan_id, status, billing_cycle, amount, currency,
                start_date, end_date, next_billing_date, auto_renew, trial_end_date, created_at
            ) VALUES (
                :subscription_id, :user_id, :plan_id, :status, :billing_cycle, :amount, :currency,
                :start_date, :end_date, :next_billing_date, :auto_renew, :trial_end_date, NOW()
            )');

            $this->db->bind(':subscription_id', $subscription_id);
            $this->db->bind(':user_id', $subscriptionData['user_id']);
            $this->db->bind(':plan_id', $subscriptionData['plan_id']);
            $this->db->bind(':status', $subscriptionData['status'] ?? 'pending');
            $this->db->bind(':billing_cycle', $subscriptionData['billing_cycle'] ?? 'monthly');
            $this->db->bind(':amount', $subscriptionData['amount']);
            $this->db->bind(':currency', $subscriptionData['currency'] ?? 'NGN');
            $this->db->bind(':start_date', $subscriptionData['start_date']);
            $this->db->bind(':end_date', $subscriptionData['end_date']);
            $this->db->bind(':next_billing_date', $subscriptionData['next_billing_date'] ?? null);
            $this->db->bind(':auto_renew', $subscriptionData['auto_renew'] ?? 1);
            $this->db->bind(':trial_end_date', $subscriptionData['trial_end_date'] ?? null);
            
            if ($this->db->execute()) {
                return $subscription_id;
            }
            
            return false;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Update subscription status
    public function updateSubscriptionStatus($subscription_id, $status, $additional_data = []) {
        try {
            $setParts = ['status = :status', 'updated_at = NOW()'];
            $params = [':subscription_id' => $subscription_id, ':status' => $status];
            
            // Handle additional data
            if (isset($additional_data['cancelled_at'])) {
                $setParts[] = 'cancelled_at = :cancelled_at';
                $params[':cancelled_at'] = $additional_data['cancelled_at'];
            }
            
            if (isset($additional_data['cancellation_reason'])) {
                $setParts[] = 'cancellation_reason = :cancellation_reason';
                $params[':cancellation_reason'] = $additional_data['cancellation_reason'];
            }
            
            if (isset($additional_data['grace_period_end'])) {
                $setParts[] = 'grace_period_end = :grace_period_end';
                $params[':grace_period_end'] = $additional_data['grace_period_end'];
            }
            
            if (isset($additional_data['next_billing_date'])) {
                $setParts[] = 'next_billing_date = :next_billing_date';
                $params[':next_billing_date'] = $additional_data['next_billing_date'];
            }
            
            $setClause = implode(', ', $setParts);
            
            $this->db->query("UPDATE UserSubscriptions SET $setClause WHERE subscription_id = :subscription_id");
            
            foreach ($params as $key => $value) {
                $this->db->bind($key, $value);
            }
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Cancel subscription
    public function cancelSubscription($subscription_id, $reason = null, $immediate = false) {
        try {
            $status = $immediate ? 'cancelled' : 'active'; // Keep active until end of billing period
            $cancelled_at = date('Y-m-d H:i:s');
            
            $additional_data = [
                'cancelled_at' => $cancelled_at,
                'cancellation_reason' => $reason
            ];
            
            // If immediate cancellation, set grace period
            if ($immediate) {
                $additional_data['grace_period_end'] = date('Y-m-d H:i:s', strtotime('+7 days'));
            }
            
            return $this->updateSubscriptionStatus($subscription_id, $status, $additional_data);
        } catch (Exception $e) {
            return false;
        }
    }

    // Upgrade/downgrade subscription
    public function changeSubscriptionPlan($user_id, $new_plan_id, $billing_cycle = 'monthly') {
        try {
            // Get current subscription
            $currentSubscription = $this->getUserSubscription($user_id);
            
            if (!$currentSubscription) {
                return false;
            }
            
            // Cancel current subscription
            $this->updateSubscriptionStatus($currentSubscription->subscription_id, 'cancelled', [
                'cancelled_at' => date('Y-m-d H:i:s'),
                'cancellation_reason' => 'Plan change'
            ]);
            
            // Get new plan details
            $planModel = new SubscriptionPlan();
            $newPlan = $planModel->getPlanById($new_plan_id);
            
            if (!$newPlan) {
                return false;
            }
            
            // Calculate new subscription dates
            $start_date = date('Y-m-d H:i:s');
            $end_date = $billing_cycle === 'yearly' 
                ? date('Y-m-d H:i:s', strtotime('+1 year'))
                : date('Y-m-d H:i:s', strtotime('+1 month'));
            
            $next_billing_date = $newPlan->price_monthly > 0 ? $end_date : null;
            $amount = $billing_cycle === 'yearly' ? $newPlan->price_yearly : $newPlan->price_monthly;
            
            // Create new subscription
            $newSubscriptionData = [
                'user_id' => $user_id,
                'plan_id' => $new_plan_id,
                'status' => 'pending',
                'billing_cycle' => $billing_cycle,
                'amount' => $amount,
                'start_date' => $start_date,
                'end_date' => $end_date,
                'next_billing_date' => $next_billing_date,
                'auto_renew' => 1
            ];
            
            $new_subscription_id = $this->createSubscription($newSubscriptionData);
            
            if ($new_subscription_id) {
                // Initialize plan usage
                $this->initializePlanUsage($user_id, $new_subscription_id, $new_plan_id);
                
                return [
                    'subscription_id' => $new_subscription_id,
                    'plan_change' => true,
                    'old_plan' => $currentSubscription->plan_id,
                    'new_plan' => $new_plan_id
                ];
            }
            
            return false;
        } catch (Exception $e) {
            return false;
        }
    }

    // Initialize plan usage tracking
    public function initializePlanUsage($user_id, $subscription_id, $plan_id) {
        try {
            $planModel = new SubscriptionPlan();
            $plan = $planModel->getPlanById($plan_id);
            
            if (!$plan) {
                return false;
            }
            
            $reset_date = date('Y-m-d', strtotime('+1 month'));
            
            // Initialize usage tracking
            $usageTypes = [
                'listings_created' => $plan->max_listings,
                'featured_used' => $plan->max_featured_per_month
            ];
            
            foreach ($usageTypes as $usage_type => $limit) {
                $this->db->query('INSERT INTO plan_usage (
                    user_id, subscription_id, usage_type, current_usage, usage_limit, reset_date
                ) VALUES (
                    :user_id, :subscription_id, :usage_type, 0, :usage_limit, :reset_date
                ) ON DUPLICATE KEY UPDATE
                    usage_limit = :usage_limit, reset_date = :reset_date');
                
                $this->db->bind(':user_id', $user_id);
                $this->db->bind(':subscription_id', $subscription_id);
                $this->db->bind(':usage_type', $usage_type);
                $this->db->bind(':usage_limit', $limit);
                $this->db->bind(':reset_date', $reset_date);
                
                $this->db->execute();
            }
            
            return true;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get user's plan usage
    public function getUserPlanUsage($user_id) {
        try {
            $this->db->query("SELECT * FROM plan_usage WHERE user_id = :user_id ORDER BY usage_type");
            $this->db->bind(':user_id', $user_id);
            
            $usage = $this->db->resultSet();
            
            // Convert to associative array
            $usageData = [];
            foreach ($usage as $item) {
                $usageData[$item->usage_type] = [
                    'current_usage' => $item->current_usage,
                    'usage_limit' => $item->usage_limit,
                    'reset_date' => $item->reset_date,
                    'percentage_used' => $item->usage_limit > 0 ? round(($item->current_usage / $item->usage_limit) * 100, 2) : 0
                ];
            }
            
            return $usageData;
        } catch (PDOException $e) {
            return [];
        }
    }

    // Update plan usage
    public function updatePlanUsage($user_id, $usage_type, $increment = 1) {
        try {
            $this->db->query('UPDATE plan_usage SET 
                             current_usage = current_usage + :increment,
                             updated_at = NOW()
                             WHERE user_id = :user_id AND usage_type = :usage_type');
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':usage_type', $usage_type);
            $this->db->bind(':increment', $increment);
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Check if user can perform action based on plan limits
    public function canPerformAction($user_id, $action) {
        try {
            $subscription = $this->getUserSubscription($user_id);
            $usage = $this->getUserPlanUsage($user_id);
            
            if (!$subscription) {
                return false;
            }
            
            switch ($action) {
                case 'create_listing':
                    if ($subscription->max_listings == -1) return true; // Unlimited
                    return isset($usage['listings_created']) && 
                           $usage['listings_created']['current_usage'] < $usage['listings_created']['usage_limit'];
                
                case 'feature_listing':
                    if ($subscription->max_featured_per_month == -1) return true; // Unlimited
                    return isset($usage['featured_used']) && 
                           $usage['featured_used']['current_usage'] < $usage['featured_used']['usage_limit'];
                
                case 'api_access':
                    return $subscription->plan_restrictions && 
                           !isset($subscription->plan_restrictions['no_api_access']);
                
                default:
                    return false;
            }
        } catch (Exception $e) {
            return false;
        }
    }

    // Get subscriptions expiring soon
    public function getExpiringSubscriptions($days = 7) {
        try {
            $this->db->query("SELECT us.*, u.email, u.full_name, sp.plan_name
                             FROM UserSubscriptions us
                             INNER JOIN initkey_rid u ON us.user_id = u.user_id
                             INNER JOIN subscription_plans sp ON us.plan_id = sp.plan_id
                             WHERE us.status = 'active' 
                             AND us.end_date BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL :days DAY)
                             ORDER BY us.end_date ASC");
            
            $this->db->bind(':days', $days);
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return [];
        }
    }

    // Renew subscription
    public function renewSubscription($subscription_id) {
        try {
            // Get subscription details
            $this->db->query("SELECT us.*, sp.price_monthly, sp.price_yearly
                             FROM UserSubscriptions us
                             INNER JOIN subscription_plans sp ON us.plan_id = sp.plan_id
                             WHERE us.subscription_id = :subscription_id");
            
            $this->db->bind(':subscription_id', $subscription_id);
            $subscription = $this->db->single();
            
            if (!$subscription) {
                return false;
            }
            
            // Calculate new dates
            $new_start = $subscription->end_date;
            $new_end = $subscription->billing_cycle === 'yearly' 
                ? date('Y-m-d H:i:s', strtotime($new_start . ' +1 year'))
                : date('Y-m-d H:i:s', strtotime($new_start . ' +1 month'));
            
            $amount = $subscription->billing_cycle === 'yearly' ? $subscription->price_yearly : $subscription->price_monthly;
            
            // Update subscription
            $this->db->query('UPDATE UserSubscriptions SET 
                             start_date = :start_date,
                             end_date = :end_date,
                             next_billing_date = :next_billing_date,
                             amount = :amount,
                             status = :status,
                             updated_at = NOW()
                             WHERE subscription_id = :subscription_id');
            
            $this->db->bind(':subscription_id', $subscription_id);
            $this->db->bind(':start_date', $new_start);
            $this->db->bind(':end_date', $new_end);
            $this->db->bind(':next_billing_date', $new_end);
            $this->db->bind(':amount', $amount);
            $this->db->bind(':status', 'active');
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    public function incrementRenewalAttempts($subscriptionId)
    {
        try {
            $this->db->query('UPDATE UserSubscriptions SET renewal_attempts = renewal_attempts + 1 WHERE subscription_id = :subscription_id');
            $this->db->bind(':subscription_id', $subscriptionId);
            return $this->db->execute();
        } catch (PDOException $e) {
            error_log("Increment renewal attempts failed for subscription {$subscriptionId}: " . $e->getMessage());
            return false;
        }
    }
}
