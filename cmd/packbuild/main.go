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
	deep := flag.Bool("deep", false, "build multi-source deep pack (Wikidata + GeoNames + IMDb + Open Food Facts)")
	minEntries := flag.Int("min-entries", 0, "drop prompts with fewer answers (deep/krillion default 10000 when flag omitted with --deep/--fetch-krillion)")
	skipDatasets := flag.Bool("skip-datasets", false, "with --deep, only query Wikidata (skip GeoNames/IMDb/OFF)")
	skipWikidata := flag.Bool("skip-wikidata", false, "with --deep, only build open datasets (skip paginated Wikidata)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *deep {
		min := *minEntries
		if !flagPassed("min-entries") {
			min = packbuild.MinEntries
		}
		outPath := *out
		if outPath == "packs/core-v1.json" {
			outPath = "packs/deep-v1.json"
		}
		pack, err := packbuild.BuildDeep(ctx, packbuild.DeepOptions{
			OutPath:      outPath,
			PackID:       strings.TrimSuffix(filepath.Base(outPath), filepath.Ext(outPath)),
			CacheDir:     *cache,
			Offline:      *offline,
			MinEntries:   min,
			SkipDatasets: *skipDatasets,
			SkipWikidata: *skipWikidata,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "packbuild: deep: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s (%d prompts)\n", outPath, len(pack.Prompts))
		upsertPack(ctx, pack)
		return
	}

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
		min := *minEntries
		if !flagPassed("min-entries") {
			min = packbuild.MinEntries
		}
		pack, err := packbuild.BuildKrillionPackOpts(packbuild.KrillionBuildOptions{
			ImportDir:  importDir,
			OutPath:    outPath,
			PackID:     packID,
			MaxEntries: packbuild.DefaultMaxEntries,
			MinEntries: min,
		})
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
		DefsDir:    *defs,
		OutPath:    *out,
		PackID:     packID,
		Only:       *only,
		Offline:    *offline,
		CacheDir:   *cache,
		MinEntries: *minEntries,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "packbuild: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d prompts)\n", *out, len(pack.Prompts))
	upsertPack(ctx, pack)
}

func flagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
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
