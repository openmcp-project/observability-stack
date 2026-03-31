#!/usr/bin/env python3
"""
Build OCM component for observability stack.

This script:
1. Loads component settings and version
2. Merges resource graph definition files
3. Pushes kustomization artifacts using flux
4. Builds the OCM component
"""

import os
import shutil
import subprocess
import sys
from pathlib import Path

# Import common functions
from common import (
    detect_repo_root,
    load_component_settings,
    read_version_file,
    merge_rgd_files,
    build_ocm_vars,
    run_command,
)


def find_component_constructor(repo_root: Path) -> Path:
    """
    Auto-detect component constructor file.

    Args:
        repo_root: Path to the repository root

    Returns:
        Path to the component constructor file

    Raises:
        SystemExit: If constructor file is not found
    """
    if "COMPONENT_CONSTRUCTOR_PATH" in os.environ:
        constructor_file = repo_root / os.environ["COMPONENT_CONSTRUCTOR_PATH"]
    elif (repo_root / "component-constructor.yaml").exists():
        constructor_file = repo_root / "component-constructor.yaml"
    else:
        print("Error: Could not find component-constructor.yaml")
        sys.exit(1)

    print(f"Using component constructor: {constructor_file}")
    return constructor_file


def push_kustomizations(repo_root: Path, version: str) -> None:
    """
    Push kustomization artifacts using flux CLI.

    Args:
        repo_root: Path to the repository root
        version: Version string for tagging artifacts

    Raises:
        SystemExit: If KUSTOMIZATIONS_LOCATION_PREFIX is not set
    """
    kustomizations_prefix = os.environ.get("KUSTOMIZATIONS_LOCATION_PREFIX")
    if not kustomizations_prefix:
        print("Error: KUSTOMIZATIONS_LOCATION_PREFIX not set")
        sys.exit(1)

    print("Pushing kustomization artifacts...")

    # Define kustomization directories
    kustomization_dirs = [
        ("opentelemetry-collector", "open-telemetry-collector"),
        ("prometheus-operator", "prometheus-operator"),
        ("prometheus", "prometheus"),
        ("metrics", "metrics"),
        ("victoria-logs", "victoria-logs"),
        ("observabiltiy-gateway", "observabiltiy-gateway")
    ]

    # Get git information
    try:
        git_origin = run_command(
            ["git", "config", "--get", "remote.origin.url"],
            cwd=repo_root,
            capture_output=True
        ).stdout.strip()

        git_tag = run_command(
            ["git", "tag", "--points-at", "HEAD"],
            cwd=repo_root,
            capture_output=True
        ).stdout.strip()

        git_sha = run_command(
            ["git", "rev-parse", "HEAD"],
            cwd=repo_root,
            capture_output=True
        ).stdout.strip()

        revision = f"{git_tag}@sha1:{git_sha}" if git_tag else f"main@sha1:{git_sha}"

    except (subprocess.CalledProcessError, OSError) as e:
        print(f"Warning: Could not get git information: {e}")
        git_origin = "unknown"
        revision = "unknown"

    # Push each kustomization
    for dir_name, artifact_name in kustomization_dirs:
        kustomization_dir = repo_root / "kustomizations" / dir_name

        if kustomization_dir.exists():
            print(f"  Pushing {artifact_name}...")

            oci_path = f"oci://{kustomizations_prefix}/{artifact_name}:{version}"

            try:
                run_command([
                    "flux", "push", "artifact", oci_path,
                    f"--path={kustomization_dir}",
                    f"--source={git_origin}",
                    f"--revision={revision}"
                ])
            except (subprocess.CalledProcessError, OSError) as e:
                print(f"  Warning: Failed to push {artifact_name}: {e}")
        else:
            print(f"  Warning: Kustomization directory not found: {kustomization_dir}")


def build_ocm_component(
    repo_root: Path,
    constructor_file: Path,
    settings_file: Path,
    version: str
) -> None:
    """
    Build the OCM component.

    Args:
        repo_root: Path to the repository root
        constructor_file: Path to the component constructor file
        settings_file: Path to the settings file
        version: Version string for the component
    """
    print("Building OCM component...")

    ctf_dir = repo_root / "ctf"

    # Remove existing CTF directory
    if ctf_dir.exists():
        print(f"Removing existing CTF directory: {ctf_dir}")
        shutil.rmtree(ctf_dir)

    # Build variable arguments for OCM command
    kustomizations_prefix = os.environ.get("KUSTOMIZATIONS_LOCATION_PREFIX", "")

    base_vars = {
        "OBSERVABILITY_STACK_VERSION": version,
        "KUSTOMIZATIONS_LOCATION_PREFIX": kustomizations_prefix,
    }

    ocm_vars = build_ocm_vars(settings_file, base_vars)

    # Build OCM command arguments
    var_args = [f"{key}={value}" for key, value in ocm_vars.items()]

    print(f"Building component with variables: {' '.join(var_args)}")

    # Execute OCM command
    cmd = [
        "ocm", "add", "componentversion",
        "--create",
        "--file", str(ctf_dir),
        str(constructor_file),
        "--"
    ] + var_args

    run_command(cmd)


def cleanup_tmp_dir(repo_root: Path) -> None:
    """
    Clean up the tmp folder.

    Args:
        repo_root: Path to the repository root
    """
    tmp_dir = repo_root / "tmp"

    if tmp_dir.exists():
        print("Cleaning up tmp folder...")
        shutil.rmtree(tmp_dir)


def main():
    """Main entry point for the build script."""
    try:
        # Detect repository root
        _, repo_root = detect_repo_root()

        # Find component constructor
        constructor_file = find_component_constructor(repo_root)

        # Load component settings
        settings_file, _ = load_component_settings(repo_root)

        # Read version from VERSION file
        version = read_version_file(repo_root)

        # Merge ResourceGraphDefinition files
        # This is a workaround until the OCM controllers allow deploying directories
        merge_rgd_files(repo_root)

        # Push kustomization artifacts
        push_kustomizations(repo_root, version)

        # Build OCM component
        build_ocm_component(repo_root, constructor_file, settings_file, version)

        # Cleanup tmp folder
        cleanup_tmp_dir(repo_root)

        print("Build completed successfully!")

    except KeyboardInterrupt:
        print("\nBuild interrupted by user")
        sys.exit(130)
    except (subprocess.CalledProcessError, OSError, ValueError) as e:
        print(f"Build failed: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
