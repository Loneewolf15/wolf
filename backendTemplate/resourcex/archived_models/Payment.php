<?php
class Payment {
    private $db;
    
    public function __construct(){
        $this->db = new Database;
    }

    // Create payment transaction
    public function createTransaction($transactionData) {
        try {
            $transaction_id = "txn_" . md5($transactionData['user_id'] . time() . rand(1000, 9999));
            
            $this->db->query('INSERT INTO payment_transactions (
                transaction_id, subscription_id, user_id, payment_method, payment_reference,
                amount, currency, status, gateway_response, created_at
            ) VALUES (
                :transaction_id, :subscription_id, :user_id, :payment_method, :payment_reference,
                :amount, :currency, :status, :gateway_response, NOW()
            )');

            $this->db->bind(':transaction_id', $transaction_id);
            $this->db->bind(':subscription_id', $transactionData['subscription_id']);
            $this->db->bind(':user_id', $transactionData['user_id']);
            $this->db->bind(':payment_method', $transactionData['payment_method']);
            $this->db->bind(':payment_reference', $transactionData['payment_reference']);
            $this->db->bind(':amount', $transactionData['amount']);
            $this->db->bind(':currency', $transactionData['currency'] ?? 'NGN');
            $this->db->bind(':status', $transactionData['status'] ?? 'pending');
            $this->db->bind(':gateway_response', isset($transactionData['gateway_response']) ? json_encode($transactionData['gateway_response']) : null);
            
            if ($this->db->execute()) {
                return $transaction_id;
            }
            
            return false;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Update transaction status
    public function updateTransactionStatus($transaction_id, $status, $gateway_response = null, $failure_reason = null) {
        try {
            $setParts = ['status = :status', 'updated_at = NOW()'];
            $params = [':transaction_id' => $transaction_id, ':status' => $status];
            
            if ($status === 'successful') {
                $setParts[] = 'payment_date = NOW()';
            }
            
            if ($gateway_response) {
                $setParts[] = 'gateway_response = :gateway_response';
                $params[':gateway_response'] = json_encode($gateway_response);
            }
            
            if ($failure_reason) {
                $setParts[] = 'failure_reason = :failure_reason';
                $params[':failure_reason'] = $failure_reason;
            }
            
            $setClause = implode(', ', $setParts);
            
            $this->db->query("UPDATE payment_transactions SET $setClause WHERE transaction_id = :transaction_id");
            
            foreach ($params as $key => $value) {
                $this->db->bind($key, $value);
            }
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get transaction by ID
    public function getTransactionById($transaction_id) {
        try {
            $this->db->query('SELECT * FROM payment_transactions WHERE transaction_id = :transaction_id');
            $this->db->bind(':transaction_id', $transaction_id);
            
            $transaction = $this->db->single();
            
            if ($transaction && $transaction->gateway_response) {
                $transaction->gateway_response = json_decode($transaction->gateway_response, true);
            }
            
            return $transaction;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get transaction by payment reference
    public function getTransactionByReference($payment_reference) {
        try {
            $this->db->query('SELECT * FROM payment_transactions WHERE payment_reference = :payment_reference');
            $this->db->bind(':payment_reference', $payment_reference);
            
            $transaction = $this->db->single();
            
            if ($transaction && $transaction->gateway_response) {
                $transaction->gateway_response = json_decode($transaction->gateway_response, true);
            }
            
            return $transaction;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get user's payment history
    public function getUserPaymentHistory($user_id, $limit = 20, $offset = 0) {
        try {
            $this->db->query("SELECT pt.*, us.plan_id, sp.plan_name
                             FROM payment_transactions pt
                             LEFT JOIN user_subscriptions us ON pt.subscription_id = us.subscription_id
                             LEFT JOIN subscription_plans sp ON us.plan_id = sp.plan_id
                             WHERE pt.user_id = :user_id
                             ORDER BY pt.created_at DESC
                             LIMIT :limit OFFSET :offset");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            $this->db->bind(':offset', $offset, PDO::PARAM_INT);
            
            $transactions = $this->db->resultSet();
            
            // Decode gateway responses
            foreach ($transactions as &$transaction) {
                if ($transaction->gateway_response) {
                    $transaction->gateway_response = json_decode($transaction->gateway_response, true);
                }
            }
            
            return $transactions;
        } catch (PDOException $e) {
            return [];
        }
    }

    // Process refund
    public function processRefund($transaction_id, $refund_amount, $reason = null) {
        try {
            $this->db->query('UPDATE payment_transactions SET 
                             refund_amount = :refund_amount,
                             refund_date = NOW(),
                             status = :status,
                             failure_reason = :reason,
                             updated_at = NOW()
                             WHERE transaction_id = :transaction_id');
            
            $this->db->bind(':transaction_id', $transaction_id);
            $this->db->bind(':refund_amount', $refund_amount);
            $this->db->bind(':status', 'refunded');
            $this->db->bind(':reason', $reason);
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Create billing invoice
    public function createInvoice($invoiceData) {
        try {
            $invoice_id = "inv_" . md5($invoiceData['user_id'] . time() . rand(1000, 9999));
            
            $this->db->query('INSERT INTO billing_history (
                invoice_id, subscription_id, user_id, plan_id, billing_period_start, billing_period_end,
                amount, tax_amount, discount_amount, total_amount, currency, status, due_date, invoice_data, created_at
            ) VALUES (
                :invoice_id, :subscription_id, :user_id, :plan_id, :billing_period_start, :billing_period_end,
                :amount, :tax_amount, :discount_amount, :total_amount, :currency, :status, :due_date, :invoice_data, NOW()
            )');

            $this->db->bind(':invoice_id', $invoice_id);
            $this->db->bind(':subscription_id', $invoiceData['subscription_id']);
            $this->db->bind(':user_id', $invoiceData['user_id']);
            $this->db->bind(':plan_id', $invoiceData['plan_id']);
            $this->db->bind(':billing_period_start', $invoiceData['billing_period_start']);
            $this->db->bind(':billing_period_end', $invoiceData['billing_period_end']);
            $this->db->bind(':amount', $invoiceData['amount']);
            $this->db->bind(':tax_amount', $invoiceData['tax_amount'] ?? 0.00);
            $this->db->bind(':discount_amount', $invoiceData['discount_amount'] ?? 0.00);
            $this->db->bind(':total_amount', $invoiceData['total_amount']);
            $this->db->bind(':currency', $invoiceData['currency'] ?? 'NGN');
            $this->db->bind(':status', $invoiceData['status'] ?? 'draft');
            $this->db->bind(':due_date', $invoiceData['due_date']);
            $this->db->bind(':invoice_data', isset($invoiceData['invoice_data']) ? json_encode($invoiceData['invoice_data']) : null);
            
            if ($this->db->execute()) {
                return $invoice_id;
            }
            
            return false;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Update invoice status
    public function updateInvoiceStatus($invoice_id, $status, $paid_date = null) {
        try {
            $setParts = ['status = :status', 'updated_at = NOW()'];
            $params = [':invoice_id' => $invoice_id, ':status' => $status];
            
            if ($paid_date) {
                $setParts[] = 'paid_date = :paid_date';
                $params[':paid_date'] = $paid_date;
            } elseif ($status === 'paid') {
                $setParts[] = 'paid_date = NOW()';
            }
            
            $setClause = implode(', ', $setParts);
            
            $this->db->query("UPDATE billing_history SET $setClause WHERE invoice_id = :invoice_id");
            
            foreach ($params as $key => $value) {
                $this->db->bind($key, $value);
            }
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get user's billing history
    public function getUserBillingHistory($user_id, $limit = 20, $offset = 0) {
        try {
            $this->db->query("SELECT bh.*, sp.plan_name
                             FROM billing_history bh
                             INNER JOIN subscription_plans sp ON bh.plan_id = sp.plan_id
                             WHERE bh.user_id = :user_id
                             ORDER BY bh.created_at DESC
                             LIMIT :limit OFFSET :offset");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            $this->db->bind(':offset', $offset, PDO::PARAM_INT);
            
            $invoices = $this->db->resultSet();
            
            // Decode invoice data
            foreach ($invoices as &$invoice) {
                if ($invoice->invoice_data) {
                    $invoice->invoice_data = json_decode($invoice->invoice_data, true);
                }
            }
            
            return $invoices;
        } catch (PDOException $e) {
            return [];
        }
    }

    // Get invoice by ID
    public function getInvoiceById($invoice_id) {
        try {
            $this->db->query("SELECT bh.*, sp.plan_name, u.full_name, u.email, u.address, u.city, u.state, u.country
                             FROM billing_history bh
                             INNER JOIN subscription_plans sp ON bh.plan_id = sp.plan_id
                             INNER JOIN initkey_rid u ON bh.user_id = u.user_id
                             WHERE bh.invoice_id = :invoice_id");
            
            $this->db->bind(':invoice_id', $invoice_id);
            
            $invoice = $this->db->single();
            
            if ($invoice && $invoice->invoice_data) {
                $invoice->invoice_data = json_decode($invoice->invoice_data, true);
            }
            
            return $invoice;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get payment statistics
    public function getPaymentStatistics($user_id = null, $start_date = null, $end_date = null) {
        try {
            $whereConditions = [];
            $params = [];
            
            if ($user_id) {
                $whereConditions[] = 'user_id = :user_id';
                $params[':user_id'] = $user_id;
            }
            
            if ($start_date) {
                $whereConditions[] = 'payment_date >= :start_date';
                $params[':start_date'] = $start_date;
            }
            
            if ($end_date) {
                $whereConditions[] = 'payment_date <= :end_date';
                $params[':end_date'] = $end_date;
            }
            
            $whereClause = !empty($whereConditions) ? 'WHERE ' . implode(' AND ', $whereConditions) : '';
            
            $this->db->query("SELECT 
                             COUNT(*) as total_transactions,
                             COUNT(CASE WHEN status = 'successful' THEN 1 END) as successful_transactions,
                             COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_transactions,
                             SUM(CASE WHEN status = 'successful' THEN amount ELSE 0 END) as total_revenue,
                             AVG(CASE WHEN status = 'successful' THEN amount ELSE NULL END) as average_transaction_amount
                             FROM payment_transactions $whereClause");
            
            foreach ($params as $key => $value) {
                $this->db->bind($key, $value);
            }
            
            return $this->db->single();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get overdue invoices
    public function getOverdueInvoices($days_overdue = 0) {
        try {
            $this->db->query("SELECT bh.*, u.email, u.full_name, sp.plan_name
                             FROM billing_history bh
                             INNER JOIN initkey_rid u ON bh.user_id = u.user_id
                             INNER JOIN subscription_plans sp ON bh.plan_id = sp.plan_id
                             WHERE bh.status IN ('sent', 'overdue') 
                             AND bh.due_date < DATE_SUB(NOW(), INTERVAL :days_overdue DAY)
                             ORDER BY bh.due_date ASC");
            
            $this->db->bind(':days_overdue', $days_overdue);
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return [];
        }
    }

    // Generate payment reference
    public function generatePaymentReference($prefix = 'SEL') {
        return $prefix . '_' . time() . '_' . rand(100000, 999999);
    }

    // Validate payment amount
    public function validatePaymentAmount($subscription_id, $amount) {
        try {
            $this->db->query("SELECT us.amount, sp.price_monthly, sp.price_yearly
                             FROM user_subscriptions us
                             INNER JOIN subscription_plans sp ON us.plan_id = sp.plan_id
                             WHERE us.subscription_id = :subscription_id");
            
            $this->db->bind(':subscription_id', $subscription_id);
            $subscription = $this->db->single();
            
            if (!$subscription) {
                return false;
            }
            
            // Check if amount matches subscription amount
            return abs($amount - $subscription->amount) < 0.01; // Allow for minor floating point differences
        } catch (PDOException $e) {
            return false;
        }
    }
}
