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

# Extract major.minor versions (e.g., 1.24 from 1.24.0)
GO_MOD_MAJOR_MINOR=$(echo "$GO_MOD_VERSION" | cut -d. -f1,2)
DOCKER_MAJOR_MINOR=$(echo "$DOCKER_VERSION" | cut -d. -f1,2)

echo ""
echo "Comparing major.minor versions:"
echo "  go.mod:     $GO_MOD_MAJOR_MINOR (from $GO_MOD_VERSION)"
echo "  Dockerfile: $DOCKER_MAJOR_MINOR (from $DOCKER_VERSION)"

# Compare major.minor versions only
if [ "$GO_MOD_MAJOR_MINOR" != "$DOCKER_MAJOR_MINOR" ]; then
    echo ""
    echo "ERROR: Go version mismatch detected!"
    echo "  go.mod:     $GO_MOD_VERSION (major.minor: $GO_MOD_MAJOR_MINOR)"
    echo "  Dockerfile: $DOCKER_VERSION (major.minor: $DOCKER_MAJOR_MINOR)"
    echo ""
    echo "Please ensure Go major.minor versions are synchronized across files."
    exit 1
fi

echo ""
echo "SUCCESS: Go major.minor versions are in sync ($GO_MOD_MAJOR_MINOR)"
echo "  go.mod:     $GO_MOD_VERSION"
echo "  Dockerfile: $DOCKER_VERSION"
