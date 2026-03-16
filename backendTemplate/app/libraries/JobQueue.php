<?php
class JobQueue
{
    private $db;
    private $queueName = 'default';
    const MAX_ATTEMPTS = 3;

    public function __construct()
    {
        $this->db = new Database;
    }

    public function push($jobType, $data)
    {
        $payload = json_encode([
            'job' => $jobType,
            'data' => $data
        ]);

        $this->db->query('INSERT INTO jobs (queue, payload, attempts, available_at, created_at) VALUES (:queue, :payload, 0, :available_at, :created_at)');
        $this->db->bind(':queue', $this->queueName);
        $this->db->bind(':payload', $payload);
        $this->db->bind(':available_at', time());
        $this->db->bind(':created_at', time());
        
        return $this->db->execute();
    }

    public function pop()
    {
        $this->db->beginTransaction();

        try {
            $this->db->query('SELECT * FROM jobs WHERE queue = :queue AND reserved_at IS NULL AND available_at <= :available_at AND attempts < :max_attempts ORDER BY id ASC LIMIT 1 FOR UPDATE');
            $this->db->bind(':queue', $this->queueName);
            $this->db->bind(':available_at', time());
            $this->db->bind(':max_attempts', self::MAX_ATTEMPTS);
            $job = $this->db->single();

            if ($job) {
                $this->db->query('UPDATE jobs SET reserved_at = :reserved_at, attempts = attempts + 1 WHERE id = :id');
                $this->db->bind(':reserved_at', time());
                $this->db->bind(':id', $job->id);
                $this->db->execute();
            }

            $this->db->commit();

            if ($job) {
                $payload = json_decode($job->payload, true);
                return [
                    'id' => $job->id,
                    'type' => $payload['job'],
                    'data' => $payload['data']
                ];
            }

            return null;
        } catch (Exception $e) {
            $this->db->rollBack();
            error_log("Job queue pop error: " . $e->getMessage());
            return null;
        }
    }

    public function delete($jobId)
    {
        $this->db->query('DELETE FROM jobs WHERE id = :id');
        $this->db->bind(':id', $jobId);
        return $this->db->execute();
    }

    public function release($jobId, $delay = 60) // Default 1 minute delay
    {
        $this->db->query('UPDATE jobs SET reserved_at = NULL, available_at = :available_at WHERE id = :id');
        $this->db->bind(':available_at', time() + $delay);
        $this->db->bind(':id', $jobId);
        return $this->db->execute();
    }
}