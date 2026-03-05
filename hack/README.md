# Hack Scripts

This directory contains utility scripts for building and pushing OCM components.

## Migration to Python

The bash scripts have been converted to Python for better maintainability, readability, and debugging. Both versions are kept for compatibility during the transition.

### Available Scripts

#### Python Scripts (Recommended)

- **[common.py](common.py)** - Shared utility functions
  - Repository and script path detection
  - Component settings loading with variable expansion
  - Version file reading with git commit appending
  - Resource graph definition merging
  - OCM variables building

- **[build-component.py](build-component.py)** - Build OCM component
  - Loads component settings and version
  - Merges resource graph definition files
  - Pushes kustomization artifacts using flux
  - Builds the OCM component

- **[push-component.py](push-component.py)** - Push component to registry
  - Transfers the built OCM component to the target registry

#### Legacy Bash Scripts

- **[common.sh](common.sh)** - Shared bash functions
- **[build-component.sh](build-component.sh)** - Build OCM component (bash)
- **[push-component.sh](push-component.sh)** - Push component to registry (bash)

## Usage

### Building the Component

```bash
# Using Python (recommended)
./hack/build-component.py

# Using bash (legacy)
./hack/build-component.sh
```

### Pushing the Component

```bash
# Using Python (recommended)
./hack/push-component.py

# Using bash (legacy)
./hack/push-component.sh
```

## Requirements

### Python Version
- Python 3.6 or higher

### Python Dependencies

Install all Python dependencies using:

```bash
pip install -r hack/requirements.txt
```

Or install manually:
- PyYAML - `pip install pyyaml`

### External Tools
- OCM CLI (`ocm`)
- Flux CLI (`flux`)
- Git

### Environment Variables

The scripts automatically load settings from `component-settings.yaml` in the repository root. You can also override settings using environment variables:

- `COMPONENT_SETTINGS_PATH` - Custom path to component settings file
- `COMPONENT_CONSTRUCTOR_PATH` - Custom path to component constructor file
- `OBSERVABILITY_STACK_VERSION` - Component version (auto-detected from VERSION file)
- `COMPONENTS_LOCATION` - Target registry for pushing components
- `KUSTOMIZATIONS_LOCATION_PREFIX` - Registry prefix for kustomization artifacts
- `REPOSITORY_CONTEXT` - Repository context (e.g., ghcr.io/openmcp-project)

## Improvements in Python Version

1. **Proper YAML Parsing**: Uses PyYAML library for robust YAML parsing instead of manual line-by-line parsing
2. **Better Error Handling**: Clear error messages and proper exception handling with try-catch blocks
3. **Type Hints**: Function signatures include type information for better IDE support and maintainability
4. **Documentation**: Comprehensive docstrings for all functions with parameter and return type descriptions
5. **Modularity**: Cleaner separation of concerns with dedicated, reusable functions
6. **Debugging**: Easier to debug with Python's rich debugging tools (pdb, IDE debuggers)
7. **Cross-platform**: Better compatibility across different operating systems using pathlib
8. **Variable Expansion**: More robust environment variable substitution with regex patterns
9. **Logging**: Better output formatting and error reporting with detailed messages
10. **Testing**: Easier to write unit tests for Python functions with proper mocking support

## Migration Notes

The Python scripts are functionally equivalent to the bash scripts. Key differences:

- **YAML Parsing**: Python uses `yaml.safe_load()` for proper YAML parsing (bash used line-by-line parsing)
- **Variable Expansion**: Python uses a regex-based approach for `${VAR}` and `$VAR` patterns
- **Command Execution**: Python uses `subprocess.run()` with better error capture and output handling
- **Path Handling**: Python uses `pathlib.Path` for cross-platform path compatibility
- **Error Messages**: Python provides more detailed and helpful error messages

## Development

To contribute or modify the scripts:

1. Edit the Python scripts in this directory
2. Test your changes locally
3. Ensure compatibility with the existing bash scripts during transition
4. Update this README if adding new functionality

## Testing

Test the Python scripts:

```bash
# Dry run - check settings and version loading
python3 -c "from hack.common import *; detect_repo_root(); load_component_settings(Path('.')); read_version_file(Path('.'))"

# Build component
./hack/build-component.py

# Verify the CTF directory was created
ls -la ctf/

# Push component (only after successful build)
./hack/push-component.py
```
