package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
)

// Platform represents an OS and architecture combination.
type Platform struct {
	OS   string
	Arch string
}

// Version represents a golangci-lint version with checksums for all platforms.
type Version struct {
	Tag       string
	Checksums map[Platform]string
}

// ParseChecksumFile parses a SHA-256 checksum file and returns a map of platforms to checksums.
func ParseChecksumFile(content []byte) (map[Platform]string, error) {
	checksums := make(map[Platform]string)
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Parse line: "<hash>  <filename>"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			log.Printf("Warning: skipping malformed line: %s", line)
			continue
		}

		hash := parts[0]
		filename := parts[len(parts)-1]

		// Validate hash is 64 hex characters
		if !isValidSHA256(hash) {
			log.Printf("Warning: skipping line with invalid SHA256: %s", line)
			continue
		}

		// Only process .tar.gz and .zip files
		if !strings.HasSuffix(filename, ".tar.gz") && !strings.HasSuffix(filename, ".zip") {
			continue
		}

		// Extract platform from filename
		platform, err := ExtractPlatformFromFilename(filename)
		if err != nil {
			log.Printf("Warning: skipping file %s: %v", filename, err)
			continue
		}

		checksums[*platform] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading checksum file: %w", err)
	}

	return checksums, nil
}

// ExtractPlatformFromFilename extracts OS and architecture from a filename.
// Expected format: golangci-lint-{version}-{os}-{arch}.{tar.gz|zip}.
func ExtractPlatformFromFilename(filename string) (*Platform, error) {
	// Pattern: golangci-lint-{version}-{os}-{arch}.tar.gz or .zip
	// Example: golangci-lint-2.6.1-linux-amd64.tar.gz
	pattern := regexp.MustCompile(`golangci-lint-[\d.]+-([\w]+)-([\w]+)\.(tar\.gz|zip)`)
	matches := pattern.FindStringSubmatch(filename)

	if len(matches) != 4 {
		return nil, fmt.Errorf("filename does not match expected pattern: %s", filename)
	}

	os := matches[1]
	arch := matches[2]

	return &Platform{
		OS:   os,
		Arch: arch,
	}, nil
}

// isValidSHA256 checks if a string is a valid SHA-256 hash (64 hex characters).
func isValidSHA256(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, c := range hash {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
