# nogo: Compile-Time Go Static Analysis

This directory provides a standard nogo configuration for Go static analysis that runs **during compilation**.

## What is nogo?

nogo is Bazel's built-in integration with Go's analysis framework (`golang.org/x/tools/go/analysis`). It runs analyzers as part of the compilation step, which means:

- ✅ **Fails the build** if issues are found (catches problems early)
- ✅ **Per-package caching** (only analyzes changed code)
- ✅ **Incremental** (faster than whole-codebase analysis)
- ✅ **Type-safe** (has access to full type information)

## Two-Layer Linting Strategy

This module provides a two-layer approach to Go static analysis:

### Layer 1: nogo (Compile-Time) - This Directory

**What it checks:**
- Critical correctness issues
- Memory safety problems
- Concurrency bugs
- Type-related errors

**When it runs:** During `bazel build` and `bazel test`

**Impact:** **Blocks compilation** if issues are found

### Layer 2: golangci-lint (Test-Time) - See `//golangci_lint`

**What it checks:**
- Code style and formatting
- Best practices
- Error handling patterns
- Additional static analysis

**When it runs:** During `bazel test` (as a separate test target)

**Impact:** Fails tests but doesn't block compilation

**Why both?** They complement each other without duplication:
- nogo handles critical compile-time checks
- golangci-lint handles style and non-blocking checks

## Usage

### In Your MODULE.bazel

```starlark
bazel_dep(name = "rules_tooling", version = "1.0.0")

# Configure Go SDK to use standard nogo
go_sdk = use_extension("@rules_go//go:extensions.bzl", "go_sdk")
go_sdk.nogo(
    nogo = "@rules_tooling//nogo:standard_nogo",
)
```

That's it! Now every Go package in your repository will be analyzed during compilation.

### Testing It

```bash
# This will run nogo analyzers during compilation
bazel build //your/package:target

# If nogo finds issues, the build will fail with analyzer output
```

## What's Included

The `standard_nogo` target includes these analyzers:

| Analyzer | What It Checks |
|----------|----------------|
| `asmdecl` | Assembly function declarations match Go signatures |
| `assign` | Useless assignments |
| `atomic` | Incorrect usage of `sync/atomic` |
| `bools` | Mistakes in boolean expressions |
| `buildtag` | Build tag syntax |
| `cgocall` | CGO pointer passing rules |
| `composite` | Unkeyed composite literals |
| `copylock` | Locks passed by value |
| `errorsas` | Correct `errors.As` usage |
| `httpresponse` | Unclosed HTTP response bodies |
| `loopclosure` | Loop variables captured by closures |
| `lostcancel` | Context cancellation |
| `nilfunc` | Useless nil function comparisons |
| `printf` | Printf-style format strings |
| `shift` | Suspicious shift operations |
| `stdmethods` | Standard method signatures |
| `structtag` | Struct field tags |
| `tests` | Test function signatures |
| `unmarshal` | Non-pointer to unmarshal |
| `unreachable` | Unreachable code |
| `unsafeptr` | Invalid `unsafe.Pointer` conversions |
| `unusedresult` | Unused results of function calls |

## Configuration

The `nogo_config.json` file allows excluding certain checks from specific files:

```json
{
  "composites": {
    "exclude_files": {
      ".*_test\\.go$": "Allow unkeyed fields in test files"
    }
  }
}
```

You can override this by creating your own nogo target:

```starlark
# your/BUILD.bazel
load("@rules_go//go:def.bzl", "nogo")

nogo(
    name = "my_nogo",
    deps = [
        "@rules_tooling//nogo:standard_nogo",  # Inherit from standard
        "//your/custom:analyzer",  # Add your own
    ],
    config = "my_nogo_config.json",
    visibility = ["//visibility:public"],
)
```

Then reference it in MODULE.bazel:
```starlark
go_sdk.nogo(nogo = "//your:my_nogo")
```

## Common Issues

### "nogo analyzer found issues" during build

This is expected! nogo is designed to fail the build when it finds problems.

**Fix the issues** or **adjust the config** to exclude specific patterns if they're false positives.

### Slow builds after enabling nogo

Nogo runs during compilation and is cached per-package. Initial builds may be slower, but incremental builds should be fast because Bazel only re-analyzes changed packages.

### Generated code failing nogo checks

Add exclusions to `nogo_config.json`:

```json
{
  "tests": {
    "exclude_files": {
      "\\.pb\\.go$": "Skip protobuf generated code",
      ".*generated.*\\.go$": "Skip other generated code"
    }
  }
}
```

## Deduplication with golangci-lint

The standard golangci-lint config (`//golangci_lint/configs:standard.yml`) has `govet` **disabled** because nogo covers all those checks. This avoids duplication and speeds up linting.

See `//golangci_lint/configs:standard.yml` for details.

## References

- [Bazel rules_go nogo documentation](https://github.com/bazelbuild/rules_go/blob/master/go/nogo.rst)
- [Go analysis framework](https://pkg.go.dev/golang.org/x/tools/go/analysis)
- [Standard analyzers](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes)
