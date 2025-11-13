"""
Runs golangci-lint on the given target.
"""

load("@rules_go//go:def.bzl", "go_context")

def _to_rlocation_path(ctx, file):
    """Convert a File object to an rlocation path for use with runfiles.bash.

    This generates paths that work correctly for both internal and external
    repository usage by using the workspace name resolved at build time.

    Args:
        ctx: The rule context
        file: A File object to convert

    Returns:
        A string path suitable for passing to the rlocation() bash function
    """
    if file.short_path.startswith("../"):
        # External repository: strip "../" prefix
        return file.short_path[3:]
    else:
        # Main repository: prepend workspace name
        return ctx.workspace_name + "/" + file.short_path

def _find_go_mod(ctx):
    """Find the go.mod file in the data attribute.

    Args:
        ctx: The rule context

    Returns:
        The go.mod File object

    Fails:
        If no go.mod is found in data
    """
    for f in ctx.files.data:
        if f.basename == "go.mod":
            return f
    fail("golangci_lint_test: no go.mod found in data=; add it to data= for the module you want to lint")

def _generate_test_script_content(gcl_path, go_tool_path, cfg_path, mod_path):
    """Generate the bash test script content.

    Args:
        gcl_path: rlocation path to golangci-lint binary
        go_tool_path: rlocation path to Go binary
        cfg_path: rlocation path to config file (empty string if none)
        mod_path: rlocation path to go.mod file

    Returns:
        The bash script content as a string
    """
    return """\
#!/bin/bash
set -euo pipefail

# --- begin runfiles.bash initialization v3 ---
set -uo pipefail; set +e; f=bazel_tools/tools/bash/runfiles/runfiles.bash
source "${{RUNFILES_DIR:-/dev/null}}/$f" 2>/dev/null || \
  source "$(grep -sm1 "^$f " "${{RUNFILES_MANIFEST_FILE:-/dev/null}}" | cut -f2- -d' ')" 2>/dev/null || \
  source "${{0}}.runfiles/$f" 2>/dev/null || \
  source "$(grep -sm1 "^$f " "${{0}}.runfiles_manifest" | cut -f2- -d' ')" 2>/dev/null || \
  source "$(grep -sm1 "^$f " "${{0}}.exe.runfiles_manifest" | cut -f2- -d' ')" 2>/dev/null || \
  {{ echo>&2 "ERROR: cannot find $f"; exit 1; }}; f=; set -e
# --- end runfiles.bash initialization v3 ---

# Resolve the hermetic Go binary from runfiles
GO_BIN="$(rlocation "{go_tool_path}")"
if [[ -z "${{GO_BIN}}" || ! -x "${{GO_BIN}}" ]]; then
  echo "Go binary not found in runfiles: {go_tool_path}" >&2
  exit 127
fi

# Prepend Go binary directory to PATH for hermetic execution
export PATH="$(dirname "${{GO_BIN}}"):$PATH"

# Resolve the golangci-lint binary from runfiles
GCL_BIN="$(rlocation "{gcl_path}")"
if [[ -z "${{GCL_BIN}}" || ! -x "${{GCL_BIN}}" ]]; then
  echo "golangci-lint not found in runfiles: {gcl_path}" >&2
  exit 127
fi

# Resolve config (if provided) from runfiles
CFG_FLAG=""
if [[ -n "{cfg_path}" ]]; then
  CFG_PATH="$(rlocation "{cfg_path}")"
  if [[ -z "${{CFG_PATH}}" || ! -f "${{CFG_PATH}}" ]]; then
    echo "golangci-lint config not found in runfiles: {cfg_path}" >&2
    exit 127
  fi
  CFG_FLAG="--config ${{CFG_PATH}}"
fi

# Change to the module directory (parent of go.mod)
MOD_DIR="$(dirname "$(rlocation "{mod_path}")")"
cd "$MOD_DIR"

# Configure hermetic caches
export HOME="${{TEST_TMPDIR}}"
export GOLANGCI_LINT_CACHE="${{TEST_TMPDIR}}/golangci-lint-cache"
export GOCACHE="${{TEST_TMPDIR}}/go-cache"
export GOMODCACHE="${{TEST_TMPDIR}}/gomodcache"
mkdir -p "${{GOLANGCI_LINT_CACHE}}" "${{GOMODCACHE}}"

# Follow symlinks for //go:embed (Go 1.25+)
export GODEBUG=embedfollowsymlinks=1

# Run golangci-lint
"$GCL_BIN" version
"$GCL_BIN" run $CFG_FLAG ./...
""".format(
        gcl_path = gcl_path,
        go_tool_path = go_tool_path,
        cfg_path = cfg_path,
        mod_path = mod_path,
    )

def _collect_runfiles(ctx, go):
    """Collect all runfiles needed for the test.

    Args:
        ctx: The rule context
        go: The go_context object

    Returns:
        A tuple of (files_list, transitive_depset)
    """
    # Direct runfiles
    files = [ctx.executable._golangci_lint, go.go] + ctx.files.data
    if ctx.file.config:
        files.append(ctx.file.config)

    # Transitive runfiles (Go SDK)
    go_sdk_files = depset(transitive = [
        go.sdk.tools,
        go.sdk.srcs,
        go.sdk.headers,
        go.stdlib.libs,
    ])

    transitive = depset(transitive = [
        ctx.attr._bash_runfiles.files,
        go_sdk_files,
    ])

    return files, transitive

def _golangci_lint_test_impl(ctx):
    """Implementation of the golangci_lint_test rule."""
    # Get Go toolchain context
    go = go_context(ctx)

    # Find required go.mod file
    mod = _find_go_mod(ctx)

    # Generate rlocation paths for all dependencies
    gcl_path = _to_rlocation_path(ctx, ctx.executable._golangci_lint)
    go_tool_path = _to_rlocation_path(ctx, go.go)
    mod_path = _to_rlocation_path(ctx, mod)
    cfg_path = _to_rlocation_path(ctx, ctx.file.config) if ctx.file.config else ""

    # Generate the test script
    script_content = _generate_test_script_content(
        gcl_path = gcl_path,
        go_tool_path = go_tool_path,
        cfg_path = cfg_path,
        mod_path = mod_path,
    )

    # Write the test script
    test_script = ctx.actions.declare_file(ctx.label.name + ".sh")
    ctx.actions.write(
        output = test_script,
        content = script_content,
        is_executable = True,
    )

    # Collect runfiles
    runfiles_files, transitive_files = _collect_runfiles(ctx, go)

    return [DefaultInfo(
        executable = test_script,
        runfiles = ctx.runfiles(
            files = runfiles_files,
            transitive_files = transitive_files,
        ),
    )]

golangci_lint_test = rule(
    implementation = _golangci_lint_test_impl,
    attrs = {
        "data": attr.label_list(
            allow_files = True,
            default = [],
            doc = "Go sources + any embedded resources. e.g. glob(['*.go','*.tmpl']).",
        ),
        "config": attr.label(
            allow_single_file = [".yml", ".yaml", ".toml", ".json"],
            doc = "Config file for golangci-lint (v2 schema).",
        ),
        "_golangci_lint": attr.label(
            default = "@golangci_lint_binary//:golangci_lint",
            executable = True,
            cfg = "exec",
            doc = "The golangci-lint binary to use.",
        ),
        "_bash_runfiles": attr.label(
            default = "@bazel_tools//tools/bash/runfiles",
        ),
        "_go_context_data": attr.label(
            default = "@rules_go//:go_context_data",
        ),
    },
    test = True,
    toolchains = ["@rules_go//go:toolchain"],
)
