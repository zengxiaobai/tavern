package storage

import (
	"context"
	"sync"

	"github.com/omalloc/tavern/api/defined/v1/storage"
	"github.com/omalloc/tavern/api/defined/v1/storage/object"
)

var (
	mu             sync.Mutex
	defaultStorage storage.Storage
)

func SetDefault(s storage.Storage) {
	mu.Lock()
	defer mu.Unlock()

	defaultStorage = s
}

func Current() storage.Storage {
	mu.Lock()
	defer mu.Unlock()

	return defaultStorage
}

func Select(ctx context.Context, cacheKey *object.ID) storage.Bucket {
	mu.Lock()
	defer mu.Unlock()

	return defaultStorage.Select(ctx, cacheKey)
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()

	return defaultStorage.Close()
}
