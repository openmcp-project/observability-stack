package extensions

import (
	"context"
	"fmt"
	"log"
	"os"

	krov1alpha1 "github.com/kubernetes-sigs/kro/api/v1alpha1"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

type Kro struct {
	// ChartVersion is the Kro chart version to install
	ChartVersion string
	// Namespace is the namespace to install Kro
	Namespace string
	// ReleaseName is the name of the Helm release
	ReleaseName string
}

func (k *Kro) Name() string {
	return "Kro"
}

func (k *Kro) Install(ctx context.Context, cfg *envconf.Config) error {
	chartVersion := k.ChartVersion
	if chartVersion == "" {
		chartVersion = "latest"
	}

	namespace := k.Namespace
	if namespace == "" {
		namespace = "kro-system"
	}

	releaseName := k.ReleaseName
	if releaseName == "" {
		releaseName = "kro"
	}

	// Set up Helm CLI settings
	settings := cli.New()
	if kubeconfig := cfg.KubeconfigFile(); kubeconfig != "" {
		settings.KubeConfig = kubeconfig
	}
	settings.SetNamespace(namespace)

	// Create action configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return fmt.Errorf("failed to initialize Helm action config: %w", err)
	}

	// Set up registry client for OCI (must be done before LocateChart)
	registryClient, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create registry client: %w", err)
	}
	actionConfig.RegistryClient = registryClient

	// Create install action
	client := action.NewInstall(actionConfig)
	client.Namespace = namespace
	client.ReleaseName = releaseName
	client.CreateNamespace = true

	// Load the chart from OCI registry
	chartRef := fmt.Sprintf("oci://registry.k8s.io/kro/charts/kro")
	if chartVersion != "latest" {
		chartRef = fmt.Sprintf("%s:%s", chartRef, chartVersion)
	}

	chartPath, err := client.ChartPathOptions.LocateChart(chartRef, settings)
	if err != nil {
		return fmt.Errorf("failed to locate chart: %w", err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	// Install the chart
	release, err := client.Run(chart, nil)
	if err != nil {
		return fmt.Errorf("Kro install failed: %w", err)
	}

	klog.Infof("Kro installed successfully: release=%s, version=%s", release.Name, release.Chart.Metadata.Version)
	return nil
}

func (k *Kro) RegisterSchemes(ctx context.Context, scheme *runtime.Scheme) error {
	if err := krov1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to register Kro scheme: %w", err)
	}
	return nil
}
