package extensions

import (
	"context"
	"fmt"
	"log"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	ocmv1alpha1 "ocm.software/open-component-model/kubernetes/controller/api/v1alpha1"
)

type OCMK8sToolkit struct {
	// ChartVersion is the OCM K8s toolkit chart version to install
	ChartVersion string
	// Namespace is the namespace to install the OCM K8s toolkit
	Namespace string
	// ReleaseName is the name of the Helm release
	ReleaseName string
}

func (t *OCMK8sToolkit) Name() string {
	return "OCM K8s Toolkit"
}

func (t *OCMK8sToolkit) Install(ctx context.Context, cfg *envconf.Config) error {
	chartVersion := t.ChartVersion
	if chartVersion == "" {
		chartVersion = "0.0.0-5b3f034"
	}

	namespace := t.Namespace
	if namespace == "" {
		namespace = "ocm-k8s-toolkit-system"
	}

	releaseName := t.ReleaseName
	if releaseName == "" {
		releaseName = "ocm-k8s-toolkit"
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
	chartRef := fmt.Sprintf("oci://ghcr.io/open-component-model/kubernetes/controller/chart:%s", chartVersion)
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
		return fmt.Errorf("OCM K8s toolkit install failed: %w", err)
	}

	klog.Infof("OCM K8s toolkit installed successfully: release=%s, version=%s", release.Name, release.Chart.Metadata.Version)
	return nil
}

func (t *OCMK8sToolkit) RegisterSchemes(ctx context.Context, scheme *runtime.Scheme) error {
	if err := ocmv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to register OCM K8s toolkit scheme: %w", err)
	}
	return nil
}
