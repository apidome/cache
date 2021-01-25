package cache

import (
	"fmt"
	"sync"
	"time"
)

type mapCache struct {
	// Holds the key/values in the cache
	cacheMap map[interface{}]interface{}

	// Holds the channels that stop the auto removal routines.
	removeChannels map[interface{}]*cacheChannel

	// Holds the channels that stop the auto update routines.
	updateChannels map[interface{}]*cacheChannel

	mutex sync.Mutex
}

var _ UpdatingExpiringCache = (*mapCache)(nil)

// Create a new Cache object that is backed by a map.
func NewMapCache() UpdatingExpiringCache {
	return &mapCache{
		cacheMap:       map[interface{}]interface{}{},
		removeChannels: map[interface{}]*cacheChannel{},
		updateChannels: map[interface{}]*cacheChannel{},
	}
}

// Store permanent value in the map.
func (m *mapCache) Store(key, val interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.store(key, val)
}

func (m *mapCache) store(key, val interface{}) error {
	if _, exists := m.cacheMap[key]; exists {
		return newError(errorTypeAlreadyExists,
			fmt.Sprintf("key %v is already in use", key))
	}

	m.cacheMap[key] = val

	return nil
}

// Get a value from the map.
func (m *mapCache) Get(key interface{}) (interface{}, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.get(key)
}

func (m *mapCache) get(key interface{}) (interface{}, error) {
	if _, exists := m.cacheMap[key]; !exists {
		return nil, newError(errorTypeDoesNotExist,
			fmt.Sprintf("key %v doesn't exist", key))
	}

	return m.cacheMap[key], nil
}

// Remove a value from the map.
func (m *mapCache) Remove(key interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.remove(key)
}

func (m *mapCache) remove(key interface{}) error {
	_, err := m.get(key)
	if err != nil {
		return err
	}

	c, exists := m.removeChannels[key]
	if exists && c != nil {
		c.signal(abort)
		delete(m.removeChannels, key)
	}

	c, exists = m.updateChannels[key]
	if exists && c != nil {
		c.signal(abort)
		delete(m.updateChannels, key)
	}

	delete(m.cacheMap, key)

	return nil
}

// Replace a value in the map.
func (m *mapCache) Replace(key, val interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.replace(key, val)
}

func (m *mapCache) replace(key, val interface{}) error {
	err := m.remove(key)
	if err != nil {
		return err
	}

	err = m.store(key, val)
	if err != nil {
		return err
	}

	return nil
}

// Clear the map.
func (m *mapCache) Clear() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.clear()
}

func (m *mapCache) clear() error {
	for key := range m.cacheMap {
		err := m.remove(key)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get cache keys.
func (m *mapCache) Keys() ([]interface{}, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.keys()
}

func (m *mapCache) keys() ([]interface{}, error) {
	keys := []interface{}{}

	for key := range m.cacheMap {
		keys = append(keys, key)
	}

	return keys, nil
}

// Store a temporary value in the map, ttl must be greater than zero.
func (m *mapCache) StoreWithExpiration(key, val interface{}, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.storeWithExpiration(key, val, ttl)
}

func (m *mapCache) storeWithExpiration(key, val interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		return newError(errorTypeNonPositivePeriod, "period must be greater than zero")
	}

	err := m.store(key, val)
	if err != nil {
		return err
	}

	c := newCacheChannel()
	m.removeChannels[key] = c

	expireSignalerRoutine := func(c *cacheChannel) {
		<-time.After(ttl)
		c.signal(proceed)
	}

	expireRoutine := func(key interface{}, c *cacheChannel) {
		msg, ok := <-c.c
		if !ok || msg == abort {
			return
		} else {
			m.mutex.Lock()
			defer m.mutex.Unlock()

			// Ignoring errors here because if the value was already
			// removed manually we shouldn't care
			delete(m.cacheMap, key)
		}
	}

	go expireSignalerRoutine(c)
	go expireRoutine(key, c)

	return nil
}

// Replace a value in the map with a temporary value, ttl must be greater than zero.
func (m *mapCache) ReplaceWithExpiration(key, val interface{},
	ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.replaceWithExpiration(key, val, ttl)
}

func (m *mapCache) replaceWithExpiration(key, val interface{},
	ttl time.Duration) error {
	if ttl <= 0 {
		return newError(errorTypeNonPositivePeriod,
			"period must be greater than zero")
	}

	err := m.remove(key)
	if err != nil {
		return err
	}

	err = m.storeWithExpiration(key, val, ttl)
	if err != nil {
		return err
	}

	return nil
}

// Update the expiration of a value in the map, ttl must be greater than zero.
func (m *mapCache) Expire(key interface{}, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.expire(key, ttl)
}

func (m *mapCache) expire(key interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		return newError(errorTypeNonPositivePeriod,
			"period must be greater than zero")
	}

	val, err := m.get(key)
	if err != nil {
		return err
	}

	err = m.remove(key)
	if err != nil {
		return err
	}

	err = m.storeWithExpiration(key, val, ttl)
	if err != nil {
		return err
	}

	return nil
}

// Store an updating value in the map.
func (m *mapCache) StoreWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{}, period time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.storeWithUpdate(key, initialValue, updateFunc, period)
}

func (m *mapCache) storeWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	if updateFunc == nil {
		return newError(errorTypeNilUpdateFunc, "updateFunc cannot be nil")
	}

	if period <= 0 {
		return newError(errorTypeNonPositivePeriod,
			"period must be bigger than zero")
	}

	err := m.store(key, initialValue)
	if err != nil {
		return err
	}

	c := newCacheChannel()
	m.updateChannels[key] = c

	updateSignalerRoutine := func(c *cacheChannel) {
		<-time.After(period)
		c.signal(proceed)
	}

	updateRoutine := func(key interface{}, c *cacheChannel) {
		msg, ok := <-c.c
		if !ok || msg == abort {
			return
		} else {
			m.mutex.Lock()
			defer m.mutex.Unlock()

			currVal, err := m.get(key)
			if err != nil {
				panic(newWrapperError(errorTypeUnexpectedError,
					"an unexpected error occurred a background routine", err))
			}

			err = m.remove(key)
			if err != nil {
				panic(newWrapperError(errorTypeUnexpectedError,
					"an unexpected error occurred a background routine", err))
			}

			err = m.storeWithUpdate(key, updateFunc(currVal), updateFunc, period)
			if err != nil {
				panic(newWrapperError(errorTypeUnexpectedError,
					"an unexpected error occurred a background routine", err))
			}
		}
	}

	go updateSignalerRoutine(c)
	go updateRoutine(key, c)

	return nil
}

// Replace a value with a continously updating value.
func (m *mapCache) ReplaceWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.replaceWithUpdate(key, initialValue, updateFunc, period)
}

func (m *mapCache) replaceWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	if updateFunc == nil {
		return newError(errorTypeNilUpdateFunc, "updateFunc cannot be nil")
	}

	if period <= 0 {
		return newError(errorTypeNonPositivePeriod,
			"period must be greater than zero")
	}

	err := m.remove(key)
	if err != nil {
		return err
	}

	err = m.storeWithUpdate(key, initialValue, updateFunc, period)
	if err != nil {
		return err
	}

	return nil
}
