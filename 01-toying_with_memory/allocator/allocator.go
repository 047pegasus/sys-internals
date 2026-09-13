//go:build linux && cgo

package allocator

/*
#include <sys/mman.h>
#include <unistd.h>
#include <stdint.h>
#include <errno.h>
#include <string.h>
#include <stdio.h>

struct block_header {
    size_t size;                 // payload size
    int free;                    // 1 = free, 0 = used
    struct block_header* next;
};

static struct block_header* head = NULL;
static size_t arena_size = 0;
static size_t page_size = 0;

static size_t align8(size_t n) { return (n + 7u) & ~7u; }

static void init_page_size(void) {
    if (!page_size) page_size = (size_t)sysconf(_SC_PAGESIZE);
}

static size_t round_pages(size_t n) {
    init_page_size();
    return (n + page_size - 1) / page_size * page_size;
}

static void split_block(struct block_header* b, size_t wanted) {
    const size_t min_payload = 16;
    const size_t header = sizeof(struct block_header);
    if (b->size < wanted + header + min_payload) return;
    char* base = (char*)b;
    struct block_header* next = (struct block_header*)(base + header + wanted);
    next->size = b->size - wanted - header;
    next->free = 1;
    next->next = b->next;
    b->size = wanted;
    b->next = next;
}

static void coalesce(void) {
    struct block_header* b = head;
    while (b && b->next) {
        if (b->free && b->next->free) {
            b->size += sizeof(struct block_header) + b->next->size;
            b->next = b->next->next;
        } else {
            b = b->next;
        }
    }
}

void* edu_malloc(size_t n) {
    if (n == 0) n = 1;
    n = align8(n);

    for (struct block_header* b = head; b; b = b->next) {
        if (b->free && b->size >= n) {
            b->free = 0;
            split_block(b, n);
            return (char*)b + sizeof(struct block_header);
        }
    }

    size_t total = round_pages(sizeof(struct block_header) + n);
    void* region = mmap(NULL, total, PROT_READ | PROT_WRITE,
                        MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (region == MAP_FAILED) return NULL;

    struct block_header* b = (struct block_header*)region;
    b->size = total - sizeof(struct block_header);
    b->free = 0;
    b->next = NULL;
    if (!head) head = b;
    else {
        struct block_header* tail = head;
        while (tail->next) tail = tail->next;
        tail->next = b;
    }
    arena_size += total;
    split_block(b, n);
    return (char*)b + sizeof(struct block_header);
}

void edu_free(void* p) {
    if (!p) return;
    struct block_header* b = (struct block_header*)((char*)p - sizeof(struct block_header));
    b->free = 1;
    coalesce();
}

void edu_dump(void) {
    printf("custom allocator blocks:\n");
    size_t i = 0;
    for (struct block_header* b = head; b; b = b->next, ++i) {
        printf("  #%zu header=%p payload=%p size=%zu %s\n",
               i, (void*)b, (void*)((char*)b + sizeof(struct block_header)),
               b->size, b->free ? "FREE" : "USED");
    }
    printf("  mapped bytes: %zu\n", arena_size);
    fflush(stdout);
}

size_t edu_arena_size(void) { return arena_size; }
size_t edu_header_size(void) { return sizeof(struct block_header); }
size_t edu_page_size(void) { init_page_size(); return page_size; }

// edu_usable_size returns the size of the block backing payload pointer p,
// i.e. how much room the allocator actually set aside for it (mirrors
// glibc's malloc_usable_size, so the demo can show "asked for X, got Y").
size_t edu_usable_size(void* p) {
    if (!p) return 0;
    struct block_header* b = (struct block_header*)((char*)p - sizeof(struct block_header));
    return b->size;
}
*/
import "C"
import "unsafe"

func Malloc(n uintptr) unsafe.Pointer     { return C.edu_malloc(C.size_t(n)) }
func Free(p unsafe.Pointer)               { C.edu_free(p) }
func Dump()                               { C.edu_dump() }
func ArenaSize() uintptr                  { return uintptr(C.edu_arena_size()) }
func HeaderSize() uintptr                 { return uintptr(C.edu_header_size()) }
func PageSize() uintptr                   { return uintptr(C.edu_page_size()) }
func UsableSize(p unsafe.Pointer) uintptr { return uintptr(C.edu_usable_size(p)) }
