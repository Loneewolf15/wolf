<?php
// Load the application environment
require_once __DIR__ . '/../bootstrap.php';

class EmailWorker extends Controller
{
    private $queue;
    protected $orderModel;

    public function __construct()
    {
        parent::__construct();
        $this->orderModel = $this->model('Order');
        $this->queue = new JobQueue();
    }

    public function process()
    {
        echo "Email worker started. Waiting for jobs...\n";
        while (true) {
            $job = $this->queue->pop();

            if ($job) {
                try {
                    echo "[" . date('Y-m-d H:i:s') . "] Processing job: {$job['id']} - Type: {$job['type']}\n";
                    $this->handleJob($job);
                    $this->queue->delete($job['id']);
                    echo "[" . date('Y-m-d H:i:s') . "] Processed and deleted job: {$job['id']}\n";
                } catch (Exception $e) {
                    error_log("Job failed: {$job['id']} - " . $e->getMessage());
                    $this->queue->release($job['id']);
                    error_log("Job released back to queue: {$job['id']}");
                }
            } else {
                sleep(5);
            }
        }
    }

    private function handleJob($job)
    {
        $data = $job['data'];
        switch ($job['type']) {
            case 'SendVerificationEmail':
                $this->sendEmailVerification($data['token'], $data);
                break;
            case 'SendWelcomeEmail':
                $this->sendWelcomeEmail($data);
                break;
            case 'SendPasswordResetEmail':
                $this->sendPasswordResetEmail($data['token'], $data);
                break;
            case 'SendAccountDeletionEmail':
                $this->sendAccountDeletionEmail($data);
                break;
            case 'SendRoleUpgradeEmail':
                $this->sendRoleUpgradeEmail($data);
                break;
            
            // Order-related jobs
            case 'SendOrderConfirmationEmail':
                $user = $this->userModel->findUserById($data['buyer_id']);
                $order = $this->orderModel->getOrderDetails($data['order_id']);
                if ($user && $order) {
                    $this->sendOrderConfirmationEmail((array)$order, (array)$user);
                }
                break;
            case 'SendNewOrderNotificationToSeller':
                $seller = $this->userModel->findUserById($data['seller_id']);
                $order = $this->orderModel->getOrderDetails($data['order_id']);
                if ($seller && $order) {
                    $this->sendSellerNewOrderEmail((array)$order, (array)$seller);
                }
                break;
            case 'SendOrderStatusNotification':
                $user = $this->userModel->findUserById($data['buyer_id']);
                $order = $this->orderModel->getOrderDetails($data['order_id']);
                if ($user && $order) {
                    $order->new_status = $data['new_status'];
                    $this->sendOrderStatusUpdateEmail((array)$order, (array)$user);
                }
                break;
            case 'SendRefundNotification':
                $user = $this->userModel->findUserById($data['buyer_id']);
                $order = $this->orderModel->getOrderDetails($data['order_id']);
                if ($user && $order) {
                    $order->amount = $data['amount'];
                    $this->sendRefundNotificationEmail((array)$order, (array)$user);
                }
                break;

            // Listing-related jobs
            case 'SendNewListingNotification':
                $user = $this->userModel->findUserById($data['user_id']);
                if ($user) {
                    $this->sendNewListingNotificationEmail($data, (array)$user);
                }
                break;

            case 'ProcessPayout':
                $payoutService = new PayoutService();
                $payoutService->processPayout($data['order_id']);
                break;

            default:
                error_log("Unknown job type: {$job['type']}");
        }
    }
}

// Run the worker
$worker = new EmailWorker();
$worker->process();
