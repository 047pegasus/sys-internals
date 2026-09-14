# malloc-demo

A small Linux/Go systems-programming experiment that makes heap allocation visible.

It demonstrates two different things:

1. **glibc `malloc()` from Go via cgo** — this is the real C allocator used by the C library. The program prints the returned address, glibc's usable allocation size, RSS, and `/proc/self/maps` information.
2. **An educational allocator implemented in C and orchestrated from Go** — it uses `mmap()` to obtain page-aligned regions, stores a small header before every block, maintains a singly-linked free list, reuses freed blocks, splits large free blocks, and coalesces adjacent free blocks.

The educational allocator intentionally does **not** try to reproduce glibc, jemalloc, tcmalloc, or mimalloc. Its purpose is to make the core allocator mechanics easy to observe.

## Requirements

- Linux
- Go 1.22+
- GCC / a C compiler
- cgo enabled (`CGO_ENABLED=1`)

Ubuntu/Debian:

```bash
sudo apt install build-essential
```

## Run

```bash
go run ./cmd/malloc-demo
```

You should see output similar to:

```text
=== 1. glibc malloc ===
malloc(64)       -> 0x..., usable=...
malloc(4096)     -> 0x..., usable=...
malloc(131072)   -> 0x..., usable=...
malloc(1048576)  -> 0x..., usable=...
malloc(10485760) -> 0x..., usable=...

=== 2. educational mmap + free-list allocator ===
...
```

Exact addresses, usable sizes, RSS, and mapping behavior are system/libc dependent.

The console output is written in plain language on purpose (e.g. "Trying to
allocate 29 KB... malloc gave us 32 KB instead") so it's readable by someone
seeing allocator internals for the first time, not just people who already
know the jargon.

## Turn the run into graphs

Every run also writes a `memory_log.csv` file (in the directory you ran the
program from) with one row per allocation/free/trim event: what was
requested, what was actually handed back, and the process's total memory
usage (RSS) at that moment.

Generate two graphs from it:

```bash
pip install matplotlib
python3 scripts/plot_memory.py
```

This produces:

- **`requested_vs_allocated.png`** — for every allocation, a bar for the size
  you asked for next to a bar for the size the allocator actually gave you.
  Makes the "allocators round up" behavior visible instead of abstract.
- **`freed_vs_held.png`** — over the course of the run, a line for how much
  memory you've *asked* to free (cumulative) next to a line for how much
  memory has *actually* gone back to the operating system, with the gap
  between them shaded in. That gap is memory `free()` released logically but
  that the allocator (glibc or the educational one) is still holding onto
  instead of returning to the OS's free pool. For glibc, that gap shrinks
  once `malloc_trim(0)` runs; for the educational allocator, it never does,
  since it deliberately never calls `munmap`.

You can point the script at a different file or output folder:

```bash
python3 scripts/plot_memory.py path/to/memory_log.csv path/to/output_dir
```

## Trace the real allocator

On Linux, `strace` makes the OS boundary visible:

```bash
strace -f -e trace=brk,mmap,munmap,madvise ./malloc-demo
```

The interesting observation is that a `malloc()` call does not necessarily correspond to one syscall. The allocator normally obtains larger regions from the kernel and services many application allocations from those regions.

## Benchmark

```bash
go test -bench=. -benchmem ./allocator
```

For a rough comparison against glibc, you can add another benchmark that calls `glibc.Malloc`/`glibc.Free` in the same way.

## Inspect the process

While the program is paused under a debugger, or while adding a sleep, inspect:

```bash
cat /proc/$PID/maps
cat /proc/$PID/smaps_rollup
```

Useful commands:

```bash
pmap -x $PID
```

and:

```bash
strace -f -e trace=brk,mmap,munmap,madvise ./malloc-demo
```

## What to notice

### 1. `malloc()` is not a syscall

`malloc()` is a user-space allocator API. The allocator talks to the kernel only when it needs more virtual memory or wants to return/release memory.

### 2. Heap and mmap are different mechanisms

Linux/glibc can obtain memory using the traditional process heap (`brk`) and anonymous mappings (`mmap`). The exact decision is implementation- and configuration-dependent.

### 3. A freed allocation may remain mapped

`free()` tells the allocator that the block can be reused. It does not imply that the pages immediately disappear from the process address space.

### 4. Fragmentation matters

The educational allocator shows how a large free block can be split to satisfy a smaller request, and how adjacent free blocks can be merged again.

### 5. Go is a separate allocator story

A normal Go `make([]byte, n)` is managed by the Go runtime, not by directly calling glibc `malloc()` for each allocation. The Go runtime itself has OS-level memory code that uses anonymous `mmap()` on Linux. This project therefore deliberately calls C `malloc()` when the goal is to study glibc.

## Safety / limitations

The educational allocator is intentionally minimal:

- no thread safety
- no `realloc`
- no `calloc`
- no arena locking
- no per-size bins
- no per-thread caches
- no returning individual free regions to Linux
- no corruption detection
- no production use

It is an educational model of allocator fundamentals, not a replacement for a production allocator.

## Reading material & suggested content:
https://dev.to/frosnerd/libmalloc-jemalloc-tcmalloc-mimalloc-exploring-different-memory-allocators-4lp3

https://go.dev/src/runtime/malloc.go

https://medium.com/@ankur_anand/a-visual-guide-to-golang-memory-allocator-from-ground-up-e132258453ed
