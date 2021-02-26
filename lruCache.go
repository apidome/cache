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
	hash UpdatingExpiringCache

	// A doubly linked list that represents the order of the items,
	// from the least recently used to the most recently used.
	list *list.List

	mutex sync.Mutex
}

// NewLruCache creates a new lruCache instance.
func NewLruCache(capacity int) *lruCache {
	return &lruCache{
		capacity: capacity,
		hash:     NewMapCache(),
		list:     list.New(),
	}
}

func (lru *lruCache) store(key, val interface{}) error {
	keys, err := lru.hash.Keys()
	if err != nil {
		// TODO: return wrapped error
	}

	numberOfCachedItems := len(keys)

	// If the cache is full, rmove the most recently used item.
	if numberOfCachedItems == lru.capacity {
		err := lru.hash.Remove(lru.list.Back().Value)
		if err != nil {
			// TODO: return wrapped error
		}
	}

	// Create a new node at the back of the linked list.
	node := lru.list.PushBack(key)

	// Store the new value.
	item := hashItem{val, node}

	// Store the new item in the hash map cache.
	err = lru.hash.Store(key, item)

	// If storing the value failed, remove the linked list node.
	if err != nil {
		lru.list.Remove(lru.list.Back())
		// TODO: return wrapped error
	}

	return nil
}

func (lru *lruCache) get(key interface{}) (interface{}, error) {
	item, err := lru.hash.Get(key)
	if err != nil {
		// TODO: return wrapped error
	}

	hashItem, _ := item.(hashItem)

	// Move the item to the head of the linked list.
	lru.list.MoveToFront(hashItem.node)
	return hashItem.value, nil
}

func (lru *lruCache) getLeastRecentlyUsed() (interface{}, error) {
	return lru.hash.Get(lru.list.Front())
}

func (lru *lruCache) remove(key interface{}) error {
	item, err := lru.hash.Get(key)
	if err != nil {
		// TODO: return wrapped error
	}

	err = lru.hash.Remove(key)
	if err != nil {
		// TODO: return wrapped error
	}

	hashItem, _ := item.(hashItem)

	lru.list.Remove(hashItem.node)
	return nil
}

func (lru *lruCache) clear() error {
	err := lru.hash.Clear()
	if err != nil {
		// TODO: return wrapped error
	}

	// Remove all nodes from linked list.
	for node := lru.list.Front(); node != nil; node = node.Next() {
		lru.list.Remove(node)
	}

	return nil
}

func (lru *lruCache) Keys() ([]interface{}, error) {
	return lru.hash.Keys()
}
