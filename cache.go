package cache

import (
	"time"
)

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

	// StoreWithUpdate stores a value and repeatedly updates it.
	StoreWithUpdate(key, initialValue interface{},
		updateFunc func(currValue interface{}) interface{},
		period time.Duration) error
}

type UpdatingExpiringCache interface {
	UpdatingCache
	ExpiringCache
}

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
	errorTypeDoesNotExist                = "DoesNotExist"
	errorTypeNonPositivePeriod           = "NonPositivePeriod"
	errorTypeNilUpdateFunc               = "NilUpdateFunc"
	errorTypeInvalidKeyType              = "InvalidKeyType"
)

func newError(errType errorType, msg string) cacheError {
	return cacheError{
		msg:     msg,
		errType: errType,
	}
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
