# Version Injection Usage

The helper functions added to `main_test.go` allow you to inject the VERSION into YAML files before applying them.

## Available Functions

### 1. `getVersion() (string, error)`
Reads the VERSION file from the repository root.

```go
version, err := getVersion()
// Returns: "v0.0.3"
```

### 2. `injectVersion(yamlContent string, version string) string`
Replaces `<VERSION_TO_TESTY>` placeholder with the actual version.

```go
content := "semver: \"<VERSION_TO_TESTY>\""
result := injectVersion(content, "v0.0.3")
// Returns: "semver: \"v0.0.3\""
```

### 3. `processYAMLFile(filePath string) (string, error)`
Reads a YAML file, injects the version, and returns the processed content.

```go
content, err := processYAMLFile("onboarding/ocm.yaml")
// Returns the file content with <VERSION_TO_TESTY> replaced
```

## Usage in Tests

### Example 1: Using processYAMLFile with kubectl apply

```go
func TestObsStack(t *testing.T) {
    obsStackTest := features.New("observability-stack test").
        Setup(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
            // Process the YAML file with version injection
            yamlContent, err := processYAMLFile("onboarding/ocm.yaml")
            if err != nil {
                t.Fatalf("failed to process YAML file: %v", err)
            }

            // Write to a temporary file
            tmpFile, err := os.CreateTemp("", "ocm-*.yaml")
            if err != nil {
                t.Fatalf("failed to create temp file: %v", err)
            }
            defer os.Remove(tmpFile.Name())

            if _, err := tmpFile.WriteString(yamlContent); err != nil {
                t.Fatalf("failed to write temp file: %v", err)
            }
            tmpFile.Close()

            // Apply the processed YAML
            cmd := exec.Command("kubectl", "apply", "-f", tmpFile.Name())
            if kubeconfig := c.KubeconfigFile(); kubeconfig != "" {
                cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
            }

            out, err := cmd.CombinedOutput()
            if err != nil {
                t.Fatalf("kubectl apply failed: %v: %s", err, string(out))
            }

            return ctx
        })

    testenv.Test(t, obsStackTest.Feature())
}
```

### Example 2: Using getVersion for dynamic values

```go
version, err := getVersion()
if err != nil {
    t.Fatalf("failed to get version: %v", err)
}

// Use version in your test
component := &ocmv1alpha1.Component{
    Spec: ocmv1alpha1.ComponentSpec{
        Semver: version,
    },
}
```

### Example 3: Processing multiple files

```go
files := []string{
    "onboarding/ocm.yaml",
    "onboarding/component.yaml",
}

for _, file := range files {
    content, err := processYAMLFile(file)
    if err != nil {
        t.Fatalf("failed to process %s: %v", file, err)
    }

    // Apply content...
}
```

## Shell Script Alternative

If you prefer to use a shell script for pre-processing:

```bash
#!/usr/bin/env bash
VERSION=$(cat ../../VERSION)
sed "s/<VERSION_TO_TESTY>/${VERSION}/g" onboarding/ocm.yaml | kubectl apply -f -
```
