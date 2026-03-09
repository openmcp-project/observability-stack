package e2e

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	kustomize1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	krov1alpha1 "github.com/kubernetes-sigs/kro/api/v1alpha1"

	ocmv1alpha1 "ocm.software/open-component-model/kubernetes/controller/api/v1alpha1"

	"github.com/openmcp-project/openmcp-testing/pkg/providers"
	"github.com/openmcp-project/openmcp-testing/pkg/setup"
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
		ServiceProviders: []providers.ServiceProviderSetup{},
	}
	testenv = env.NewWithConfig(envconf.New().WithNamespace(openmcp.Namespace))
	openmcp.Bootstrap(testenv)
	testenv.Setup(
		installFlux, registerFluxSchemes,
		installKro, registerKroSchemes,
		installOCMK8sToolkit, registerOCMK8sToolkitSchemes,
	)
	os.Exit(testenv.Run(m))
}

func installFlux(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	args := []string{"install"}
	if kubeconfig := cfg.KubeconfigFile(); kubeconfig != "" {
		args = append(args, "--kubeconfig", kubeconfig)
	}
	out, err := exec.Command("flux", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("flux install failed: %w: %s", err, string(out))
	}
	klog.Infof("flux install output: %s", string(out))
	return ctx, nil
}

func registerFluxSchemes(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	scheme := cfg.Client().Resources().GetScheme()
	if err := helmv2.AddToScheme(scheme); err != nil {
		return ctx, fmt.Errorf("failed to register helm-controller scheme: %w", err)
	}
	if err := kustomize1.AddToScheme(scheme); err != nil {
		return ctx, fmt.Errorf("failed to register kustomization-controller scheme: %w", err)
	}
	if err := sourcev1.AddToScheme(scheme); err != nil {
		return ctx, fmt.Errorf("failed to register source-controller scheme: %w", err)
	}
	return ctx, nil
}

func installKro(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	args := []string{
		"install", "kro",
		"oci://registry.k8s.io/kro/charts/kro",
		"--namespace", "kro-system", "--create-namespace",
	}
	if kubeconfig := cfg.KubeconfigFile(); kubeconfig != "" {
		args = append(args, "--kubeconfig", kubeconfig)
	}
	out, err := exec.Command("helm", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kro install failed: %w: %s", err, string(out))
	}
	klog.Infof("kro install output: %s", string(out))
	return ctx, nil
}

func registerKroSchemes(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	scheme := cfg.Client().Resources().GetScheme()
	if err := krov1alpha1.AddToScheme(scheme); err != nil {
		return ctx, fmt.Errorf("failed to register kro scheme: %w", err)
	}
	return ctx, nil
}

func installOCMK8sToolkit(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	args := []string{
		"install", "ocm-k8s-toolkit",
		"oci://ghcr.io/open-component-model/kubernetes/controller/chart:0.0.0-5b3f034",
		"--namespace", "ocm-k8s-toolkit-system", "--create-namespace",
	}
	if kubeconfig := cfg.KubeconfigFile(); kubeconfig != "" {
		args = append(args, "--kubeconfig", kubeconfig)
	}
	out, err := exec.Command("helm", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("OCM K8s toolkit install failed: %w: %s", err, string(out))
	}
	klog.Infof("OCM K8s toolkit install output: %s", string(out))
	return ctx, nil
}

func registerOCMK8sToolkitSchemes(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	scheme := cfg.Client().Resources().GetScheme()
	if err := ocmv1alpha1.AddToScheme(scheme); err != nil {
		return ctx, fmt.Errorf("failed to register OCM K8s toolkit scheme: %w", err)
	}
	return ctx, nil
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

	return injectVersion(string(data), version), nil
}
