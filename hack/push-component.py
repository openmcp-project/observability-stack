#!/usr/bin/env python3
"""
Push OCM component to registry.

This script transfers the built OCM component to the target registry.
"""

import os
import subprocess
import sys
from pathlib import Path

# Import common functions
from common import (
    detect_repo_root,
    load_component_settings,
    run_command,
)


def push_component(repo_root: Path, components_location: str) -> None:
    """
    Push the OCM component to the registry.

    Args:
        repo_root: Path to the repository root
        components_location: Target registry location

    Raises:
        SystemExit: If CTF directory is not found
    """
    ctf_dir = repo_root / "ctf"

    if not ctf_dir.exists():
        print(f"Error: CTF directory not found at {ctf_dir}")
        print("Please run build-component.py first to build the component.")
        sys.exit(1)

    repository_context = os.environ.get("REPOSITORY_CONTEXT", "")
    print(f"Pushing component to {repository_context or components_location}...")

    run_command([
        "ocm", "transfer", "ctf",
        "--copy-resources",
        "--no-update",
        str(ctf_dir),
        components_location
    ])


def main():
    """Main entry point for the push script."""
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
        push_component(repo_root, components_location)

        print("Push completed successfully!")

    except KeyboardInterrupt:
        print("\nPush interrupted by user")
        sys.exit(130)
    except (subprocess.CalledProcessError, OSError, ValueError) as e:
        print(f"Push failed: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
