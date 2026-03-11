#!/usr/bin/env python3
"""
Common functions shared across build and push scripts.
"""

import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Dict, Optional, Tuple

try:
    import yaml
except ImportError:
    print("Error: PyYAML is required. Install it with: pip install pyyaml", flush=True)
    sys.exit(1)


def detect_repo_root() -> Tuple[Path, Path]:
    """
    Detect script directory and repository root.

    Returns:
        Tuple of (script_dir, repo_root)
    """
    script_dir = Path(__file__).resolve().parent
    repo_root = script_dir.parent

    print(f"Script directory: {script_dir}", flush=True)
    print(f"Repository root: {repo_root}", flush=True)

    return script_dir, repo_root


def load_component_settings(repo_root: Path) -> Tuple[Optional[Path], Dict[str, str]]:
    """
    Auto-detect and load component settings file using YAML parser.

    Args:
        repo_root: Path to the repository root

    Returns:
        Tuple of (settings_file_path, environment_variables)
    """
    settings_file = None
    env_vars = {}

    # Auto-detect component settings file
    if "COMPONENT_SETTINGS_PATH" in os.environ:
        settings_file = repo_root / os.environ["COMPONENT_SETTINGS_PATH"]
    elif (repo_root / "component-settings.yaml").exists():
        settings_file = repo_root / "component-settings.yaml"

    # Load environment variables from component-settings.yaml if it exists
    if settings_file and settings_file.exists():
        print(f"Loading environment variables from {settings_file}...", flush=True)

        with open(settings_file, 'r', encoding='utf-8') as f:
            try:
                data = yaml.safe_load(f)

                if data and isinstance(data, dict):
                    for key, value in data.items():
                        # Convert value to string
                        value_str = str(value)

                        # Perform variable substitution
                        value_str = expand_variables(value_str, env_vars)

                        # Store and export the variable
                        env_vars[key] = value_str
                        os.environ[key] = value_str
                        print(f"  {key}={value_str}", flush=True)
                else:
                    print("Warning: Settings file is empty or not a valid YAML dictionary", flush=True)

            except yaml.YAMLError as e:
                print(f"Error parsing YAML file: {e}", flush=True)
                sys.exit(1)

        print("Environment variables loaded.", flush=True)
    else:
        print("Warning: Settings file not found. Using existing environment variables.", flush=True)
        settings_file = None

    return settings_file, env_vars


def expand_variables(value: str, env_vars: Dict[str, str]) -> str:
    """
    Expand environment variables in a string.

    Args:
        value: String that may contain variable references
        env_vars: Dictionary of environment variables

    Returns:
        String with variables expanded
    """
    # Replace ${VAR} and $VAR patterns
    def replace_var(match):
        var_name = match.group(1) or match.group(2)
        # Check local env_vars first, then os.environ
        return env_vars.get(var_name, os.environ.get(var_name, match.group(0)))

    # Match ${VAR} and $VAR patterns
    pattern = r'\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)'
    return re.sub(pattern, replace_var, value)


def read_version_file(repo_root: Path) -> str:
    """
    Read version from VERSION file.

    Args:
        repo_root: Path to the repository root

    Returns:
        Version string

    Raises:
        SystemExit: If VERSION file is not found
    """
    version_file = repo_root / "VERSION"

    if not version_file.exists():
        print(f"Error: VERSION file not found at {version_file}", flush=True)
        sys.exit(1)

    version = version_file.read_text().strip()

    # Append git commit SHA if it's a development version without a SHA already
    # Check if version ends with a git SHA pattern (7 hex characters after -dev-)
    if "-dev" in version and not re.search(r'-dev-[0-9a-f]{7}$', version):
        try:
            git_commit = subprocess.check_output(
                ["git", "-C", str(repo_root), "rev-parse", "--short", "HEAD"],
                stderr=subprocess.DEVNULL,
                text=True
            ).strip()
        except subprocess.CalledProcessError:
            git_commit = "unknown"

        version = f"{version}-{git_commit}"

    os.environ["OBSERVABILITY_STACK_VERSION"] = version
    print(f"Observability stack version: {version}", flush=True)

    return version


def merge_rgd_files(repo_root: Path) -> Path:
    """
    Merge all YAML files from resource-graph-definitions directory.

    Args:
        repo_root: Path to the repository root

    Returns:
        Path to the merged YAML file
    """
    print("Merging RGD files...", flush=True)

    rgd_dir = repo_root / "resource-graph-definitions"
    tmp_dir = repo_root / "tmp"
    tmp_dir.mkdir(exist_ok=True)

    merged_file = tmp_dir / "rgd-merged.yaml"
    merged_file.unlink(missing_ok=True)

    if not rgd_dir.exists():
        print(f"Warning: RGD directory not found at {rgd_dir}", flush=True)
        return merged_file

    # Merge all yaml files from rgd folder
    yaml_files = sorted(rgd_dir.glob("*.yaml"))

    if yaml_files:
        with open(merged_file, 'w', encoding='utf-8') as outfile:
            for i, yaml_file in enumerate(yaml_files):
                if i > 0:
                    outfile.write("---\n")
                outfile.write(yaml_file.read_text())

        print(f"RGD files merged into {merged_file}", flush=True)
    else:
        print(f"Warning: No YAML files found in {rgd_dir}", flush=True)

    return merged_file


def build_ocm_vars(settings_file: Optional[Path], base_vars: Dict[str, str]) -> Dict[str, str]:
    """
    Build OCM variables dictionary from component settings using YAML parser.

    Args:
        settings_file: Path to the settings file (or None)
        base_vars: Base variables to include

    Returns:
        Dictionary of OCM variables
    """
    ocm_vars = dict(base_vars)

    if settings_file and settings_file.exists():
        with open(settings_file, 'r', encoding='utf-8') as f:
            try:
                data = yaml.safe_load(f)

                if data and isinstance(data, dict):
                    for key, value in data.items():
                        # Convert value to string
                        value_str = str(value)

                        # Perform variable substitution
                        value_str = expand_variables(value_str, ocm_vars)
                        ocm_vars[key] = value_str

            except yaml.YAMLError as e:
                print(f"Warning: Error parsing YAML file: {e}", flush=True)

    return ocm_vars


def run_command(
    cmd: list,
    cwd: Optional[Path] = None,
    check: bool = True,
    capture_output: bool = False
) -> subprocess.CompletedProcess:
    """
    Run a shell command with real-time output streaming.

    Args:
        cmd: Command as a list of strings
        cwd: Working directory for the command
        check: Whether to raise an exception on non-zero exit code
        capture_output: If True, capture output instead of streaming
                       (useful for parsing command output)

    Returns:
        CompletedProcess object

    Raises:
        subprocess.CalledProcessError: If command fails and check=True
    """
    print(f"Running: {' '.join(cmd)}", flush=True)

    try:
        if capture_output:
            # Capture mode: for commands where we need to parse output
            result = subprocess.run(
                cmd,
                cwd=cwd,
                check=check,
                text=True,
                capture_output=True
            )

            if result.stdout:
                print(result.stdout, flush=True)
            if result.stderr:
                print(result.stderr, file=sys.stderr, flush=True)

            return result
        else:
            # Streaming mode: real-time output for long-running commands
            result = subprocess.run(
                cmd,
                cwd=cwd,
                check=check,
                text=True,
                stdout=None,  # Inherit parent's stdout for streaming
                stderr=None   # Inherit parent's stderr for streaming
            )

            return result

    except subprocess.CalledProcessError as e:
        print(f"Error: Command failed with exit code {e.returncode}",
              file=sys.stderr, flush=True)
        if capture_output:
            if e.stdout:
                print(f"stdout: {e.stdout}", flush=True)
            if e.stderr:
                print(f"stderr: {e.stderr}", file=sys.stderr, flush=True)
        raise
