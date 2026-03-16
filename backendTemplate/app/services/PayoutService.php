<?php
class PayoutService
{
    private $orderModel;
    private $listingModel;
    private $walletModel;
    private $userModel;
    private $db;

    public function __construct()
    {
        $this->db = new Database;
        $this->orderModel = new Order;
        $this->listingModel = new Listing;
        $this->walletModel = new Wallet;
        $this->userModel = new User;
    }

    public function processPayout($orderId)
    {
        try {
            $this->db->beginTransaction();

            $order = $this->orderModel->getOrderDetails($orderId);
            if (!$order) {
                throw new Exception("Order not found");
            }

            // Payout logic
            $totalPrice = $order->total_price;
            $platformFee = $totalPrice * 0.01;
            $sellerPayout = $totalPrice - $platformFee;

            // Process Reseller Commission
            if ($order->reseller_id) {
                $listing = $this->listingModel->getSingleListing($order->listing_id);
                if ($listing && $listing->is_resellable) {
                    $resellerCommission = $totalPrice * ($listing->reseller_commission_percent / 100);
                    $sellerPayout -= $resellerCommission;
                    $this->walletModel->creditWallet($order->reseller_id, $resellerCommission, 'reseller_commission', $order->order_id);
                }
            }

            // Process Referral Commission
            $buyer = $this->userModel->findUserById($order->buyer_id);
            if ($buyer && $buyer->referrer_id) {
                $referralCommissionRate = $this->orderModel->getReferralCommissionRate();
                $referralCommission = $platformFee * ($referralCommissionRate / 100);
                $this->walletModel->creditWallet($buyer->referrer_id, $referralCommission, 'referral_commission', $order->order_id);
            }
            
            // Finalize transaction
            $this->walletModel->creditWallet($order->seller_id, $sellerPayout, 'order_payout', $order->order_id);
            $this->orderModel->updateOrderStatus($order->order_id, 'delivered');
            $this->orderModel->updateEscrowTransactionStatus($order->order_id, 'released_to_seller');

            $this->db->commit();
            return true;

        } catch (Exception $e) {
            $this->db->rollBack();
            error_log("Payout processing failed for order {$orderId}: " . $e->getMessage());
            return false;
        }
    }
}
