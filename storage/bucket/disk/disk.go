package disk

import (
	"context"

	"github.com/omalloc/tavern/api/defined/v1/storage"
	"github.com/omalloc/tavern/api/defined/v1/storage/object"
	"github.com/omalloc/tavern/conf"
)

var _ storage.Bucket = (*diskBucket)(nil)

type diskBucket struct {
	path      string
	driver    string
	storeType string
	weight    int
	sharedkv  storage.SharedKV
}

func New(config *conf.Bucket, sharedkv storage.SharedKV) (storage.Bucket, error) {
	return &diskBucket{
		path:      config.Path,
		driver:    config.Driver,
		storeType: config.Type,
		weight:    100, // default weight
		sharedkv:  sharedkv,
	}, nil
}

// Allow implements storage.Bucket.
func (d *diskBucket) Allow() int {
	panic("unimplemented")
}

// Close implements storage.Bucket.
func (d *diskBucket) Close() error {
	panic("unimplemented")
}

// Discard implements storage.Bucket.
func (d *diskBucket) Discard(ctx context.Context, id *object.ID) error {
	panic("unimplemented")
}

// DiscardWithHash implements storage.Bucket.
func (d *diskBucket) DiscardWithHash(ctx context.Context, hash object.IDHash) error {
	panic("unimplemented")
}

// DiscardWithMessage implements storage.Bucket.
func (d *diskBucket) DiscardWithMessage(ctx context.Context, id *object.ID, msg string) error {
	panic("unimplemented")
}

// DiscardWithMetadata implements storage.Bucket.
func (d *diskBucket) DiscardWithMetadata(ctx context.Context, meta *object.Metadata) error {
	panic("unimplemented")
}

// Exist implements storage.Bucket.
func (d *diskBucket) Exist(ctx context.Context, id []byte) bool {
	panic("unimplemented")
}

// Expired implements storage.Bucket.
func (d *diskBucket) Expired(ctx context.Context, id *object.ID, md *object.Metadata) bool {
	panic("unimplemented")
}

// HasBad implements storage.Bucket.
func (d *diskBucket) HasBad() bool {
	return false
}

// ID implements storage.Bucket.
func (d *diskBucket) ID() string {
	return d.path
}

// Iterate implements storage.Bucket.
func (d *diskBucket) Iterate(ctx context.Context, fn func(*object.Metadata) error) error {
	panic("unimplemented")
}

// Lookup implements storage.Bucket.
func (d *diskBucket) Lookup(ctx context.Context, id *object.ID) (*object.Metadata, error) {
	// find indexdb metadata by cache-key
	return &object.Metadata{}, nil
}

// Remove implements storage.Bucket.
func (d *diskBucket) Remove(ctx context.Context, id *object.ID) error {
	panic("unimplemented")
}

// Store implements storage.Bucket.
func (d *diskBucket) Store(ctx context.Context, meta *object.Metadata) error {
	panic("unimplemented")
}

// StoreType implements storage.Bucket.
func (d *diskBucket) StoreType() string {
	return d.storeType
}

// Type implements storage.Bucket.
func (d *diskBucket) Type() string {
	return d.driver
}

// UseAllow implements storage.Bucket.
func (d *diskBucket) UseAllow() bool {
	// TODO: check disk usage if the bucket is full, return false
	return true
}

// Weight implements storage.Bucket.
func (d *diskBucket) Weight() int {
	return d.weight
}
