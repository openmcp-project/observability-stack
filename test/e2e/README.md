# E2E Tests

This directory contains end-to-end tests for the observability-stack.

## Prerequisites

1. **GitHub Credentials**: You need GitHub credentials to access the OCI registry at `ghcr.io/openmcp-project/components`

2. **Required tools**:
   - Go 1.21+
   - kubectl
   - flux CLI
   - helm
   - Kind (or another Kubernetes cluster)

## Setup

Set your GitHub credentials as environment variables. Use the `OCM_` prefixed variables to avoid conflicts with other tools (like Helm):

```bash
export OCM_GITHUB_USERNAME="your-github-username"
export OCM_GITHUB_TOKEN="your-github-pat-token"
```

Alternatively, you can use the generic `GITHUB_USERNAME` and `GITHUB_TOKEN` variables, but be aware they may conflict with Helm OCI authentication.

The test framework will automatically inject these into the OCM configuration secret.

## Running Tests

### Run all tests

```bash
cd test/e2e
go test -v ./...
```

### Run specific test

```bash
cd test/e2e
go test -v -run TestObsStack
```

### With verbose logging

```bash
cd test/e2e
go test -v -args -v=4
```

## How It Works

### Test Flow

1. **Setup Phase** ([main_test.go:31-56](../../test/e2e/main_test.go#L31-L56)):
   - Creates an openMCP cluster using the test framework
   - Installs Flux CD
   - Installs Kro (Kubernetes Resource Orchestrator)
   - Installs OCM Kubernetes Toolkit

2. **Test Execution** ([obs_stack_test.go:23-107](../../test/e2e/obs_stack_test.go#L23-L107)):
   - Creates platform cluster resources
   - Processes and applies onboarding YAML files (with version injection)
   - Waits for Repository, Component, Resource, and Deployer to be ready
   - Verifies all components are functioning correctly

3. **Teardown Phase**:
   - Cleans up all created resources

### Version Injection

The test framework automatically replaces placeholders in YAML files:

- `<VERSION_TO_TEST>` → Replaced with the version from `../../VERSION`
- `${GITHUB_USERNAME}` → Replaced with `$GITHUB_USERNAME` environment variable
- `${GITHUB_TOKEN}` → Replaced with `$GITHUB_TOKEN` environment variable

This is handled by the `processYAMLFile()` function in [main_test.go:160-174](../../test/e2e/main_test.go#L160-L174).

## CI/CD Integration

For GitHub Actions or other CI systems:

```yaml
- name: Run E2E Tests
  env:
    OCM_GITHUB_USERNAME: ${{ secrets.GITHUB_USERNAME }}
    OCM_GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: |
    cd test/e2e
    go test -v ./...
```

## Troubleshooting

### "Failed to create object: secrets 'ocm-config' already exists"

The secret is created automatically by the test. If you manually created it, delete it first:

```bash
kubectl delete secret ocm-config -n obs-stack
```

### "Repository not ready: authentication required"

Check that your GitHub credentials are set correctly:

```bash
echo $OCM_GITHUB_USERNAME
echo $OCM_GITHUB_TOKEN  # Should show your PAT
```

### "Component not ready: component not found"

Verify the version in the `VERSION` file exists in the OCI registry:

```bash
cat ../../VERSION
```

## Adding New Tests

To add a new test, follow the pattern in [obs_stack_test.go](../../test/e2e/obs_stack_test.go):

1. Create a new test function: `func TestMyFeature(t *testing.T) { ... }`
2. Use `features.New()` to define Setup, Assess, and Teardown phases
3. Use the helper functions for processing YAML files
4. Register the test with `testenv.Test(t, myFeature.Feature())`
