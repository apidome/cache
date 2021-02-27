# cache
A Golang package for data caching.

It provides several caching mechanisms, which follow these general rules:

- Thread safety
- Expiring values are removed by background routines
- Updating values are updated by background routines  

You can find in this package two kinds of cache mechanisms:
- **Concrete cache** - Those cache implementations use various system resources to cache your concrete data (memory, storage, network, etc)
- **Behavioural cache** - Those cache implementations wrap the concrete cache types and meant to implement advanced caching algorithms such as _Least recenlty used_ or _Least Frequently Used_. Behavioural cache type are not independent, they can rely on each cache type that implements this package's interface.

## Get it
To install, use `go get`, preferrably from a tagged release, for example `v0.1.15`
```
go get github.com/apidome/cache@v0.1.15
```

## Concrete Cache
You can use the following concrete cache types:
- Map Cache
- Directory Cache
  
## Behavioural Cache
You can wrap your concrete cache with the following behavioural cache types:
- LFU Cache (Least Recently Used)
- *COMING SOON* - LFU Cache (Least Frequently Used)
# Usage
## MapCache
A cache that stores your data in the process's memory.
```go
import (
  "github.com/apidome/cache"
  "time"
 )

func main() {
    // MapCache is currently the only implemented cache backend
    mc := cache.NewMapCache()

    // Keys and values can be of any type
    var key, val interface{} = "key", "val"

    // Store a persistent value
    err := mc.Store(key, val)

    // Get a value
    v, err = mc.Get(key)

    // Remove a value
    err = mc.Remove(key)

    // Replace a value
    err = mc.Replace(key, val.(string)+"2")

    // Clear the cache (remove all values and stop all background routines)
    err = mc.Clear()

    // Gets all keys in the cache
    keys, err := mc.Keys()

    // Store an expiring value, it will be removed after a minute
    err = mc.StoreWithExpiration(key, val, time.Minute)

    // Replace a value with an expiring value, it will be removed a minute
    // after this call
    err = mc.ReplaceWithExpiration(key, val.(string)+"2", time.Minute)

    // Set an expiration duration for a value, it will be removed a minute
    // after this call
    err = mc.Expire(key, time.Minute)

    // Store a continuosly updating value, it will be updated every minute
    // using the provided update function
    err = mc.StoreWithUpdate(key, val, func(currValue interface{}) interface{} {
        return currVal.(string)+"."
    }, time.Minute)

    // Replace a value with a continously updating value, it will be updated every
    // minute using the provided update function
    err = mc.ReplaceWithUpdate(key, val, func(currValue interface{}) interface{} {
        return currVal.(string)+"."
    }, time.Minute)
```
## DirectoryCache
A cache that store your data in a certain directory in the file system.
```go
import (
  "github.com/apidome/cache"
  "time"
  "fmt"
 )

func main() {
    // An example, can be any directory
    cacheDir := fmt.Sprintf("%s/%s", os.TempDir(), "dir-cache")

    dc, err := cache.NewDirectoryCache(cacheDir, func(key string, err error) {
        fmt.Println("Something happened in a background routine")
    })

    // Values of DirectoryCache must be any of:
    // - Maps
    // - Slices
    // - Structs that can be fully marshalled to JSON
    type exampleValue struct {
        Str string `json:"str"`
    }

    // Keys of DirectoryCache must be strings
    var key string = "key"
    var val exampleStruct = exampleStruct{"example"}

    // Store a value
    err = dc.Store(key, val)

    // Get a value
    v, err = dc.Get(key)

    // Remove a value
    err = dc.Remove(key)

    // Replace a value
    err = dc.Replace(key, exampleStruct{"newExample"})

    // Clear the cache, it will not be usable once cleared
    err = dc.Clear()

    // Gets all keys in the cache
    keys, err = dc.Keys()

    // Store an expiring value, it will be removed after a minute
    err = dc.StoreWithExpiration(key, val, time.Minute)

    // Replace an expiring value
    err = dc.ReplaceWithExpiration(key, exampleStruct{"newExample"}, time.Minute)

    // Set an expiration time for a value
    err = dc.Expire(key, 2*time.Minute)

    // Store a continously updating value
    err = dc.StoreWithUpdate(key, val, func(currValue interface{}) interface{} {
        return exampleStruct{"newExample"}
    }, time.Minute)

    // Replace a value with a continously updating one
    err = dc.ReplaceWithUpdate(key, val, func(currValue interface{}) interface{} {
        return exampleStruct{"newExample"}
    }, time.Minute)
}
```
## LRU Cache
An implementation of Least Recently Used cache algorithm. Although behavioural cache types are not independent, LRU cache will work with MapCache by default.
```go
import (
  "github.com/apidome/cache"
  "fmt"
 )

func main() {
    // An LRU cache requires a predefined capacity.
    lru := NewLru(3)

    // Get the amount of stored items 
    numberOfItems := lru.Count()

    // Check if lru cache is full
    isFull := lru.IsFull()

    // Check if lru cache is emptyv
    isEmpty := lru.IsEmpty()

    // Get the most recently used key
    mostRecent := lru.GetMostRecentlyUsedKey()

    // Get the least recently used key
    leastRecent := lru.GetLeastRecentlyUsedKey()
}
```
It is also possible to create an LRU Cache that works with other type of cache, DirectoryCache for example.
```go
func main() {
    // Initialize your own cache.
    dc := NewDirectoryCache()

    // Create a new lru with a given instance,
    lru := NewLruWithCustomCache(3, dc)
}
```

***NOTE***: When creating an LRU cache with custom concrete cache, the given concrete cahce must be empty!