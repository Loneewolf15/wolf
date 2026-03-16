<?php
class SubscriptionPlan {
    private $db;
    
    public function __construct(){
        $this->db = new Database;
    }

    // Get all active subscription plans
    public function getAllPlans($include_inactive = false) {
        try {
            $whereClause = $include_inactive ? '' : 'WHERE is_active = 1';
            
            $this->db->query("SELECT * FROM SubscriptionPlans $whereClause ORDER BY price_monthly ASC");
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return [];
        }
    }

    // Get plan by ID
    public function getPlanById($plan_id) {
        try {
            $this->db->query('SELECT * FROM SubscriptionPlans WHERE plan_id = :plan_id');
            $this->db->bind(':plan_id', $plan_id);
            
            return $this->db->single();
        } catch (PDOException $e) {
            return false;
        }
    }


     public function getPlanByIdx($plan_id) {
        try {
            $this->db->query('SELECT * FROM SubscriptionPlans WHERE plan_id = :plan_id AND is_active = 1');
            $this->db->bind(':plan_id', $plan_id);
            
            return $this->db->single();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get plans by type
    public function getPlansByType($plan_type) {
        try {
            $this->db->query('SELECT * FROM SubscriptionPlans WHERE plan_type = :plan_type AND is_active = 1 ORDER BY price_monthly ASC');
            $this->db->bind(':plan_type', $plan_type);
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return [];
        }
    }

    // Get plan features and restrictions
    public function getPlanDetails($plan_id) {
        try {
            $plan = $this->getPlanById($plan_id);
            
            if (!$plan) {
                return false;
            }
            
            // Decode JSON fields
            $plan->plan_features = $plan->plan_features ? json_decode($plan->plan_features, true) : [];
            $plan->plan_restrictions = $plan->plan_restrictions ? json_decode($plan->plan_restrictions, true) : [];
            
            return $plan;
        } catch (Exception $e) {
            return false;
        }
    }

    // Check if plan allows specific feature
    public function planAllowsFeature($plan_id, $feature) {
        try {
            $plan = $this->getPlanDetails($plan_id);
            
            if (!$plan) {
                return false;
            }
            
            // Check common features
            switch ($feature) {
                case 'featured_listings':
                    return $plan->max_featured_per_week > 0;
                case 'priority_placement':
                    return $plan->priority_placement == 1;
                case 'analytics':
                    return $plan->analytics_access !== 'none';
                case 'api_access':
                    return $plan->api_access == 1;
                case 'whatsapp_integration':
                    return $plan->whatsapp_integration == 1;
                case 'auto_renewal':
                    return $plan->auto_renewal == 1;
                case 'social_promotion':
                    return $plan->social_promotion == 1;
                case 'homepage_feature':
                    return $plan->homepage_feature == 1;
                case 'verified_badge':
                    return $plan->verified_badge == 1;
                case 'custom_branding':
                    return $plan->custom_branding == 1;
                case 'bulk_upload':
                    return $plan->bulk_upload == 1;
                case 'crm_integration':
                    return $plan->crm_integration == 1;
                case 'lead_generation':
                    return $plan->lead_generation == 1;
                default:
                    return false;
            }
        } catch (Exception $e) {
            return false;
        }
    }

    // Get plan limits
    public function getPlanLimits($plan_id) {
        try {
            $plan = $this->getPlanById($plan_id);
            
            if (!$plan) {
                return false;
            }
            
            return [
                'max_listings' => $plan->max_listings,
                'listing_duration_days' => $plan->listing_duration_days,
                'max_featured_per_week' => $plan->max_featured_per_week,
                'max_featured_per_month' => $plan->max_featured_per_month,
                'analytics_access' => $plan->analytics_access,
                'support_level' => $plan->support_level
            ];
        } catch (Exception $e) {
            return false;
        }
    }

    // Create new plan (admin function)
    public function createPlan($planData) {
        try {
            $this->db->query('INSERT INTO SubscriptionPlans (
                plan_id, plan_name, plan_type, price_monthly, price_yearly, currency,
                max_listings, listing_duration_days, max_featured_per_week, max_featured_per_month,
                priority_placement, analytics_access, profile_visibility, support_level,
                api_access, whatsapp_integration, auto_renewal, social_promotion,
                homepage_feature, verified_badge, custom_branding, bulk_upload,
                crm_integration, lead_generation, plan_features, plan_restrictions,
                is_active, created_at
            ) VALUES (
                :plan_id, :plan_name, :plan_type, :price_monthly, :price_yearly, :currency,
                :max_listings, :listing_duration_days, :max_featured_per_week, :max_featured_per_month,
                :priority_placement, :analytics_access, :profile_visibility, :support_level,
                :api_access, :whatsapp_integration, :auto_renewal, :social_promotion,
                :homepage_feature, :verified_badge, :custom_branding, :bulk_upload,
                :crm_integration, :lead_generation, :plan_features, :plan_restrictions,
                :is_active, NOW()
            )');

            // Bind all parameters
            $this->db->bind(':plan_id', $planData['plan_id']);
            $this->db->bind(':plan_name', $planData['plan_name']);
            $this->db->bind(':plan_type', $planData['plan_type']);
            $this->db->bind(':price_monthly', $planData['price_monthly']);
            $this->db->bind(':price_yearly', $planData['price_yearly'] ?? null);
            $this->db->bind(':currency', $planData['currency'] ?? 'NGN');
            $this->db->bind(':max_listings', $planData['max_listings'] ?? 1);
            $this->db->bind(':listing_duration_days', $planData['listing_duration_days'] ?? 7);
            $this->db->bind(':max_featured_per_week', $planData['max_featured_per_week'] ?? 0);
            $this->db->bind(':max_featured_per_month', $planData['max_featured_per_month'] ?? 0);
            $this->db->bind(':priority_placement', $planData['priority_placement'] ?? 0);
            $this->db->bind(':analytics_access', $planData['analytics_access'] ?? 'none');
            $this->db->bind(':profile_visibility', $planData['profile_visibility'] ?? 'basic');
            $this->db->bind(':support_level', $planData['support_level'] ?? 'community');
            $this->db->bind(':api_access', $planData['api_access'] ?? 0);
            $this->db->bind(':whatsapp_integration', $planData['whatsapp_integration'] ?? 0);
            $this->db->bind(':auto_renewal', $planData['auto_renewal'] ?? 0);
            $this->db->bind(':social_promotion', $planData['social_promotion'] ?? 0);
            $this->db->bind(':homepage_feature', $planData['homepage_feature'] ?? 0);
            $this->db->bind(':verified_badge', $planData['verified_badge'] ?? 0);
            $this->db->bind(':custom_branding', $planData['custom_branding'] ?? 0);
            $this->db->bind(':bulk_upload', $planData['bulk_upload'] ?? 0);
            $this->db->bind(':crm_integration', $planData['crm_integration'] ?? 0);
            $this->db->bind(':lead_generation', $planData['lead_generation'] ?? 0);
            $this->db->bind(':plan_features', isset($planData['plan_features']) ? json_encode($planData['plan_features']) : null);
            $this->db->bind(':plan_restrictions', isset($planData['plan_restrictions']) ? json_encode($planData['plan_restrictions']) : null);
            $this->db->bind(':is_active', $planData['is_active'] ?? 1);
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Update plan (admin function)
    public function updatePlan($plan_id, $planData) {
        try {
            $setParts = [];
            $params = [':plan_id' => $plan_id];
            
            // Build dynamic update query
            $allowedFields = [
                'plan_name', 'plan_type', 'price_monthly', 'price_yearly', 'currency',
                'max_listings', 'listing_duration_days', 'max_featured_per_week', 'max_featured_per_month',
                'priority_placement', 'analytics_access', 'profile_visibility', 'support_level',
                'api_access', 'whatsapp_integration', 'auto_renewal', 'social_promotion',
                'homepage_feature', 'verified_badge', 'custom_branding', 'bulk_upload',
                'crm_integration', 'lead_generation', 'plan_features', 'plan_restrictions', 'is_active'
            ];
            
            foreach ($allowedFields as $field) {
                if (isset($planData[$field])) {
                    $setParts[] = "$field = :$field";
                    if ($field === 'plan_features' || $field === 'plan_restrictions') {
                        $params[":$field"] = json_encode($planData[$field]);
                    } else {
                        $params[":$field"] = $planData[$field];
                    }
                }
            }
            
            if (empty($setParts)) {
                return false;
            }
            
            $setParts[] = 'updated_at = NOW()';
            $setClause = implode(', ', $setParts);
            
            $this->db->query("UPDATE SubscriptionPlans SET $setClause WHERE plan_id = :plan_id");
            
            foreach ($params as $key => $value) {
                $this->db->bind($key, $value);
            }
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Deactivate plan (admin function)
    public function deactivatePlan($plan_id) {
        try {
            $this->db->query('UPDATE SubscriptionPlans SET is_active = 0, updated_at = NOW() WHERE plan_id = :plan_id');
            $this->db->bind(':plan_id', $plan_id);
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get plan comparison data
    public function getPlanComparison($plan_ids = null) {
        try {
            if ($plan_ids) {
                $placeholders = str_repeat('?,', count($plan_ids) - 1) . '?';
                $this->db->query("SELECT * FROM SubscriptionPlans WHERE plan_id IN ($placeholders) AND is_active = 1 ORDER BY price_monthly ASC");
                
                foreach ($plan_ids as $index => $plan_id) {
                    $this->db->bind($index + 1, $plan_id);
                }
            } else {
                $this->db->query('SELECT * FROM SubscriptionPlans WHERE is_active = 1 ORDER BY price_monthly ASC');
            }
            
            $plans = $this->db->resultSet();
            
            // Process features for comparison
            foreach ($plans as &$plan) {
                $plan->plan_features = $plan->plan_features ? json_decode($plan->plan_features, true) : [];
                $plan->plan_restrictions = $plan->plan_restrictions ? json_decode($plan->plan_restrictions, true) : [];
            }
            
            return $plans;
        } catch (PDOException $e) {
            return [];
        }
    }

    // Calculate prorated amount for plan changes
    public function calculateProratedAmount($current_plan_id, $new_plan_id, $days_remaining) {
        try {
            $currentPlan = $this->getPlanById($current_plan_id);
            $newPlan = $this->getPlanById($new_plan_id);
            
            if (!$currentPlan || !$newPlan) {
                return false;
            }
            
            // Calculate daily rates
            $currentDailyRate = $currentPlan->price_monthly / 30;
            $newDailyRate = $newPlan->price_monthly / 30;
            
            // Calculate refund for unused days of current plan
            $refund = $currentDailyRate * $days_remaining;
            
            // Calculate charge for new plan
            $newCharge = $newDailyRate * $days_remaining;
            
            // Net amount (positive = charge, negative = refund)
            $netAmount = $newCharge - $refund;
            
            return [
                'current_plan_refund' => round($refund, 2),
                'new_plan_charge' => round($newCharge, 2),
                'net_amount' => round($netAmount, 2),
                'days_remaining' => $days_remaining
            ];
        } catch (Exception $e) {
            return false;
        }
    }
}
