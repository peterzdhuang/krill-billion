// Command heapcheck loads packs/ and prints prompt/entry counts plus heap size.
package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

func main() {
	dir := "packs"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	store, err := data.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load: %v\n", err)
		os.Exit(1)
	}
	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	delta := int64(after.HeapInuse) - int64(before.HeapInuse)
	if delta < 0 {
		delta = int64(after.HeapInuse)
	}
	fmt.Printf("prompts=%d entries=%d packs=%d\n", store.PromptCount(), store.EntryCount(), store.PackCount())
	fmt.Printf("heap_inuse=%.1f MiB (delta≈%.1f MiB) heap_alloc=%.1f MiB sys=%.1f MiB\n",
		float64(after.HeapInuse)/(1024*1024),
		float64(delta)/(1024*1024),
		float64(after.HeapAlloc)/(1024*1024),
		float64(after.Sys)/(1024*1024),
	)
}
