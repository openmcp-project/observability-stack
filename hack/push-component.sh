#!/usr/bin/env bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Detect repository root
detect_repo_root

# Load component settings
load_component_settings

# Use COMPONENTS_LOCATION from settings
if [ -z "$COMPONENTS_LOCATION" ]; then
    echo "Error: COMPONENTS_LOCATION not set"
    exit 1
fi

echo "Pushing component to $REPOSITORY_CONTEXT..."
ocm transfer ctf --copy-resources --recursive --lookup ghcr.io/openmcp-project/components ./ctf "$COMPONENTS_LOCATION"

