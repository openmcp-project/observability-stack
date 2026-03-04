#!/usr/bin/env bash

# Common functions shared across build and push scripts

# Detect script directory and repository root
detect_repo_root() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[1]}")" && pwd)"
    REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
    echo "Script directory: $SCRIPT_DIR"
    echo "Repository root: $REPO_ROOT"
}

# Auto-detect and load component settings file
load_component_settings() {
    # Auto-detect component settings file
    if [ -n "$COMPONENT_SETTINGS_PATH" ]; then
        SETTINGS_FILE="$REPO_ROOT/$COMPONENT_SETTINGS_PATH"
    elif [ -f "$REPO_ROOT/component-settings.yaml" ]; then
        SETTINGS_FILE="$REPO_ROOT/component-settings.yaml"
    else
        SETTINGS_FILE=""
    fi

    # Load environment variables from component-settings.yaml if it exists
    if [ -n "$SETTINGS_FILE" ] && [ -f "$SETTINGS_FILE" ]; then
        echo "Loading environment variables from $SETTINGS_FILE..."
        # Parse YAML and export as environment variables
        while IFS=': ' read -r key value; do
            # Skip comments and empty lines
            [[ "$key" =~ ^#.*$ || -z "$key" ]] && continue
            # Remove quotes and trim whitespace
            value=$(echo "$value" | sed -e 's/^"//' -e 's/"$//' -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
            # Perform variable substitution using eval
            value=$(eval echo "$value")
            # Export the variable
            export "$key=$value"
            echo "  $key=$value"
        done < <(grep -v '^#' "$SETTINGS_FILE" | grep -v '^[[:space:]]*$')
        echo "Environment variables loaded."
    else
        echo "Warning: Settings file not found. Using existing environment variables."
        SETTINGS_FILE=""
    fi
}

# Read version from VERSION file
read_version_file() {
    VERSION_FILE="$REPO_ROOT/VERSION"
    if [ -f "$VERSION_FILE" ]; then
        OBSERVABILITY_STACK_VERSION=$(cat "$VERSION_FILE")

        # Append git commit SHA if it's a development version
        if [[ "$OBSERVABILITY_STACK_VERSION" == *"-dev"* ]]; then
            GIT_COMMIT=$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo "unknown")
            OBSERVABILITY_STACK_VERSION="${OBSERVABILITY_STACK_VERSION}-${GIT_COMMIT}"
        fi

        export OBSERVABILITY_STACK_VERSION
        echo "Observability stack version: $OBSERVABILITY_STACK_VERSION"
    else
        echo "Error: VERSION file not found at $VERSION_FILE"
        exit 1
    fi
}

# Merge all YAML files from resource-graph-definitions directory
merge_rgd_files() {
    echo "Merging RGD files..."
    RGD_DIR="$REPO_ROOT/resource-graph-definitions"
    TMP_DIR="$REPO_ROOT/tmp"
    mkdir -p "$TMP_DIR"
    rm -f "$TMP_DIR/rgd-merged.yaml"

    # Merge all yaml files from rgd folder
    if [ -d "$RGD_DIR" ]; then
        first_file=true
        for file in "$RGD_DIR"/*.yaml; do
            if [ -f "$file" ]; then
                if [ "$first_file" = true ]; then
                    cat "$file" > "$TMP_DIR/rgd-merged.yaml"
                    first_file=false
                else
                    echo "---" >> "$TMP_DIR/rgd-merged.yaml"
                    cat "$file" >> "$TMP_DIR/rgd-merged.yaml"
                fi
            fi
        done
        echo "RGD files merged into $TMP_DIR/rgd-merged.yaml"
    else
        echo "Warning: RGD directory not found at $RGD_DIR"
    fi
}

# Build OCM variables string from component settings
build_ocm_vars() {
    local ocm_vars="$1"

    if [ -n "$SETTINGS_FILE" ] && [ -f "$SETTINGS_FILE" ]; then
        while IFS=': ' read -r key value; do
            # Skip comments and empty lines
            [[ "$key" =~ ^#.*$ || -z "$key" ]] && continue
            # Remove quotes and trim whitespace
            value=$(echo "$value" | sed -e 's/^"//' -e 's/"$//' -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
            # Perform variable substitution using eval
            value=$(eval echo "$value")
            # Add to OCM variables
            ocm_vars="$ocm_vars $key=$value"
        done < <(grep -v '^#' "$SETTINGS_FILE" | grep -v '^[[:space:]]*$')
    fi

    echo "$ocm_vars"
}
