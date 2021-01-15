# cache
A Golang package for in memory caching

## Usage

```
import (
  "github.com/apidome/cache"
  "time"
 )

func main() {
    // MapCache is currently the only implemented cache backend
    c := cache.NewMapCache()
  
    // Store a value that will be removed after 1 minute
    key, val := "key", "val"
    duration := time.Minute
    c.Store(key, val, duration)
  
    // Store a persistent value
    c.Store(key, val, 0)

    // Get a value
    v := c.Get(key)

    // Remove a value
    c.Remove(key)

    // Replace a value
    newVal := "newval"
    newDuration := 2 * time.Minute
    c.Replace(key, newVal, newDuration)

    // Update expiration of a value
    c.Expire(key, newDuration)

    // Store an updating value
    c.StoreWithUpdate(key, func(currValue interface{}) interface{} {
        if currValue == nil {
            return 0
        }
        return currValue.(int) + 1
    }, duration)

```
