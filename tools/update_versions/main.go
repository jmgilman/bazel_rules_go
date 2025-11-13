// Package main provides a tool for updating golangci-lint version information
// in Bazel Starlark files by fetching releases from GitHub and generating
// checksum data for all supported platforms.
package main

import (
	"context"
	"flag"
	"log"
	"os"
)

var (
	count      = flag.Int("count", 10, "Number of versions to process")
	cacheDir   = flag.String("cache-dir", "tools/update_versions/cache/checksums", "Cache directory for checksum files")
	outputFile = flag.String("output", "golangci_lint/private/versions.bzl", "Output file path for generated Starlark")
)

func main() {
	flag.Parse()

	if *count <= 0 {
		log.Fatal("count must be positive")
	}

	// Determine workspace root
	// When running via `bazel run`, Bazel sets BUILD_WORKSPACE_DIRECTORY
	workspaceRoot := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if workspaceRoot == "" {
		// Fallback to current working directory if not running via Bazel
		var err error
		workspaceRoot, err = os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %v", err)
		}
	}

	// Create configuration
	config := Config{
		Count:         *count,
		CacheDir:      *cacheDir,
		OutputFile:    *outputFile,
		WorkspaceRoot: workspaceRoot,
	}

	// Initialize GitHub client
	client := NewGitHubClient()

	// Create runner and execute
	runner := NewRunner(config, client)
	ctx := context.Background()

	if err := runner.Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
