package cache

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testStruct struct {
	Str string `json:"str"`
	Int int    `json:"int"`
}

var _ = Describe("Directory Cache", func() {
	var (
		c   *directoryCache
		key string     = "key"
		val testStruct = testStruct{"Test", 0}
	)

	BeforeEach(func() {
		cacheDir := fmt.Sprintf("%s/%s", os.TempDir(), "dir-cache")
		Expect(os.RemoveAll(cacheDir)).ToNot(HaveOccurred())

		ic, err := NewDirectoryCache(cacheDir, func(key string, err error) {
			defer GinkgoRecover()
			Fail(err.Error())
		})
		Expect(err).ToNot(HaveOccurred())

		c = ic
	})

	AfterEach(func() {
		err := c.Clear()
		if err != nil {
			Expect(IsClearedCache(err)).To(BeTrue())
		}
	})

	Context("Store", func() {
		It("should store a value", func() {
			Expect(c.Store(key, val)).ToNot(HaveOccurred())
			v, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(val))
		})

		It("should return an error when attempting to store a value of an invalid type", func() {
			Expect(IsInvalidValueType(c.Store(key, "val"))).To(BeTrue())
			Expect(IsInvalidValueType(c.Store(key, 1))).To(BeTrue())
			Expect(IsInvalidValueType(c.Store(key, 0.1))).To(BeTrue())
			Expect(IsInvalidValueType(c.Store(key, true))).To(BeTrue())
		})
	})

	Context("Get", func() {
		BeforeEach(func() {
			Expect(c.Store(key, val)).ToNot(HaveOccurred())
		})

		It("should get an existing value from the cache", func() {
			v, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(val))
		})

		It("should return an error when attmpting to get a non-existent value", func() {
			_, err := c.Get("IDoNotExist")
			Expect(IsDoesNotExist(err)).To(BeTrue())
		})
	})

	Context("Remove", func() {
		BeforeEach(func() {
			Expect(c.Store(key, val)).ToNot(HaveOccurred())
		})

		It("should remove a value from the cache", func() {
			Expect(c.Remove(key)).ToNot(HaveOccurred())
			_, err := c.Get(key)
			Expect(IsDoesNotExist(err)).To(BeTrue())
		})

		It("should return an error when attempting to remove a non-existent value", func() {
			Expect(IsDoesNotExist(c.Remove("IDoNotExist"))).To(BeTrue())
		})
	})

	Context("Replace", func() {
		BeforeEach(func() {
			Expect(c.Store(key, val)).ToNot(HaveOccurred())
		})

		It("should replace a permanent value with a differnt permanent value", func() {
			Expect(c.Replace(key, testStruct{"New", 0})).ToNot(HaveOccurred())
			v, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(testStruct{"New", 0}))
		})

		It("should replace a temporary value with a permanent value", func() {
			newKey := "new"
			Expect(c.storeWithExpiration(newKey, val, 3*time.Second)).ToNot(HaveOccurred())
			Expect(c.Replace(newKey, testStruct{"New", 0})).ToNot(HaveOccurred())
			Consistently(func() bool {
				v, err := c.Get(newKey)
				Expect(err).ToNot(HaveOccurred())
				return v == testStruct{"New", 0}
			}, 5*time.Second)
		})
	})

	Context("Clear", func() {
		It("should clear the cache", func() {
			Expect(c.Clear()).ToNot(HaveOccurred())
			_, err := c.Get(key)
			Expect(IsClearedCache(err)).To(BeTrue())
		})
	})

	Context("StoreWithExpiration", func() {
		It("should store a temporary value", func() {
			Expect(c.StoreWithExpiration(key, val, 3*time.Second))

			Consistently(func() bool {
				v, err := c.Get(key)
				return v == val && err == nil
			}, 2*time.Second).Should(BeTrue())

			Eventually(func() bool {
				_, err := c.Get(key)
				return IsDoesNotExist(err)
			}, testTimeout).Should(BeTrue())
		})
	})

	Context("ReplaceWithExpiration", func() {
		It("should replace a permanent value with a temporary one", func() {
			Expect(c.Store(key, val)).ToNot(HaveOccurred())
			Expect(c.ReplaceWithExpiration(key, testStruct{"New", 0}, 3*time.Second)).ToNot(HaveOccurred())

			Consistently(func() bool {
				v, err := c.Get(key)
				return v == testStruct{"New", 0} && err == nil
			}, 2*time.Second).Should(BeTrue())

			Eventually(func() bool {
				_, err := c.Get(key)
				return IsDoesNotExist(err)
			}, testTimeout).Should(BeTrue())
		})

		It("should replace a temporary value with a temporary one", func() {
			Expect(c.StoreWithExpiration(key, val, 2*time.Second)).ToNot(HaveOccurred())
			Expect(c.ReplaceWithExpiration(key, testStruct{"New", 0}, 10*time.Second)).ToNot(HaveOccurred())

			Consistently(func() bool {
				v, err := c.Get(key)
				return v == testStruct{"New", 0} && err == nil
			}, 4*time.Second).Should(BeTrue())

			Eventually(func() bool {
				_, err := c.Get(key)
				return IsDoesNotExist(err)
			}, testTimeout).Should(BeTrue())
		})
	})

	Context("Exprie", func() {
		It("should put an expiration duration on a permanent value", func() {
			Expect(c.Store(key, val)).ToNot(HaveOccurred())
			Expect(c.Expire(key, 2*time.Second)).ToNot(HaveOccurred())

			Eventually(func() bool {
				_, err := c.Get(key)
				return IsDoesNotExist(err)
			}, testTimeout).Should(BeTrue())
		})

		It("should update the expiration duration on a temporary value", func() {
			Expect(c.StoreWithExpiration(key, val, 3*time.Second)).ToNot(HaveOccurred())
			Expect(c.Expire(key, 5*time.Second)).ToNot(HaveOccurred())

			Consistently(func() bool {
				v, err := c.Get(key)
				return v == val && err == nil
			}, 2*time.Second).Should(BeTrue())

			Eventually(func() bool {
				_, err := c.Get(key)
				return IsDoesNotExist(err)
			}, testTimeout).Should(BeTrue())
		})
	})

	Context("StoreWithUpdate", func() {
		It("should store a value and continously update it", func() {
			updateFunc := func(currValue interface{}) interface{} {
				intVal := currValue.(testStruct).Int
				return testStruct{"Test", intVal + 1}
			}

			Expect(c.StoreWithUpdate(key, val, updateFunc, 5*time.Second)).ToNot(HaveOccurred())

			Consistently(func() bool {
				currVal, err := c.Get(key)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					v, err := c.Get(key)
					Expect(err).ToNot(HaveOccurred())
					return v.(testStruct).Int > currVal.(testStruct).Int
				}, testTimeout).Should(BeTrue())

				return true
			}, 6*time.Second).Should(BeTrue())
		})
	})
})
