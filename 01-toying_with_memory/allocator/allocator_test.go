//go:build linux && cgo

package allocator

import "testing"

func BenchmarkMallocFree64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p := Malloc(64)
		if p == nil {
			b.Fatal("allocation failed")
		}
		Free(p)
	}
}
