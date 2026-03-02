<?php
// app/commands/ProcessUnconfirmedPayments.php

// Bootstrap the application
require_once dirname(__DIR__) . '/bootstrap.php';

class ProcessUnconfirmedPaymentsCommand
{
    private $orderModel;
    private $payoutService;

    public function __construct()
    {
        $this->orderModel = new Order();
        $this->payoutService = new PayoutService();
    }

    public function execute()
    {
        echo "Starting automated payout process for unconfirmed orders...\n";

        $page = 1;
        $processedCount = 0;

        do {
            $result = $this->orderModel->getUnconfirmedDeliveryByBuyer(3, $page);
            $ordersToPay = $result['data'];

            if (empty($ordersToPay)) {
                break;
            }

            foreach ($ordersToPay as $order) {
                echo "Processing order ID: {$order->order_id}\n";
                
                if ($this->payoutService->processPayout($order->order_id)) {
                    echo "Order {$order->order_id} automatically marked as delivered and funds disbursed.\n";
                    $processedCount++;
                } else {
                    echo "Failed to process payout for order {$order->order_id}.\n";
                }
            }

            $page++;

        } while ($result['pagination']['has_next']);

        echo "Automated payout process complete. Processed {$processedCount} orders.\n";
    }
}

$command = new ProcessUnconfirmedPaymentsCommand();
$command->execute();
