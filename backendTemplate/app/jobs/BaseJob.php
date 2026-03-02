<?php

/**
 * Base Job Class
 * 
 * All scheduled jobs should extend this class
 */
abstract class BaseJob
{
    protected $db;
    protected $logger;

    public function __construct()
    {
        $this->db = new Database();
        $this->logger = new Logger();
    }

    /**
     * Execute the job
     * Must be implemented by child classes
     */
    abstract public function execute(): string;

    /**
     * Log job activity
     */
    protected function log(string $message): void
    {
        $this->logger->log(Logger::INFO, $message, [
            'job' => get_class($this)
        ]);
    }
}
