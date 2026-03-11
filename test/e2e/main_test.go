package e2e

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	"github.com/openmcp-project/openmcp-testing/pkg/providers"
	"github.com/openmcp-project/openmcp-testing/pkg/setup"
	"github.com/openmcp-project/openmcp-testing/pkg/setup/extensions"
	"github.com/openmcp-project/openmcp-testing/pkg/setup/extensions/fluxcd"
	localextensions "github.com/openmcp-project/observability-stack/extensions"
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

// injectVersion replaces version placeholders in YAML files
func injectVersion(yamlContent string, version string) string {
	return strings.ReplaceAll(yamlContent, "<VERSION_TO_TEST>", version)
}

// injectEnvVars replaces environment variable placeholders in YAML files
func injectEnvVars(yamlContent string) string {
	result := yamlContent
	// Try OCM-specific env vars first, fallback to generic GITHUB_ vars
	username := os.Getenv("OCM_GITHUB_USERNAME")
	if username == "" {
		username = os.Getenv("GITHUB_USERNAME")
	}
	token := os.Getenv("OCM_GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	result = strings.ReplaceAll(result, "${GITHUB_USERNAME}", username)
	result = strings.ReplaceAll(result, "${GITHUB_TOKEN}", token)
	return result
}

// processYAMLFile reads a YAML file, injects the version, and returns the content
func processYAMLFile(filePath string) (string, error) {
	version, err := getVersion()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	result := injectVersion(string(data), version)
	result = injectEnvVars(result)
	return result, nil
}
