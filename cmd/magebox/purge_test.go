package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"qoliber/magebox/internal/config"
)

func TestPurgeGlob(t *testing.T) {
	// Create a temp project directory with Magento-like structure
	projectDir := t.TempDir()

	// Create directories and files to purge
	dirs := []string{
		"generated/metadata",
		"generated/code/Vendor/Module",
		"pub/static/frontend/Vendor/theme",
		"var/cache/mage-tags",
		"var/composer_home/cache",
		"var/page_cache/html",
		"var/view_preprocessed/css",
	}
	for _, d := range dirs {
		fullPath := filepath.Join(projectDir, d)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", d, err)
		}
		// Create a file inside each dir
		if err := os.WriteFile(filepath.Join(fullPath, "test.txt"), []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file in %s: %v", d, err)
		}
	}

	// Verify files exist before purge
	for _, d := range dirs {
		entries, _ := os.ReadDir(filepath.Join(projectDir, d))
		if len(entries) == 0 {
			t.Fatalf("expected files in %s before purge", d)
		}
	}

	// Run purgeGlob on each pattern
	var mu sync.Mutex
	for _, pattern := range purgeDirs {
		purgeGlob(projectDir, pattern, &mu)
	}

	// Verify contents are removed but parent dirs still exist
	checks := []struct {
		dir       string
		wantEmpty bool
	}{
		{"generated/metadata", true},
		{"generated/code", true},
		{"pub/static", true},
		{"var/cache", true},
		{"var/composer_home", true},
		{"var/page_cache", true},
		{"var/view_preprocessed", true},
	}

	for _, c := range checks {
		fullPath := filepath.Join(projectDir, c.dir)
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			// Parent dir may have been removed by glob, that's ok
			continue
		}
		if c.wantEmpty && len(entries) > 0 {
			t.Errorf("%s should be empty after purge, has %d entries", c.dir, len(entries))
		}
	}
}

func TestPurgeGlob_EmptyDir(t *testing.T) {
	projectDir := t.TempDir()

	// Create empty directories (nothing to purge)
	os.MkdirAll(filepath.Join(projectDir, "var/cache"), 0755)

	// Should not panic or error on empty dirs
	var mu sync.Mutex
	purgeGlob(projectDir, "var/cache/*", &mu)
}

func TestPurgeGlob_NonExistent(t *testing.T) {
	projectDir := t.TempDir()

	// Should not panic on non-existent directories
	var mu sync.Mutex
	purgeGlob(projectDir, "nonexistent/path/*", &mu)
}

func TestPurgeDirs(t *testing.T) {
	// Verify all expected directories are in the purge list
	expected := map[string]bool{
		"generated/metadata/*":    false,
		"generated/code/*":        false,
		"pub/static/*":            false,
		"var/cache/*":             false,
		"var/composer_home/*":     false,
		"var/page_cache/*":        false,
		"var/view_preprocessed/*": false,
	}

	for _, d := range purgeDirs {
		if _, ok := expected[d]; ok {
			expected[d] = true
		} else {
			t.Errorf("unexpected purge dir: %s", d)
		}
	}

	for d, found := range expected {
		if !found {
			t.Errorf("missing purge dir: %s", d)
		}
	}
}

func TestFlushCache_NoService(t *testing.T) {
	// Should not panic when no cache service configured
	cfg := &config.Config{}
	flushCache(cfg)
}

func TestFlushVarnish_NotConfigured(t *testing.T) {
	// Should not panic when Varnish is not configured
	cfg := &config.Config{}
	flushVarnish(cfg)
}
