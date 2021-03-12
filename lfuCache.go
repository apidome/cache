package cache

import (
	"container/heap"
	"sync"
)

type lfuHeapItem struct {
	// The item's value is the key of a specific
	// value stored in lfuCache.
	value interface{}

	// The amount of time that a certain key has been accessed.
	frequency int

	// The index of the item in the heap.
	// It is needed by update and is maintained by the
	// heap.Interface methods.
	index int
}

// A slice of lfuItems that behaves is a min heap.
type lfuHeap []*lfuHeapItem

var _ heap.Interface = (*lfuHeap)(nil)

func (h lfuHeap) Len() int {
	return len(h)
}

func (h lfuHeap) Less(i, j int) bool {
	return h[i].frequency < h[j].frequency
}

func (h lfuHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = j
	h[j].index = i
}

func (h *lfuHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*lfuHeapItem)
	item.index = n
	*h = append(*h, item)
}

func (h *lfuHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]
	return item
}

type lfuItem struct {
	heapItem *lfuHeapItem
	value    interface{}
}

type lfuCache struct {
	capacity int

	numberOfItems int

	storage Cache

	heap lfuHeap

	mutex sync.Mutex
}

var _ Cache = (*lfuCache)(nil)

// NewLfu creates a new lfuCache instance using mapCache.
func NewLfu(capacity int) *lfuCache {
	return &lfuCache{
		capacity: capacity,
		storage:  NewMapCache(),
		heap:     lfuHeap{},
	}
}

// NewLfuWithCustomCache creates a new lfuCache with custom cache.
func NewLfuWithCustomCache(capacity int, cache Cache) (*lfuCache, error) {
	keys, err := cache.Keys()
	if err != nil {
		return nil, err
	}

	if len(keys) > 0 {
		return nil, newError(errorTypeCacheNotEmpty, "supplied cache must be empty")
	}

	return &lfuCache{
		capacity: capacity,
		storage:  cache,
		heap:     lfuHeap{},
	}, nil
}

func (lfu *lfuCache) Store(key, val interface{}) error {
	lfu.mutex.Lock()
	defer lfu.mutex.Unlock()

	return lfu.store(key, val)
}

func (lfu *lfuCache) store(key, val interface{}) error {
	// Create a new lfu heap item.
	heapItem := &lfuHeapItem{
		value:     key,
		frequency: 0,
	}

	// Create a new lfu item.
	item := lfuItem{heapItem, val}

	// Store the new item in the inner cache.
	err := lfu.storage.Store(key, item)
	if err != nil {
		return err
	}

	// Add the new key to the heap.
	lfu.heap.Push(heapItem)

	// If the inner cache is full, remove the least frequently used.
	if lfu.isFull() {
		heapItem := lfu.heap.Pop().(*lfuHeapItem)
		err := lfu.storage.Remove(heapItem.value)
		if err != nil {
			return err
		}
	} else {
		lfu.numberOfItems++
	}

	return nil
}

func (lfu *lfuCache) Get(key interface{}) (interface{}, error) {
	lfu.mutex.Lock()
	defer lfu.mutex.Unlock()

	return lfu.get(key)
}

func (lfu *lfuCache) get(key interface{}) (interface{}, error) {
	item, err := lfu.storage.Get(key)
	if err != nil {
		return nil, err
	}

	// Increase itme's frequency.
	lfuItem := item.(lfuItem)
	lfuItem.heapItem.frequency++

	return lfuItem.value, nil
}

// GetLeastFrequentlyUsedKey returns the key from the back of the linked list.
func (lfu *lfuCache) GetLeastFrequentlyUsedKey() interface{} {
	return lfu.heap[0].value
}

func (lfu *lfuCache) Remove(key interface{}) error {
	lfu.mutex.Lock()
	defer lfu.mutex.Unlock()

	return lfu.remove(key)
}

func (lfu *lfuCache) remove(key interface{}) error {

}

func (lfu *lfuCache) Replace(key, value interface{}) error {
	lfu.mutex.Lock()
	defer lfu.mutex.Unlock()

	return lfu.replace(key, value)
}

func (lfu *lfuCache) replace(key, value interface{}) error {

}

func (lfu *lfuCache) Clear() error {
	lfu.mutex.Lock()
	defer lfu.mutex.Unlock()

	return lfu.clear()
}

func (lfu *lfuCache) clear() error {

}

func (lfu *lfuCache) Keys() ([]interface{}, error) {
	return lfu.storage.Keys()
}

func (lfu *lfuCache) Count() int {
	lfu.mutex.Lock()
	defer lfu.mutex.Unlock()

	return lfu.count()
}

func (lfu *lfuCache) count() int {
	return lfu.numberOfItems
}

func (lfu *lfuCache) IsFull() bool {
	lfu.mutex.Lock()
	defer lfu.mutex.Unlock()

	return lfu.isFull()
}

func (lfu *lfuCache) isFull() bool {
	return lfu.numberOfItems >= lfu.capacity
}

func (lfu *lfuCache) IsEmpty() bool {
	lfu.mutex.Lock()
	defer lfu.mutex.Unlock()

	return lfu.isEmpty()
}

func (lfu *lfuCache) isEmpty() bool {
	return lfu.numberOfItems < 0
}
