package cache

type Cache interface {
	Add()
	Remove()
	Get()
	Clear()
}

type mapCache struct {
	cache map[interface{}][]interface{}
}

func NewCache() Cache {
	return &mapCache{
		cache: map[interface{}][]interface{}{},
	}
}

func (mc *mapCache) Add() {
}

func (mc *mapCache) Remove() {
}

func (mc *mapCache) Get() {
}

func (mc *mapCache) Clear() {
}
