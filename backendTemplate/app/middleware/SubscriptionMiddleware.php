<?php
class SubscriptionMiddleware
{
    private $userModel;
    private $subscriptionModel;

    public function __construct()
    {
        $this->userModel = new User();
        $this->subscriptionModel = new Subscription();
    }

    public function checkSubscription($userId, $planName)
    {
        $user = $this->userModel->findUserById($userId);
        if (!$user) {
            return ['status' => false, 'message' => 'User not found.'];
        }

        $subscription = $this->subscriptionModel->getUserSubscription($userId);

        if (!$subscription || $subscription->plan_name !== $planName || $subscription->status !== 'active') {
            return ['status' => false, 'message' => 'Active subscription for this plan not found.'];
        }

        return ['status' => true, 'message' => 'Subscription is active.'];
    }
}