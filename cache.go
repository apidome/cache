package cache

import (
	"fmt"
	"sync"
	"time"
)

// -------------------------------------------------------------------------------
type Cache interface {
	// Store a value for the specified duration.
	//
	// If duration is 0, the value will not be removed automatically.
	Store(key, val interface{}, timeout time.Duration) error

	// Remove a value.
	Remove(key interface{}) error

	// Fetch a value.
	Get(key interface{}) (interface{}, error)

	// Replaces the value of a key.
	//
	// If duration is 0, the value will not be removed automatically.
	Replace(key, val interface{}, timeout time.Duration) error

	// Expire resets and updates the life span of a value.
	//
	// If duration is 0, the value will be removed immediately
	Expire(key interface{}, duration time.Duration) error
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
	errorTypeAlreadyExists errorType = "AlreadyExists"
	errorTypeDoesntExist   errorType = "DoesntExist"
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

// -------------------------------------------------------------------------------

// -------------------------------------------------------------------------------
type mapCache struct {
	cacheMap    map[interface{}]interface{}
	keyChannels map[interface{}]chan struct{}
	mutex       sync.Mutex
}

var _ Cache = (*mapCache)(nil)

// Create a new Cache object that is backed by a map
func NewMapCache() Cache {
	return &mapCache{
		cacheMap:    map[interface{}]interface{}{},
		keyChannels: map[interface{}]chan struct{}{},
	}
}

func (m *mapCache) Store(key, val interface{}, duration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.cacheMap[key]; exists {
		return newError(errorTypeAlreadyExists, fmt.Sprintf("key %v is already in use", key))
	}

	m.cacheMap[key] = val

	if duration > 0 {
		m.runExpirationRoutine(key, duration)
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

func (m *mapCache) Get(key interface{}) (interface{}, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.cacheMap[key]; !exists {
		return nil, newError(errorTypeDoesntExist, fmt.Sprintf("key %v doesn't exist", key))
	}

	return m.cacheMap[key], nil
}

func (m *mapCache) Replace(key, value interface{}, duration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.cacheMap[key]
	if !exists {
		return newError(errorTypeDoesntExist, fmt.Sprintf("key %#v doesn't exist", key))
	}

	m.runExpirationRoutine(key, duration)
	m.cacheMap[key] = value

	return nil
}

func (m *mapCache) Expire(key interface{}, duration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, exists := m.cacheMap[key]
	if !exists {
		return newError(errorTypeDoesntExist, fmt.Sprintf("key %#v doesn't exist", key))
	}

	if duration > 0 {
		m.runExpirationRoutine(key, duration)
	} else {
		delete(m.cacheMap, key)
	}

	return nil
}

// -------------------------------------------------------------------------------
// -------------------------------------------------------------------------------
func (m *mapCache) killRoutineIfExists(key interface{}) {
	keyChan, exists := m.keyChannels[key]
	if exists {
		keyChan <- struct{}{}
		close(keyChan)
	}
}

func (m *mapCache) runExpirationRoutine(key interface{}, duration time.Duration) {
	m.killRoutineIfExists(key)
	m.keyChannels[key] = make(chan struct{})

	go func() {
		select {
		case <-m.keyChannels[key]:
			// This routine might be killed if the value was modified by 'Replace'
			return
		case <-time.After(duration):
			m.mutex.Lock()
			defer m.mutex.Unlock()

			// Ignoring errors here because if the value was already removed manually we shouldn't care
			delete(m.cacheMap, key)
		}
	}()
}

// -------------------------------------------------------------------------------
