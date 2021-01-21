# cache
A Golang package for in memory caching

## Usage

```go
import (
  "github.com/apidome/cache"
  "time"
 )

func main() {
    // MapCache is currently the only implemented cache backend
    c := cache.NewMapCache()
  
    // Store a value that will be removed after 1 minute
    key, val := "key", "val"
    c.Store(key, val, time.Minute)

    // Store an updating value
    c.StoreWithUpdate(key, val, func(currValue interface{}) interface{} {
        return "newval"
    }, ttl)

    // Store a persistent value
    c.Store(key, val)

    // Get a value
    v, err := c.Get(key)

    // Remove a value
    err = c.Remove(key)

    // Replace a value
    newVal := "newval"
    newTTL := 2 * time.Minute
    c.Replace(key, newVal, newTTL)

    // Update expiration of a value
    c.Expire(key, newTTL)

```
