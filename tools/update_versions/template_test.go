package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareTemplateData(t *testing.T) {
	t.Run("empty versions list", func(t *testing.T) {
		data := PrepareTemplateData([]Version{})

		assert.Empty(t, data.DefaultVersion, "PrepareTemplateData() with empty list should have empty DefaultVersion")
		assert.Empty(t, data.Versions, "PrepareTemplateData() with empty list should have empty Versions")
		assert.NotEmpty(t, data.GeneratedAt, "PrepareTemplateData() should set GeneratedAt timestamp")
	})

	t.Run("single version", func(t *testing.T) {
		versions := []Version{
			{
				Tag: "v2.6.1",
				Checksums: map[Platform]string{
					{OS: "linux", Arch: "amd64"}: "abc123",
					{OS: "darwin", Arch: "arm64"}: "def456",
				},
			},
		}

		data := PrepareTemplateData(versions)

		assert.Equal(t, "v2.6.1", data.DefaultVersion, "PrepareTemplateData() should set DefaultVersion to first version")
		require.Len(t, data.Versions, 1, "PrepareTemplateData() should return 1 version")
		assert.Equal(t, "v2.6.1", data.Versions[0].Tag, "PrepareTemplateData() should preserve version tag")
	})

	t.Run("multiple versions - first is default", func(t *testing.T) {
		versions := []Version{
			{Tag: "v2.6.1", Checksums: map[Platform]string{{OS: "linux", Arch: "amd64"}: "abc"}},
			{Tag: "v2.6.0", Checksums: map[Platform]string{{OS: "linux", Arch: "amd64"}: "def"}},
			{Tag: "v2.5.0", Checksums: map[Platform]string{{OS: "linux", Arch: "amd64"}: "ghi"}},
		}

		data := PrepareTemplateData(versions)

		assert.Equal(t, "v2.6.1", data.DefaultVersion, "PrepareTemplateData() should set DefaultVersion to first version")
		assert.Len(t, data.Versions, 3, "PrepareTemplateData() should return all versions")
	})

	t.Run("checksums organized by OS", func(t *testing.T) {
		versions := []Version{
			{
				Tag: "v2.6.1",
				Checksums: map[Platform]string{
					{OS: "linux", Arch: "amd64"}:   "abc123",
					{OS: "linux", Arch: "arm64"}:   "def456",
					{OS: "darwin", Arch: "amd64"}:  "ghi789",
					{OS: "darwin", Arch: "arm64"}:  "jkl012",
					{OS: "windows", Arch: "amd64"}: "mno345",
				},
			},
		}

		data := PrepareTemplateData(versions)

		versionData := data.Versions[0]
		assert.Len(t, versionData.ChecksumsByOS, 3, "PrepareTemplateData() should organize checksums by OS")
		assert.Len(t, versionData.ChecksumsByOS["linux"], 2, "PrepareTemplateData() should preserve all architectures per OS")
		assert.Equal(t, "abc123", versionData.ChecksumsByOS["linux"]["amd64"], "PrepareTemplateData() should preserve checksum values")
	})

	t.Run("generated timestamp is recent", func(t *testing.T) {
		versions := []Version{{Tag: "v2.6.1", Checksums: map[Platform]string{}}}
		data := PrepareTemplateData(versions)

		// Parse the timestamp
		timestamp, err := time.Parse(time.RFC3339, data.GeneratedAt)
		require.NoError(t, err, "PrepareTemplateData() GeneratedAt timestamp should be valid RFC3339")

		// Check it's within the last minute
		assert.WithinDuration(t, time.Now(), timestamp, time.Minute, "PrepareTemplateData() GeneratedAt timestamp should be recent")
	})
}

func TestOrganizePlatformsByOS(t *testing.T) {
	t.Run("empty checksums", func(t *testing.T) {
		result := organizePlatformsByOS(map[Platform]string{})

		assert.Empty(t, result, "organizePlatformsByOS() with empty input should return empty map")
	})

	t.Run("single platform", func(t *testing.T) {
		checksums := map[Platform]string{
			{OS: "linux", Arch: "amd64"}: "abc123",
		}

		result := organizePlatformsByOS(checksums)

		require.Len(t, result, 1, "organizePlatformsByOS() should return 1 OS")
		assert.Len(t, result["linux"], 1, "organizePlatformsByOS() should preserve architectures")
		assert.Equal(t, "abc123", result["linux"]["amd64"], "organizePlatformsByOS() should preserve checksum values")
	})

	t.Run("multiple platforms same OS", func(t *testing.T) {
		checksums := map[Platform]string{
			{OS: "linux", Arch: "amd64"}: "abc123",
			{OS: "linux", Arch: "arm64"}: "def456",
			{OS: "linux", Arch: "386"}:   "ghi789",
		}

		result := organizePlatformsByOS(checksums)

		require.Len(t, result, 1, "organizePlatformsByOS() should return 1 OS")
		assert.Len(t, result["linux"], 3, "organizePlatformsByOS() should group all architectures for same OS")
	})

	t.Run("multiple platforms different OSes", func(t *testing.T) {
		checksums := map[Platform]string{
			{OS: "linux", Arch: "amd64"}:   "abc123",
			{OS: "linux", Arch: "arm64"}:   "def456",
			{OS: "darwin", Arch: "amd64"}:  "ghi789",
			{OS: "darwin", Arch: "arm64"}:  "jkl012",
			{OS: "windows", Arch: "amd64"}: "mno345",
		}

		result := organizePlatformsByOS(checksums)

		assert.Len(t, result, 3, "organizePlatformsByOS() should return all OSes")
		assert.Len(t, result["linux"], 2, "organizePlatformsByOS() should group linux architectures")
		assert.Len(t, result["darwin"], 2, "organizePlatformsByOS() should group darwin architectures")
		assert.Len(t, result["windows"], 1, "organizePlatformsByOS() should group windows architectures")
	})
}

func TestSortedOSKeys(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		keys := SortedOSKeys(map[string]map[string]string{})
		assert.Empty(t, keys, "SortedOSKeys() with empty map should return empty slice")
	})

	t.Run("single key", func(t *testing.T) {
		m := map[string]map[string]string{
			"linux": {},
		}
		keys := SortedOSKeys(m)
		assert.Equal(t, []string{"linux"}, keys, "SortedOSKeys() should return single key")
	})

	t.Run("multiple keys sorted alphabetically", func(t *testing.T) {
		m := map[string]map[string]string{
			"windows": {},
			"darwin":  {},
			"linux":   {},
			"freebsd": {},
		}
		keys := SortedOSKeys(m)

		expected := []string{"darwin", "freebsd", "linux", "windows"}
		assert.Equal(t, expected, keys, "SortedOSKeys() should return keys in alphabetical order")
	})

	t.Run("deterministic ordering", func(t *testing.T) {
		m := map[string]map[string]string{
			"z": {}, "a": {}, "m": {}, "b": {},
		}

		// Run multiple times to ensure consistency
		keys1 := SortedOSKeys(m)
		keys2 := SortedOSKeys(m)
		keys3 := SortedOSKeys(m)

		assert.Equal(t, keys1, keys2, "SortedOSKeys() should be deterministic")
		assert.Equal(t, keys2, keys3, "SortedOSKeys() should be deterministic")
	})
}

func TestSortedArchKeys(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		keys := SortedArchKeys(map[string]string{})
		assert.Empty(t, keys, "SortedArchKeys() with empty map should return empty slice")
	})

	t.Run("single key", func(t *testing.T) {
		m := map[string]string{"amd64": "hash"}
		keys := SortedArchKeys(m)
		assert.Equal(t, []string{"amd64"}, keys, "SortedArchKeys() should return single key")
	})

	t.Run("multiple keys sorted alphabetically", func(t *testing.T) {
		m := map[string]string{
			"armv7": "hash1",
			"386":   "hash2",
			"arm64": "hash3",
			"amd64": "hash4",
			"armv6": "hash5",
		}
		keys := SortedArchKeys(m)

		expected := []string{"386", "amd64", "arm64", "armv6", "armv7"}
		assert.Equal(t, expected, keys, "SortedArchKeys() should return keys in alphabetical order")
	})
}

func TestGenerateStarlarkFile(t *testing.T) {
	t.Run("generates valid file", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "test_output.bzl")

		data := &TemplateData{
			GeneratedAt:    "2025-11-11T00:00:00Z",
			DefaultVersion: "v2.6.1",
			Versions: []VersionData{
				{
					Tag: "v2.6.1",
					ChecksumsByOS: map[string]map[string]string{
						"linux": {
							"amd64": "abc123",
							"arm64": "def456",
						},
						"darwin": {
							"arm64": "ghi789",
						},
					},
				},
			},
		}

		err := GenerateStarlarkFile(data, outputFile)
		require.NoError(t, err, "GenerateStarlarkFile() should succeed")

		// Verify file was created
		_, err = os.Stat(outputFile)
		require.NoError(t, err, "GenerateStarlarkFile() should create output file")

		// Read and verify content
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err, "Failed to read generated file")

		contentStr := string(content)

		// Check for key elements
		checks := []string{
			"# Code generated by //tools/update_versions. DO NOT EDIT.",
			"DEFAULT_VERSION = \"v2.6.1\"",
			"GOLANGCI_VERSIONS = {",
			"\"v2.6.1\": {",
			"\"linux\": {",
			"\"amd64\": \"abc123\"",
			"\"darwin\": {",
			"\"arm64\": \"ghi789\"",
			"def get_golangci_version_info(version = None):",
		}

		for _, check := range checks {
			assert.Contains(t, contentStr, check, "GenerateStarlarkFile() output should contain %q", check)
		}
	})

	t.Run("atomic write - temp file is removed on success", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "test_output.bzl")

		data := &TemplateData{
			GeneratedAt:    "2025-11-11T00:00:00Z",
			DefaultVersion: "v2.6.1",
			Versions:       []VersionData{},
		}

		err := GenerateStarlarkFile(data, outputFile)
		require.NoError(t, err, "GenerateStarlarkFile() should succeed")

		// Check temp file was removed
		tempFile := outputFile + ".tmp"
		_, err = os.Stat(tempFile)
		assert.ErrorIs(t, err, os.ErrNotExist, "GenerateStarlarkFile() should remove temp file")
	})

	t.Run("handles invalid output path", func(t *testing.T) {
		data := &TemplateData{
			GeneratedAt:    "2025-11-11T00:00:00Z",
			DefaultVersion: "v2.6.1",
			Versions:       []VersionData{},
		}

		// Try to write to an invalid path
		err := GenerateStarlarkFile(data, "/nonexistent/directory/output.bzl")
		assert.Error(t, err, "GenerateStarlarkFile() should error with invalid output path")
	})
}

func TestGenerateStarlarkFile_MultipleVersions(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "test_output.bzl")

	data := &TemplateData{
		GeneratedAt:    "2025-11-11T00:00:00Z",
		DefaultVersion: "v2.6.1",
		Versions: []VersionData{
			{
				Tag: "v2.6.1",
				ChecksumsByOS: map[string]map[string]string{
					"linux": {"amd64": "abc123"},
				},
			},
			{
				Tag: "v2.6.0",
				ChecksumsByOS: map[string]map[string]string{
					"linux": {"amd64": "def456"},
				},
			},
			{
				Tag: "v2.5.0",
				ChecksumsByOS: map[string]map[string]string{
					"linux": {"amd64": "ghi789"},
				},
			},
		},
	}

	err := GenerateStarlarkFile(data, outputFile)
	require.NoError(t, err, "GenerateStarlarkFile() should succeed")

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err, "Failed to read generated file")

	contentStr := string(content)

	// Verify all versions are present
	for _, version := range []string{"v2.6.1", "v2.6.0", "v2.5.0"} {
		assert.Contains(t, contentStr, "\""+version+"\": {", "GenerateStarlarkFile() should contain version %q", version)
	}
}

func TestEnsureOutputDirectory(t *testing.T) {
	t.Run("creates directory if it doesn't exist", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "subdir", "output.bzl")

		err := EnsureOutputDirectory(outputFile)
		require.NoError(t, err, "EnsureOutputDirectory() should succeed")

		// Verify directory was created
		dirPath := filepath.Dir(outputFile)
		_, err = os.Stat(dirPath)
		assert.NoError(t, err, "EnsureOutputDirectory() should create directory")
	})

	t.Run("handles existing directory", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "output.bzl")

		err := EnsureOutputDirectory(outputFile)
		require.NoError(t, err, "EnsureOutputDirectory() should succeed with existing directory")
	})

	t.Run("handles nested directories", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "a", "b", "c", "output.bzl")

		err := EnsureOutputDirectory(outputFile)
		require.NoError(t, err, "EnsureOutputDirectory() should succeed")

		// Verify nested directories were created
		dirPath := filepath.Dir(outputFile)
		_, err = os.Stat(dirPath)
		assert.NoError(t, err, "EnsureOutputDirectory() should create nested directories")
	})
}
