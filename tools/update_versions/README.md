# golangci-lint Version Manager

Automated utility to fetch golangci-lint releases from GitHub, cache checksums, and generate version data for hermetic Bazel builds.

## Quick Start

```bash
# Prerequisites: Bazel 8.4.2+, rules_go workspace
# From workspace root, generate versions.bzl with the latest 10 releases
bazel run //tools/update_versions -- --count=10

# Verify the extension works
bazel build @golangci_lint_binary//...
```

The utility downloads checksum files from GitHub, caches them in `cache/checksums/`, and generates `golangci_lint/private/versions.bzl` with SHA-256 mappings for all platforms.

## Usage

Common tasks:

```bash
# 1) Update to latest 10 versions (monthly maintenance)
bazel run //tools/update_versions -- --count=10

# 2) Fetch more versions for historical compatibility
bazel run //tools/update_versions -- --count=20

# 3) Run with custom paths (useful for testing)
bazel run //tools/update_versions -- \
  --count=10 \
  --cache-dir=tools/update_versions/cache/checksums \
  --output=golangci_lint/private/versions.bzl
```

After running, review and commit:
```bash
git diff golangci_lint/private/versions.bzl
git add golangci_lint/private/versions.bzl tools/update_versions/cache/
git commit -m "Update golangci-lint versions to v2.X.Y"
```

More workflows → **DESIGN.md**.

## Configuration

| Option        | Default                                    | Description                          |
| ------------- | ------------------------------------------ | ------------------------------------ |
| `--count`     | 10                                         | Number of recent versions to process |
| `--cache-dir` | `tools/update_versions/cache/checksums`    | Checksum cache directory             |
| `--output`    | `golangci_lint/private/versions.bzl`       | Generated Starlark file path         |

All paths are relative to workspace root.

See implementation details → **DESIGN.md**, **TASKS.md**.

## Troubleshooting

* **"Failed to fetch releases"**: Network issue or GitHub rate limit. Check connectivity; wait if rate limited; use GitHub token for higher limits.
* **"Failed to download checksum file"**: Release missing checksum or network issue. Utility skips problematic releases automatically.
* **Generated file in wrong location**: Use `bazel run` instead of `go run .` to ensure correct working directory.
* **Extension fails after update**: Run `bazel clean --expunge` and rebuild. Verify generated `versions.bzl` syntax is valid Starlark.

## License

Part of `rules_go`. See **../../LICENSE** for details.
