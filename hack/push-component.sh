#!/usr/bin/env bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Detect repository root
detect_repo_root

# Load component settings
load_component_settings

# Use REPOSITORY_CONTEXT from settings, or default
if [ -z "$REPOSITORY_CONTEXT" ]; then
    echo "Error: REPOSITORY_CONTEXT not set"
    exit 1
fi

echo "Pushing component to $REPOSITORY_CONTEXT..."
ocm transfer ctf --copy-resources --recursive --lookup ghcr.io/openmcp-project/components ./ctf "$REPOSITORY_CONTEXT"

