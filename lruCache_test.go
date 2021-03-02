package cache

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const CacheSize = 3

var _ = Describe("LRU Cache", func() {
	var (
		c            *lruCache
		keys, values []string
	)

	for i := 0; i < CacheSize; i++ {
		keys = append(keys, fmt.Sprintf("test-key-%v", i))
		values = append(values, fmt.Sprintf("test-value-%v", i))
	}

	BeforeEach(func() {
		c = NewLru(CacheSize)
	})

	Context("Store", func() {
		It("should store a value", func() {
			Expect(c.Store(keys[0], values[0])).ToNot(HaveOccurred(), "failed storing a value")
			v, err := c.Get(keys[0])
			Expect(v).To(Equal(values[0]), fmt.Sprintf("stored value is incorrect for key %v", keys))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should count each cached value", func() {
			for i := 0; i < CacheSize; i++ {
				Expect(c.Count()).To(Equal(i), "number of items mismatch")
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
			Expect(c.Count()).To(Equal(CacheSize), "number of items mismatch")
		})

		It("should not increase items counter when cache is full", func() {
			for i := 0; i < CacheSize; i++ {
				Expect(c.Count()).To(Equal(i), "number of items mismatch")
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
			Expect(c.Count()).To(Equal(CacheSize), "number of items mismatch")
			c.Store("extra-key", "extra-value")
			Expect(c.Count()).To(Equal(CacheSize), "number of items increased even though cache is full")
		})

		It("should remove the least recently used value when the cache is full and the user stores a new value", func() {
			for i := 0; i < CacheSize; i++ {
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}

			Expect(c.Store("extra-key", "extra-value")).ToNot(HaveOccurred(), "failed to store extra value")

			_, err := c.Get(keys[0])
			Expect(err).To(HaveOccurred(), "managed to getfrom cache a key that should not be cached")
		})
	})

	Context("Get", func() {
		It("should move an item to the front of the linked list after it has been accessed", func() {
			for i := 0; i < CacheSize; i++ {
				Expect(c.Count()).To(Equal(i), "number of items mismatch")
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
			Expect(c.GetMostRecentlyUsedKey()).To(Equal(keys[CacheSize-1]))

			val, err := c.Get(keys[0])
			Expect(err).ToNot(HaveOccurred(), "failed to get a key")
			Expect(val).To(Equal(values[0]), "got the wrong value from cache")
			Expect(c.GetMostRecentlyUsedKey()).To(Equal(keys[0]), "most recent value is not getting updated")
		})
	})

	Context("Remove", func() {
		BeforeEach(func() {
			for i := 0; i < CacheSize; i++ {
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
		})

		It("should remove an item", func() {
			Expect(c.Remove(keys[0])).ToNot(HaveOccurred(), "failed to remove item")
			_, err := c.Get(keys[0])
			Expect(err).To(HaveOccurred(), "item has not been deleted")
		})

		It("should decrease items counter", func() {
			Expect(c.Remove(keys[0])).ToNot(HaveOccurred(), "failed to remove item")
			Expect(c.Count()).To(Equal(CacheSize - 1))
		})

		It("should not decrease items counter when cache is empty", func() {
			Expect(c.Clear()).ToNot(HaveOccurred())
			Expect(c.Remove(keys[0])).To(HaveOccurred())
			Expect(c.Count()).To(Equal(0))
		})
	})

	Context("Replace", func() {
		BeforeEach(func() {
			c.Store(keys[0], values[0])
		})

		It("should not change items counter", func() {
			Expect(c.Replace(keys[0], "new-value")).ToNot(HaveOccurred())
			Expect(c.Count()).To(Equal(1))
		})
	})

	Context("Clear", func() {
		BeforeEach(func() {
			for i := 0; i < CacheSize; i++ {
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
		})

		It("should remove all items from cache", func() {
			Expect(c.Clear()).ToNot(HaveOccurred())
			Expect(c.Count()).To(Equal(0))
		})
	})

	Context("Count", func() {
		It("should return 0 after creating a new instance", func() {
			Expect(c.Count()).To(Equal(0))
		})

		It("should return the correct amount of cached items", func() {
			for i := 0; i < CacheSize; i++ {
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
			Expect(c.Count()).To(Equal(CacheSize))
		})
	})

	Context("IsFull", func() {
		It("should return true when cache is full", func() {
			for i := 0; i < CacheSize; i++ {
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
			Expect(c.IsFull()).To(Equal(true))
		})

		It("should return false when cache is not full", func() {
			Expect(c.IsFull()).To(Equal(false))
		})
	})

	Context("IsEmpty", func() {
		It("should return true when cache is empty", func() {
			Expect(c.IsEmpty()).To(Equal(true))
		})

		It("should return false cache is not empty", func() {
			Expect(c.Store(keys[0], values[0])).ToNot(HaveOccurred(), "failed storing a value")
			Expect(c.IsEmpty()).To(Equal(false))
		})
	})

	Context("NewLruWithCustomCache", func() {
		It("should return an error when being supplied with a non empty cache", func() {
			mapCache := NewMapCache()
			Expect(mapCache.Store(keys[0], values[0])).ToNot(HaveOccurred(), "failed storing a value in map cache")
			_, err := NewLruWithCustomCache(CacheSize, mapCache)
			Expect(err).To(HaveOccurred())
		})
	})
})
