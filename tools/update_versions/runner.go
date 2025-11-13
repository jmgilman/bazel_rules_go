package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Config holds configuration for the version updater.
type Config struct {
	Count         int
	CacheDir      string
	OutputFile    string
	WorkspaceRoot string
}

// Runner orchestrates the version update workflow.
type Runner struct {
	config Config
	client GitHubAPI
}

// NewRunner creates a new Runner with the given configuration and GitHub client.
func NewRunner(config Config, client GitHubAPI) *Runner {
	return &Runner{
		config: config,
		client: client,
	}
}

// Run executes the version update workflow.
func (r *Runner) Run(ctx context.Context) error {
	log.Printf("golangci-lint version updater starting...")
	log.Printf("Workspace root: %s", r.config.WorkspaceRoot)
	log.Printf("Will process %d versions", r.config.Count)
	log.Printf("Cache directory: %s", r.config.CacheDir)
	log.Printf("Output file: %s", r.config.OutputFile)

	// Convert relative paths to absolute paths based on workspace root
	absCacheDir, absOutputFile := r.resolveAbsolutePaths()
	log.Printf("Absolute cache directory: %s", absCacheDir)
	log.Printf("Absolute output file: %s", absOutputFile)

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(absCacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(absOutputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Fetch releases from GitHub
	log.Println("Fetching releases from GitHub...")
	releases, err := r.client.GetLatestReleases(ctx, r.config.Count)
	if err != nil {
		return fmt.Errorf("failed to fetch releases: %w", err)
	}
	log.Printf("Found %d releases", len(releases))

	// Process each release
	versions := r.processReleases(ctx, releases, absCacheDir)

	if len(versions) == 0 {
		return fmt.Errorf("no versions were successfully processed")
	}

	log.Printf("Successfully processed %d versions", len(versions))

	// Prepare template data
	log.Println("Generating Starlark file...")
	templateData := PrepareTemplateData(versions)

	// Generate output file
	if err := GenerateStarlarkFile(templateData, absOutputFile); err != nil {
		return fmt.Errorf("failed to generate output file: %w", err)
	}

	log.Printf("Successfully generated %s", absOutputFile)
	log.Printf("Default version: %s", templateData.DefaultVersion)
	log.Println("Done!")

	return nil
}

// resolveAbsolutePaths converts relative paths to absolute based on workspace root.
func (r *Runner) resolveAbsolutePaths() (absCacheDir, absOutputFile string) {
	if filepath.IsAbs(r.config.CacheDir) {
		absCacheDir = r.config.CacheDir
	} else {
		absCacheDir = filepath.Join(r.config.WorkspaceRoot, r.config.CacheDir)
	}

	if filepath.IsAbs(r.config.OutputFile) {
		absOutputFile = r.config.OutputFile
	} else {
		absOutputFile = filepath.Join(r.config.WorkspaceRoot, r.config.OutputFile)
	}

	return absCacheDir, absOutputFile
}

// processReleases downloads and parses checksums for each release.
func (r *Runner) processReleases(ctx context.Context, releases []Release, cacheDir string) []Version {
	versions := make([]Version, 0, len(releases))

	for _, release := range releases {
		tag := release.TagName
		if tag == "" {
			log.Printf("Warning: skipping release with empty tag")
			continue
		}

		log.Printf("Processing %s...", tag)

		// Check cache
		cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s.txt", tag))

		checksumData, err := r.loadFromCacheOrDownload(ctx, cacheFile, tag)
		if err != nil {
			log.Printf("  Warning: %v", err)
			continue
		}

		// Parse checksum file
		checksums, err := ParseChecksumFile(checksumData)
		if err != nil {
			log.Printf("  Warning: failed to parse checksum file: %v", err)
			continue
		}
		log.Printf("  Found checksums for %d platforms", len(checksums))

		versions = append(versions, Version{
			Tag:       tag,
			Checksums: checksums,
		})
	}

	return versions
}

// loadFromCacheOrDownload attempts to load checksum data from cache, or downloads if not cached.
func (r *Runner) loadFromCacheOrDownload(ctx context.Context, cacheFile, tag string) ([]byte, error) {
	// Try cache first
	if _, err := os.Stat(cacheFile); err == nil {
		log.Printf("  Using cached checksum file")
		data, err := os.ReadFile(cacheFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read cache file: %w", err)
		}
		return data, nil
	}

	// Cache miss - download
	log.Printf("  Downloading checksum file...")

	// Strip 'v' prefix from tag if present for URL
	version := tag
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}

	url := fmt.Sprintf("https://github.com/golangci/golangci-lint/releases/download/%s/golangci-lint-%s-checksums.txt", tag, version)
	data, err := r.client.DownloadAsset(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksum file: %w", err)
	}

	// Save to cache
	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		log.Printf("  Warning: failed to save to cache: %v", err)
		// Continue anyway - we have the data
	} else {
		log.Printf("  Cached checksum file")
	}

	return data, nil
}
