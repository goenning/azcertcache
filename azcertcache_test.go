package azcertcache_test

import (
	"context"
	"io/ioutil"
	"reflect"
	"testing"

	"golang.org/x/crypto/acme/autocert"

	"github.com/goenning/azcertcache"
)

func expectNotNil(t *testing.T, v interface{}) {
	if v == nil {
		t.Errorf("should not be nil")
	}
}

func expectNil(t *testing.T, v interface{}) {
	if v != nil {
		t.Errorf("should be nil, but was %v", v)
	}
}

func expectEquals(t *testing.T, v interface{}, expected interface{}) {
	if !reflect.DeepEqual(v, expected) {
		t.Errorf("should be %v, but was %v", expected, v)
	}
}

func newCache(t *testing.T, containerName string) *azcertcache.Cache {
	accountName := "devstoreaccount1"
	accountKey := "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
	endpointURL := "http://localhost:10000/devstoreaccount1"

	cache, err := azcertcache.NewWithEndpoint(accountName, accountKey, containerName, endpointURL)
	expectNil(t, err)

	err = cache.DeleteContainer(context.Background())
	expectNil(t, err)

	err = cache.CreateContainer(context.Background())
	expectNil(t, err)
	return cache
}

func TestNew(t *testing.T) {
	cache := newCache(t, "testcontainer")
	expectNotNil(t, cache)
}

func TestGet_UnkownKey(t *testing.T) {
	cache := newCache(t, "testcontainer")
	expectNotNil(t, cache)
	data, err := cache.Get(context.Background(), "my-key")
	expectEquals(t, err, autocert.ErrCacheMiss)
	expectEquals(t, len(data), 0)
}

func TestGet_AfterPut(t *testing.T) {
	cache := newCache(t, "testcontainer")
	expectNotNil(t, cache)

	actual, _ := ioutil.ReadFile("./LICENSE")
	err := cache.Put(context.Background(), "my-key", actual)
	expectNil(t, err)

	data, err := cache.Get(context.Background(), "my-key")
	expectNil(t, err)
	expectEquals(t, data, actual)
}

func TestGet_AfterDelete(t *testing.T) {
	cache := newCache(t, "testcontainer")
	expectNotNil(t, cache)

	actual := []byte{1, 2, 3, 4}
	err := cache.Put(context.Background(), "my-key", actual)
	expectNil(t, err)

	err = cache.Delete(context.Background(), "my-key")
	expectNil(t, err)

	data, err := cache.Get(context.Background(), "my-key")
	expectEquals(t, err, autocert.ErrCacheMiss)
	expectEquals(t, len(data), 0)
}

func TestDelete_UnkownKey(t *testing.T) {
	cache := newCache(t, "testcontainer")
	expectNotNil(t, cache)

	var err error

	err = cache.Delete(context.Background(), "my-key1")
	expectNil(t, err)
	err = cache.Delete(context.Background(), "other-key")
	expectNil(t, err)
	err = cache.Delete(context.Background(), "hello-world")
	expectNil(t, err)
}

func TestPut_Overwrite(t *testing.T) {
	cache := newCache(t, "testcontainer")
	expectNotNil(t, cache)

	data1 := []byte{1, 2, 3, 4}
	err := cache.Put(context.Background(), "thekey", data1)
	expectNil(t, err)
	data, err := cache.Get(context.Background(), "thekey")
	expectEquals(t, data, data1)

	data2 := []byte{5, 6, 7, 8}
	err = cache.Put(context.Background(), "thekey", data2)
	expectNil(t, err)
	data, err = cache.Get(context.Background(), "thekey")
	expectEquals(t, data, data2)
}

func TestDifferentContainer(t *testing.T) {
	cache1 := newCache(t, "testcontainer1")
	cache2 := newCache(t, "testcontainer2")

	input := []byte{1, 2, 3, 4}
	err := cache1.Put(context.Background(), "thekey.txt", input)
	expectNil(t, err)

	data, err := cache1.Get(context.Background(), "thekey.txt")
	expectEquals(t, data, input)
	expectNil(t, err)

	data, err = cache2.Get(context.Background(), "thekey.txt")
	expectEquals(t, len(data), 0)
	expectEquals(t, err, autocert.ErrCacheMiss)
}

func TestGet_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cache := newCache(t, "testcontainer")
	expectNotNil(t, cache)
	data, err := cache.Get(ctx, "my-key")
	expectNotNil(t, err)
	expectEquals(t, len(data), 0)
}
