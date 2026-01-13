#!/bin/bash

set -e

echo "Checking Go version consistency across files..."
echo ""

# Extract Go version from go.mod
GO_MOD_VERSION=$(grep '^go ' go.mod | awk '{print $2}')
echo "INFO: go.mod: $GO_MOD_VERSION"

# Extract Go version from Dockerfile (handle ubi9/go-toolset:X.Y format)
DOCKER_VERSION=$(grep 'FROM registry.access.redhat.com/ubi9/go-toolset:' Dockerfile | head -n 1 | sed -E 's/.*go-toolset:([0-9]+\.[0-9]+).*/\1/')
echo "INFO: Dockerfile: $DOCKER_VERSION"

# Extract Go version from gotests workflow
GOTESTS_VERSION=$(grep 'go-version:' .github/workflows/gotests.yml | sed -E 's/.*go-version:\s*"?([0-9]+\.[0-9]+)"?.*/\1/')
echo "INFO: gotests.yml: $GOTESTS_VERSION"

# Extract major.minor versions (e.g., 1.24 from 1.24.0)
GO_MOD_MAJOR_MINOR=$(echo "$GO_MOD_VERSION" | cut -d. -f1,2)
DOCKER_MAJOR_MINOR=$(echo "$DOCKER_VERSION" | cut -d. -f1,2)
GOTESTS_MAJOR_MINOR=$(echo "$GOTESTS_VERSION" | cut -d. -f1,2)

echo ""
echo "Comparing major.minor versions:"
echo "  go.mod:      $GO_MOD_MAJOR_MINOR (from $GO_MOD_VERSION)"
echo "  Dockerfile:  $DOCKER_MAJOR_MINOR (from $DOCKER_VERSION)"
echo "  gotests.yml: $GOTESTS_MAJOR_MINOR (from $GOTESTS_VERSION)"

# Compare major.minor versions only
MISMATCH=0

if [ "$GO_MOD_MAJOR_MINOR" != "$DOCKER_MAJOR_MINOR" ]; then
    echo ""
    echo "ERROR: Go version mismatch between go.mod and Dockerfile!"
    echo "  go.mod:     $GO_MOD_VERSION (major.minor: $GO_MOD_MAJOR_MINOR)"
    echo "  Dockerfile: $DOCKER_VERSION (major.minor: $DOCKER_MAJOR_MINOR)"
    MISMATCH=1
fi

if [ "$GO_MOD_MAJOR_MINOR" != "$GOTESTS_MAJOR_MINOR" ]; then
    echo ""
    echo "ERROR: Go version mismatch between go.mod and gotests.yml!"
    echo "  go.mod:      $GO_MOD_VERSION (major.minor: $GO_MOD_MAJOR_MINOR)"
    echo "  gotests.yml: $GOTESTS_VERSION (major.minor: $GOTESTS_MAJOR_MINOR)"
    MISMATCH=1
fi

if [ $MISMATCH -eq 1 ]; then
    echo ""
    echo "Please ensure Go major.minor versions are synchronized across all files."
    exit 1
fi

echo ""
echo "SUCCESS: All Go major.minor versions are in sync ($GO_MOD_MAJOR_MINOR)"
echo "  go.mod:      $GO_MOD_VERSION"
echo "  Dockerfile:  $DOCKER_VERSION"
echo "  gotests.yml: $GOTESTS_VERSION"
