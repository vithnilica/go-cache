package cache

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
	//"go.uber.org/goleak"
)

func TestCache(t *testing.T) {
	c := New[string, string](0, 0, 0)

	c.Set("key1", "val1")
	c.Set("key2", "val2")
	c.Set("key3", "val3")

	if size := c.Size(); size != 3 {
		t.Errorf("Size() = %v, want %v", size, 3)
	}

	if val1, _ := c.Get("key1"); val1 != "val1" {
		t.Errorf("Get() = %v, want %v", val1, "val1")
	}

	if val1 := c.GetValue("key1"); val1 != "val1" {
		t.Errorf("GetValue() = %v, want %v", val1, "val1")
	}

	c.Clear()
	if size := c.Size(); size != 0 {
		t.Errorf("Size() = %v, want %v", size, 0)
	}

	c.SetAll(map[string]string{
		"klic1": "val1",
		"klic2": "val2",
		"klic3": "val3",
	})

	if size := c.Size(); size != 3 {
		t.Errorf("Size() = %v, want %v", size, 3)
	}
}

func TestWithSize(t *testing.T) {
	c := New[string, string](0, 0, 1)

	c.Set("key1", "val1")
	c.Set("key2", "val2")
	c.Set("key3", "val3")

	//musi byt porad jeden protoze maximalni velikost
	if size := c.Size(); size != 1 {
		t.Errorf("Size() = %v, want %v", size, 1)
	}

	c.Set("key1", "val1")

	if val1, _ := c.Get("key1"); val1 != "val1" {
		t.Errorf("Get() = %v, want %v", val1, "val1")
	}

	if val1 := c.GetValue("key1"); val1 != "val1" {
		t.Errorf("GetValue() = %v, want %v", val1, "val1")
	}

	c.Clear()
	if size := c.Size(); size != 0 {
		t.Errorf("Size() = %v, want %v", size, 0)
	}

	c.SetAll(map[string]string{
		"klic1": "val1",
		"klic2": "val2",
		"klic3": "val3",
	})

	//tady to maximalni velikost preroste
	if size := c.Size(); size != 3 {
		t.Errorf("Size() = %v, want %v", size, 3)
	}
}

func TestCacheExpiration(t *testing.T) {
	c := New[string, string](100*time.Millisecond, 300*time.Millisecond, 0)

	c.Set("key1", "val1")
	c.SetWithTTL("key2", "val2", 300*time.Millisecond)
	c.SetWithTTL("key3", "val3", time.Second)

	if size := c.Size(); size != 3 {
		t.Errorf("Size() = %v, want %v", size, 3)
	}

	//key1 by se mel smazat
	time.Sleep(100 * time.Millisecond)
	c.CleanExpired()

	if size := c.Size(); size != 2 {
		t.Errorf("Size() = %v, want %v", size, 2)
	}

	//key2 by se mel smazat
	time.Sleep(250 * time.Millisecond)

	if size := c.Size(); size != 1 {
		t.Errorf("Size() = %v, want %v", size, 1)
	}

	c.Close()

}

// test jestli funguje finalizer
// je potreba pridat zavislost, kterou v projektu jen kvuli testu nechci
// go get go.uber.org/goleak
// func TestGoleak(t *testing.T) {
// 	debug.SetGCPercent(-1)
// 	defer goleak.VerifyNone(t)
// 	{
// 		c := New[string, string](time.Second, time.Second, 0)
// 		c.Set("aaa", "bbb")
// 	}
// 	runtime.GC()
// 	debug.FreeOSMemory()
// }

//testy rychlosti zapisu bez a s kontrolou velikosti a ano, nakej cas to zere (ale rust do nekonecna taky)
//go test -bench .
// BenchmarkSetWithoutSizeKeys10-16         4780008               241.9 ns/op
// BenchmarkSetWithoutSizeKeysN-16          1312898               821.2 ns/op
// BenchmarkSetWithSize100Keys10-16         3823531               336.1 ns/op
// BenchmarkSetWithSize100KeysN-16          1632226               747.7 ns/op
// BenchmarkSetWithSize10000Keys10-16       3056233               354.3 ns/op
// BenchmarkSetWithSize10000KeysN-16        1381782               805.9 ns/op

func newCache4Benchmark(defaultTTL time.Duration, cleanupInterval time.Duration, maxSize int) Cache[string, string] {
	c := New[string, string](defaultTTL, cleanupInterval, maxSize)
	for i := 0; i < 10000; i++ {
		c.Set("a"+strconv.Itoa(i), "val")
	}
	return c
}
func BenchmarkSetWithoutSizeKeys10(b *testing.B) {
	c := newCache4Benchmark(0, 0, 0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Set("b"+strconv.Itoa(rand.Int()%10), "val")
		}
	})
}

func BenchmarkSetWithoutSizeKeysN(b *testing.B) {
	c := newCache4Benchmark(0, 0, 0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Set("b"+strconv.Itoa(rand.Int()), "val")
		}
	})
}

func BenchmarkSetWithSize100Keys10(b *testing.B) {
	c := newCache4Benchmark(0, 0, 100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Set("b"+strconv.Itoa(rand.Int()%10), "val")
		}
	})
}

func BenchmarkSetWithSize100KeysN(b *testing.B) {
	c := newCache4Benchmark(0, 0, 100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Set("b"+strconv.Itoa(rand.Int()), "val")
		}
	})
}

func BenchmarkSetWithSize10000Keys10(b *testing.B) {
	c := newCache4Benchmark(0, 0, 10000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Set("b"+strconv.Itoa(rand.Int()%10), "val")
		}
	})
}

func BenchmarkSetWithSize10000KeysN(b *testing.B) {
	c := newCache4Benchmark(0, 0, 10000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Set("b"+strconv.Itoa(rand.Int()), "val")
		}
	})
}
