package cache

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sync"
	"time"
)

// -----------------------------------------

const (
	errorTypeUrecoverableValue errorType = "UnrecoverableValue"
	errorTypeInvalidValueType            = "InvalidValueType"
	errorTypeNilValue                    = "NilValue"
	errorTypeClearedCache                = "ClearedCache"
)

func IsUnrecoverableValue(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeUrecoverableValue
}

func IsInvalidValueType(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeInvalidValueType
}

func IsNilValue(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeNilValue
}

func IsClearedCache(err error) bool {
	cacheErr, isCacheErr := err.(cacheError)
	return isCacheErr && cacheErr.errType == errorTypeClearedCache
}

// -----------------------------------------

type directoryCache struct {
	// Directory to store value files.
	cacheDir string

	// Holds the channels that stop the auto removal routines.
	removeChannels map[string]*cacheChannel

	// Holds the channels that stop the auto update routines.
	updateChannels map[string]*cacheChannel

	// Holds pointers to stored structs to allow recovery from a file.
	valueTypes map[string]reflect.Type

	// Indication if the cache was cleared, if it was, it should not be usable.
	cleared bool

	mutex sync.Mutex
}

var _ UpdatingExpiringCache = (*directoryCache)(nil)

// Create a new Cache object that is backed up by a directory.
//
// If dir does not exist, it will be created.
func NewDirectoryCache(dir string) (*directoryCache, error) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.Mkdir(dir, os.ModeDir)
		if err != nil {
			return nil, err
		}

		err = os.Chmod(dir, 0700)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &directoryCache{
		cacheDir:       dir,
		removeChannels: map[string]*cacheChannel{},
		updateChannels: map[string]*cacheChannel{},
		valueTypes:     map[string]reflect.Type{},
	}, nil
}

// Store a permanent value in the cache.
func (dc *directoryCache) Store(key, val interface{}) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.store(key, val)
}

func (dc *directoryCache) store(key, val interface{}) error {
	if dc.cleared {
		return newError(errorTypeClearedCache, "cannot reuse a cleared cache")
	}

	if err := dc.verifyInputs(key, val); err != nil {
		return err
	}

	strKey := key.(string)

	if dc.fileExists(key) {
		return newError(errorTypeAlreadyExists,
			fmt.Sprintf("key file [%s] already exists", strKey))
	}

	err := dc.writeValueToFile(val, strKey)
	if err != nil {
		return err
	}

	dc.valueTypes[strKey] = reflect.TypeOf(val)

	return nil
}

// Get a value from the cache.
func (dc *directoryCache) Get(key interface{}) (interface{}, error) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.get(key)
}

func (dc *directoryCache) get(key interface{}) (interface{}, error) {
	if dc.cleared {
		return nil, newError(errorTypeClearedCache,
			"cannot reuse a cleared cache")
	}

	err := dc.verifyKey(key)
	if err != nil {
		return nil, err
	}

	if !dc.fileExists(key) {
		return nil, newError(errorTypeDoesNotExist,
			fmt.Sprintf("key [%s] does not exist", key.(string)))
	}

	return dc.readValueFromFile(key)
}

// Remove a value from the cache.
func (dc *directoryCache) Remove(key interface{}) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.remove(key)
}

// If fromRoutine is true, it will not send on the channels to prevent deadlock
func (dc *directoryCache) remove(key interface{}) error {
	if dc.cleared {
		return newError(errorTypeClearedCache, "cannot reuse a cleared cache")
	}

	err := dc.verifyKey(key)
	if err != nil {
		return err
	}

	if !dc.fileExists(key) {
		return newError(errorTypeDoesNotExist,
			fmt.Sprintf("key [%s] does not exist",
				key.(string)))
	}

	strKey := key.(string)

	err = os.Remove(path.Join(dc.cacheDir, strKey))
	if err != nil {
		return err
	}

	c, exists := dc.removeChannels[strKey]
	if exists && c != nil {
		c.signal(abort)
		delete(dc.removeChannels, strKey)
	}

	c, exists = dc.updateChannels[strKey]
	if exists && c != nil {
		c.signal(abort)
		delete(dc.updateChannels, strKey)
	}

	return nil
}

// Replace a value in the cache with a permanent value.
func (dc *directoryCache) Replace(key, val interface{}) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.replace(key, val)
}

func (dc *directoryCache) replace(key, val interface{}) error {
	err := dc.remove(key)
	if err != nil {
		return err
	}

	err = dc.store(key, val)
	if err != nil {
		return err
	}

	return nil
}

// Clears the cache, cache should not be used again once it has been cleared.
func (dc *directoryCache) Clear() error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.clear()
}

func (dc *directoryCache) clear() error {
	if dc.cleared {
		return newError(errorTypeClearedCache, "cannot reuse a cleared cache")
	}

	files, err := ioutil.ReadDir(dc.cacheDir)
	if err != nil {
		return err
	}

	for _, finf := range files {
		err = dc.remove(finf.Name())
		if err != nil && !IsDoesNotExist(err) {
			return err
		}
	}

	err = os.RemoveAll(dc.cacheDir)
	if err != nil {
		return err
	}

	dc.cleared = true

	return nil
}

// Get all keys in the cache.
func (dc *directoryCache) Keys() ([]interface{}, error) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.keys()
}

func (dc *directoryCache) keys() ([]interface{}, error) {
	if dc.cleared {
		return nil, newError(errorTypeClearedCache, "cannot reuse a cleared cache")
	}

	files, err := ioutil.ReadDir(dc.cacheDir)
	if err != nil {
		return nil, err
	}

	keys := []interface{}{}
	for _, file := range files {
		keys = append(keys, file.Name())
	}

	return keys, nil
}

// Stores a temporary value in the cache, ttl must be greater than zero.
func (dc *directoryCache) StoreWithExpiration(key, val interface{},
	ttl time.Duration) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.storeWithExpiration(key, val, ttl)
}

func (dc *directoryCache) storeWithExpiration(key, val interface{},
	ttl time.Duration) error {
	if ttl <= 0 {
		return newError(errorTypeNonPositivePeriod,
			"period must be greater than zero")
	}

	err := dc.store(key, val)
	if err != nil {
		return err
	}

	keyStr := key.(string)
	c := newCacheChannel()
	dc.removeChannels[keyStr] = c

	expireSignalerRoutine := func(c *cacheChannel) {
		<-time.After(ttl)
		c.signal(proceed)
	}

	expireRoutine := func(key string, c *cacheChannel) {
		msg, ok := <-c.c
		if !ok || msg == abort {
			return
		} else {
			dc.mutex.Lock()
			defer dc.mutex.Unlock()

			if dc.cleared {
				return
			}

			// Delete the file from the directory
			err := dc.remove(key)
			if err != nil {
				panic(newWrapperError(errorTypeUnexpectedError,
					"an unexpected error occurred a background routine", err))
			}
		}
	}

	go expireSignalerRoutine(c)
	go expireRoutine(keyStr, c)

	return nil
}

// Replaces a value in the map with a temporary one, ttl must be greater than zero.
func (dc *directoryCache) ReplaceWithExpiration(key, val interface{},
	ttl time.Duration) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.replaceWithExpiration(key, val, ttl)
}

func (dc *directoryCache) replaceWithExpiration(key, val interface{},
	ttl time.Duration) error {
	if ttl <= 0 {
		return newError(errorTypeNonPositivePeriod,
			"period must be greater than zero")
	}

	err := dc.remove(key)
	if err != nil {
		return err
	}

	err = dc.storeWithExpiration(key, val, ttl)
	if err != nil {
		return err
	}

	return nil
}

// Expire a value in the cache, ttl must be greater than zero.
func (dc *directoryCache) Expire(key interface{}, ttl time.Duration) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.expire(key, ttl)
}

func (dc *directoryCache) expire(key interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		return newError(errorTypeNonPositivePeriod,
			"period must be greater than zero")
	}

	val, err := dc.get(key)
	if err != nil {
		return err
	}

	err = dc.remove(key)
	if err != nil {
		return err
	}

	err = dc.storeWithExpiration(key, val, ttl)
	if err != nil {
		return err
	}

	return nil
}

// Stores an updating value in the map, period must be greater than zero.
func (dc *directoryCache) StoreWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.storeWithUpdate(key, initialValue, updateFunc, period)
}

func (dc *directoryCache) storeWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	if period <= 0 {
		return newError(errorTypeNonPositivePeriod,
			"period must be greater than zero")
	}

	err := dc.store(key, initialValue)
	if err != nil {
		return err
	}

	keyStr := key.(string)
	c := newCacheChannel()
	dc.updateChannels[keyStr] = c

	updateSignalerRoutine := func(c *cacheChannel) {
		<-time.After(period)
		c.signal(proceed)
	}

	updateRoutine := func(key string, c *cacheChannel) {
		msg, ok := <-c.c
		if !ok || msg == abort {
			return
		} else {
			dc.mutex.Lock()
			defer dc.mutex.Unlock()

			if dc.cleared {
				return
			}

			// Update the value using the update func
			currVal, err := dc.get(key)
			if err != nil {
				panic(newWrapperError(errorTypeUnexpectedError,
					"an unexpected error occurred a background routine", err))
			}

			err = dc.remove(key)
			if err != nil {
				panic(newWrapperError(errorTypeUnexpectedError,
					"an unexpected error occurred a background routine", err))
			}

			err = dc.storeWithUpdate(key, updateFunc(currVal),
				updateFunc, period)
			if err != nil {
				panic(newWrapperError(errorTypeUnexpectedError,
					"an unexpected error occurred a background routine", err))
			}
		}
	}

	go updateSignalerRoutine(c)
	go updateRoutine(keyStr, c)

	return nil
}

// Replace a value with a continously updating value.
func (dc *directoryCache) ReplaceWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	return dc.replaceWithUpdate(key, initialValue, updateFunc, period)
}

func (dc *directoryCache) replaceWithUpdate(key, initialValue interface{},
	updateFunc func(currValue interface{}) interface{},
	period time.Duration) error {
	if updateFunc == nil {
		return newError(errorTypeNilUpdateFunc, "updateFunc cannot be nil")
	}

	if period <= 0 {
		return newError(errorTypeNonPositivePeriod,
			"period must be greater than zero")
	}

	err := dc.remove(key)
	if err != nil {
		return err
	}

	err = dc.storeWithUpdate(key, initialValue, updateFunc, period)
	if err != nil {
		return err
	}

	return nil
}

func (dc *directoryCache) verifyKey(key interface{}) error {
	_, isStr := key.(string)
	if !isStr {
		return newError(errorTypeInvalidKeyType,
			fmt.Sprintf("invalid key type, expected: [string] found: [%s]",
				reflect.TypeOf(key).Name()))
	}

	return nil
}

func (dc *directoryCache) verifyValue(val interface{}) error {
	// TODO allow struct pointers as well
	if reflect.TypeOf(val).Kind() != reflect.Struct &&
		reflect.TypeOf(val).Kind() != reflect.Map &&
		reflect.TypeOf(val).Kind() != reflect.Slice &&
		reflect.TypeOf(val).Kind() != reflect.Array {
		return newError(errorTypeInvalidValueType,
			fmt.Sprintf("invalid value type, expected either of:"+
				" [Struct, Map, Slice, Array] found: [%s]",
				reflect.TypeOf(val).String()))
	}

	jsonData, err := json.Marshal(val)
	if err != nil {
		return err
	}

	tmpVal := reflect.New(reflect.TypeOf(val)).Interface()
	err = json.Unmarshal(jsonData, tmpVal)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(val, reflect.ValueOf(tmpVal).Elem().Interface()) {
		return newError(errorTypeUrecoverableValue,
			"value cannot be fully recovered after marshalled to json,"+
				" make sure val's type has json tags")
	}

	return nil
}

func (dc *directoryCache) verifyInputs(key, val interface{}) error {
	err := dc.verifyKey(key)
	if err != nil {
		return err
	}

	err = dc.verifyValue(val)
	if err != nil {
		return err
	}

	return nil
}

func (dc *directoryCache) fileExists(key interface{}) bool {
	_, err := os.Stat(path.Join(dc.cacheDir, key.(string)))
	return err == nil
}

func (dc *directoryCache) writeValueToFile(val interface{}, strKey string) error {
	fileName := path.Join(dc.cacheDir, strKey)

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	err = os.Chmod(fileName, 0600)
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(val)
	if err != nil {
		return err
	}

	_, err = file.Write(jsonData)
	if err != nil {
		return err
	}

	err = file.Sync()
	if err != nil {
		return err
	}

	return nil
}

func (dc *directoryCache) readValueFromFile(key interface{}) (interface{}, error) {
	fileName := path.Join(dc.cacheDir, key.(string))
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return nil, newError(errorTypeDoesNotExist,
			fmt.Sprintf("file for key [%s] does not exist", key.(string)))
	} else if err == nil {
		file, err := os.Open(fileName)
		if err != nil {
			return nil, err
		}

		jsonData, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}

		valStruct := reflect.New(dc.valueTypes[key.(string)]).Interface()

		err = json.Unmarshal(jsonData, valStruct)
		if err != nil {
			return nil, err
		}

		return reflect.Indirect(reflect.ValueOf(valStruct)).Interface(), nil
	} else {
		return nil, err
	}
}

// -----------------------------------------
