#!/usr/bin/env python3
"""
Push OCM component to registry.

This script transfers the built OCM component to the target registry.
"""

import argparse
import os
import subprocess
import sys
from pathlib import Path

# Import common functions
from common import (
    detect_repo_root,
    load_component_settings,
    read_version_file,
    run_command,
)

COMPONENT_NAME = "github.com/openmcp-project/observability-stack"


def push_component(repo_root: Path, components_location: str, copy_resources: bool = False) -> None:
    """
    Push the OCM component to the registry.

    Args:
        repo_root: Path to the repository root
        components_location: Target registry location
        copy_resources: When True, copy all resources into the target registry.
                        When False, copy local blobs only and keep references (PR mode).

    Raises:
        SystemExit: If CTF directory is not found
    """
    ctf_dir = repo_root / "ctf"

    if not ctf_dir.exists():
        print(f"Error: CTF directory not found at {ctf_dir}")
        print("Please run build-component.py first to build the component.")
        sys.exit(1)

    repository_context = os.environ.get("REPOSITORY_CONTEXT", "")
    component = f"{COMPONENT_NAME}:{read_version_file(repo_root)}"
    print(f"Pushing {component} to {repository_context or components_location}...")

    # ocm replaces existing versions and has no flag to prevent it.
    if os.environ.get("OVERWRITE_COMPONENTS", "false") != "true":
        exists = subprocess.run(
            ["ocm", "get", "cv", f"{components_location}//{component}"],
            check=False,
        )
        if exists.returncode == 0:
            print(f"Error: {component} already exists. Set OVERWRITE_COMPONENTS to 'true' to overwrite it.")
            sys.exit(1)

    resource_flags = ["--copy-resources", "--upload-as", "ociArtifact"] if copy_resources else []
    run_command([
        "ocm", "transfer", "cv",
        *resource_flags,
        f"ctf::{ctf_dir}//{component}",
        components_location
    ])


def main():
    """Main entry point for the push script."""
    parser = argparse.ArgumentParser(description="Push OCM component to registry.")
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument(
        "--copy-local-resources",
        action="store_true",
        dest="copy_local_resources",
        help="Copy only local resource refs (PR validation mode).",
    )
    mode.add_argument(
        "--copy-resources",
        action="store_true",
        dest="copy_resources",
        help="Fetch and push all resources as blobs (release/main mode).",
    )
    args = parser.parse_args()

    try:
        # Detect repository root
        _, repo_root = detect_repo_root()

        # Load component settings
        _, _ = load_component_settings(repo_root)

        # Get COMPONENTS_LOCATION from environment
        components_location = os.environ.get("COMPONENTS_LOCATION")

        if not components_location:
            print("Error: COMPONENTS_LOCATION not set")
            print(
                "Please ensure component-settings.yaml is loaded "
                "or set the environment variable."
            )
            sys.exit(1)

        # Push component
        push_component(repo_root, components_location, copy_resources=args.copy_resources)

        print("Push completed successfully!")

    except KeyboardInterrupt:
        print("\nPush interrupted by user")
        sys.exit(130)
    except (subprocess.CalledProcessError, OSError, ValueError) as e:
        print(f"Push failed: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
