package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/peterzdhuang/krill-billion/internal/packbuild"
)

func main() {
	defs := flag.String("defs", "packs/defs", "directory of prompt YAML defs")
	out := flag.String("out", "packs/core-v1.json", "output pack JSON path")
	only := flag.String("only", "", "build only this prompt id")
	offline := flag.Bool("offline", false, "use disk cache only (no network)")
	cache := flag.String("cache", ".packbuild-cache", "disk cache directory")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	packID := strings.TrimSuffix(filepath.Base(*out), filepath.Ext(*out))

	pack, err := packbuild.Build(ctx, packbuild.Options{
		DefsDir:  *defs,
		OutPath:  *out,
		PackID:   packID,
		Only:     *only,
		Offline:  *offline,
		CacheDir: *cache,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "packbuild: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d prompts)\n", *out, len(pack.Prompts))
}
