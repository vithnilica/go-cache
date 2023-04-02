# go-cache

The Go library is a simple implementation of a cache with support for expiration of values. It uses a map to store key-value pairs and a mutex to synchronize access to the map. It also has a background routine that periodically cleans up expired values from the map. Inspired by [patrickmn/go-cache](https://github.com/patrickmn/go-cache).

go doc [doc.md](doc.md)

## Getting started

```bash
go get gitlab.seznam.net/go/cache
```

```golang
// Cache with a default expiration of 1 second, purging old items every 2 seconds and a maximum size of 5 entries.
c := cache.New[string, string](time.Second, time.Second*2, 5)
defer c.Close()

// Set a value with the default expiration.
c.Set("key", "x")

// Set a value with an expiration time of 1 hour.
c.SetWithTTL("key_1h", "x", time.Hour)

// Set multiple values at once with the default expiration.
c.SetAll(map[string]string{
    "key_map1": "x",
    "key_map2": "x",
    "key_map3": "x",
})

// Read an item from the cache.
if val, found := c.Get("key"); found {
    fmt.Println("value:", val)
    // value: x
}

// Read an item from the cache and get an empty value if it does not exist.
val1 := c.GetValue("zzz")

// Read an item from the cache and require that it has not expired.
if val, found := c.GetSafe("key"); found {
    fmt.Println("value:", val)
    // value: x
}

// Wait for the cache to clean up.
time.Sleep(time.Second * 3)
fmt.Println("number of items after cleanup:", c.Size())
// number of items after cleanup: 1

```
