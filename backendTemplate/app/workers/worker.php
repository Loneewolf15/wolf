#!/usr/bin/env php
<?php
/**
 * Unified Worker Script - Processes all async jobs from JobQueue
 * 
 * Handles: Emails, SMS, Image Processing, Exports, Orders, etc.
 * 
 * Run with: php worker.php
 * Or use supervisor for production
 */

require_once __DIR__ . '/../bootstrap.php';

// Initialize services
$queue = new JobQueue();
$logger = new Logger();

echo "🚀 Unified Worker started at " . date('Y-m-d H:i:s') . "\n";
echo "Waiting for jobs...\n\n";

while (true) {
    try {
        $job = $queue->pop();

        if ($job) {
            echo "[" . date('H:i:s') . "] 📋 Processing: {$job['type']}\n";

            $data = $job['data'];

            switch ($job['type']) {
                // ===== Email Jobs =====
                case 'SendEmail':
                    $emailService = new EmailService();
                    $emailService->sendEmail(
                        $data['to'],
                        $data['subject'],
                        $data['body']
                    );
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Email sent\n";
                    break;

                case 'SendVerificationEmail':
                    $controller = new Controller();
                    $controller->sendEmailVerification($data['token'], $data);
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Verification email sent\n";
                    break;

                case 'SendWelcomeEmail':
                    $controller = new Controller();
                    $controller->sendWelcomeEmail($data);
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Welcome email sent\n";
                    break;

                case 'SendPasswordResetEmail':
                    $controller = new Controller();
                    $controller->sendPasswordResetEmail($data['token'], $data);
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Password reset email sent\n";
                    break;

                case 'SendAccountDeletionEmail':
                    $controller = new Controller();
                    $controller->sendAccountDeletionEmail($data);
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Account deletion email sent\n";
                    break;

                case 'SendRoleUpgradeEmail':
                    $controller = new Controller();
                    $controller->sendRoleUpgradeEmail($data);
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Role upgrade email sent\n";
                    break;


                // ===== Order Jobs =====
                case 'SendOrderConfirmationEmail':
                    $controller = new Controller();
                    $userModel = $controller->model('User');
                    $orderModel = $controller->model('Order');

                    $user = $userModel->findUserById($data['buyer_id']);
                    $order = $orderModel->getOrderDetails($data['order_id']);

                    if ($user && $order) {
                        $controller->sendOrderConfirmationEmail((array)$order, (array)$user);
                    }
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Order confirmation sent\n";
                    break;

                case 'SendNewOrderNotificationToSeller':
                    $controller = new Controller();
                    $userModel = $controller->model('User');
                    $orderModel = $controller->model('Order');

                    $seller = $userModel->findUserById($data['seller_id']);
                    $order = $orderModel->getOrderDetails($data['order_id']);

                    if ($seller && $order) {
                        $controller->sendSellerNewOrderEmail((array)$order, (array)$seller);
                    }
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Seller notification sent\n";
                    break;

                case 'SendOrderStatusNotification':
                    $controller = new Controller();
                    $userModel = $controller->model('User');
                    $orderModel = $controller->model('Order');

                    $user = $userModel->findUserById($data['buyer_id']);
                    $order = $orderModel->getOrderDetails($data['order_id']);

                    if ($user && $order) {
                        $order->new_status = $data['new_status'];
                        $controller->sendOrderStatusUpdateEmail((array)$order, (array)$user);
                    }
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Order status update sent\n";
                    break;

                case 'SendRefundNotification':
                    $controller = new Controller();
                    $userModel = $controller->model('User');
                    $orderModel = $controller->model('Order');

                    $user = $userModel->findUserById($data['buyer_id']);
                    $order = $orderModel->getOrderDetails($data['order_id']);

                    if ($user && $order) {
                        $order->amount = $data['amount'];
                        $controller->sendRefundNotificationEmail((array)$order, (array)$user);
                    }
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Refund notification sent\n";
                    break;

                // ===== Listing Jobs =====
                case 'SendNewListingNotification':
                    $controller = new Controller();
                    $userModel = $controller->model('User');

                    $user = $userModel->findUserById($data['user_id']);
                    if ($user) {
                        $controller->sendNewListingNotificationEmail($data, (array)$user);
                    }
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Listing notification sent\n";
                    break;

                // ===== Payment Jobs =====
                case 'ProcessPayout':
                    $payoutService = new PayoutService();
                    $payoutService->processPayout($data['order_id']);
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Payout processed\n";
                    break;

                // ===== SMS Jobs =====
                case 'SendSMS':
                    $smsService = new SMSService();
                    $smsService->send(
                        $data['phone'],
                        $data['message']
                    );
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ SMS sent\n";
                    break;

                // ===== Image Processing =====
                case 'ProcessImage':
                    $processor = new ImageProcessor();
                    $processor->resize($data['file_path'], 800, 800);
                    $processor->compress($data['file_path'], 85);
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Image processed\n";
                    break;

                // ===== File Operations =====
                case 'BulkFileCleanup':
                    $deleted = 0;
                    foreach ($data['files'] as $file) {
                        if (@unlink($file)) {
                            $deleted++;
                        }
                    }
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ Deleted {$deleted} files\n";
                    break;

                // ===== Data Export =====
                case 'ProcessExport':
                    require_once APPROOT . '/jobs/ProcessExportJob.php';
                    $exportJob = new ProcessExportJob();
                    $result = $exportJob->processExport($data['export_id']);
                    $queue->delete($job['id']);
                    echo "[" . date('H:i:s') . "] ✅ {$result}\n";
                    break;

                default:
                    echo "[" . date('H:i:s') . "] ⚠️  Unknown job type: {$job['type']}\n";
                    $queue->delete($job['id']);
            }
        } else {
            // No jobs, wait before checking again
            sleep(5);
        }
    } catch (Exception $e) {
        echo "[" . date('H:i:s') . "] ❌ Error: " . $e->getMessage() . "\n";

        // Log error
        $logger->logError('Worker error', $e, [
            'job' => $job ?? null
        ]);

        // Release job for retry (if exists)
        if (isset($job)) {
            $queue->release($job['id'], 300); // Retry in 5 minutes
            echo "[" . date('H:i:s') . "] 🔄 Job released for retry\n";
        }

        sleep(5);
    }
}
