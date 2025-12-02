package storage

import (
	"errors"

	"github.com/omalloc/tavern/api/defined/v1/storage"
	"github.com/omalloc/tavern/conf"
	"github.com/omalloc/tavern/storage/bucket/disk"
	"github.com/omalloc/tavern/storage/bucket/empty"
)

type GlobalBucketOption struct {
	AsyncLoad       bool
	EvictionPolicy  string
	SelectionPolicy string
	Driver          string
}

// implements storage.Bucket map.
var bucketMap = map[string]func(opt *conf.Bucket, sharedkv storage.SharedKV) (storage.Bucket, error){
	"empty":  empty.New,
	"native": disk.New, // disk is an alias of native
}

func NewBucket(opt *conf.Bucket, sharedkv storage.SharedKV) (storage.Bucket, error) {
	factory, exist := bucketMap[opt.Driver]
	if !exist {
		return nil, errors.New("bucket factory not found")
	}

	return factory(opt, sharedkv)
}

func mergeConfig(global *GlobalBucketOption, bucket *conf.Bucket) *conf.Bucket {
	// copied from conf bucket.
	copied := &conf.Bucket{
		Path:   bucket.Path,
		Driver: bucket.Driver,
		Type:   bucket.Type,
	}

	if copied.Driver == "" {
		copied.Driver = global.Driver
	}
	if copied.Type == "" {
		copied.Type = "normal"
	}
	return copied
}
