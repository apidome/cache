package cache

import (
	"fmt"
	"sync"
	"time"
)

// -------------------------------------------------------------------------------
type Cache interface {
	// Store a value permanently.
	Store(key, val interface{}) error

	// Store a value that will be removed after the specified ttl.
	StoreWithExpiration(key, val interface{}, ttl time.Duration) error

	// StoreWithUpdate stores a value and repeatedly updates it.
	StoreWithUpdate(key, initialValue interface{},
		updateFunc func(currValue interface{}) interface{},
		period time.Duration) error

	// Fetch a value.
	Get(key interface{}) (interface{}, error)

	// Replaces the value of a key.
	Replace(key, val interface{}, ttl time.Duration) error

	// Expire resets and updates the ttl of a value.
	Expire(key interface{}, ttl time.Duration) error

	// Remove a value.
	Remove(key interface{}) error
}

// -------------------------------------------------------------------------------

// -------------------------------------------------------------------------------
type cacheError struct {
	msg     string
	errType errorType
}

func (ce cacheError) Error() string {
	return ce.msg
}

type errorType string

const (
	errorTypeAlreadyExists     errorType = "AlreadyExists"
	errorTypeDoesntExist       errorType = "DoesntExist"
	errorTypeNonPositivePeriod errorType = "NonPositivePeriod"
	errorTypeNilUpdateFunc     errorType = "NilUpdateFunc"
)

func newError(errType errorType, msg string) cacheError {
	return cacheError{
		msg:     msg,
		errType: errType,
	}
}

func IsAlreadyExistsError(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeAlreadyExists
}

func IsDoesntExistError(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeDoesntExist
}

func IsNonPositivePeriodError(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeNonPositivePeriod
}

func IsNilUpdateFuncError(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeNilUpdateFunc
}

// -------------------------------------------------------------------------------

// -------------------------------------------------------------------------------
type mapCache struct {
	cacheMap       map[interface{}]interface{}
	removeChannels map[interface{}]chan struct{}
	updateChannels map[interface{}]chan struct{}
	mutex          sync.Mutex
}

var _ Cache = (*mapCache)(nil)

// Create a new Cache object that is backed by a map
func NewMapCache() Cache {
	return &mapCache{
		cacheMap:       map[interface{}]interface{}{},
		removeChannels: map[interface{}]chan struct{}{},
		updateChannels: map[interface{}]chan struct{}{},
	}
}

func (m *mapCache) Store(key, val interface{}) error {
	return m.StoreWithExpiration(key, val, 0)
}

// If ttl is 0, the value is permanently stored.
func (m *mapCache) StoreWithExpiration(key, val interface{}, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.cacheMap[key]; exists {
		return newError(errorTypeAlreadyExists, fmt.Sprintf("key %v is already in use", key))
	}

	m.cacheMap[key] = val

	if ttl > 0 {
		m.runExpirationRoutine(key, ttl)
	}

	return nil
}

func (m *mapCache) StoreWithUpdate(key, initialValue interface{}, updateFunc func(currValue interface{}) interface{}, period time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.cacheMap[key]; exists {
		return newError(errorTypeAlreadyExists, fmt.Sprintf("key %v is already in use", key))
	}

	if updateFunc == nil {
		return newError(errorTypeNilUpdateFunc, "updateFunc cannot be nil")
	}

	if period <= 0 {
		return newError(errorTypeNonPositivePeriod, "period must be bigger than zero")
	}

	m.cacheMap[key] = initialValue
	m.runUpdateRoutine(key, updateFunc, period)

	return nil
}

func (m *mapCache) Get(key interface{}) (interface{}, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.cacheMap[key]; !exists {
		return nil, newError(errorTypeDoesntExist, fmt.Sprintf("key %v doesn't exist", key))
	}

	return m.cacheMap[key], nil
}

// If ttl is 0, the value will not be removed automatically.
func (m *mapCache) Replace(key, value interface{}, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.cacheMap[key]
	if !exists {
		return newError(errorTypeDoesntExist, fmt.Sprintf("key %#v doesn't exist", key))
	}

	m.runExpirationRoutine(key, ttl)
	m.cacheMap[key] = value

	return nil
}

// If ttl is 0, the value will be removed immediately.
func (m *mapCache) Expire(key interface{}, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.cacheMap[key]
	if !exists {
		return newError(errorTypeDoesntExist, fmt.Sprintf("key %#v doesn't exist", key))
	}

	if ttl > 0 {
		m.runExpirationRoutine(key, ttl)
	} else {
		delete(m.cacheMap, key)
	}

	return nil
}

func (m *mapCache) Remove(key interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.cacheMap[key]; !exists {
		return newError(errorTypeDoesntExist, fmt.Sprintf("key %v doesn't exist", key))
	}

	m.killRoutineIfExists(key)
	delete(m.cacheMap, key)

	return nil
}

// -------------------------------------------------------------------------------

// -------------------------------------------------------------------------------
func (m *mapCache) killRoutineIfExists(key interface{}) {
	keyChan, exists := m.removeChannels[key]
	if exists {
		keyChan <- struct{}{}
		close(keyChan)
	}

	keyChan, exists = m.updateChannels[key]
	if exists {
		keyChan <- struct{}{}
		close(keyChan)
	}
}

func (m *mapCache) runExpirationRoutine(key interface{}, ttl time.Duration) {
	m.killRoutineIfExists(key)
	m.removeChannels[key] = make(chan struct{})

	go func() {
		select {
		case <-m.removeChannels[key]:
			// This routine might be killed if the value was modified by 'Replace' or 'Remove'
			return
		case <-time.After(ttl):
			m.mutex.Lock()

			// Ignoring errors here because if the value was already removed manually we shouldn't care
			delete(m.cacheMap, key)

			m.mutex.Unlock()
		}
	}()
}

func (m *mapCache) runUpdateRoutine(key interface{}, updateFunc func(currValue interface{}) interface{}, period time.Duration) {
	m.killRoutineIfExists(key)
	m.updateChannels[key] = make(chan struct{})

	go func() {
		for {
			select {
			case <-m.updateChannels[key]:
				// This routine might be killed if the value was modified by 'Replace' or 'Remove'
				return
			case <-time.After(period):
				m.mutex.Lock()

				// Update the value using the update func
				m.cacheMap[key] = updateFunc(m.cacheMap[key])

				m.mutex.Unlock()
			}
		}
	}()
}

// -------------------------------------------------------------------------------
