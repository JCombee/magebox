package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/config"
)

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge all generated files and caches",
	Long: `Removes all generated files, preprocessed views, and caches.

Clears:
  - generated/metadata/*
  - generated/code/*
  - pub/static/*
  - var/cache/*
  - var/composer_home/*
  - var/page_cache/*
  - var/view_preprocessed/*
  - Redis/Valkey (if configured)
  - Varnish (if configured)

Examples:
  magebox purge`,
	RunE: runPurge,
}

func init() {
	rootCmd.AddCommand(purgeCmd)
}

// purgeDirs are the Magento directories to clean, relative to project root
var purgeDirs = []string{
	"generated/metadata/*",
	"generated/code/*",
	"pub/static/*",
	"var/cache/*",
	"var/composer_home/*",
	"var/page_cache/*",
	"var/view_preprocessed/*",
}

func runPurge(cmd *cobra.Command, args []string) error {
	cwd, err := getCwd()
	if err != nil {
		return err
	}

	cfg, ok := loadProjectConfig(cwd)
	if !ok {
		return nil
	}

	cli.PrintTitle("Purge")
	fmt.Println()

	// Remove generated files and caches in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, pattern := range purgeDirs {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			purgeGlob(cwd, p, &mu)
		}(pattern)
	}
	wg.Wait()

	// Flush Redis/Valkey
	flushCache(cfg)

	// Flush Varnish
	flushVarnish(cfg)

	fmt.Println()
	cli.PrintSuccess("All caches purged!")
	return nil
}

// purgeGlob removes all files matching a glob pattern relative to the project root
func purgeGlob(cwd, pattern string, mu *sync.Mutex) {
	fullPattern := filepath.Join(cwd, pattern)
	matches, _ := filepath.Glob(fullPattern)

	for _, match := range matches {
		os.RemoveAll(match)
	}

	mu.Lock()
	fmt.Printf("  %s %s\n", cli.Success(cli.SymbolCheck), pattern)
	mu.Unlock()
}

// flushCache flushes Redis or Valkey if configured
func flushCache(cfg *config.Config) {
	if cfg.Services.HasRedis() {
		flushRedis("magebox-redis")
	} else if cfg.Services.HasValkey() {
		flushRedis("magebox-valkey")
	}
}

// flushRedis sends FLUSHALL to a Redis-compatible container
func flushRedis(containerName string) {
	cmd := exec.Command("docker", "exec", containerName, "redis-cli", "flushall")
	if err := cmd.Run(); err != nil {
		fmt.Printf("  %s Redis flush (not running)\n", cli.Warning(cli.SymbolWarning))
		return
	}
	fmt.Printf("  %s Redis caches flushed\n", cli.Success(cli.SymbolCheck))
}

// flushVarnish bans all URLs if Varnish is configured
func flushVarnish(cfg *config.Config) {
	if !cfg.Services.HasVarnish() {
		return
	}

	cmd := exec.Command("docker", "exec", "magebox-varnish", "varnishadm", "ban", "req.url", "~", ".")
	if err := cmd.Run(); err != nil {
		fmt.Printf("  %s Varnish flush (not running)\n", cli.Warning(cli.SymbolWarning))
		return
	}
	fmt.Printf("  %s Varnish caches flushed\n", cli.Success(cli.SymbolCheck))
}
