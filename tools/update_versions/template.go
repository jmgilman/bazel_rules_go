package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"
)

//go:embed template.bzl.tmpl
var templateFS embed.FS

// TemplateData holds the data for generating the Starlark file.
type TemplateData struct {
	GeneratedAt    string
	DefaultVersion string
	Versions       []VersionData
}

// VersionData represents version data organized for template rendering.
type VersionData struct {
	Tag           string
	ChecksumsByOS map[string]map[string]string // os -> arch -> sha256
}

// EnsureOutputDirectory ensures the output directory exists.
func EnsureOutputDirectory(outputPath string) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}

// GenerateStarlarkFile generates the versions.bzl file from template.
func GenerateStarlarkFile(data *TemplateData, outputPath string) error {
	// Create template with custom functions
	funcMap := template.FuncMap{
		"SortedOSKeys":   SortedOSKeys,
		"SortedArchKeys": SortedArchKeys,
	}

	// Parse template
	tmpl, err := template.New("template.bzl.tmpl").Funcs(funcMap).ParseFS(templateFS, "template.bzl.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create temporary file for atomic write
	tempFile := outputPath + ".tmp"
	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Execute template
	if err := tmpl.Execute(f, data); err != nil {
		_ = os.Remove(tempFile) // Best-effort cleanup
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tempFile) // Best-effort cleanup
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, outputPath); err != nil {
		_ = os.Remove(tempFile) // Best-effort cleanup
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// PrepareTemplateData converts Version structs to TemplateData.
func PrepareTemplateData(versions []Version) *TemplateData {
	if len(versions) == 0 {
		return &TemplateData{
			GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
			DefaultVersion: "",
			Versions:       []VersionData{},
		}
	}

	versionData := make([]VersionData, 0, len(versions))
	for _, v := range versions {
		vd := VersionData{
			Tag:           v.Tag,
			ChecksumsByOS: organizePlatformsByOS(v.Checksums),
		}
		versionData = append(versionData, vd)
	}

	return &TemplateData{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		DefaultVersion: versions[0].Tag, // First version is latest
		Versions:       versionData,
	}
}

// SortedArchKeys returns sorted architecture keys for deterministic output.
func SortedArchKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SortedOSKeys returns sorted OS keys for deterministic output.
func SortedOSKeys(m map[string]map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// organizePlatformsByOS converts flat Platform map to nested OS -> Arch -> SHA256 map.
func organizePlatformsByOS(checksums map[Platform]string) map[string]map[string]string {
	result := make(map[string]map[string]string)

	for platform, hash := range checksums {
		if result[platform.OS] == nil {
			result[platform.OS] = make(map[string]string)
		}
		result[platform.OS][platform.Arch] = hash
	}

	return result
}
