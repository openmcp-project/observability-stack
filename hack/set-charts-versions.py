#!/usr/bin/env python3

from common import (
    detect_repo_root,
    read_version_file,
    set_all_charts_versions,
)

if __name__ == "__main__":
    # Detect repository root
    _, repo_root = detect_repo_root()

    # Read version from VERSION file
    version = read_version_file(repo_root)

    # Set helm chart versions to match component version
    set_all_charts_versions(repo_root, version)
