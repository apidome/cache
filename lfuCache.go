package cache

import (
	"sync"
)

type lfuItem struct {
	value interface{}

	frequency int

	index int
}

type lfuHeap []*lfuItem

func (h lfuHeap) Len() int {
	return len(h)
}

func (h lfuHeap) Less(i, j int) bool {

}

func (h lfuHeap) Swap(i, j int) {

}

func (h *lfuHeap) Push(x interface{}) {

}

func (h *lfuHeap) Pop() interface{} {

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

}

// NewLfuWithCustomCache creates a new lfuCache with custom cache.
func NewLfuWithCustomCache(capacity int, cache Cache) (*lfuCache, error) {

}

func (lfu *lfuCache) Store(key, val interface{}) error {

}

func (lfu *lfuCache) store(key, val interface{}) error {

}

func (lfu *lfuCache) Get(key interface{}) (interface{}, error) {

}

func (lfu *lfuCache) get(key interface{}) (interface{}, error) {

}

// GetMostFrequentlyUsedKey returns the key from the front of the linked list.
func (lfu *lfuCache) GetMostFrequentlyUsedKey() interface{} {

}

// GetLeastFrequentlyUsedKey returns the key from the back of the linked list.
func (lfu *lfuCache) GetLeastFrequentlyUsedKey() interface{} {

}

func (lfu *lfuCache) Remove(key interface{}) error {

}

func (lfu *lfuCache) remove(key interface{}) error {

}

func (lfu *lfuCache) Replace(key, value interface{}) error {

}

func (lfu *lfuCache) replace(key, value interface{}) error {

}

func (lfu *lfuCache) Clear() error {

}

func (lfu *lfuCache) clear() error {

}

func (lfu *lfuCache) Keys() ([]interface{}, error) {

}

func (lfu *lfuCache) Count() int {

}

func (lfu *lfuCache) count() int {

}

func (lfu *lfuCache) IsFull() bool {

}

func (lfu *lfuCache) isFull() bool {

}

func (lfu *lfuCache) IsEmpty() bool {

}

func (lfu *lfuCache) isEmpty() bool {

}
