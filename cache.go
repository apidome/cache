package cache

import (
	"sync"
	"time"
)

// -----------------------------------------

type Cache interface {
	// Store a value permanently.
	Store(key, val interface{}) error

	// Get a value.
	Get(key interface{}) (interface{}, error)

	// Remove a value.
	Remove(key interface{}) error

	// Replace a value.
	Replace(key, val interface{}) error

	// Clears the cache.
	Clear() error

	// Get all keys from the cache.
	Keys() ([]interface{}, error)
}

type ExpiringCache interface {
	Cache

	// Store a value that will be removed after the specified ttl.
	StoreWithExpiration(key, val interface{}, ttl time.Duration) error

	// Replaces the value of a key.
	ReplaceWithExpiration(key, val interface{}, ttl time.Duration) error

	// Expire resets and updates the ttl of a value.
	Expire(key interface{}, ttl time.Duration) error
}

type UpdatingCache interface {
	Cache

	// Stores a value and repeatedly updates it.
	StoreWithUpdate(key, initialValue interface{},
		updateFunc func(currValue interface{}) interface{},
		period time.Duration) error

	// Replaces a value and repeatedly updates it.
	ReplaceWithUpdate(key, initialValue interface{},
		updateFunc func(currValue interface{}) interface{},
		period time.Duration) error
}

type UpdatingExpiringCache interface {
	UpdatingCache
	ExpiringCache
}

// -----------------------------------------

type timedMessage string

const (
	abort   timedMessage = "Abort"
	proceed              = "Proceed"
)

// -----------------------------------------

type cacheChannel struct {
	c    chan timedMessage
	once sync.Once
}

func newCacheChannel() *cacheChannel {
	return &cacheChannel{
		c: make(chan timedMessage),
	}
}

func (cc *cacheChannel) signal(msg timedMessage) {
	cc.once.Do(func() {
		cc.c <- msg
		close(cc.c)
	})
}

// -----------------------------------------

type cacheError struct {
	msg         string
	errType     errorType
	nestedError error
}

func (ce cacheError) Error() string {
	return ce.msg
}

type errorType string

const (
	errorTypeUnexpectedError   errorType = "UnexpectedError"
	errorTypeAlreadyExists               = "AlreadyExists"
	errorTypeDoesNotExist                = "DoesNotExist"
	errorTypeNonPositivePeriod           = "NonPositivePeriod"
	errorTypeNilUpdateFunc               = "NilUpdateFunc"
	errorTypeInvalidKeyType              = "InvalidKeyType"
	errorTypeInvalidMessage              = "InvalidMessage"
)

func newError(errType errorType, msg string) cacheError {
	return cacheError{
		msg:     msg,
		errType: errType,
	}
}

func newWrapperError(errType errorType, msg string, nestedError error) cacheError {
	return cacheError{
		msg:         msg,
		errType:     errType,
		nestedError: nestedError,
	}
}

func IsUnexpectedError(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeUnexpectedError
}

func IsAlreadyExists(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeAlreadyExists
}

func IsDoesNotExist(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeDoesNotExist
}

func IsNonPositivePeriod(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeNonPositivePeriod
}

func IsNilUpdateFunc(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeNilUpdateFunc
}

func IsInvalidKeyType(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeInvalidKeyType
}

func IsInvalidMessage(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeInvalidMessage
}

// -----------------------------------------
