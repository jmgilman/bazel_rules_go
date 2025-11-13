"""Public definitions for the golangci_lint extension."""

load(
    "//golangci_lint/private:golangci_lint_test.bzl",
    _golangci_lint_test = "golangci_lint_test",
)

golangci_lint_test = _golangci_lint_test
