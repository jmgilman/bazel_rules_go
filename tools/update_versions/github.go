package main

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-github/v62/github"
)

// Release represents a GitHub release with basic information.
type Release struct {
	TagName string
}

// GitHubAPI defines the interface for interacting with GitHub.
type GitHubAPI interface {
	GetLatestReleases(ctx context.Context, count int) ([]Release, error)
	DownloadAsset(ctx context.Context, url string) ([]byte, error)
}

// GitHubClient wraps the GitHub API client for fetching golangci-lint releases.
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new GitHub API client.
func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		client: github.NewClient(nil),
	}
}

// GetLatestReleases fetches the last N releases from the golangci-lint repository.
func (c *GitHubClient) GetLatestReleases(ctx context.Context, count int) ([]Release, error) {
	opts := &github.ListOptions{
		PerPage: count,
	}

	ghReleases, _, err := c.client.Repositories.ListReleases(ctx, "golangci", "golangci-lint", opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	// Convert to our Release type
	releases := make([]Release, 0, len(ghReleases))
	for _, r := range ghReleases {
		releases = append(releases, Release{
			TagName: r.GetTagName(),
		})
	}

	return releases, nil
}

// DownloadAsset downloads an asset from a URL and returns the contents.
func (c *GitHubClient) DownloadAsset(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download asset: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}
