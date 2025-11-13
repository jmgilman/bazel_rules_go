package main

import (
	"context"
	"fmt"
)

// MockGitHubClient is a mock implementation of GitHubAPI for testing.
type MockGitHubClient struct {
	Releases      []Release
	AssetContents map[string][]byte
	GetReleasesError error
	DownloadError error
}

// NewMockGitHubClient creates a new mock GitHub client.
func NewMockGitHubClient() *MockGitHubClient {
	return &MockGitHubClient{
		Releases:      []Release{},
		AssetContents: make(map[string][]byte),
	}
}

// GetLatestReleases returns the pre-configured releases or an error.
func (m *MockGitHubClient) GetLatestReleases(_ context.Context, count int) ([]Release, error) {
	if m.GetReleasesError != nil {
		return nil, m.GetReleasesError
	}

	// Return up to 'count' releases
	if count > len(m.Releases) {
		count = len(m.Releases)
	}

	return m.Releases[:count], nil
}

// DownloadAsset returns the pre-configured asset content for the given URL or an error.
func (m *MockGitHubClient) DownloadAsset(_ context.Context, url string) ([]byte, error) {
	if m.DownloadError != nil {
		return nil, m.DownloadError
	}

	content, ok := m.AssetContents[url]
	if !ok {
		return nil, fmt.Errorf("asset not found: %s", url)
	}

	return content, nil
}

// AddRelease adds a release to the mock client.
func (m *MockGitHubClient) AddRelease(tag string) {
	m.Releases = append(m.Releases, Release{TagName: tag})
}

// AddAsset adds asset content for a specific URL.
func (m *MockGitHubClient) AddAsset(url string, content []byte) {
	m.AssetContents[url] = content
}
