<?php
class Cache
{
    private $cacheDir;

    public function __construct()
    {
        $this->cacheDir = APPROOT . '/cache';
        if (!is_dir($this->cacheDir)) {
            mkdir($this->cacheDir, 0755, true);
        }
    }

    private function getFilePath($key)
    {
        return $this->cacheDir . '/' . sha1($key) . '.cache';
    }

    public function get($key)
    {
        $filePath = $this->getFilePath($key);
        if (file_exists($filePath)) {
            $content = file_get_contents($filePath);
            $data = unserialize($content);
            if (time() < $data['expires']) {
                return $data['value'];
            } else {
                // Cache has expired, delete it
                unlink($filePath);
            }
        }
        return false;
    }

    public function set($key, $value, $ttl = 3600) // Default TTL is 1 hour
    {
        $filePath = $this->getFilePath($key);
        $data = [
            'expires' => time() + $ttl,
            'value' => $value
        ];
        return file_put_contents($filePath, serialize($data)) !== false;
    }

    public function del($key)
    {
        $filePath = $this->getFilePath($key);
        if (file_exists($filePath)) {
            return unlink($filePath);
        }
        return true;
    }

    public function exists($key)
    {
        $filePath = $this->getFilePath($key);
        if (file_exists($filePath)) {
            $content = file_get_contents($filePath);
            $data = unserialize($content);
            if (time() < $data['expires']) {
                return true;
            } else {
                unlink($filePath);
            }
        }
        return false;
    }

    public function increment($key, $value = 1)
    {
        $currentValue = $this->get($key);
        if ($currentValue === false) {
            $currentValue = 0;
        }
        $newValue = $currentValue + $value;
        $this->set($key, $newValue);
        return $newValue;
    }

    public function expire($key, $ttl)
    {
        $currentValue = $this->get($key);
        if ($currentValue !== false) {
            $this->set($key, $currentValue, $ttl);
        }
    }
}
