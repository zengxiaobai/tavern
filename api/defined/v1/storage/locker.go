package storage

type ResourceLocker interface {
	// Key returns the lock key.
	Key() string
	// Lock acquires the lock.
	Lock()
	// Unlock releases the lock.
	Unlock()
	// RLock acquires a read lock.
	RLock()
	// RUnlock releases a read lock.
	RUnlock()
}
