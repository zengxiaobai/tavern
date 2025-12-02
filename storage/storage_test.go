package storage_test

import (
	"context"
	"testing"

	"github.com/omalloc/tavern/api/defined/v1/storage/object"
	"github.com/omalloc/tavern/conf"
	"github.com/omalloc/tavern/contrib/log"
	"github.com/omalloc/tavern/storage"
)

func TestSelect(t *testing.T) {
	s, err := storage.New(&conf.Storage{
		Driver:          "native",
		AsyncLoad:       true,
		EvictionPolicy:  "lru",
		SelectionPolicy: "hashring",
		Buckets: []*conf.Bucket{
			{Path: "/cache1", Type: "normal"},
			{Path: "/cache2", Type: "normal"},
		},
	}, log.DefaultLogger)
	if err != nil {
		t.Fatal(err)
	}

	cacheKey := object.NewID("http://www.example.com/path/to/1K.bin")

	bucket := s.Select(context.Background(), cacheKey)

	if bucket == nil {
		t.Fatal("no bucket selected")
	}

	md, err := bucket.Lookup(context.Background(), cacheKey)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("object metadata: %+v", md)
}
