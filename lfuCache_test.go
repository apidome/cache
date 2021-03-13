package cache

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const LFUCacheSize = 3

var _ = Describe("LFU Cache", func() {
	var (
		c            *lfuCache
		keys, values []string
	)

	for i := 0; i < LFUCacheSize; i++ {
		keys = append(keys, fmt.Sprintf("test-key-%v", i))
		values = append(values, fmt.Sprintf("test-value-%v", i))
	}

	BeforeEach(func() {
		c = NewLfu(LFUCacheSize)
	})

	Context("Store", func() {
		BeforeEach(func() {
			for i := 0; i < LFUCacheSize; i++ {
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
		})

		It("should store a value", func() {
			c.Clear()
			Expect(c.Store(keys[0], values[0])).ToNot(HaveOccurred(), "failed storing a value")
			v, err := c.Get(keys[0])
			Expect(v).To(Equal(values[0]), fmt.Sprintf("stored value is incorrect for key %v", keys[0]))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should remove the least frequently used value when the cache is full and the user stores a new value", func() {
			fmt.Println(c.Keys())
			// Enforce keys[2] to be the lfu item.
			_, err := c.Get(keys[0])
			Expect(err).ToNot(HaveOccurred())
			_, err = c.Get(keys[1])
			Expect(err).ToNot(HaveOccurred())

			// Storing a new value should pop keys[2] out of the cache.
			Expect(c.Store("extra-key", "extra-value")).ToNot(HaveOccurred(), "failed storing extra value")

			_, err = c.Get(keys[2])
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Get", func() {
		BeforeEach(func() {
			for i := 0; i < LFUCacheSize; i++ {
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
		})

		It("should update the heap after each access", func() {
			lfuKey := c.GetLeastFrequentlyUsedKey()
			_, err := c.Get(lfuKey)
			Expect(err).ToNot(HaveOccurred())
			newLfuKey := c.GetLeastFrequentlyUsedKey()
			Expect(newLfuKey).ToNot(Equal(lfuKey))
		})

		It("should return an error when accessing a key that does not exist", func() {
			_, err := c.Get("non-existent-key")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Remove", func() {
		BeforeEach(func() {
			for i := 0; i < LFUCacheSize; i++ {
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
		})

		It("should remove an item", func() {
			Expect(c.Remove(keys[0])).ToNot(HaveOccurred(), "failed to remove item")
			_, err := c.Get(keys[0])
			Expect(err).To(HaveOccurred(), "item has not been deleted")
		})
	})

	Context("Clear", func() {
		BeforeEach(func() {
			for i := 0; i < LFUCacheSize; i++ {
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
		})

		It("should remove all items from cache", func() {
			Expect(c.Clear()).ToNot(HaveOccurred())
			Expect(c.Count()).To(Equal(0))
		})

		It("should be able to store more items after clear", func() {
			Expect(c.Clear()).ToNot(HaveOccurred())
			Expect(c.Count()).To(Equal(0))
			Expect(c.Store(keys[0], values[0])).ToNot(HaveOccurred(), "failed storing a value")
		})
	})

	Context("Count", func() {
		It("should return 0 after creating a new instance", func() {
			Expect(c.Count()).To(Equal(0))
		})

		It("should return the correct amount of cached items", func() {
			for i := 0; i < LFUCacheSize; i++ {
				Expect(c.Store(keys[i], values[i])).ToNot(HaveOccurred(), "failed storing a value")
			}
			Expect(c.Count()).To(Equal(LFUCacheSize))
		})
	})

	Context("IsFull", func() {
		It("should return true when cache is full", func() {
			for i := 0; i < LFUCacheSize; i++ {
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

	Context("NewLfuWithCustomCache", func() {
		It("should return an error when being supplied with a non empty cache", func() {
			mapCache := NewMapCache()
			Expect(mapCache.Store(keys[0], values[0])).ToNot(HaveOccurred(), "failed storing a value in map cache")
			_, err := NewLruWithCustomCache(LFUCacheSize, mapCache)
			Expect(err).To(HaveOccurred())
		})
	})
})
