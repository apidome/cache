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
	has      *mapCache
	queue    *List
	mutex    sync.Mutex
}

func NewLRU(capacity int) *LRU {
	return &LRU{
		capacity,
		NewMapCache(),
		list.New(),
	}
}

func (lru *LRU) Store(key, val interface{}) error {
	numberOfCachedItems := len(lru.hash.Keys())
	if numberOfCachedItems == lru.capacity {
		err := lru.Remove(lru.queue.Back().Value)
		if err != nil {
			// TODO: return wrapped error
		}
	}

}

func (lru *LRU) Get(key interface{}) (interface{}, error) {
	return nil, nil
}

func (lru *LRU) Remove(key interface{}) error {
	return nil
}

func Clear() error {
	return nil
}

func Keys() ([]interface{}, error) {
	return nil, nil
}

func updateHead() {}

func updateTail() {}