package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner_Run_SuccessfulWorkflowWithSingleVersion(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	outputFile := filepath.Join(tempDir, "versions.bzl")

	config := Config{
		Count:         1,
		CacheDir:      cacheDir,
		OutputFile:    outputFile,
		WorkspaceRoot: tempDir,
	}

	// Setup mock client
	mock := NewMockGitHubClient()
	mock.AddRelease("v2.6.1")
	mock.AddAsset(
		"https://github.com/golangci/golangci-lint/releases/download/v2.6.1/golangci-lint-2.6.1-checksums.txt",
		[]byte("aee6e16af4dfa60dd3c4e39536edc905f28369fda3c138090db00c8233cfe450  golangci-lint-2.6.1-darwin-amd64.tar.gz\n"),
	)

	runner := NewRunner(config, mock)
	ctx := context.Background()

	err := runner.Run(ctx)
	require.NoError(t, err, "Runner.Run() should succeed")

	// Verify cache file was created
	cacheFile := filepath.Join(cacheDir, "v2.6.1.txt")
	_, err = os.Stat(cacheFile)
	assert.NoError(t, err, "Runner.Run() should create cache file")

	// Verify output file was created
	_, err = os.Stat(outputFile)
	require.NoError(t, err, "Runner.Run() should create output file")

	// Verify output content
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Failed to read output file")

	contentStr := string(content)
	assert.Contains(t, contentStr, "v2.6.1", "Runner.Run() output should contain version v2.6.1")
	assert.Contains(t, contentStr, "darwin", "Runner.Run() output should contain darwin platform")
}

func TestRunner_Run_SuccessfulWorkflowWithMultipleVersions(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	outputFile := filepath.Join(tempDir, "versions.bzl")

	config := Config{
		Count:         3,
		CacheDir:      cacheDir,
		OutputFile:    outputFile,
		WorkspaceRoot: tempDir,
	}

	// Setup mock client with 3 releases
	mock := NewMockGitHubClient()
	for _, tag := range []string{"v2.6.1", "v2.6.0", "v2.5.0"} {
		mock.AddRelease(tag)
		version := tag[1:] // Strip 'v'
		url := fmt.Sprintf("https://github.com/golangci/golangci-lint/releases/download/%s/golangci-lint-%s-checksums.txt", tag, version)
		mock.AddAsset(url, []byte(fmt.Sprintf(
			"aaa1111111111111111111111111111111111111111111111111111111111111  golangci-lint-%s-linux-amd64.tar.gz\nbbb2222222222222222222222222222222222222222222222222222222222222  golangci-lint-%s-darwin-arm64.tar.gz\n",
			version, version,
		)))
	}

	runner := NewRunner(config, mock)
	ctx := context.Background()

	err := runner.Run(ctx)
	require.NoError(t, err, "Runner.Run() should succeed")

	// Verify all cache files were created
	for _, tag := range []string{"v2.6.1", "v2.6.0", "v2.5.0"} {
		cacheFile := filepath.Join(cacheDir, tag+".txt")
		_, err := os.Stat(cacheFile)
		assert.NoError(t, err, "Runner.Run() should create cache file for %s", tag)
	}

	// Verify output contains all versions
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Failed to read output file")

	contentStr := string(content)
	for _, version := range []string{"v2.6.1", "v2.6.0", "v2.5.0"} {
		assert.Contains(t, contentStr, version, "Runner.Run() output should contain version %s", version)
	}
}

func TestRunner_Run_CacheHit(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	outputFile := filepath.Join(tempDir, "versions.bzl")

	// Pre-populate cache
	err := os.MkdirAll(cacheDir, 0755)
	require.NoError(t, err, "Failed to create cache dir")
	cacheContent := []byte("ccc3333333333333333333333333333333333333333333333333333333333333  golangci-lint-2.6.1-linux-amd64.tar.gz\n")
	err = os.WriteFile(filepath.Join(cacheDir, "v2.6.1.txt"), cacheContent, 0644)
	require.NoError(t, err, "Failed to write cache file")

	config := Config{
		Count:         1,
		CacheDir:      cacheDir,
		OutputFile:    outputFile,
		WorkspaceRoot: tempDir,
	}

	// Setup mock client - should not download
	mock := NewMockGitHubClient()
	mock.AddRelease("v2.6.1")
	// Don't add asset - if it tries to download, test will fail

	runner := NewRunner(config, mock)
	ctx := context.Background()

	err = runner.Run(ctx)
	require.NoError(t, err, "Runner.Run() should succeed")

	// Verify output was generated from cache
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Failed to read output file")

	contentStr := string(content)
	assert.Contains(t, contentStr, "ccc3333333333333333333333333333333333333333333333333333333333333", "Runner.Run() should use cached checksum")
}

func TestRunner_Run_HandlesGitHubAPIError(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{
		Count:         1,
		CacheDir:      filepath.Join(tempDir, "cache"),
		OutputFile:    filepath.Join(tempDir, "versions.bzl"),
		WorkspaceRoot: tempDir,
	}

	// Setup mock to return error
	mock := NewMockGitHubClient()
	mock.GetReleasesError = fmt.Errorf("API rate limit exceeded")

	runner := NewRunner(config, mock)
	ctx := context.Background()

	err := runner.Run(ctx)
	require.Error(t, err, "Runner.Run() should return error when GitHub API fails")
	assert.Contains(t, err.Error(), "failed to fetch releases", "Error should mention failed to fetch releases")
}

func TestRunner_Run_HandlesDownloadError(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{
		Count:         1,
		CacheDir:      filepath.Join(tempDir, "cache"),
		OutputFile:    filepath.Join(tempDir, "versions.bzl"),
		WorkspaceRoot: tempDir,
	}

	// Setup mock with download error
	mock := NewMockGitHubClient()
	mock.AddRelease("v2.6.1")
	mock.DownloadError = fmt.Errorf("network timeout")

	runner := NewRunner(config, mock)
	ctx := context.Background()

	err := runner.Run(ctx)
	require.Error(t, err, "Runner.Run() should return error when download fails")
	assert.Contains(t, err.Error(), "no versions", "Error should mention no versions")
}

func TestRunner_Run_HandlesEmptyReleasesList(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{
		Count:         10,
		CacheDir:      filepath.Join(tempDir, "cache"),
		OutputFile:    filepath.Join(tempDir, "versions.bzl"),
		WorkspaceRoot: tempDir,
	}

	// Setup mock with no releases
	mock := NewMockGitHubClient()
	// Don't add any releases

	runner := NewRunner(config, mock)
	ctx := context.Background()

	err := runner.Run(ctx)
	require.Error(t, err, "Runner.Run() should return error when no releases found")
}

func TestRunner_Run_SkipsInvalidChecksums(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	outputFile := filepath.Join(tempDir, "versions.bzl")

	config := Config{
		Count:         2,
		CacheDir:      cacheDir,
		OutputFile:    outputFile,
		WorkspaceRoot: tempDir,
	}

	// Setup mock: first release has invalid checksums, second is valid
	mock := NewMockGitHubClient()
	mock.AddRelease("v2.6.1")
	mock.AddRelease("v2.6.0")

	// Invalid checksum file for v2.6.1
	mock.AddAsset(
		"https://github.com/golangci/golangci-lint/releases/download/v2.6.1/golangci-lint-2.6.1-checksums.txt",
		[]byte("invalid checksum file\n"),
	)

	// Valid checksum file for v2.6.0
	mock.AddAsset(
		"https://github.com/golangci/golangci-lint/releases/download/v2.6.0/golangci-lint-2.6.0-checksums.txt",
		[]byte("ddd4444444444444444444444444444444444444444444444444444444444444  golangci-lint-2.6.0-linux-amd64.tar.gz\n"),
	)

	runner := NewRunner(config, mock)
	ctx := context.Background()

	err := runner.Run(ctx)
	require.NoError(t, err, "Runner.Run() should succeed with at least one valid release")

	// Verify output contains only the valid version
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Failed to read output file")

	contentStr := string(content)
	assert.Contains(t, contentStr, "v2.6.0", "Runner.Run() output should contain v2.6.0")
}

func TestRunner_ResolveAbsolutePaths(t *testing.T) {
	t.Run("converts relative paths", func(t *testing.T) {
		config := Config{
			CacheDir:      "cache",
			OutputFile:    "output.bzl",
			WorkspaceRoot: "/workspace",
		}

		runner := NewRunner(config, nil)
		absCacheDir, absOutputFile := runner.resolveAbsolutePaths()

		assert.Equal(t, "/workspace/cache", absCacheDir, "resolveAbsolutePaths() should convert relative cache path")
		assert.Equal(t, "/workspace/output.bzl", absOutputFile, "resolveAbsolutePaths() should convert relative output path")
	})

	t.Run("preserves absolute paths", func(t *testing.T) {
		config := Config{
			CacheDir:      "/absolute/cache",
			OutputFile:    "/absolute/output.bzl",
			WorkspaceRoot: "/workspace",
		}

		runner := NewRunner(config, nil)
		absCacheDir, absOutputFile := runner.resolveAbsolutePaths()

		assert.Equal(t, "/absolute/cache", absCacheDir, "resolveAbsolutePaths() should preserve absolute cache path")
		assert.Equal(t, "/absolute/output.bzl", absOutputFile, "resolveAbsolutePaths() should preserve absolute output path")
	})
}

func TestRunner_ProcessReleases(t *testing.T) {
	t.Run("processes valid releases", func(t *testing.T) {
		tempDir := t.TempDir()

		mock := NewMockGitHubClient()
		mock.AddAsset(
			"https://github.com/golangci/golangci-lint/releases/download/v2.6.1/golangci-lint-2.6.1-checksums.txt",
			[]byte("eee5555555555555555555555555555555555555555555555555555555555555  golangci-lint-2.6.1-windows-amd64.zip\n"),
		)

		config := Config{WorkspaceRoot: tempDir}
		runner := NewRunner(config, mock)

		releases := []Release{{TagName: "v2.6.1"}}
		ctx := context.Background()

		versions := runner.processReleases(ctx, releases, tempDir)

		require.Len(t, versions, 1, "processReleases() should return 1 version")
		assert.Equal(t, "v2.6.1", versions[0].Tag, "processReleases() should have correct tag")
	})

	t.Run("skips releases with empty tags", func(t *testing.T) {
		tempDir := t.TempDir()

		mock := NewMockGitHubClient()
		config := Config{WorkspaceRoot: tempDir}
		runner := NewRunner(config, mock)

		releases := []Release{{TagName: ""}}
		ctx := context.Background()

		versions := runner.processReleases(ctx, releases, tempDir)

		assert.Empty(t, versions, "processReleases() should skip releases with empty tags")
	})
}
