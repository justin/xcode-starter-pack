binary := "build/xcode-starter-pack"

# Show available recipes.
default:
  @just --list

# Compile the project-creation CLI.
build:
  mkdir -p build
  go build -o {{binary}} ./cmd/xcode-starter-pack

# Interactively create a project. Optionally supply its output directory.
run output='': build
  {{binary}} new {{if output != '' { '--output "' + output + '"' } else { '' }}}

# Run the Go test suite.
test:
  go test ./...

# Run Go's static analyzer.
vet:
  go vet ./...

# Run the local verification suite.
check: test vet

# Create a disposable example project in a fresh temporary directory.
generate-example: build
  #!/usr/bin/env bash
  set -euo pipefail
  example_directory="$(mktemp -d "${TMPDIR:-/tmp}/xcode-starter-pack-example.XXXXXX")"
  {{binary}} new --output "$example_directory/Example" --name Example --platform iOS --platform macOS --app-target Example --test-target "Example Tests"
  echo "Example created at $example_directory/Example"

# Build release archives without publishing them.
release-snapshot:
  goreleaser release --snapshot --clean
