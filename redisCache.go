package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	errorTypeRedisError errorType = "RedisError"
)

// RedisCache is a client that implements Cache interface.
type RedisCache struct {
	// This dictionary is maintained in order to keep track of this
	// instance's keys for functions like clear.
	keysSet map[string]struct{}

	// Holds the channels that stop the auto removal routines.
	removeChannels map[interface{}]*cacheChannel

	client *redis.Client

	mutex sync.Mutex
}

var _ (ExpiringCache) = (*RedisCache)(nil)

// --------------------------------------------------------------------------

// NewRedisCache creates and returns a reference to a RedisCache instance.
func NewRedisCache(address, password string, db int) *RedisCache {
	return &RedisCache{
		keysSet:        map[string]struct{}{},
		removeChannels: map[interface{}]*cacheChannel{},
		client: redis.NewClient(&redis.Options{
			Addr:     address,
			Password: password,
			DB:       db,
		}),
	}
}

func (r *RedisCache) store(key, val interface{}, ttl time.Duration) error {
	strKey := fmt.Sprintf("%v", key)
	err := r.client.Set(context.TODO(), strKey, val, ttl).Err()

	if err != nil {
		return newError(errorTypeRedisError, fmt.Sprintf("could not store key %v", strKey))
	}

	r.keysSet[strKey] = struct{}{}

	return nil
}

func (r *RedisCache) get(key interface{}) (interface{}, error) {
	strKey := fmt.Sprintf("%v", key)

	if _, ok := r.keysSet[strKey]; !ok {
		return nil, newError(errorTypeDoesNotExist,
			fmt.Sprintf("cannot get key %v", strKey))
	}

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

	if _, ok := r.keysSet[strKey]; !ok {
		return newError(errorTypeDoesNotExist,
			fmt.Sprintf("cannot remove key %v", strKey))
	}

	res := r.client.Del(context.TODO(), strKey).Val()
	if res < 1 {
		return newError(errorTypeDoesNotExist,
			fmt.Sprintf("could not delete key %v", strKey))
	}

	return nil
}

func (r *RedisCache) replace(key, val interface{}) error {
	return r.store(key, val, 0)
}

func (r *RedisCache) clear() error {
	for key := range r.keysSet {
		err := r.remove(key)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RedisCache) keys() ([]interface{}, error) {
	keys := []interface{}{}

	for key := range r.keysSet {
		keys = append(keys, key)
	}

	return keys, nil
}

func (r *RedisCache) storeWithExpiration(key, val interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		return newError(errorTypeNonPositivePeriod, "period must be greater than zero")
	}

	r.store(key, val, ttl)

	r.createExpirationRoutine(key, ttl)

	return nil
}

func (r *RedisCache) replaceWithExpiration(key, val interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		return newError(errorTypeNonPositivePeriod, "period must be greater than zero")
	}

	err := r.storeWithExpiration(key, val, ttl)
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisCache) expire(key interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		return newError(errorTypeNonPositivePeriod, "period must be greater than zero")
	}

	strKey := fmt.Sprintf("%v", key)

	res := r.client.Expire(context.TODO(), strKey, ttl).Val()
	if !res {
		return newError(errorTypeRedisError, fmt.Sprintf("could not expire key %v", strKey))
	}

	r.createExpirationRoutine(key, ttl)

	return nil
}

func (r *RedisCache) createExpirationRoutine(key interface{}, ttl time.Duration) {
	c := newCacheChannel()
	r.removeChannels[key] = c

	expireSignalerRoutine := func(c *cacheChannel) {
		<-time.After(ttl)
		c.signal(proceed)
	}

	expireRoutine := func(key interface{}, c *cacheChannel) {
		msg, ok := <-c.c
		if !ok || msg == abort {
			return
		}

		r.mutex.Lock()
		defer r.mutex.Unlock()

		delete(r.keysSet, fmt.Sprintf("%v", key))
	}

	go expireSignalerRoutine(c)
	go expireRoutine(key, c)
}

// --------------------------------------------------------------------------

// Store permanent value in redis.
func (r *RedisCache) Store(key, val interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.store(key, val, 0)
}

// Get a value from redis.
func (r *RedisCache) Get(key interface{}) (interface{}, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.get(key)
}

// Remove a value from redis.
func (r *RedisCache) Remove(key interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.remove(key)
}

// Replace an existing value in redis.
func (r *RedisCache) Replace(key, val interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.replace(key, val)
}

// Clear all values that maintained by this RedisCache instance.
func (r *RedisCache) Clear() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.clear()
}

// Keys return all keys that maintained by this RedisCache instance.
func (r *RedisCache) Keys() ([]interface{}, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.keys()
}

// StoreWithExpiration stores a key-value pair in redis for limited time.
func (r *RedisCache) StoreWithExpiration(key, val interface{}, ttl time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.storeWithExpiration(key, val, ttl)
}

// ReplaceWithExpiration replaces a key-value pair in redis for limited time.
func (r *RedisCache) ReplaceWithExpiration(key, val interface{}, ttl time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.replaceWithExpiration(key, val, ttl)
}

// Expire a key-value pair.
func (r *RedisCache) Expire(key interface{}, ttl time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.expire(key, ttl)
}
