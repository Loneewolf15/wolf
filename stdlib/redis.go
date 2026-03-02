package stdlib

import (
	"fmt"
	"sync"
	"time"
)

// Redis implements a Wolf Redis client interface.
// In production this wraps go-redis; here we define the API surface.
type Redis struct {
	host     string
	port     int
	password string
	db       int
	conn     RedisConn
}

// RedisConn is the interface for Redis connections (allows mocking).
type RedisConn interface {
	Get(key string) (string, error)
	Set(key string, value interface{}, expiration time.Duration) error
	Del(keys ...string) (int64, error)
	Exists(keys ...string) (int64, error)
	Expire(key string, expiration time.Duration) (bool, error)
	LPush(key string, values ...interface{}) (int64, error)
	RPush(key string, values ...interface{}) (int64, error)
	LRange(key string, start, stop int64) ([]string, error)
	HSet(key, field string, value interface{}) (int64, error)
	HGet(key, field string) (string, error)
	HGetAll(key string) (map[string]string, error)
	Close() error
	Ping() error
}

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// NewRedis creates a new Redis client.
func NewRedis(config *RedisConfig) *Redis {
	if config.Port == 0 {
		config.Port = 6379
	}
	if config.Host == "" {
		config.Host = "localhost"
	}
	return &Redis{
		host:     config.Host,
		port:     config.Port,
		password: config.Password,
		db:       config.DB,
	}
}

// SetConn sets the underlying Redis connection (for dependency injection/testing).
func (r *Redis) SetConn(conn RedisConn) {
	r.conn = conn
}

// GlobalRedis is the default instance used by Wolf_Redis static method calls.
var GlobalRedis *Redis

func init() {
	GlobalRedis = NewRedis(&RedisConfig{})
	// Mock by default so examples work without an actual Redis server
	GlobalRedis.SetConn(NewMemoryRedis())
}

// Connect implements Wolf_Redis::Connect().
func (r *Redis) Connect() error {
	// Dummy connect function to satisfy Wolf_Redis::Connect() AST calls.
	return nil
}

// Get retrieves a string value by key.
func (r *Redis) Get(key string) string {
	if r.conn == nil {
		return ""
	}
	val, _ := r.conn.Get(key)
	return val
}

// Set stores a key-value pair with optional TTL.
func (r *Redis) Set(key string, value interface{}, ttl time.Duration) error {
	if r.conn == nil {
		return fmt.Errorf("wolf: redis not connected")
	}
	return r.conn.Set(key, value, ttl)
}

// Del deletes keys.
func (r *Redis) Del(keys ...string) (int64, error) {
	if r.conn == nil {
		return 0, fmt.Errorf("wolf: redis not connected")
	}
	return r.conn.Del(keys...)
}

// Exists checks if keys exist.
func (r *Redis) Exists(keys ...string) (int64, error) {
	if r.conn == nil {
		return 0, fmt.Errorf("wolf: redis not connected")
	}
	return r.conn.Exists(keys...)
}

// HSet sets a hash field.
func (r *Redis) HSet(key, field string, value interface{}) (int64, error) {
	if r.conn == nil {
		return 0, fmt.Errorf("wolf: redis not connected")
	}
	return r.conn.HSet(key, field, value)
}

// HGet gets a hash field.
func (r *Redis) HGet(key, field string) (string, error) {
	if r.conn == nil {
		return "", fmt.Errorf("wolf: redis not connected")
	}
	return r.conn.HGet(key, field)
}

// HGetAll returns all fields in a hash.
func (r *Redis) HGetAll(key string) (map[string]string, error) {
	if r.conn == nil {
		return nil, fmt.Errorf("wolf: redis not connected")
	}
	return r.conn.HGetAll(key)
}

// Close closes the Redis connection.
func (r *Redis) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// ========== In-Memory Mock (for testing) ==========

// MemoryRedis implements RedisConn using an in-memory map.
type MemoryRedis struct {
	mu   sync.RWMutex
	data map[string]string
	hash map[string]map[string]string
}

// NewMemoryRedis creates an in-memory Redis for testing.
func NewMemoryRedis() *MemoryRedis {
	return &MemoryRedis{
		data: make(map[string]string),
		hash: make(map[string]map[string]string),
	}
}

func (m *MemoryRedis) Get(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.data[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("key not found: %s", key)
}

func (m *MemoryRedis) Set(key string, value interface{}, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = fmt.Sprintf("%v", value)
	return nil
}

func (m *MemoryRedis) Del(keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, k := range keys {
		if _, ok := m.data[k]; ok {
			delete(m.data, k)
			count++
		}
	}
	return count, nil
}

func (m *MemoryRedis) Exists(keys ...string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var count int64
	for _, k := range keys {
		if _, ok := m.data[k]; ok {
			count++
		}
	}
	return count, nil
}

func (m *MemoryRedis) Expire(_ string, _ time.Duration) (bool, error) {
	return true, nil // noop for memory
}

func (m *MemoryRedis) LPush(key string, values ...interface{}) (int64, error) {
	return int64(len(values)), nil
}

func (m *MemoryRedis) RPush(key string, values ...interface{}) (int64, error) {
	return int64(len(values)), nil
}

func (m *MemoryRedis) LRange(_ string, _, _ int64) ([]string, error) {
	return nil, nil
}

func (m *MemoryRedis) HSet(key, field string, value interface{}) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.hash[key]; !ok {
		m.hash[key] = make(map[string]string)
	}
	m.hash[key][field] = fmt.Sprintf("%v", value)
	return 1, nil
}

func (m *MemoryRedis) HGet(key, field string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if h, ok := m.hash[key]; ok {
		if v, ok := h[field]; ok {
			return v, nil
		}
	}
	return "", fmt.Errorf("field not found")
}

func (m *MemoryRedis) HGetAll(key string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if h, ok := m.hash[key]; ok {
		return h, nil
	}
	return map[string]string{}, nil
}

func (m *MemoryRedis) Close() error { return nil }
func (m *MemoryRedis) Ping() error  { return nil }
