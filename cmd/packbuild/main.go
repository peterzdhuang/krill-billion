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

	"github.com/peterzdhuang/krill-billion/internal/data"
	"github.com/peterzdhuang/krill-billion/internal/packbuild"
	"github.com/peterzdhuang/krill-billion/internal/supabank"
)

func main() {
	defs := flag.String("defs", "packs/defs", "directory of prompt YAML defs")
	out := flag.String("out", "packs/core-v1.json", "output pack JSON path")
	only := flag.String("only", "", "build only this prompt id")
	offline := flag.Bool("offline", false, "use disk cache only (no network)")
	cache := flag.String("cache", ".packbuild-cache", "disk cache directory")
	krillionImport := flag.String("krillion-import", "", "directory of dated Krillion reveal JSON (build krillion pack)")
	fetchKrillion := flag.Bool("fetch-krillion", false, "download Krillion reveals into --krillion-import then build")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *fetchKrillion || *krillionImport != "" {
		importDir := *krillionImport
		if importDir == "" {
			importDir = "packs/import/krillion"
		}
		outPath := *out
		if outPath == "packs/core-v1.json" {
			outPath = "packs/krillion-v1.json"
		}
		if *fetchKrillion {
			if err := packbuild.FetchKrillion(ctx, importDir, nil); err != nil {
				fmt.Fprintf(os.Stderr, "packbuild: fetch-krillion: %v\n", err)
				os.Exit(1)
			}
		}
		packID := strings.TrimSuffix(filepath.Base(outPath), filepath.Ext(outPath))
		pack, err := packbuild.BuildKrillionPack(importDir, outPath, packID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "packbuild: krillion: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s (%d prompts)\n", outPath, len(pack.Prompts))
		upsertPack(ctx, pack)
		return
	}

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
	upsertPack(ctx, pack)
}

func upsertPack(ctx context.Context, pack *data.Pack) {
	cfg, ok := supabank.FromEnv()
	if !ok {
		fmt.Fprintf(os.Stderr, "supabase not configured; skipped upsert\n")
		return
	}
	fmt.Fprintf(os.Stderr, "upserting %d prompts to supabase (%s)\n", len(pack.Prompts), cfg.Mode())
	if err := supabank.Upsert(ctx, cfg, pack); err != nil {
		fmt.Fprintf(os.Stderr, "packbuild: supabase upsert: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "supabase upsert ok\n")
}
