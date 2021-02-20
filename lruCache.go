import (
	"sync"
	"container/list"
)

type LRU struct {
	capacity int
	hash     map[interface{}]interface{}
	queue    *List
	mutex    sync.Mutex
}

func NewLRU(capacity int) *LRU {
	return &LRU{
		capacity,
		make(map[interface{}]interface{}, capacity),
		list.New(),
	}
}

func (lru *LRU) Store(key, val interface{}) error {
	return nil
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