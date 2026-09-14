package forkdemo

/*
#include <errno.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <sys/wait.h>
#include <unistd.h>

static long read_kb(const char *field) {
    FILE *f = fopen("/proc/self/status", "r");
    if (!f) return -1;
    char line[256];
    long value = -1;
    while (fgets(line, sizeof(line), f)) {
        long kb;
        if (sscanf(line, "%*[^:]: %ld kB", &kb) == 1 && strstr(line, field)) {
            value = kb;
            break;
        }
    }
    fclose(f);
    return value;
}

static long smaps_kb(const char *field) {
    FILE *f = fopen("/proc/self/smaps", "r");
    if (!f) return -1;
    char line[512];
    long total = 0, value;
    while (fgets(line, sizeof(line), f)) {
        if (sscanf(line, "%[^:]: %ld kB", (char[64]){0}, &value) == 2) {
            char key[64];
            if (sscanf(line, "%63[^:]: %ld kB", key, &value) == 2 && strcmp(key, field) == 0)
                total += value;
        }
    }
    fclose(f);
    return total;
}

static void log_state(const char *who, const char *stage) {
    printf("FORK_STATE,%s,%s,%ld,%ld,%ld,%ld\n", who, stage,
           read_kb("VmRSS"), smaps_kb("Private_Dirty"),
           smaps_kb("Shared_Dirty"), smaps_kb("Pss"));
    fflush(stdout);
}

int run_fork_demo(size_t megabytes) {
    size_t bytes = megabytes * 1024UL * 1024UL;
    size_t pagesize = (size_t)sysconf(_SC_PAGESIZE);
    if (pagesize == 0) return 2;

    unsigned char *memory = mmap(NULL, bytes, PROT_READ | PROT_WRITE,
                                 MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (memory == MAP_FAILED) return 3;

    printf("FORK_LOG,Parent reserves %zu MB of anonymous memory.\n", megabytes);
    printf("FORK_LOG,Parent touches every page so the memory is backed by physical pages.\n");
    fflush(stdout);
    for (size_t i = 0; i < bytes; i += pagesize) memory[i] = 1;
    log_state("parent", "before_fork");

    printf("FORK_LOG,Calling fork(). Linux does not eagerly copy every page.\n");
    printf("FORK_LOG,The parent and child initially point at the same physical pages using copy-on-write.\n");
    fflush(stdout);

    pid_t pid = fork();
    if (pid < 0) { munmap(memory, bytes); return 4; }

    if (pid == 0) {
        log_state("child", "after_fork_before_write");
        printf("FORK_LOG,Child now writes to half of the pages. Those writes trigger copy-on-write page copies.\n");
        fflush(stdout);
        for (size_t i = 0; i < bytes / 2; i += pagesize) memory[i] = 2;
        log_state("child", "after_writing_half");
        printf("FORK_LOG,Child exits. The parent still has its own unchanged version of those pages.\n");
        fflush(stdout);
        _exit(0);
    }

    log_state("parent", "after_fork_before_child_write");
    int status = 0;
    waitpid(pid, &status, 0);
    log_state("parent", "after_child_exit");
    printf("FORK_LOG,Child finished. Parent memory is unchanged because copy-on-write kept the copies separate.\n");
    fflush(stdout);

    munmap(memory, bytes);
    return WIFEXITED(status) ? WEXITSTATUS(status) : 5;
}
*/
import "C"

import "fmt"

// Run starts a Linux fork()+copy-on-write demonstration.
// The actual fork happens in C because calling fork directly from Go and then
// continuing to execute Go runtime code in the child is unsafe.
func Run(megabytes int) error {
	if megabytes < 1 {
		return fmt.Errorf("memory size must be at least 1 MB")
	}
	if rc := C.run_fork_demo(C.size_t(megabytes)); rc != 0 {
		return fmt.Errorf("fork demonstration failed with code %d", int(rc))
	}
	return nil
}
