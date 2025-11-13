package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseChecksumFile(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		wantPlatforms int
		wantError     bool
	}{
		{
			name:          "valid checksum file",
			filename:      "testdata/checksums/valid.txt",
			wantPlatforms: 7, // 3 darwin, 2 linux, 2 windows
			wantError:     false,
		},
		{
			name:          "invalid hashes are skipped with warning",
			filename:      "testdata/checksums/invalid_hash.txt",
			wantPlatforms: 2, // Only the 2 valid ones
			wantError:     false,
		},
		{
			name:          "malformed lines are skipped with warning",
			filename:      "testdata/checksums/malformed.txt",
			wantPlatforms: 2, // Only the 2 valid ones
			wantError:     false,
		},
		{
			name:          "only packages (deb/rpm/source) returns empty",
			filename:      "testdata/checksums/only_packages.txt",
			wantPlatforms: 0, // All filtered out
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(tt.filename)
			require.NoError(t, err, "Failed to read test file")

			checksums, err := ParseChecksumFile(content)
			if tt.wantError {
				assert.Error(t, err, "ParseChecksumFile() should return error")
			} else {
				assert.NoError(t, err, "ParseChecksumFile() should not return error")
				assert.Len(t, checksums, tt.wantPlatforms, "ParseChecksumFile() should return correct number of platforms")
			}
		})
	}
}

func TestParseChecksumFile_EmptyFile(t *testing.T) {
	checksums, err := ParseChecksumFile([]byte{})
	assert.NoError(t, err, "ParseChecksumFile() with empty content should not error")
	assert.Empty(t, checksums, "ParseChecksumFile() with empty content should return empty map")
}

func TestParseChecksumFile_ValidEntry(t *testing.T) {
	content := []byte("aee6e16af4dfa60dd3c4e39536edc905f28369fda3c138090db00c8233cfe450  golangci-lint-2.6.1-darwin-amd64.tar.gz\n")

	checksums, err := ParseChecksumFile(content)
	require.NoError(t, err, "ParseChecksumFile() should not error")
	require.Len(t, checksums, 1, "ParseChecksumFile() should return 1 entry")

	platform := Platform{OS: "darwin", Arch: "amd64"}
	hash, ok := checksums[platform]
	require.True(t, ok, "ParseChecksumFile() should contain darwin-amd64 platform")
	assert.Equal(t, "aee6e16af4dfa60dd3c4e39536edc905f28369fda3c138090db00c8233cfe450", hash, "ParseChecksumFile() should return correct hash")
}

func TestExtractPlatformFromFilename(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantOS      string
		wantArch    string
		wantError   bool
	}{
		{
			name:      "valid tar.gz linux-amd64",
			filename:  "golangci-lint-2.6.1-linux-amd64.tar.gz",
			wantOS:    "linux",
			wantArch:  "amd64",
			wantError: false,
		},
		{
			name:      "valid tar.gz darwin-arm64",
			filename:  "golangci-lint-2.6.1-darwin-arm64.tar.gz",
			wantOS:    "darwin",
			wantArch:  "arm64",
			wantError: false,
		},
		{
			name:      "valid zip windows-amd64",
			filename:  "golangci-lint-2.6.1-windows-amd64.zip",
			wantOS:    "windows",
			wantArch:  "amd64",
			wantError: false,
		},
		{
			name:      "valid with multi-part version",
			filename:  "golangci-lint-1.64.8-linux-amd64.tar.gz",
			wantOS:    "linux",
			wantArch:  "amd64",
			wantError: false,
		},
		{
			name:      "valid armv6",
			filename:  "golangci-lint-2.6.1-linux-armv6.tar.gz",
			wantOS:    "linux",
			wantArch:  "armv6",
			wantError: false,
		},
		{
			name:      "valid armv7",
			filename:  "golangci-lint-2.6.1-freebsd-armv7.tar.gz",
			wantOS:    "freebsd",
			wantArch:  "armv7",
			wantError: false,
		},
		{
			name:      "valid 386 architecture",
			filename:  "golangci-lint-2.6.1-windows-386.zip",
			wantOS:    "windows",
			wantArch:  "386",
			wantError: false,
		},
		{
			name:      "invalid - source tarball",
			filename:  "golangci-lint-2.6.1-source.tar.gz",
			wantError: true,
		},
		{
			name:      "invalid - deb package",
			filename:  "golangci-lint-2.6.1-linux-amd64.deb",
			wantError: true,
		},
		{
			name:      "invalid - rpm package",
			filename:  "golangci-lint-2.6.1-linux-amd64.rpm",
			wantError: true,
		},
		{
			name:      "invalid - wrong format",
			filename:  "some-other-file.tar.gz",
			wantError: true,
		},
		{
			name:      "invalid - missing version",
			filename:  "golangci-lint-linux-amd64.tar.gz",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform, err := ExtractPlatformFromFilename(tt.filename)

			if tt.wantError {
				assert.Error(t, err, "ExtractPlatformFromFilename() should return error")
				return
			}

			require.NoError(t, err, "ExtractPlatformFromFilename() should not return error")
			assert.Equal(t, tt.wantOS, platform.OS, "ExtractPlatformFromFilename() should return correct OS")
			assert.Equal(t, tt.wantArch, platform.Arch, "ExtractPlatformFromFilename() should return correct Arch")
		})
	}
}

func TestIsValidSHA256(t *testing.T) {
	tests := []struct {
		name  string
		hash  string
		valid bool
	}{
		{
			name:  "valid lowercase hash",
			hash:  "aee6e16af4dfa60dd3c4e39536edc905f28369fda3c138090db00c8233cfe450",
			valid: true,
		},
		{
			name:  "valid uppercase hash",
			hash:  "AEE6E16AF4DFA60DD3C4E39536EDC905F28369FDA3C138090DB00C8233CFE450",
			valid: true,
		},
		{
			name:  "valid mixed case hash",
			hash:  "Aee6e16Af4dfa60dd3c4e39536edc905f28369fda3c138090db00c8233cfe450",
			valid: true,
		},
		{
			name:  "invalid - too short",
			hash:  "aee6e16af4dfa60dd3c4e39536edc905",
			valid: false,
		},
		{
			name:  "invalid - too long",
			hash:  "aee6e16af4dfa60dd3c4e39536edc905f28369fda3c138090db00c8233cfe450extra",
			valid: false,
		},
		{
			name:  "invalid - contains non-hex characters",
			hash:  "aee6e16af4dfa60dd3c4e39536edc905f28369fda3c138090db00c8233cfe45z",
			valid: false,
		},
		{
			name:  "invalid - contains spaces",
			hash:  "aee6e16af4dfa60dd3c4e39536edc905 f28369fda3c138090db00c8233cfe450",
			valid: false,
		},
		{
			name:  "invalid - empty string",
			hash:  "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidSHA256(tt.hash)
			assert.Equal(t, tt.valid, got, "isValidSHA256() should return correct validation result")
		})
	}
}

func TestParseChecksumFile_WindowsZipFiles(t *testing.T) {
	content := []byte(`
d47312b0bd87fa4d0b161001bcebaaaf59203d13444e624b00d2dd240b168dc8  golangci-lint-2.6.1-windows-386.zip
b6edeea3d1d52331e98dc6378f710cfe2d752ca1ba09032fe60e62a87a27a25f  golangci-lint-2.6.1-windows-amd64.zip
eff5849a62c2b0076ab55a4b40379c8636028bccfdb8af3cc54af155e18f25dd  golangci-lint-2.6.1-windows-arm64.zip
`)

	checksums, err := ParseChecksumFile(content)
	require.NoError(t, err, "ParseChecksumFile() should not error")
	assert.Len(t, checksums, 3, "ParseChecksumFile() should return 3 Windows platforms")

	// Verify all Windows platforms are present
	expectedPlatforms := []Platform{
		{OS: "windows", Arch: "386"},
		{OS: "windows", Arch: "amd64"},
		{OS: "windows", Arch: "arm64"},
	}

	for _, platform := range expectedPlatforms {
		_, ok := checksums[platform]
		assert.True(t, ok, "ParseChecksumFile() should contain platform %s-%s", platform.OS, platform.Arch)
	}
}

func TestParseChecksumFile_MultiplePlatformsPerOS(t *testing.T) {
	content := []byte(`
c22e188e46aff9b140588abe6828ba271b600ae82b2d6a4f452196a639c17ec0  golangci-lint-2.6.1-linux-amd64.tar.gz
1c22b899f2dd84f9638e0e0352a319a2867b0bb082c5323ad50d8713b65bb793  golangci-lint-2.6.1-linux-arm64.tar.gz
b52331fb224cdc987f8f703120d546a98114c400a453c61a2b51a86d0d669dbe  golangci-lint-2.6.1-linux-armv6.tar.gz
e4b2151c569eb481cd9482f6b1bbf70cf129959e75b918aa5f3cb6acb0745ede  golangci-lint-2.6.1-linux-armv7.tar.gz
79bb6342726ccea96abb99a77bece01961f4bece7e44601855f30e01d3efba27  golangci-lint-2.6.1-linux-386.tar.gz
`)

	checksums, err := ParseChecksumFile(content)
	require.NoError(t, err, "ParseChecksumFile() should not error")
	assert.Len(t, checksums, 5, "ParseChecksumFile() should return 5 Linux platforms")

	// Verify all platforms have different hashes
	seenHashes := make(map[string]bool)
	for _, hash := range checksums {
		assert.False(t, seenHashes[hash], "ParseChecksumFile() should not have duplicate hashes")
		seenHashes[hash] = true
	}
}
