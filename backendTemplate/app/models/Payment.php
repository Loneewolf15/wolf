<?php

/**
 * Payment Model
 * 
 * Handles database operations for payment transactions
 */
class Payment
{
    private $db;

    public function __construct()
    {
        $this->db = new Database();
    }

    /**
     * Create a new payment transaction
     */
    public function createTransaction(array $data): bool
    {
        $this->db->query('INSERT INTO payments (
            user_id, provider, reference, amount, currency, status, metadata, created_at
        ) VALUES (
            :user_id, :provider, :reference, :amount, :currency, :status, :metadata, NOW()
        )');

        $this->db->bind(':user_id', $data['user_id']);
        $this->db->bind(':provider', $data['provider']);
        $this->db->bind(':reference', $data['reference']);
        $this->db->bind(':amount', $data['amount']);
        $this->db->bind(':currency', $data['currency']);
        $this->db->bind(':status', $data['status']);
        $this->db->bind(':metadata', $data['metadata'] ?? null);

        return $this->db->execute();
    }

    /**
     * Find transaction by reference
     */
    public function findByReference(string $reference): ?object
    {
        $this->db->query('SELECT * FROM payments WHERE reference = :reference');
        $this->db->bind(':reference', $reference);
        return $this->db->single();
    }

    /**
     * Find transaction by ID
     */
    public function findById(int $id): ?object
    {
        $this->db->query('SELECT * FROM payments WHERE id = :id');
        $this->db->bind(':id', $id);
        return $this->db->single();
    }

    /**
     * Update transaction status
     */
    public function updateTransaction(int $id, array $data): bool
    {
        $fields = [];
        $bindings = [':id' => $id];

        foreach ($data as $key => $value) {
            $fields[] = "$key = :$key";
            $bindings[":$key"] = $value;
        }

        $sql = 'UPDATE payments SET ' . implode(', ', $fields) . ' WHERE id = :id';
        $this->db->query($sql);

        foreach ($bindings as $param => $value) {
            $this->db->bind($param, $value);
        }

        return $this->db->execute();
    }

    /**
     * Get user transactions
     */
    public function getUserTransactions(int $userId, int $limit = 20, int $offset = 0): array
    {
        $this->db->query('SELECT * FROM payments 
                         WHERE user_id = :user_id 
                         ORDER BY created_at DESC 
                         LIMIT :limit OFFSET :offset');

        $this->db->bind(':user_id', $userId);
        $this->db->bind(':limit', $limit);
        $this->db->bind(':offset', $offset);

        return $this->db->resultset();
    }

    /**
     * Get successful transactions for a user
     */
    public function getSuccessfulTransactions(int $userId): array
    {
        $this->db->query('SELECT * FROM payments 
                         WHERE user_id = :user_id AND status = :status 
                         ORDER BY created_at DESC');

        $this->db->bind(':user_id', $userId);
        $this->db->bind(':status', 'successful');

        return $this->db->resultset();
    }

    /**
     * Get total amount paid by user
     */
    public function getTotalPaid(int $userId): float
    {
        $this->db->query('SELECT SUM(amount) as total 
                         FROM payments 
                         WHERE user_id = :user_id AND status = :status');

        $this->db->bind(':user_id', $userId);
        $this->db->bind(':status', 'successful');

        $result = $this->db->single();
        return (float)($result->total ?? 0);
    }

    /**
     * Check if reference exists
     */
    public function referenceExists(string $reference): bool
    {
        $this->db->query('SELECT COUNT(*) as count FROM payments WHERE reference = :reference');
        $this->db->bind(':reference', $reference);

        $result = $this->db->single();
        return ($result->count > 0);
    }

    /**
     * Get transaction stats for user
     */
    public function getTransactionStats(int $userId): array
    {
        $this->db->query('SELECT 
            COUNT(*) as total_transactions,
            SUM(CASE WHEN status = "successful" THEN 1 ELSE 0 END) as successful_count,
            SUM(CASE WHEN status = "successful" THEN amount ELSE 0 END) as total_amount,
            MAX(created_at) as last_transaction
            FROM payments 
            WHERE user_id = :user_id');

        $this->db->bind(':user_id', $userId);

        $result = $this->db->single();

        return [
            'total_transactions' => (int)$result->total_transactions,
            'successful_transactions' => (int)$result->successful_count,
            'total_amount_paid' => (float)$result->total_amount,
            'last_transaction_date' => $result->last_transaction
        ];
    }
}
