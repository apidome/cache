package cache

import (
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	client *redis.Client
}

var _ (UpdatingExpiringCache) = (*RedisCache)(nil)

func NewRedisCache() *RedisCache {
	return nil
}

func (r *RedisCache) store(key, val interface{}) error {
	return nil
}

func (r *RedisCache) get(key interface{}) (interface{}, error) {
	return nil, nil
}

func (r *RedisCache) remove(key interface{}) error {
	return nil
}

func (r *RedisCache) replace(key, val interface{}) error {
	return nil
}

func (r *RedisCache) clear() error {
	return nil
}

func (r *RedisCache) keys() ([]interface{}, error) {
	return nil, nil
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

func (r *RedisCache) Store(key, val interface{}) error {
	return r.store(key, val)
}

func (r *RedisCache) Get(key interface{}) (interface{}, error) {
	return r.get(key)
}

func (r *RedisCache) Remove(key interface{}) error {
	return r.remove(key)
}

func (r *RedisCache) Replace(key, val interface{}) error {
	return r.replace(key, val)
}

func (r *RedisCache) Clear() error {
	return r.clear()
}

func (r *RedisCache) Keys() ([]interface{}, error) {
	return r.keys()
}

func (r *RedisCache) StoreWithExpiration(key, val interface{}, ttl time.Duration) error {
	return r.storeWithExpiration(key, val, ttl)
}

func (r *RedisCache) ReplaceWithExpiration(key, val interface{}, ttl time.Duration) error {
	return r.replaceWithExpiration(key, val, ttl)
}

func (r *RedisCache) Expire(key interface{}, ttl time.Duration) error {
	return r.expire(key, ttl)
}

func (r *RedisCache) StoreWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	return r.storeWithUpdate(key, initialValue, updateFunc, period)
}

func (r *RedisCache) ReplaceWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	return r.replaceWithUpdate(key, initialValue, updateFunc, period)
}
