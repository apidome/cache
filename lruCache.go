package cache

import (
	"container/list"
	"sync"
)

type hashItem struct {
	value interface{}
	node  *list.Element
}

type lruCache struct {
	// The maximal amount of cached items.
	capacity int

	// A hash map that holds tha actual cached data.
	hash *mapCache

	// A doubly linked list that represents the order of the items,
	// from the least recently used to the most recently used.
	list *list.List

	mutex sync.Mutex
}

var _ Cache = (*lruCache)(nil)

// NewLruCache creates a new lruCache instance.
func NewLruCache(capacity int) *lruCache {
	return &lruCache{
		capacity: capacity,
		hash:     NewMapCache(),
		list:     list.New(),
	}
}

// Cache a new value.
func (lru *lruCache) Store(key, val interface{}) error {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.store(key, val)
}

func (lru *lruCache) store(key, val interface{}) error {
	keys, err := lru.hash.Keys()
	if err != nil {
		return err
	}

	numberOfCachedItems := len(keys)

	// If the cache is full, rmove the most recently used item.
	if numberOfCachedItems == lru.capacity {
		err := lru.hash.Remove(lru.list.Back().Value)
		if err != nil {
			return err
		}
	}

	// Create a new node at the front of the linked list.
	node := lru.list.PushFront(key)

	// Store the new value.
	item := hashItem{val, node}

	// Store the new item in the hash map cache.
	err = lru.hash.Store(key, item)

	// If storing the value failed, remove the linked list node.
	if err != nil {
		lru.list.Remove(lru.list.Back())
		return err
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
	item, err := lru.hash.Get(key)
	if err != nil {
		return nil, err
	}

	hashItem, _ := item.(hashItem)

	// Move the item to the head of the linked list.
	lru.list.MoveToFront(hashItem.node)
	return hashItem.value, nil
}

func (lru *lruCache) getLeastRecentlyUsed() (interface{}, error) {
	return lru.hash.Get(lru.list.Front())
}

// Remove a cahced value.
func (lru *lruCache) Remove(key interface{}) error {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.remove(key)
}

func (lru *lruCache) remove(key interface{}) error {
	item, err := lru.hash.Get(key)
	if err != nil {
		return err
	}

	err = lru.hash.Remove(key)
	if err != nil {
		return err
	}

	hashItem, _ := item.(hashItem)

	lru.list.Remove(hashItem.node)
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
	err := lru.hash.Clear()
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
	return lru.hash.Keys()
}
