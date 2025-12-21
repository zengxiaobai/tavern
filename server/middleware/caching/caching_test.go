package caching

import "testing"

func BenchmarkWithPooling(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c := cachingPool.Get().(*Caching)
		c.reset()
	}
}
