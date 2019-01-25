// Package azcertcache implements an autocert.Cache to store certificate data within an Azure Blob Storage container
//
// See https://godoc.org/golang.org/x/crypto/acme/autocert
package azcertcache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"golang.org/x/crypto/acme/autocert"
)

func isNotFound(err error) bool {
	if err != nil {
		if azError, ok := err.(azblob.ResponseError); ok {
			return azError.Response().StatusCode == http.StatusNotFound
		}
	}
	return false
}

// ErrEmptyContainerName is returned when given container name is empty
var ErrEmptyContainerName = errors.New("containerName must not be empty")

// Making sure that we're adhering to the autocert.Cache interface.
var _ autocert.Cache = (*Cache)(nil)

// Cache provides an Azure Blob Storage backend to the autocert cache.
type Cache struct {
	containerURL azblob.ContainerURL
}

// New creates an cache instance that can be used with autocert.Cache.
// It returns any errors that could happen while connecting to Azure Blob Storage.
func New(accountName, accountKey, containerName string) (*Cache, error) {
	return NewWithEndpoint(accountName, accountKey, containerName, fmt.Sprintf("https://%s.blob.core.windows.net", accountName))
}

// NewWithEndpoint creates an cache instance that can be used with autocert.Cache.
// Endpoint can be used to target a different environment that is not Azure
// It returns any errors that could happen while connecting to Azure Blob Storage.
func NewWithEndpoint(accountName, accountKey, containerName, endpointURL string) (*Cache, error) {
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(containerName) == "" {
		return nil, ErrEmptyContainerName
	}

	pipeline := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	endpoint, _ := url.Parse(endpointURL)
	serviceURL := azblob.NewServiceURL(*endpoint, pipeline)
	containerURL := serviceURL.NewContainerURL(containerName)

	return &Cache{
		containerURL: containerURL,
	}, nil
}

// DeleteContainer based on current configured container
func (c *Cache) DeleteContainer(ctx context.Context) error {
	_, err := c.containerURL.Delete(ctx, azblob.ContainerAccessConditions{})
	if isNotFound(err) {
		return nil
	}
	return err
}

// CreateContainer based on current configured container
func (c *Cache) CreateContainer(ctx context.Context) error {
	_, err := c.containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	return err
}

// Get returns a certificate data for the specified key.
// If there's no such key, Get returns ErrCacheMiss.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	blobURL := c.containerURL.NewBlockBlobURL(key)
	get, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)

	if isNotFound(err) {
		return nil, autocert.ErrCacheMiss
	}

	if err != nil {
		return nil, err
	}

	reader := get.Body(azblob.RetryReaderOptions{})
	defer reader.Close()
	data := &bytes.Buffer{}
	data.ReadFrom(reader)

	return data.Bytes(), nil
}

// Put stores the data in the cache under the specified key.
func (c *Cache) Put(ctx context.Context, key string, data []byte) error {
	blobURL := c.containerURL.NewBlockBlobURL(key)
	_, err := blobURL.Upload(ctx, bytes.NewReader(data), azblob.BlobHTTPHeaders{
		ContentType: "application/x-pem-file",
	}, azblob.Metadata{}, azblob.BlobAccessConditions{})
	return err
}

// Delete removes a certificate data from the cache under the specified key.
// If there's no such key in the cache, Delete returns nil.
func (c *Cache) Delete(ctx context.Context, key string) error {
	blobURL := c.containerURL.NewBlockBlobURL(key)
	_, err := blobURL.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	if isNotFound(err) {
		return nil
	}
	return err
}
