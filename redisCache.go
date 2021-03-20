package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	errorTypeRedisError errorType = "RedisError"
)

// RedisCache is a client that implements Cache interface.
type RedisCache struct {
	client   *redis.Client
	keysList []string
}

var _ (ExpiringCache) = (*RedisCache)(nil)

// NewRedisCache creates and returns a reference to a RedisCache instance.
func NewRedisCache(address, password string, db int) *RedisCache {
	return &RedisCache{
		redis.NewClient(&redis.Options{
			Addr:     address,
			Password: password,
			DB:       db,
		}),
		[]string{},
	}
}

func (r *RedisCache) store(key, val interface{}) error {
	strKey := fmt.Sprintf("%v", key)
	status := r.client.Set(context.TODO(), strKey, val, 0)

	r.keysList = append(r.keysList, strKey)

}

func (r *RedisCache) get(key interface{}) (interface{}, error) {
	strKey := fmt.Sprintf("%v", key)

	val, err := r.client.Get(context.TODO(), strKey).Result()
	if err == redis.Nil {
		return nil, newError(errorTypeDoesNotExist,
			fmt.Sprintf("key %v doesn't exist", strKey))
	}

	if err != nil {
		return nil, newError(errorTypeRedisError,
			fmt.Sprintf("failed to get %v from redis", strKey))
	}

	return val, nil
}

func (r *RedisCache) remove(key interface{}) error {
	strKey := fmt.Sprintf("%v", key)

	res := r.client.Del(context.TODO(), strKey).Val()
	if res < 1 {
		return newError(errorTypeDoesNotExist,
			fmt.Sprintf("could not delete key %v", strKey))
	}

	return nil
}

func (r *RedisCache) replace(key, val interface{}) error {
	err := r.remove(key)
	if err != nil {
		return err
	}

	err = r.store(key, val)
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisCache) clear() error {
	for key := range r.keysList {
		err := r.remove(key)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RedisCache) keys() ([]interface{}, error) {
	return []interface(r.keysList), nil
}

func (r *RedisCache) storeWithExpiration(key, val interface{}, ttl time.Duration) error {
	return nil
}

func (r *RedisCache) replaceWithExpiration(key, val interface{}, ttl time.Duration) error {
	return nil
}

func (r *RedisCache) expire(key interface{}, ttl time.Duration) error {
	return nil
}

func (r *RedisCache) storeWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	return nil
}

func (r *RedisCache) replaceWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	return nil
}

// --------------------------------------------------------------------------

// Store permanent value in redis.
func (r *RedisCache) Store(key, val interface{}) error {
	return r.store(key, val)
}

// Get a value from redis.
func (r *RedisCache) Get(key interface{}) (interface{}, error) {
	return r.get(key)
}

// Remove a value from redis.
func (r *RedisCache) Remove(key interface{}) error {
	return r.remove(key)
}

// Replace an existing value in redis.
func (r *RedisCache) Replace(key, val interface{}) error {
	return r.replace(key, val)
}

// Clear all values that maintained by this RedisCache instance.
func (r *RedisCache) Clear() error {
	return r.clear()
}

// Keys return all keys that maintained by this RedisCache instance.
func (r *RedisCache) Keys() ([]interface{}, error) {
	return r.keys()
}

// StoreWithExpiration stores a key-value pair in redis for limited time.
func (r *RedisCache) StoreWithExpiration(key, val interface{}, ttl time.Duration) error {
	return r.storeWithExpiration(key, val, ttl)
}

// ReplaceWithExpiration replaces a key-value pair in redis for limited time.
func (r *RedisCache) ReplaceWithExpiration(key, val interface{}, ttl time.Duration) error {
	return r.replaceWithExpiration(key, val, ttl)
}

// Expire a key-value pair.
func (r *RedisCache) Expire(key interface{}, ttl time.Duration) error {
	return r.expire(key, ttl)
}
