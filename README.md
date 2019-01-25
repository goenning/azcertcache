[![GoDoc](https://godoc.org/github.com/goenning/azcertcache?status.svg)](https://godoc.org/github.com/goenning/azcertcache) [![Go Report Card](https://goreportcard.com/badge/github.com/goenning/azcertcache)](https://goreportcard.com/report/github.com/goenning/azcertcache)

# azcertcache

[Azure Blob Storage](https://azure.microsoft.com/en-us/services/storage/blobs/) cache for [acme/autocert](https://godoc.org/golang.org/x/crypto/acme/autocert) written in Go.

## Example

```go
containerName := "autocertcache"
cache, err := azcertcache.New("<account name>", "<account key>", containerName)
if err != nil {
  // Handle error
}

m := autocert.Manager{
  Prompt:     autocert.AcceptTOS,
  Cache:      cache,
}

s := &http.Server{
  Addr:      ":https",
  TLSConfig: &tls.Config{GetCertificate: m.GetCertificate},
}

s.ListenAndServeTLS("", "")
```

## Performance

This is just a reminder that autocert has an internal in-memory cache that is used before quering this long-term cache.
So you don't need to worry about your Azure Blob Storage instance being hit many times just to get the same certificate. It should only do once per process+key.

## Thanks

Inspired by https://github.com/danilobuerger/autocert-s3-cache

## License

MIT