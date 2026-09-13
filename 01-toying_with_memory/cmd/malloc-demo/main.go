//go:build linux && cgo

package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"01-toying_with_memory/allocator"
	"01-toying_with_memory/glibc"
)

// ---------------------------------------------------------------------------
// Small helpers for plain-language, human-friendly output.
// ---------------------------------------------------------------------------

// friendlySize turns a raw byte count into something like "29 KB" or "1.0 MB"
// instead of a big number of bytes, which is what actually confuses people
// new to this topic.
func friendlySize(n uint64) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%d bytes", n)
	}
}

func rssKB() uint64 {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) >= 2 && fields[0] == "VmRSS:" {
			n, _ := strconv.ParseUint(fields[1], 10, 64)
			return n
		}
	}
	return 0
}

func mapsSummary() {
	data, err := os.ReadFile("/proc/self/maps")
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	heap, anon := 0, 0
	for _, line := range lines {
		if strings.Contains(line, "[heap]") {
			heap++
		}
		if strings.Contains(line, "rw-p") && !strings.Contains(line, "/") && !strings.Contains(line, "[stack") && !strings.Contains(line, "[heap]") {
			anon++
		}
	}
	fmt.Printf("(behind the scenes: the process currently has %d traditional heap mapping(s) and roughly %d other anonymous memory mapping(s))\n", heap, anon)
}

func touch(p unsafe.Pointer, n uintptr) {
	b := unsafe.Slice((*byte)(p), n)
	for i := 0; i < len(b); i += 4096 {
		b[i] = byte(i)
	}
	if len(b) > 0 {
		b[len(b)-1] = 1
	}
}

// ---------------------------------------------------------------------------
// CSV logging so the accompanying Python script can turn the run into graphs.
// Every row is one "moment in time": what we asked for, what we actually got,
// and how much memory the whole process was using at that instant.
// ---------------------------------------------------------------------------

type memLogger struct {
	w    *csv.Writer
	f    *os.File
	step int
}

func newMemLogger(path string) *memLogger {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "note: could not open %s for writing (%v) -- continuing without a log file\n", path, err)
		return &memLogger{}
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"step", "section", "event", "label", "requested_bytes", "allocated_bytes", "rss_kb"})
	w.Flush()
	return &memLogger{w: w, f: f}
}

func (m *memLogger) log(section, event, label string, requestedBytes, allocatedBytes, rssKBVal uint64) {
	if m.w == nil {
		return
	}
	m.step++
	_ = m.w.Write([]string{
		strconv.Itoa(m.step),
		section,
		event,
		label,
		strconv.FormatUint(requestedBytes, 10),
		strconv.FormatUint(allocatedBytes, 10),
		strconv.FormatUint(rssKBVal, 10),
	})
	m.w.Flush()
}

func (m *memLogger) close() {
	if m.f == nil {
		return
	}
	m.w.Flush()
	m.f.Close()
}

// ---------------------------------------------------------------------------
// Demo 1: the real allocator, glibc's malloc()
// ---------------------------------------------------------------------------

func glibcDemo(log *memLogger) {
	fmt.Println("\n=== 1. The real thing: glibc's malloc() ===")
	fmt.Println("This is the exact malloc() that C (and most Linux programs) use every day.")

	startRSS := rssKB()
	fmt.Printf("Memory this process is using right now: %s\n", friendlySize(startRSS*1024))
	log.log("glibc", "start", "process start", 0, 0, startRSS)

	sizes := []uintptr{64, 4096, 128 * 1024, 1024 * 1024, 10 * 1024 * 1024}
	ptrs := make([]unsafe.Pointer, 0, len(sizes))

	for _, n := range sizes {
		fmt.Printf("\nTrying to allocate %s...\n", friendlySize(uint64(n)))
		p := glibc.Malloc(n)
		if p == nil {
			panic("glibc malloc failed")
		}
		touch(p, n)
		usable := uint64(glibc.UsableSize(p))
		fmt.Printf("  -> malloc gave us %s instead\n", friendlySize(usable))
		if usable > uint64(n) {
			fmt.Println("     (malloc rounds requests up to sizes it manages internally, so you usually get a little more than you asked for)")
		}
		rss := rssKB()
		fmt.Printf("  -> total memory this process is using now: %s\n", friendlySize(rss*1024))
		log.log("glibc", "alloc", fmt.Sprintf("malloc(%s)", friendlySize(uint64(n))), uint64(n), usable, rss)
		ptrs = append(ptrs, p)
	}

	fmt.Println("\nNow freeing every block we just allocated...")
	for i, p := range ptrs {
		glibc.Free(p)
		rss := rssKB()
		fmt.Printf("  -> freed the %s block, total memory in use now: %s\n", friendlySize(uint64(sizes[i])), friendlySize(rss*1024))
		log.log("glibc", "free", fmt.Sprintf("free() the %s block", friendlySize(uint64(sizes[i]))), uint64(sizes[i]), 0, rss)
	}

	fmt.Println("\nNotice that the memory total did not drop back down to where it started.")
	fmt.Println("free() means \"I'm done with this,\" not \"give it back to the operating system")
	fmt.Println("right now.\" malloc keeps that memory reserved for itself in case the program")
	fmt.Println("asks for more soon -- handing pages back to the OS is comparatively slow.")

	fmt.Println("\nAsking malloc directly to hand back any spare memory it's holding (malloc_trim)...")
	glibc.Trim(0)
	rss := rssKB()
	fmt.Printf("  -> total memory this process is using now: %s\n", friendlySize(rss*1024))
	log.log("glibc", "trim", "malloc_trim(0)", 0, 0, rss)
	mapsSummary()
}

// ---------------------------------------------------------------------------
// Demo 2: a tiny educational allocator we wrote ourselves
// ---------------------------------------------------------------------------

func logCustomAlloc(log *memLogger, label string, p unsafe.Pointer, requested uint64) {
	usable := uint64(allocator.UsableSize(p))
	fmt.Printf("  -> %s: asked for %s, the allocator set aside a %s slot for it\n", label, friendlySize(requested), friendlySize(usable))
	if usable != requested {
		fmt.Println("     (rounded up so every block lines up on an 8-byte boundary)")
	}
	log.log("custom", "alloc", label, requested, usable, rssKB())
}

func customDemo(log *memLogger) {
	fmt.Println("\n=== 2. A tiny allocator we wrote ourselves ===")
	fmt.Println("Same idea as malloc, but simplified so every step is visible.")
	fmt.Printf("The operating system only hands out memory in %s chunks (called pages).\n", friendlySize(uint64(allocator.PageSize())))
	fmt.Printf("Every block we track also carries a small %s bookkeeping tag before it.\n", friendlySize(uint64(allocator.HeaderSize())))

	log.log("custom", "start", "process start", 0, 0, rssKB())

	fmt.Println("\nAsking for three blocks: 64 bytes, 128 bytes, and 256 bytes...")
	a := allocator.Malloc(64)
	b := allocator.Malloc(128)
	c := allocator.Malloc(256)
	if a == nil || b == nil || c == nil {
		panic("custom allocator failed")
	}
	touch(a, 64)
	touch(b, 128)
	touch(c, 256)
	logCustomAlloc(log, "block A", a, 64)
	logCustomAlloc(log, "block B", b, 128)
	logCustomAlloc(log, "block C", c, 256)

	fmt.Println("\nHere's what the allocator's internal notebook looks like right now:")
	allocator.Dump()

	fmt.Println("\nFreeing block B (128 bytes)...")
	allocator.Free(b)
	log.log("custom", "free", "free() block B", 128, 0, rssKB())
	fmt.Println("Block B is now marked as a free hole the allocator can reuse later:")
	allocator.Dump()

	fmt.Println("\nAsking for 100 bytes -- the allocator should reuse that hole instead of asking the OS for more memory:")
	d := allocator.Malloc(100)
	touch(d, 100)
	logCustomAlloc(log, "block D", d, 100)
	allocator.Dump()

	fmt.Println("\nFreeing block A and block D -- since they sit right next to each other, the")
	fmt.Println("allocator can merge (\"coalesce\") them back into one bigger free hole:")
	allocator.Free(a)
	log.log("custom", "free", "free() block A", 64, 0, rssKB())
	allocator.Free(d)
	log.log("custom", "free", "free() block D", 100, 0, rssKB())
	allocator.Dump()

	fmt.Println("\nFreeing the last block (block C, 256 bytes)...")
	allocator.Free(c)
	log.log("custom", "free", "free() block C", 256, 0, rssKB())
	fmt.Println("Everything is freed now, but this simple allocator never gives pages back to")
	fmt.Println("the OS -- it keeps the whole arena mapped, ready to reuse:")
	allocator.Dump()
	fmt.Printf("Total memory this process is using: %s (it did not shrink after freeing everything)\n", friendlySize(rssKB()*1024))
	log.log("custom", "end", "after freeing everything", 0, 0, rssKB())
}

func main() {
	if runtime.GOOS != "linux" {
		fmt.Println("This demo requires Linux.")
		os.Exit(1)
	}

	fmt.Printf("malloc internals demo | pid=%d | Go=%s\n", os.Getpid(), runtime.Version())

	logPath := "memory_log.csv"
	log := newMemLogger(logPath)
	defer log.close()

	glibcDemo(log)
	customDemo(log)

	fmt.Printf("\nA step-by-step record of every allocation/free above was written to %s\n", logPath)
	fmt.Println("Run `python3 scripts/plot_memory.py` to turn it into graphs.")

	var lim syscall.Rlimit
	_ = syscall.Getrlimit(syscall.RLIMIT_AS, &lim)
	fmt.Printf("\n(for reference, this process's virtual memory address-space limit is: %v)\n", lim.Max)
}
