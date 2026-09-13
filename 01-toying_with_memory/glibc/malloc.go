//go:build linux && cgo

package glibc

/*
#include <stdlib.h>
#include <malloc.h>
#include <stdint.h>

void* demo_malloc(size_t n) { return malloc(n); }
void  demo_free(void* p) { free(p); }
size_t demo_usable_size(void* p) { return malloc_usable_size(p); }
size_t demo_malloc_trim(size_t pad) { return malloc_trim(pad); }
*/
import "C"
import "unsafe"

func Malloc(n uintptr) unsafe.Pointer     { return C.demo_malloc(C.size_t(n)) }
func Free(p unsafe.Pointer)               { C.demo_free(p) }
func UsableSize(p unsafe.Pointer) uintptr { return uintptr(C.demo_usable_size(p)) }
func Trim(pad uintptr) bool               { return C.demo_malloc_trim(C.size_t(pad)) != 0 }
