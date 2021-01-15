package tests_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/apidome/cache"
)

var _ = Describe("Map Cache", func() {
	var (
		c                        cache.Cache
		key, val, nonExistentKey string = "test-key", "test-val", "non-existent"
	)

	BeforeEach(func() {
		c = cache.NewMapCache()
	})

	Context("Store", func() {
		It("should add a value", func() {
			c.Store(key, val, time.Minute)
			Expect(c.Get(key)).To(Equal(val), "value was not stored in cache")
		})

		It("should remove a value after timeout", func() {
			timeout := time.Second * 3
			c.Store(key, val, timeout)

			Eventually(func() bool {
				time.Sleep(time.Second)
				_, err := c.Get(key)
				return cache.IsDoesntExistError(err)
			}, testTimeout).Should(Equal(true), "value was not removed from cache after timeout")
		})

		It("should return an error when attempting to override a value", func() {
			c.Store(key, val, 0)
			Expect(c.Store(key, "new-val", 0)).To(HaveOccurred(), "expected an error when attempting to override a value")
			storedVal, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(storedVal).To(Equal(val), "a value was overriden")
		})
	})

	Context("Remove", func() {
		BeforeEach(func() {
			c.Store(key, val, 0)
		})

		It("should remove a value", func() {
			Expect(c.Remove(key)).ToNot(HaveOccurred(), "value was not removed")
			Expect(c.Store(key, val, 0)).NotTo(HaveOccurred(), "value could not be added eventhough it's key is available")
		})

		It("should return an error when attempting to remove a non-existent value", func() {
			Expect(c.Remove("non-existent")).To(HaveOccurred(), "an error was not returned when attempting to remove a non-existent value")
		})
	})

	Context("Get", func() {
		BeforeEach(func() {
			c.Store(key, val, 0)
		})

		It("should return a stored value", func() {
			val, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(val), "the returned value is not equal to the stored value")
		})

		It("should return an error when attempting to get a non-existent value", func() {
			_, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred(), "an error was not returned when attempting to get a non-existent value")
		})

		It("should not remove a value after it was fethced", func() {
			_, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			_, err = c.Get(key)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Replace", func() {
		BeforeEach(func() {
			c.Store(key, val, 0)
		})

		It("should replace a value", func() {
			newVal := "new"
			c.Replace(key, newVal, 0)

			val, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(newVal), "a value was not replaced")
		})

		It("should fail to replace a non-existent value", func() {
			Expect(c.Replace(nonExistentKey, nonExistentKey, 0)).To(HaveOccurred())
		})
	})

	Context("Expire", func() {
		BeforeEach(func() {
			Expect(c.Store(key, val, 0)).ToNot(HaveOccurred())
		})

		It("should be removed after expiration time is set", func() {
			c.Expire(key, 3*time.Second)

			Eventually(func() bool {
				_, err := c.Get(key)
				if err != nil {
					if cache.IsDoesntExistError(err) {
						return true
					}
				}

				return false
			}, testTimeout).Should(BeTrue(), "value was not removed when expiration time was set")
		})

		It("should be removed immediately when expiration duration is 0", func() {
			c.Expire(key, 0)
			_, err := c.Get(key)
			Expect(err).To(HaveOccurred())
			Expect(cache.IsDoesntExistError(err)).To(BeTrue())
		})

		It("should return an error when attempting to expire a non-existent key", func() {
			err := c.Expire(nonExistentKey, time.Second)
			Expect(err).To(HaveOccurred(), "expected an error when expiring a non-existent key")
		})

		It("should update duration of a value", func() {
			newKey, newVal := "newKey", "newVal"
			Expect(c.Store(newKey, newVal, 5*time.Second)).ToNot(HaveOccurred())
			Expect(c.Expire(newKey, 20*time.Second)).ToNot(HaveOccurred())

			time.Sleep(10 * time.Second)

			_, err := c.Get(newKey)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				_, err := c.Get(newKey)
				if err != nil {
					if cache.IsDoesntExistError(err) {
						return true
					}
				}

				return false
			}, testTimeout).Should(BeTrue())
		})
	})

	Context("StoreWithUpdate", func() {
		It("should continuosly update the value after the specified duration", func() {
			c.StoreWithUpdate(key, func(currValue interface{}) interface{} {
				if currValue == nil {
					return 0
				}

				return currValue.(int) + 1
			}, 1*time.Second)

			Consistently(func() bool {
				currValue, err := c.Get(key)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					val, err := c.Get(key)
					Expect(err).ToNot(HaveOccurred())
					return val.(int) > currValue.(int)
				}, testTimeout, 500*time.Millisecond).Should(BeTrue(), "value should have been updated")

				return true
			}, 10*time.Second, 2*time.Second).Should(BeTrue(), "value should have been updated")
		})
	})
})
