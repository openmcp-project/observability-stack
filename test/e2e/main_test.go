package e2e

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	localextensions "github.com/openmcp-project/observability-stack/extensions"
	"github.com/openmcp-project/openmcp-testing/pkg/platformservices"
	"github.com/openmcp-project/openmcp-testing/pkg/providers"
	"github.com/openmcp-project/openmcp-testing/pkg/setup"
	"github.com/openmcp-project/openmcp-testing/pkg/setup/extensions"
	"github.com/openmcp-project/openmcp-testing/pkg/setup/extensions/fluxcd"
)

var testenv env.Environment

func TestMain(m *testing.M) {
	initLogging()
	openmcp := setup.OpenMCPSetup{
		Namespace: "openmcp-system",
		Operator: setup.OpenMCPOperatorSetup{
			Name:         "openmcp-operator",
			Image:        "ghcr.io/openmcp-project/images/openmcp-operator:v0.17.1",
			Environment:  "debug",
			PlatformName: "platform",
		},
		ClusterProviders: []providers.ClusterProviderSetup{
			{
				Name:  "kind",
				Image: "ghcr.io/openmcp-project/images/cluster-provider-kind:v0.0.15",
			},
		},
		PlatformServices: []platformservices.PlatformServiceSetup{
			{
				Name:                      "gateway",
				Image:                     "ghcr.io/openmcp-project/images/platform-service-gateway:v0.0.9",
				PlatformServiceConfigsDir: "platform/gateway",
			},
		},
		Extensions: []extensions.Extension{
			&fluxcd.FluxCD{},
			&localextensions.Kro{},
			&localextensions.OCMK8sToolkit{},
		},
		ServiceProviders: []providers.ServiceProviderSetup{},
	}
	testenv = env.NewWithConfig(envconf.New().WithNamespace(openmcp.Namespace))
	openmcp.Bootstrap(testenv)
	os.Exit(testenv.Run(m))
}

func initLogging() {
	klog.InitFlags(nil)
	if err := flag.Set("v", "2"); err != nil {
		panic(err)
	}
	flag.Parse()
}

// getVersion reads the VERSION file from the repository root
func getVersion() (string, error) {
	versionPath := filepath.Join("..", "..", "VERSION")
	data, err := os.ReadFile(versionPath)
	if err != nil {
		return "", fmt.Errorf("failed to read VERSION file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// processYAMLContent processes YAML content by executing it as a Go template.
// It supports {{.VERSION}} for version and {{.VAR_NAME}} for environment variables.
func processYAMLContent(yamlContent string, version string) (string, error) {
	// Create a data map with version and all environment variables
	data := make(map[string]string)

	// Add version to the data map
	data["VERSION"] = version

	// Add all environment variables to the data map
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			data[parts[0]] = parts[1]
		}
	}

	// Create and execute template directly
	tmpl, err := template.New("yaml").Parse(yamlContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// processYAMLFile reads a YAML file, processes version and environment variables, and returns the content
func processYAMLFile(filePath string) (string, error) {
	version, err := getVersion()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	result, err := processYAMLContent(string(data), version)
	if err != nil {
		return "", fmt.Errorf("failed to process YAML content: %w", err)
	}

	return result, nil
}

// applyYAMLFilesFromGlob processes and applies YAML files matching a glob pattern to the cluster.
// It returns a list of created objects and any error encountered.
func applyYAMLFilesFromGlob(ctx context.Context, t *testing.T, cfg *envconf.Config, pattern string) ([]k8s.Object, error) {
	var objectList []k8s.Object

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
	}

	for _, file := range files {
		content, err := processYAMLFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to process %s: %w", file, err)
		}

		// Decode and create objects from the processed YAML content
		objs, err := decoder.DecodeAll(ctx, bytes.NewReader([]byte(content)))
		if err != nil {
			return nil, fmt.Errorf("failed to decode %s: %w", file, err)
		}

		for _, obj := range objs {
			if err := cfg.Client().Resources().Create(ctx, obj); err != nil {
				return nil, fmt.Errorf("failed to create object from %s: %w", file, err)
			}
			objectList = append(objectList, obj)
		}
	}

	return objectList, nil
}
