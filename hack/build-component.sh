#!/usr/bin/env bash

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Detect repository root
detect_repo_root

# Auto-detect component constructor file
if [ -n "$COMPONENT_CONSTRUCTOR_PATH" ]; then
    CONSTRUCTOR_FILE="$REPO_ROOT/$COMPONENT_CONSTRUCTOR_PATH"
elif [ -f "$REPO_ROOT/component-constructor.yaml" ]; then
    CONSTRUCTOR_FILE="$REPO_ROOT/component-constructor.yaml"
else
    echo "Error: Could not find component-constructor.yaml"
    exit 1
fi
echo "Using component constructor: $CONSTRUCTOR_FILE"

# Load component settings
load_component_settings

# Read version from VERSION file
read_version_file

# Merge ResourceGraphDefinition files
# This is a workaround until the OCM controllers allow to deployer directories
merge_rgd_files

# Push kustomization artifacts
if [ -z "$KUSTOMIZATIONS_LOCATION_PREFIX" ]; then
    echo "Error: KUSTOMIZATIONS_LOCATION_PREFIX not set"
    exit 1
fi

echo "Pushing kustomization artifacts..."

# Define kustomization directories relative to repo root
KUSTOMIZATION_DIRS=(
    "opentelemetry-collector:open-telemetry-collector"
    "prometheus-operator:prometheus-operator"
    "prometheus:prometheus"
    "metrics:metrics"
)

# Push kustomizations directly with the flux cli until there is a better way with ocm directly
for entry in "${KUSTOMIZATION_DIRS[@]}"; do
    IFS=':' read -r dir_name artifact_name <<< "$entry"
    kustomization_dir="$REPO_ROOT/kustomizations/$dir_name"

    if [ -d "$kustomization_dir" ]; then
        echo "  Pushing $artifact_name..."
        flux push artifact "oci://${KUSTOMIZATIONS_LOCATION_PREFIX}/${artifact_name}:${OBSERVABILITY_STACK_VERSION}" \
            --path="$kustomization_dir" \
            --source="$(git config --get remote.origin.url)" \
            --revision="$(git tag --points-at HEAD)@sha1:$(git rev-parse HEAD)"
    else
        echo "  Warning: Kustomization directory not found: $kustomization_dir"
    fi
done

# Build OCM component
echo "Building OCM component..."
CTF_DIR="$REPO_ROOT/ctf"
rm -rf "$CTF_DIR"

# Build variable arguments for OCM command
OCM_VARS="OBSERVABILITY_STACK_VERSION=${OBSERVABILITY_STACK_VERSION} KUSTOMIZATIONS_LOCATION_PREFIX=${KUSTOMIZATIONS_LOCATION_PREFIX}"
OCM_VARS=$(build_ocm_vars "$OCM_VARS")

echo "Building component with variables: $OCM_VARS"

# Execute OCM command with explicit variable passing
ocm add componentversion --create --file "$CTF_DIR" "$CONSTRUCTOR_FILE" -- $OCM_VARS

# Cleanup tmp folder
echo "Cleaning up tmp folder..."
rm -rf "$TMP_DIR"
echo "Build completed successfully!"
