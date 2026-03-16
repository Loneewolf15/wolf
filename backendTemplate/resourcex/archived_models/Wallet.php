<?php
class Wallet {
    private $db;

    public function __construct(){
        $this->db = new Database;
    }

    public function createWallet($userId, $walletTag){
        try {
            $walletId = $this->generateUuid();
            $accountNumber = $this->generateAccountNumber();

            $this->db->query('INSERT INTO Wallets (wallet_id, user_id, account_number, wallet_tag, balance) VALUES (:wallet_id, :user_id, :account_number, :wallet_tag, 0)');
            $this->db->bind(':wallet_id', $walletId);
            $this->db->bind(':user_id', $userId);
            $this->db->bind(':account_number', $accountNumber);
            $this->db->bind(':wallet_tag', $walletTag);
            
            return $this->db->execute();
        } catch (Exception $e) {
            error_log('createWallet error: ' . $e->getMessage());
            return false;
        }
    }

    public function getWalletByUserId($userId)
    {
        $this->db->query('SELECT * FROM Wallets WHERE user_id = :user_id');
        $this->db->bind(':user_id', $userId);
        return $this->db->single();
    }

    public function hasSufficientFunds($userId, $amount)
    {
        $wallet = $this->getWalletByUserId($userId);
        if ($wallet && $wallet->balance >= $amount) {
            return true;
        }
        return false;
    }

    public function processWalletPayment($userId, $amount, $description)
    {
        if (!$this->hasSufficientFunds($userId, $amount)) {
            return false;
        }

        try {
            $this->db->query('UPDATE Wallets SET balance = balance - :amount WHERE user_id = :user_id');
            $this->db->bind(':amount', $amount);
            $this->db->bind(':user_id', $userId);
            if ($this->db->execute()) {
                // Log the transaction
                // You might want to create a dedicated transaction logging method
                return true;
            }
            return false;
        } catch (Exception $e) {
            error_log('processWalletPayment error: ' . $e->getMessage());
            return false;
        }
    }

    private function generateUuid() {
        return sprintf( '%04x%04x-%04x-%04x-%04x-%04x%04x%04x',
            mt_rand( 0, 0xffff ), mt_rand( 0, 0xffff ),
            mt_rand( 0, 0xffff ),
            mt_rand( 0, 0x0fff ) | 0x4000,
            mt_rand( 0, 0x3fff ) | 0x8000,
            mt_rand( 0, 0xffff ), mt_rand( 0, 0xffff ), mt_rand( 0, 0xffff )
        );
    }

    private function generateAccountNumber() {
        // Generate a 10-digit account number
        $number = mt_rand(1000000000, 9999999999);
        return 'MP-' . $number;
    }
}
