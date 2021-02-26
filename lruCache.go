import (
	"sync"
	"container/list"
)

type hashItem struct {
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

	// Create a new node at the back of the linked list,
	node := lru.list.PushBack(key)

	// Store the new value.
	item := hashItem{val, node}

	// Store the new item in the hash map cache.
	err := lru.hash.Store(key, item)

	// If storing the value failed, remove the linked list node.
	if err != nil {
		lru.list.Remove(lru.list.Back())
		// TODO: return wrapped error
	}

	return nil
}

func (lru *LRU) get(key interface{}) (interface{}, error) {
	item, err := lru.hash.Get(key)
	if err != nil {
		// TODO: return wrapped error
	}

	// Move the item to the head of the linked list.
	lru.list.MoveToFront(item.node)
	return item.value, nil
}

func (lru *LRU) getLeastRecentlyUsed() (interface{}, error) {
	return lru.hash.Get(lru.list.Front())
}

func (lru *LRU) remove(key interface{}) error {
	item, err := lru.hash.Get(key)
	if err != nil {
		// TODO: return wrapped error
	}

	err = lru.hash.Remove(key)
	if err != nil {
		// TODO: return wrapped error
	}

	lru.list.Remove(item.node)
	return nil
}

func (lru *LRU) clear() error {
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

func (lru *LRU) Keys() ([]interface{}, error) {
	return lru.hash.Keys()
}