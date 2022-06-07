package azcertcache_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/acme/autocert"

	"github.com/goenning/azcertcache"
)

const accountName = "devstoreaccount1"
const accountKey = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="

// Bad account key encoded in base64 that would fail the authentication check
const badAccountKey = "YmFkY3JlZGVudGlhbA==" // base64: "badcredential"

func newCache(t *testing.T, containerPrefix string, accountKey string) (*azcertcache.Cache, error) {
	// Append test name after the container prefix to make them different.
	// This avoids "ContainerBeingDeleted" error when creating new containers
	// in different unit tests.
	containerName := fmt.Sprintf("%s-%s", containerPrefix, t.Name())

	// Sanitize: container names must only contain lower case, numbers, or hyphens
	containerName = strings.ToLower(containerName)
	containerName = strings.Replace(containerName, "_", "", -1)

	endpointURL := fmt.Sprintf("https://%s.blob.core.windows.net", accountName)
	cache, err := azcertcache.NewWithEndpoint(accountName, accountKey, containerName, endpointURL)
	if err != nil {
		// Failed to create new blob storage endpoint. Return the error
		return nil, err
	}

	err = cache.CreateContainer(context.Background())
	assert.Nil(t, err)

	t.Cleanup(func() {
		err = cache.DeleteContainer(context.Background())
		assert.Nil(t, err)
	})
	return cache, nil
}

func TestNew(t *testing.T) {
	cache, _ := newCache(t, "testcontainer", accountKey)
	assert.NotNil(t, cache)
}

func TestNew_BadCredential(t *testing.T) {
	cache, err := newCache(t, "testcontainer", badAccountKey)
	assert.Nil(t, cache)
	assert.NotNil(t, err)
	stgErr, _ := err.(azblob.StorageError)
	assert.Equal(t, stgErr.ServiceCode(), azblob.ServiceCodeAuthenticationFailed)
}

func TestGet_UnkownKey(t *testing.T) {
	cache, _ := newCache(t, "testcontainer", accountKey)
	assert.NotNil(t, cache)
	data, err := cache.Get(context.Background(), "my-key")
	assert.Equal(t, err, autocert.ErrCacheMiss)
	assert.Equal(t, len(data), 0)
}

func TestGet_AfterPut(t *testing.T) {
	cache, _ := newCache(t, "testcontainer", accountKey)
	assert.NotNil(t, cache)

	actual, _ := ioutil.ReadFile("./LICENSE")
	err := cache.Put(context.Background(), "my-key", actual)
	assert.Nil(t, err)

	data, err := cache.Get(context.Background(), "my-key")
	assert.Nil(t, err)
	assert.Equal(t, data, actual)
}

func TestGet_AfterDelete(t *testing.T) {
	cache, _ := newCache(t, "testcontainer", accountKey)
	assert.NotNil(t, cache)

	actual := []byte{1, 2, 3, 4}
	err := cache.Put(context.Background(), "my-key", actual)
	assert.Nil(t, err)

	err = cache.Delete(context.Background(), "my-key")
	assert.Nil(t, err)

	data, err := cache.Get(context.Background(), "my-key")
	assert.Equal(t, err, autocert.ErrCacheMiss)
	assert.Equal(t, len(data), 0)
}

func TestDelete_UnkownKey(t *testing.T) {
	cache, _ := newCache(t, "testcontainer", accountKey)
	assert.NotNil(t, cache)

	var err error

	err = cache.Delete(context.Background(), "my-key1")
	assert.Nil(t, err)
	err = cache.Delete(context.Background(), "other-key")
	assert.Nil(t, err)
	err = cache.Delete(context.Background(), "hello-world")
	assert.Nil(t, err)
}

func TestPut_Overwrite(t *testing.T) {
	cache, _ := newCache(t, "testcontainer", accountKey)
	assert.NotNil(t, cache)

	data1 := []byte{1, 2, 3, 4}
	err := cache.Put(context.Background(), "thekey", data1)
	assert.Nil(t, err)
	data, _ := cache.Get(context.Background(), "thekey")
	assert.Equal(t, data, data1)

	data2 := []byte{5, 6, 7, 8}
	err = cache.Put(context.Background(), "thekey", data2)
	assert.Nil(t, err)
	data, _ = cache.Get(context.Background(), "thekey")
	assert.Equal(t, data, data2)
}

func TestDifferentContainer(t *testing.T) {
	cache1, _ := newCache(t, "testcontainer1", accountKey)
	cache2, _ := newCache(t, "testcontainer2", accountKey)

	input := []byte{1, 2, 3, 4}
	err := cache1.Put(context.Background(), "thekey.txt", input)
	assert.Nil(t, err)

	data, err := cache1.Get(context.Background(), "thekey.txt")
	assert.Equal(t, data, input)
	assert.Nil(t, err)

	data, err = cache2.Get(context.Background(), "thekey.txt")
	assert.Equal(t, len(data), 0)
	assert.Equal(t, err, autocert.ErrCacheMiss)
}

func TestGet_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cache, _ := newCache(t, "testcontainer", accountKey)
	assert.NotNil(t, cache)
	data, err := cache.Get(ctx, "my-key")
	assert.NotNil(t, err)
	assert.Equal(t, len(data), 0)
}
