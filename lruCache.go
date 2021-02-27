package cache

import (
	"container/list"
	"sync"
)

type lruItem struct {
	// The cached data.
	value interface{}

	// A reference to the corresponding element in the linked list.
	node *list.Element
}

type lruCache struct {
	// The maximal amount of cached items.
	capacity int

	// // Current number of cached items.
	numberOfItems int

	// A cache that holds tha data.
	storage Cache

	// A doubly linked list that represents the order of the items,
	// from the most recently used to the least recently used.
	list *list.List

	mutex sync.Mutex
}

var _ Cache = (*lruCache)(nil)

// NewLru creates a new lruCache instance using mapCache.
func NewLru(capacity int) *lruCache {
	return &lruCache{
		capacity: capacity,
		storage:  NewMapCache(),
		list:     list.New(),
	}
}

// NewLruWithCustomCache creates a new lruCache with custom cache.
func NewLruWithCustomCache(capacity int, cache Cache) (*lruCache, error) {
	keys, err := cache.Keys()
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, newError(errorTypeCacheNotEmpty, "supplied cache must be empty")
	}

	return &lruCache{
		capacity: capacity,
		storage:  cache,
		list:     list.New(),
	}, nil
}

// Cache a new value.
func (lru *lruCache) Store(key, val interface{}) error {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.store(key, val)
}

func (lru *lruCache) store(key, val interface{}) error {
	// Create a new node at the front of the linked list.
	node := lru.list.PushFront(key)

	// Store the new value.
	item := lruItem{val, node}

	// Store the new item in the hash map cache.
	err := lru.storage.Store(key, item)

	// If storing the value failed, remove the linked list node.
	if err != nil {
		lru.list.Remove(lru.list.Back())
		return err
	}

	// If the cache is full, remove the least recently used item.
	if lru.numberOfItems == lru.capacity {
		err := lru.storage.Remove(lru.list.Back().Value)
		if err != nil {
			return err
		}
	} else {
		// Count the new item.
		lru.numberOfItems++
	}

	return nil
}

// Get a cached value.
func (lru *lruCache) Get(key interface{}) (interface{}, error) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.get(key)
}

func (lru *lruCache) get(key interface{}) (interface{}, error) {
	item, err := lru.storage.Get(key)
	if err != nil {
		return nil, err
	}

	lruItem, _ := item.(lruItem)

	// Move the item to the head of the linked list.
	lru.list.MoveToFront(lruItem.node)

	return lruItem.value, nil
}

// GetMostRecentlyUsedKey returns the key from the front of the linked list.
func (lru *lruCache) GetMostRecentlyUsedKey() interface{} {
	return lru.list.Front().Value
}

// GetLeastRecentlyUsedKey returns the key from the back of the linked list.
func (lru *lruCache) GetLeastRecentlyUsedKey() interface{} {
	return lru.list.Back().value
}

// Remove a cahced value.
func (lru *lruCache) Remove(key interface{}) error {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.remove(key)
}

func (lru *lruCache) remove(key interface{}) error {
	item, err := lru.storage.Get(key)
	if err != nil {
		return err
	}

	err = lru.storage.Remove(key)
	if err != nil {
		return err
	}

	lruItem, _ := item.(lruItem)
	lru.list.Remove(lruItem.node)
	lru.numberOfItems--

	return nil
}

// Replace a cached value.
func (lru *lruCache) Replace(key, val interface{}) error {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.replace(key, val)
}

func (lru *lruCache) replace(key, val interface{}) error {
	err := lru.remove(key)
	if err != nil {
		return err
	}

	err = lru.store(key, val)
	if err != nil {
		return err
	}

	return nil
}

// Clear all values from lru cache.
func (lru *lruCache) Clear() error {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.clear()
}

func (lru *lruCache) clear() error {
	err := lru.storage.Clear()
	if err != nil {
		return err
	}

	// Remove all nodes from linked list.
	for node := lru.list.Front(); node != nil; node = node.Next() {
		lru.list.Remove(node)
	}

	return nil
}

// Get all keys.
func (lru *lruCache) Keys() ([]interface{}, error) {
	return lru.storage.Keys()
}

// Count return the number of cached items,
func (lru *lruCache) Count() int {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.count()
}

func (lru *lruCache) count() int {
	return lru.numberOfItems
}

// IsFull returns true if cache is full.
func (lru *lruCache) IsFull() bool {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.isFull()
}

func (lru *lruCache) isFull() bool {
	return lru.capacity == lru.numberOfItems
}

// IsEmpty return false if cache is empty.
func (lru *lruCache) IsEmpty() bool {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.isEmpty()
}

func (lru *lruCache) isEmpty() bool {
	return lru.numberOfItems == 0
}
