load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("//golangci_lint/private:versions.bzl", "get_golangci_version_info", "DEFAULT_VERSION")

# Tag class for configuring golangci-lint version
_config_tag = tag_class(
    attrs = {
        "version": attr.string(
            doc = "Version of golangci-lint to use (e.g., 'v2.6.1'). If not specified, uses the default version.",
        ),
    },
)

def _golangci_lint_extension_impl(ctx):
    """Downloads golangci-lint binary for the current platform."""

    # Collect version from tags across all modules
    requested_version = None
    for mod in ctx.modules:
        for config_tag in mod.tags.config:
            if config_tag.version:
                if requested_version and requested_version != config_tag.version:
                    fail("Multiple modules requested different golangci-lint versions: {} and {}".format(
                        requested_version,
                        config_tag.version,
                    ))
                requested_version = config_tag.version

    # Use requested version or default
    version_to_use = requested_version if requested_version else None

    # Get version and checksums from generated versions file
    version, checksums = get_golangci_version_info(version_to_use)

    # Detect current platform
    os, arch = _detect_platforms(ctx)

    # Check if platform is supported
    if os not in checksums or arch not in checksums[os]:
        fail("Unsupported platform: {}-{}. Available platforms: {}".format(
            os,
            arch,
            ", ".join(["{}-{}".format(o, a) for o in checksums.keys() for a in checksums[o].keys()]),
        ))

    # Get SHA256 checksum for this platform
    sha256 = checksums[os][arch]

    # Strip 'v' prefix from version for URL and strip_prefix
    version_no_v = version[1:] if version.startswith("v") else version

    # Construct download URL
    url = "https://github.com/golangci/golangci-lint/releases/download/{version}/golangci-lint-{version_no_v}-{platform}-{arch}.tar.gz".format(
        version = version,
        version_no_v = version_no_v,
        platform = os,
        arch = arch,
    )

    http_archive(
        name = "golangci_lint_binary",
        url = url,
        sha256 = sha256,
        strip_prefix = "golangci-lint-{version}-{platform}-{arch}".format(
            version = version_no_v,
            platform = os,
            arch = arch,
        ),
        build_file = "//golangci_lint:private/golangci_lint_binary.BUILD.bazel",
    )

    return ctx.extension_metadata(
        root_module_direct_deps = ["golangci_lint_binary"],
        root_module_direct_dev_deps = [],
        reproducible = True,
    )

def _detect_platforms(ctx):
    """Detects the platforms of the current build."""

    os_map = {
        "linux": "linux",
        "mac os x": "darwin",
        "windows": "windows",
    }

    arch_map = {
        "x86_64": "amd64",
        "amd64": "amd64",  # Some systems report as amd64 instead of x86_64
        "aarch64": "arm64",
        "arm64": "arm64",
    }

    os_name = ctx.os.name.lower()
    os_arch = ctx.os.arch.lower()

    if os_name not in os_map:
        fail("Unsupported operating system: {}. Supported: {}".format(os_name, ", ".join(os_map.keys())))

    if os_arch not in arch_map:
        fail("Unsupported architecture: {}. Supported: {}".format(os_arch, ", ".join(arch_map.keys())))

    return os_map[os_name], arch_map[os_arch]

golangci_lint = module_extension(
    implementation = _golangci_lint_extension_impl,
    tag_classes = {
        "config": _config_tag,
    },
)
