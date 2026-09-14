package main

import (
    "bufio"
    "encoding/csv"
    "fmt"
    "os"
    "strconv"
    "strings"

    "02-your_kernel_lies/forkdemo"
)

func main() {
    fmt.Println("Linux fork() and Copy-on-Write demonstration")
    fmt.Println("------------------------------------------------")
    fmt.Println("This experiment makes the kernel's lazy memory copying visible.")
    fmt.Println()

    if err := forkdemo.Run(32); err != nil {
        fmt.Fprintln(os.Stderr, "error:", err)
        os.Exit(1)
    }
}

// Keep the CSV schema documented here because scripts/plot_memory.py consumes it.
// Expected lines from the C side:
// FORK_STATE,who,stage,VmRSS_kB,Private_Dirty_kB,Shared_Dirty_kB,Pss_kB
var _ = csv.ErrFieldCount
var _ = bufio.ErrInvalidUnreadByte
var _ = strconv.IntSize
var _ = strings.Builder{}
