#!/bin/bash

set -e

echo "Checking Go version consistency across files..."
echo ""

# Extract Docker image name from Dockerfile
DOCKER_IMAGE=$(grep 'FROM registry.access.redhat.com/ubi9/go-toolset:' Dockerfile | head -n 1 | sed -E 's/.*FROM ([^ ]+).*/\1/')

if [ -z "$DOCKER_IMAGE" ]; then
    echo "ERROR: Could not extract Docker image from Dockerfile"
    exit 1
fi

echo "INFO: Inspecting Docker image: $DOCKER_IMAGE"

# Use skopeo to inspect the image and extract Go version from 'io.k8s.display-name' label
# Example label value: "Go 1.25.3"
DOCKER_GO_VERSION=$(skopeo inspect docker://$DOCKER_IMAGE | jq -r '.Labels["io.k8s.display-name"]' | sed -E 's/Go ([0-9]+\.[0-9]+).*/\1/')

if [ -z "$DOCKER_GO_VERSION" ] || [ "$DOCKER_GO_VERSION" == "null" ]; then
    echo "ERROR: Could not extract Go version from Docker image"
    exit 1
fi

echo "INFO: Docker image Go version: $DOCKER_GO_VERSION"

# Extract Go version from gotests workflow
GOTESTS_VERSION=$(grep 'go-version:' .github/workflows/gotests.yml | sed -E 's/.*go-version:\s*"?([0-9]+\.[0-9]+)"?.*/\1/')

if [ -z "$GOTESTS_VERSION" ]; then
    echo "ERROR: Could not extract Go version from gotests.yml"
    exit 1
fi

echo "INFO: gotests.yml: $GOTESTS_VERSION"

# Extract major.minor versions (e.g., 1.24 from 1.24.0)
DOCKER_MAJOR_MINOR=$(echo "$DOCKER_GO_VERSION" | cut -d. -f1,2)
GOTESTS_MAJOR_MINOR=$(echo "$GOTESTS_VERSION" | cut -d. -f1,2)

echo ""
echo "Comparing major.minor versions:"
echo "  Docker image: $DOCKER_MAJOR_MINOR (from $DOCKER_GO_VERSION)"
echo "  gotests.yml:  $GOTESTS_MAJOR_MINOR (from $GOTESTS_VERSION)"

# Compare major.minor versions only
if [ "$DOCKER_MAJOR_MINOR" != "$GOTESTS_MAJOR_MINOR" ]; then
    echo ""
    echo "ERROR: Go version mismatch between Docker image and gotests.yml!"
    echo "  Docker image: $DOCKER_GO_VERSION (major.minor: $DOCKER_MAJOR_MINOR)"
    echo "  gotests.yml:  $GOTESTS_VERSION (major.minor: $GOTESTS_MAJOR_MINOR)"
    echo ""
    echo "Please ensure Go major.minor versions are synchronized."
    echo "Update the go-version in .github/workflows/gotests.yml to match the Docker image."
    exit 1
fi

echo ""
echo "SUCCESS: Go major.minor versions are in sync ($DOCKER_MAJOR_MINOR)"
echo "  Docker image: $DOCKER_GO_VERSION"
echo "  gotests.yml:  $GOTESTS_VERSION"
