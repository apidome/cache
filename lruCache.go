import (
	"sync"
	"container/list"
)

type hasItem struct {
	value interface{}
	node  *list.Element
}

type LRU struct {
	capacity int
	hash     *mapCache
	list     *List
	mutex    sync.Mutex
}

func NewLRU(capacity int) *LRU {
	return &LRU{
		capacity,
		NewMapCache(),
		list.New(),
	}
}

func (lru *LRU) store(key, val interface{}) error {
	numberOfCachedItems := len(lru.hash.Keys())

	// If the cache is full, rmove the most recently used item.
	if numberOfCachedItems == lru.capacity {
		err := lru.hash.Remove(lru.queue.Back().Value)
		if err != nil {
			// TODO: return wrapped error
		}
	}

	// Store the new value.
	err := lru.hash.Store(key, val)
	if err != nil {
		// TODO: return wrapped error
	}

	// Put the new value in the back of the list.
	lru.list.PushBack(key)
	return nil
}

func (lru *LRU) get(key interface{}) (interface{}, error) {
	return nil, nil
}

func (lru *LRU) remove(key interface{}) error {
	return nil
}

func (lru *LRU) clear() error {
	return nil
}

func (lru *LRU) Keys() ([]interface{}, error) {
	return lru.hash.Keys()
}

func (lru *LRU) updateHead() {}

func (lru *LRU) updateTail() {}