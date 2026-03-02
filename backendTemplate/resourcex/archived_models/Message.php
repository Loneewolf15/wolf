<?php
class Message {
    private $db;
    
    public function __construct(){
        $this->db = new Database;
    }

    // Create or get thread between two users
    public function getOrCreateThread($user1_id, $user2_id, $listing_id = null) {
        try {
            // Ensure consistent ordering for unique constraint
            $participant_1 = $user1_id < $user2_id ? $user1_id : $user2_id;
            $participant_2 = $user1_id < $user2_id ? $user2_id : $user1_id;
            
            // Check if thread exists
            $this->db->query("SELECT thread_id FROM message_threads 
                             WHERE participant_1 = :p1 AND participant_2 = :p2 
                             AND (related_listing_id = :listing_id OR (related_listing_id IS NULL AND :listing_id IS NULL))");
            
            $this->db->bind(':p1', $participant_1);
            $this->db->bind(':p2', $participant_2);
            $this->db->bind(':listing_id', $listing_id);
            
            $existing = $this->db->single();
            
            if ($existing) {
                return $existing->thread_id;
            }
            
            // Create new thread
            $thread_id = "thread_" . md5($participant_1 . $participant_2 . ($listing_id ?: '') . time());
            
            $this->db->query('INSERT INTO message_threads (
                thread_id, 
                participant_1, 
                participant_2, 
                related_listing_id, 
                created_at
            ) VALUES (
                :thread_id, 
                :participant_1, 
                :participant_2, 
                :related_listing_id, 
                NOW()
            )');

            $this->db->bind(':thread_id', $thread_id);
            $this->db->bind(':participant_1', $participant_1);
            $this->db->bind(':participant_2', $participant_2);
            $this->db->bind(':related_listing_id', $listing_id);
            
            if ($this->db->execute()) {
                return $thread_id;
            }
            
            return false;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Send a message
    public function sendMessage($sender_id, $receiver_id, $message_text, $subject = null, $listing_id = null, $message_type = 'general') {
        try {
            $message_id = "msg_" . md5($sender_id . $receiver_id . time() . rand(1000, 9999));
            $thread_id = $this->getOrCreateThread($sender_id, $receiver_id, $listing_id);
            
            if (!$thread_id) {
                return false;
            }
            
            $this->db->query('INSERT INTO messages (
                message_id, 
                thread_id, 
                sender_id, 
                receiver_id, 
                subject, 
                message_text, 
                related_listing_id, 
                message_type, 
                created_at
            ) VALUES (
                :message_id, 
                :thread_id, 
                :sender_id, 
                :receiver_id, 
                :subject, 
                :message_text, 
                :related_listing_id, 
                :message_type, 
                NOW()
            )');

            $this->db->bind(':message_id', $message_id);
            $this->db->bind(':thread_id', $thread_id);
            $this->db->bind(':sender_id', $sender_id);
            $this->db->bind(':receiver_id', $receiver_id);
            $this->db->bind(':subject', $subject);
            $this->db->bind(':message_text', $message_text);
            $this->db->bind(':related_listing_id', $listing_id);
            $this->db->bind(':message_type', $message_type);
            
            if ($this->db->execute()) {
                // Update thread's last message
                $this->updateThreadLastMessage($thread_id, $message_id);
                return $message_id;
            }
            
            return false;
        } catch (PDOException $e) {
            return false;
        }
    }

    // Update thread's last message
    private function updateThreadLastMessage($thread_id, $message_id) {
        try {
            $this->db->query('UPDATE message_threads SET 
                             last_message_id = :message_id, 
                             last_activity = NOW() 
                             WHERE thread_id = :thread_id');
            
            $this->db->bind(':message_id', $message_id);
            $this->db->bind(':thread_id', $thread_id);
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get user's message threads
    public function getUserThreads($user_id, $limit = 20, $offset = 0) {
        try {
            $this->db->query("SELECT mt.*, 
                             CASE 
                                 WHEN mt.participant_1 = :user_id THEN u2.full_name 
                                 ELSE u1.full_name 
                             END as other_user_name,
                             CASE 
                                 WHEN mt.participant_1 = :user_id THEN mt.participant_2 
                                 ELSE mt.participant_1 
                             END as other_user_id,
                             l.title as listing_title,
                             lm.message_text as last_message_text,
                             lm.created_at as last_message_time,
                             COUNT(CASE WHEN m.receiver_id = :user_id AND m.is_read = 0 THEN 1 END) as unread_count
                             FROM message_threads mt
                             LEFT JOIN initkey_rid u1 ON mt.participant_1 = u1.user_id
                             LEFT JOIN initkey_rid u2 ON mt.participant_2 = u2.user_id
                             LEFT JOIN listings l ON mt.related_listing_id = l.listing_id
                             LEFT JOIN messages lm ON mt.last_message_id = lm.message_id
                             LEFT JOIN messages m ON mt.thread_id = m.thread_id
                             WHERE mt.participant_1 = :user_id OR mt.participant_2 = :user_id
                             GROUP BY mt.thread_id
                             ORDER BY mt.last_activity DESC
                             LIMIT :limit OFFSET :offset");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            $this->db->bind(':offset', $offset, PDO::PARAM_INT);
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return [];
        }
    }

    // Get messages in a thread
    public function getThreadMessages($thread_id, $user_id, $limit = 50, $offset = 0) {
        try {
            // Verify user is participant in thread
            $this->db->query("SELECT 1 FROM message_threads 
                             WHERE thread_id = :thread_id 
                             AND (participant_1 = :user_id OR participant_2 = :user_id)");
            
            $this->db->bind(':thread_id', $thread_id);
            $this->db->bind(':user_id', $user_id);
            
            if (!$this->db->single()) {
                return false; // User not authorized to view this thread
            }
            
            $this->db->query("SELECT m.*, 
                             s.full_name as sender_name, 
                             r.full_name as receiver_name,
                             l.title as listing_title
                             FROM messages m
                             LEFT JOIN initkey_rid s ON m.sender_id = s.user_id
                             LEFT JOIN initkey_rid r ON m.receiver_id = r.user_id
                             LEFT JOIN listings l ON m.related_listing_id = l.listing_id
                             WHERE m.thread_id = :thread_id
                             ORDER BY m.created_at ASC
                             LIMIT :limit OFFSET :offset");
            
            $this->db->bind(':thread_id', $thread_id);
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            $this->db->bind(':offset', $offset, PDO::PARAM_INT);
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Mark messages as read
    public function markMessagesAsRead($thread_id, $user_id) {
        try {
            $this->db->query('UPDATE messages SET is_read = 1 
                             WHERE thread_id = :thread_id 
                             AND receiver_id = :user_id 
                             AND is_read = 0');
            
            $this->db->bind(':thread_id', $thread_id);
            $this->db->bind(':user_id', $user_id);
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get unread message count for user
    public function getUnreadCount($user_id) {
        try {
            $this->db->query("SELECT COUNT(*) as unread_count FROM messages 
                             WHERE receiver_id = :user_id AND is_read = 0");
            
            $this->db->bind(':user_id', $user_id);
            $result = $this->db->single();
            
            return $result ? $result->unread_count : 0;
        } catch (PDOException $e) {
            return 0;
        }
    }

    // Search messages
    public function searchMessages($user_id, $search_term, $limit = 20) {
        try {
            $this->db->query("SELECT m.*, 
                             s.full_name as sender_name, 
                             r.full_name as receiver_name,
                             l.title as listing_title
                             FROM messages m
                             LEFT JOIN initkey_rid s ON m.sender_id = s.user_id
                             LEFT JOIN initkey_rid r ON m.receiver_id = r.user_id
                             LEFT JOIN listings l ON m.related_listing_id = l.listing_id
                             WHERE (m.sender_id = :user_id OR m.receiver_id = :user_id)
                             AND (m.message_text LIKE :search OR m.subject LIKE :search OR l.title LIKE :search)
                             ORDER BY m.created_at DESC
                             LIMIT :limit");
            
            $this->db->bind(':user_id', $user_id);
            $this->db->bind(':search', '%' . $search_term . '%');
            $this->db->bind(':limit', $limit, PDO::PARAM_INT);
            
            return $this->db->resultSet();
        } catch (PDOException $e) {
            return [];
        }
    }

    // Delete message (soft delete by archiving)
    public function deleteMessage($message_id, $user_id) {
        try {
            // Only sender can delete their own messages
            $this->db->query('UPDATE messages SET is_archived = 1 
                             WHERE message_id = :message_id 
                             AND sender_id = :user_id');
            
            $this->db->bind(':message_id', $message_id);
            $this->db->bind(':user_id', $user_id);
            
            return $this->db->execute();
        } catch (PDOException $e) {
            return false;
        }
    }

    // Get message by ID (with authorization check)
    public function getMessageById($message_id, $user_id) {
        try {
            $this->db->query("SELECT m.*, 
                             s.full_name as sender_name, 
                             r.full_name as receiver_name,
                             l.title as listing_title
                             FROM messages m
                             LEFT JOIN initkey_rid s ON m.sender_id = s.user_id
                             LEFT JOIN initkey_rid r ON m.receiver_id = r.user_id
                             LEFT JOIN listings l ON m.related_listing_id = l.listing_id
                             WHERE m.message_id = :message_id
                             AND (m.sender_id = :user_id OR m.receiver_id = :user_id)");
            
            $this->db->bind(':message_id', $message_id);
            $this->db->bind(':user_id', $user_id);
            
            return $this->db->single();
        } catch (PDOException $e) {
            return false;
        }
    }
}
