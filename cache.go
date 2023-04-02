package cache

import (
	"runtime"
	"sync"
	"time"
)

type Cache[K comparable, V any] interface {
	// Get searches for a key in the cache and returns its value and a boolean indicating whether the key was found or not.
	Get(key K) (value V, found bool)
	// GetSafe is similar to Get but also checks if the value associated with the key has expired or not.
	GetSafe(key K) (V, bool)
	// GetValue returns the value associated with the key or a default value if the key is not present in the cache.
	GetValue(key K) V
	// Set inserts a key-value pair into the cache with a default duration.
	Set(key K, value V)
	// SetWithTTL inserts a key-value pair into the cache with a specified duration (TTL - Time To Live).
	SetWithTTL(key K, value V, ttl time.Duration)
	// SetAll inserts a map of key-value pairs into the cache with a default duration.
	SetAll(m map[K]V)
	// SetAllWithTTL inserts a map of key-value pairs into the cache with a specified duration.
	SetAllWithTTL(m map[K]V, ttl time.Duration)
	// Remove removes a key-value pair from the cache.
	Remove(key K)
	// IsEmpty returns true if the cache is empty.
	IsEmpty() bool
	// Size returns the number of key-value pairs in the cache.
	Size() int
	// Clear removes all key-value pairs from the cache.
	Clear()
	// CleanExpired removes all expired key-value pairs from the cache.
	CleanExpired()
	// Close stops the cleanup routine and closes the cache object.
	Close()
	// resetItems
	resetItems()
}

// New returns a new cache object that can store key-value pairs of any comparable key type and any value type.
//
// defaultTTL - default duration after which the values will expire (<=0 for no expiration)
//
// cleanupInterval - cleanup interval for expired values (<=0 wihout cleanup)
//
// maxSize - maximum size (0 wihout limit)
func New[K comparable, V any](
	defaultTTL time.Duration,
	cleanupInterval time.Duration,
	maxSize int,
) Cache[K, V] {
	c := &cache[K, V]{
		items:      make(map[K]*item[V]),
		defaultTTL: defaultTTL,
	}

	var wrapped Cache[K, V]
	if maxSize > 0 {
		//obal pro omezovani velikosti pri zapisu
		wrapped = &cacheWithSize[K, V]{
			cache:   c,
			maxSize: maxSize,
		}
	} else {
		//obal jen aby fungoval finalizer
		wrapped = &cacheWithoutSize[K, V]{
			cache: c,
		}
	}

	wrapped.resetItems()

	runtime.SetFinalizer(wrapped, func(c Cache[K, V]) {
		c.Close()
	})

	//nastartovat uklid
	if cleanupInterval > 0 {
		ticker := time.NewTicker(cleanupInterval)

		c.running = true
		c.done = make(chan struct{})

		go func() {
			for {
				select {
				case <-c.done:
					ticker.Stop()
					return
				case <-ticker.C:
					c.CleanExpired()
				}
			}
		}()
	}

	return wrapped

}

type item[V any] struct {
	value      V
	expiration int64
}

// isExpired test jestli je polozka uz stara
// now - time.Now().UnixNano()
func (item item[V]) isExpired(now int64) bool {
	if item.expiration == 0 {
		return false
	}
	return now > item.expiration
}

type cache[K comparable, V any] struct {
	items      map[K]*item[V]
	mu         sync.RWMutex
	defaultTTL time.Duration
	running    bool
	done       chan struct{}
}

type cacheWithSize[K comparable, V any] struct {
	*cache[K, V]
	maxSize int
}

type cacheWithoutSize[K comparable, V any] struct {
	*cache[K, V]
}

func (c *cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	if item, found := c.items[key]; found {
		c.mu.RUnlock()
		return item.value, true
	}
	c.mu.RUnlock()
	return *new(V), false
}

func (c *cache[K, V]) GetSafe(key K) (V, bool) {
	c.mu.RLock()
	if item, found := c.items[key]; found {
		if !item.isExpired(time.Now().UnixNano()) {
			c.mu.RUnlock()
			return item.value, true
		}
	}
	c.mu.RUnlock()
	return *new(V), false
}

func (c *cache[K, V]) GetValue(key K) V {
	c.mu.RLock()
	if item, found := c.items[key]; found {
		c.mu.RUnlock()
		return item.value
	}
	c.mu.RUnlock()
	return *new(V)
}

func (c *cache[K, V]) Remove(key K) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *cache[K, V]) IsEmpty() bool {
	return c.Size() == 0
}

func (c *cache[K, V]) Size() int {
	c.mu.RLock()
	size := len(c.items)
	c.mu.RUnlock()
	return size
}

func (c *cache[K, V]) CleanExpired() {
	//nejdriv projdu vsechny klice jen se zamkem pro cteni, odlozim si je a pak nahodim zamek i pro zapis a smaznu je
	now := time.Now().UnixNano()
	var keys []K
	c.mu.RLock()
	for key, item := range c.items {
		// "Inlining" of expired
		if item.isExpired(now) {
			keys = append(keys, key)
		}
	}
	c.mu.RUnlock()

	c.mu.Lock()
	for _, key := range keys {
		delete(c.items, key)
	}
	c.mu.Unlock()
}

func (c *cache[K, V]) Close() {
	if c.running {
		c.running = false
		close(c.done)
	}
	c.items = make(map[K]*item[V], 0)
}

func (c *cache[K, V]) resetItems() {
	c.items = make(map[K]*item[V])
}
func (c *cacheWithoutSize[K, V]) Clear() {
	c.mu.Lock()
	c.resetItems()
	c.mu.Unlock()
}

func (c *cacheWithSize[K, V]) resetItems() {
	c.items = make(map[K]*item[V], c.maxSize)
}
func (c *cacheWithSize[K, V]) Clear() {
	c.mu.Lock()
	c.resetItems()
	c.mu.Unlock()
}

// Set bez kontroly velikosti
func (c *cacheWithoutSize[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL bez kontroly velikosti
func (c *cacheWithoutSize[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()

	c.items[key] = &item[V]{
		value:      value,
		expiration: expiration,
	}

	c.mu.Unlock()
}

// SetAll bez kontroly velikosti
func (c *cacheWithoutSize[K, V]) SetAll(m map[K]V) {
	c.SetAllWithTTL(m, c.defaultTTL)
}

// SetAllWithTTL bez kontroly velikosti
func (c *cacheWithoutSize[K, V]) SetAllWithTTL(m map[K]V, ttl time.Duration) {
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()

	for key, value := range m {
		c.items[key] = &item[V]{
			value:      value,
			expiration: expiration,
		}
	}

	c.mu.Unlock()
}

// Set s kontrolou velikosti
func (c *cacheWithSize[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL s kontrolou velikosti
func (c *cacheWithSize[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()

	//omezeni velikosti
	if len(c.items)+1 > c.maxSize {
		if _, found := c.items[key]; !found {
			//smaze prvni klic v mape, je to nefer:)
			//chtel sem vybrat nahodne pred reflect.ValueOf(c.items).MapKeys() ale to je pomale
			//neco jako linked hash map se mi delat nechce
			for k := range c.items {
				delete(c.items, k)
				break
			}
		}
	}

	c.items[key] = &item[V]{
		value:      value,
		expiration: expiration,
	}

	c.mu.Unlock()
}

// SetAll s kontrolou velikosti
func (c *cacheWithSize[K, V]) SetAll(m map[K]V) {
	c.SetAllWithTTL(m, c.defaultTTL)
}

// SetAllWithTTL s kontrolou velikosti
func (c *cacheWithSize[K, V]) SetAllWithTTL(m map[K]V, ttl time.Duration) {
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()

	//omezeni velikosti
	if newSize := len(c.items) + len(m); newSize > c.maxSize {
		//smaze x nahodnych polozek
		x := newSize - c.maxSize
		if len(c.items) <= x {
			//smaze vsechno a rovnou pripravi prostor pro nove polozky
			c.items = make(map[K]*item[V], len(m))
		} else {
			//smaze x polozek
			i := 0
			for k := range c.items {
				delete(c.items, k)
				i++
				if i >= x || len(c.items) == 0 {
					break
				}
			}
		}
	}

	for key, value := range m {
		c.items[key] = &item[V]{
			value:      value,
			expiration: expiration,
		}
	}

	c.mu.Unlock()
}
