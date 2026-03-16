<?php
class Delivery
{
    private $db;

    public function __construct()
    {
        $this->db = new Database;
    }

    /**
     * Finds registered delivery agents operating in the specified local government area.
     * @param string $localGovernmentArea The LGA of the seller.
     * @return array List of available delivery agents.
     */
    public function findNearbyAgents($localGovernmentArea)
    {
        $this->db->query('SELECT
            da.agent_id,
            da.rating,
            da.rate_per_km,
            u.name AS agent_name,
            u.profile_pic_url,
            u.phone AS contact_number
            FROM DeliveryAgents da
            JOIN initkey_rid u ON da.agent_id = u.user_id
            WHERE u.local_government_area = :lga
            AND da.current_location IS NOT NULL  -- Assumes agents update their current location
            ORDER BY da.rating DESC');

        $this->db->bind(':lga', $localGovernmentArea);
        $rows = $this->db->resultSet();
        return $rows;
    }

    /**
     * Creates a new delivery record.
     * @param array $data Associative array containing delivery details.
     * @return string|false The new delivery_id on success, false on failure.
     */
    public function createDelivery($data)
    {
        try {
            $delivery_id = "del_" . md5($data['order_id'] . time() . rand(1000, 9999));
            $tracking_code = strtoupper(substr(md5(uniqid(mt_rand(), true)), 0, 10));

            $this->db->query('INSERT INTO Deliveries (
                delivery_id, order_id, agent_id, seller_id, buyer_id,
                pickup_address, delivery_address, status, tracking_code,
                estimated_delivery_date, created_at
            ) VALUES (
                :delivery_id, :order_id, :agent_id, :seller_id, :buyer_id,
                :pickup_address, :delivery_address, :status, :tracking_code,
                :estimated_delivery_date, NOW()
            )');

            $this->db->bind(':delivery_id', $delivery_id);
            $this->db->bind(':order_id', $data['order_id']);
            $this->db->bind(':agent_id', $data['agent_id']);
            $this->db->bind(':seller_id', $data['seller_id']);
            $this->db->bind(':buyer_id', $data['buyer_id']);
            $this->db->bind(':pickup_address', $data['pickup_address']);
            $this->db->bind(':delivery_address', $data['delivery_address']);
            $this->db->bind(':status', $data['status'] ?? 'pending');
            $this->db->bind(':tracking_code', $tracking_code);
            $this->db->bind(':estimated_delivery_date', $data['estimated_delivery_date'] ?? null);

            if ($this->db->execute()) {
                return $delivery_id;
            }
            return false;
        } catch (PDOException $e) {
            error_log("Error creating delivery: " . $e->getMessage());
            return false;
        }
    }

    /**
     * Updates the status of an existing delivery.
     * @param string $deliveryId The ID of the delivery to update.
     * @param string $status The new status for the delivery.
     * @param int|null $agentId Optional. The ID of the agent performing the update, for security.
     * @return bool True on success, false on failure.
     */
    public function updateDeliveryStatus($deliveryId, $status, $agentId = null)
    {
        try {
            $query = 'UPDATE Deliveries SET status = :status, updated_at = NOW() ';
            $params = [':delivery_id' => $deliveryId, ':status' => $status];

            if ($agentId !== null) {
                $query .= ' WHERE delivery_id = :delivery_id AND agent_id = :agent_id';
                $params[':agent_id'] = $agentId;
            } else {
                $query .= ' WHERE delivery_id = :delivery_id';
            }

            // If status is 'delivered', set actual_delivery_date
            if ($status === 'delivered') {
                $query = str_replace('updated_at = NOW()', 'updated_at = NOW(), actual_delivery_date = NOW()', $query);
            }

            $this->db->query($query);

            foreach ($params as $key => $value) {
                $this->db->bind($key, $value);
            }

            return $this->db->execute();
        } catch (PDOException $e) {
            error_log("Error updating delivery status: " . $e->getMessage());
            return false;
        }
    }

    /**
     * Retrieves the details of a specific delivery.
     * @param string $deliveryId The ID of the delivery to retrieve.
     * @return object|false Delivery details on success, false if not found or on failure.
     */
    public function getDeliveryDetails($deliveryId)
    {
        try {
            $this->db->query('SELECT
                d.*,
                s.name AS seller_name,
                b.name AS buyer_name,
                a.name AS agent_name
                FROM Deliveries d
                LEFT JOIN initkey_rid s ON d.seller_id = s.user_id
                LEFT JOIN initkey_rid b ON d.buyer_id = b.user_id
                LEFT JOIN initkey_rid a ON d.agent_id = a.user_id
                WHERE d.delivery_id = :delivery_id');
            $this->db->bind(':delivery_id', $deliveryId);
            
            return $this->db->single();
        } catch (PDOException $e) {
            error_log("Error retrieving delivery details: " . $e->getMessage());
            return false;
        }
    }

    /**
     * Registers a new delivery agent.
     * @param array $data Associative array containing agent details (user_id, rating, rate_per_km, etc.).
     * @return bool True on success, false on failure.
     */
    public function registerDeliveryAgent($data)
    {
        try {
            $this->db->query('INSERT INTO DeliveryAgents (
                agent_id, rating, rate_per_km, current_location, is_available, created_at
            ) VALUES (
                :agent_id, :rating, :rate_per_km, :current_location, :is_available, NOW()
            )');

            $this->db->bind(':agent_id', $data['user_id']); // Assuming agent_id is the user_id
            $this->db->bind(':rating', $data['rating'] ?? 0);
            $this->db->bind(':rate_per_km', $data['rate_per_km'] ?? 0);
            $this->db->bind(':current_location', $data['current_location'] ?? null);
            $this->db->bind(':is_available', $data['is_available'] ?? 1);

            return $this->db->execute();
        } catch (PDOException $e) {
            error_log("Error registering delivery agent: " . $e->getMessage());
            return false;
        }
    }

    /**
     * Updates an existing delivery agent's profile.
     * @param int $agentId The ID of the agent to update.
     * @param array $data Associative array containing fields to update (e.g., rating, rate_per_km, current_location, is_available).
     * @return bool True on success, false on failure.
     */
    public function updateDeliveryAgentProfile($agentId, $data)
    {
        try {
            $setParts = [];
            $params = [':agent_id' => $agentId];

            if (isset($data['rating'])) {
                $setParts[] = 'rating = :rating';
                $params[':rating'] = $data['rating'];
            }
            if (isset($data['rate_per_km'])) {
                $setParts[] = 'rate_per_km = :rate_per_km';
                $params[':rate_per_km'] = $data['rate_per_km'];
            }
            if (isset($data['current_location'])) {
                $setParts[] = 'current_location = :current_location';
                $params[':current_location'] = $data['current_location'];
            }
            if (isset($data['is_available'])) {
                $setParts[] = 'is_available = :is_available';
                $params[':is_available'] = $data['is_available'];
            }

            if (empty($setParts)) {
                return false; // No data to update
            }

            $setParts[] = 'updated_at = NOW()';
            $setClause = implode(', ', $setParts);

            $this->db->query("UPDATE DeliveryAgents SET $setClause WHERE agent_id = :agent_id");

            foreach ($params as $key => $value) {
                $this->db->bind($key, $value);
            }

            return $this->db->execute();
        } catch (PDOException $e) {
            error_log("Error updating delivery agent profile: " . $e->getMessage());
            return false;
        }
    }

    /**
     * Retrieves the full profile of a delivery agent.
     * @param int $agentId The ID of the agent to retrieve.
     * @return object|false Agent details on success, false if not found or on failure.
     */
    public function getDeliveryAgentProfile($agentId)
    {
        try {
            $this->db->query('SELECT
                da.agent_id,
                da.rating,
                da.rate_per_km,
                da.current_location,
                da.is_available,
                u.name AS agent_name,
                u.email,
                u.phone,
                u.profile_pic_url,
                u.local_government_area
                FROM DeliveryAgents da
                JOIN initkey_rid u ON da.agent_id = u.user_id
                WHERE da.agent_id = :agent_id');
            $this->db->bind(':agent_id', $agentId);
            
            return $this->db->single();
        } catch (PDOException $e) {
            error_log("Error retrieving delivery agent profile: " . $e->getMessage());
            return false;
        }
    }
}