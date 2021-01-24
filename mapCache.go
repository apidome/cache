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
	removeRoutineKillers map[interface{}]chan struct{}

	// Holds the channels that stop the auto update routines.
	updateRoutineKillers map[interface{}]chan struct{}

	mutex sync.Mutex
}

var _ UpdatingExpiringCache = (*mapCache)(nil)

// Create a new Cache object that is backed by a map.
func NewMapCache() UpdatingExpiringCache {
	return &mapCache{
		cacheMap:             map[interface{}]interface{}{},
		removeRoutineKillers: map[interface{}]chan struct{}{},
		updateRoutineKillers: map[interface{}]chan struct{}{},
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
	if _, exists := m.cacheMap[key]; !exists {
		return newError(errorTypeDoesNotExist,
			fmt.Sprintf("key %v doesn't exist", key))
	}

	if m.removeRoutineKillers[key] != nil {
		close(m.removeRoutineKillers[key])
		m.removeRoutineKillers[key] = nil
	}

	if m.updateRoutineKillers[key] != nil {
		close(m.updateRoutineKillers[key])
		m.updateRoutineKillers[key] = nil
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

	m.removeRoutineKillers[key] = make(chan struct{})

	expirationRoutine := func() {
		select {
		case <-m.removeRoutineKillers[key]:
			// This routine might be killed if the value was modified by
			// 'Replace' or 'Remove'
			return
		case <-time.After(ttl):
			m.mutex.Lock()

			// Ignoring errors here because if the value was already
			// removed manually we shouldn't care
			delete(m.cacheMap, key)

			m.mutex.Unlock()
		}
	}

	go expirationRoutine()

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
	updateFunc func(currValue interface{}) interface{}, period time.Duration) error {
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

	m.updateRoutineKillers[key] = make(chan struct{})

	updateRoutine := func() {
		for {
			select {
			case <-m.updateRoutineKillers[key]:
				// This routine might be killed if the value was
				// modified by 'Replace' or 'Remove'
				return
			case <-time.After(period):
				m.mutex.Lock()

				// Update the value using the update func
				m.cacheMap[key] = updateFunc(m.cacheMap[key])

				m.mutex.Unlock()
			}
		}
	}

	go updateRoutine()

	return nil
}
