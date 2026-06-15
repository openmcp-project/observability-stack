package e2e

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	krov1alpha1 "github.com/kubernetes-sigs/kro/api/v1alpha1"
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	localaccess "github.com/openmcp-project/opencontrolplane-runtime/pkg/serviceprovider/clusteraccess"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	commonapi "github.com/openmcp-project/openmcp-operator/api/common"
	ocmv1alpha1 "ocm.software/open-component-model/kubernetes/controller/api/v1alpha1"

	"github.com/openmcp-project/openmcp-testing/pkg/clusterutils"
	"github.com/openmcp-project/openmcp-testing/pkg/conditions"
	"github.com/openmcp-project/openmcp-testing/pkg/resources"
)

const (
	obsStackNamespace  = "obs-stack"
	conditionTypeReady = "Ready"
)

func TestObsStack(t *testing.T) {
	var (
		objectList       []k8s.Object
		onboardingList   *unstructured.UnstructuredList
		onboardingConfig *envconf.Config
	)
	obsStackTest := features.New("observability-stack test").
		Setup(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			// create the obs-stack namespace
			namespace := &corev1.Namespace{}
			namespace.SetName(obsStackNamespace)

			if err := c.Client().Resources().Create(ctx, namespace); err != nil {
				t.Errorf("failed to create namespace: %v", err)
			}

			objectList = append(objectList, namespace)

			// Process and apply the platform manifests
			objs, err := applyYAMLFilesFromGlob(ctx, t, c, "platform/manifests/*.yaml")
			if err != nil {
				t.Fatalf("failed to apply YAML files: %v", err)
			}
			objectList = append(objectList, objs...)

			onboardingConfig, err = clusterutils.OnboardingConfig()
			if err != nil {
				t.Error(err)
				return ctx
			}

			onboardingList, err = resources.CreateObjectsFromDir(ctx, onboardingConfig, "onboarding")
			if err != nil {
				t.Error(err)
			}

			return ctx
		}).
		Assess("verify that the repository, component, resource and deployment are ready", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			repository := &ocmv1alpha1.Repository{}
			repository.SetName("obs-stack-repository")
			repository.SetNamespace(obsStackNamespace)

			component := &ocmv1alpha1.Component{}
			component.SetName("obs-stack-component")
			component.SetNamespace(obsStackNamespace)

			resource := &ocmv1alpha1.Resource{}
			resource.SetName("resource-graph-definition")
			resource.SetNamespace(obsStackNamespace)

			deployer := &ocmv1alpha1.Deployer{}
			deployer.SetName("resource-graph-definition")
			deployer.SetNamespace(obsStackNamespace)

			if err := wait.For(conditions.Match(repository, config, conditionTypeReady, corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("Repository not ready: %v", err)
			}

			if err := wait.For(conditions.Match(component, config, conditionTypeReady, corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("Component not ready: %v", err)
			}

			if err := wait.For(conditions.Match(resource, config, conditionTypeReady, corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("Resource not ready: %v", err)
			}

			if err := wait.For(conditions.Match(repository, config, conditionTypeReady, corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("Repository not ready: %v", err)
			}

			if err := wait.For(conditions.Match(deployer, config, conditionTypeReady, corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("Deployer not ready: %v", err)
			}

			return ctx
		}).
		Assess("resource graph definition is ready", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			rgd := &krov1alpha1.ResourceGraphDefinition{}
			rgd.SetName("obs-stack")

			if err := wait.For(conditions.Match(rgd, config, conditionTypeReady, corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("ResourceGraphDefinition not ready: %v", err)
			}

			return ctx
		}).
		WithStep("create Observability Stack", features.LevelAssess, func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			// Process and apply the platform observability-stack manifests
			objs, err := applyYAMLFilesFromGlob(ctx, t, config, "platform/observability-stack/*.yaml")
			if err != nil {
				t.Fatalf("failed to apply YAML files: %v", err)
			}
			objectList = append(objectList, objs...)
			return ctx
		}).
		Assess("observability stack is ready", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			obsStack := &unstructured.Unstructured{}
			obsStack.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "kro.run",
				Version: "v1alpha1",
				Kind:    "ObservabilityStack",
			})
			obsStack.SetName("stack")
			obsStack.SetNamespace(obsStackNamespace)

			if err := wait.For(func(context.Context) (done bool, err error) {
				if getErr := config.Client().Resources().Get(ctx, obsStack.GetName(), obsStack.GetNamespace(), obsStack); getErr != nil {
					return false, err
				}

				ready, hasReady, err := unstructured.NestedBool(obsStack.Object, "status", "ready")
				if err != nil {
					return false, err
				}
				if !hasReady {
					return false, nil
				}

				return ready, nil
			}, wait.WithTimeout(20*time.Minute)); err != nil {
				t.Errorf("ObservabilityStack not ready: %v", err)
			}

			return ctx
		}).
		WithStep("make observability gateway routable from workload cluster(s)", features.LevelAssess, func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			// add types to scheme
			if err := gatewayv1.Install(config.Client().Resources().GetScheme()); err != nil {
				t.Fatalf("failed to install gateway types: %v", err)
			}
			if err := clustersv1alpha1.AddToScheme(config.Client().Resources().GetScheme()); err != nil {
				t.Fatalf("failed to install cluster types: %v", err)
			}

			// get gateway IP address
			gatewayIP := getGatewayIP(ctx, t, config, "observability-gateway", "observability-gateway-system")
			// get hostname of the gateway http route
			hostname := getHostname(ctx, t, config, "victoria-logs-otlp", "victoria-logs-system")

			// fetch clusters
			clusterList := &clustersv1alpha1.ClusterList{}
			clusterList.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "clusters.openmcp.cloud",
				Version: "v1alpha1",
				Kind:    "ClusterList",
			})
			if err := config.Client().Resources().List(ctx, clusterList); err != nil {
				t.Fatalf("failed to list clusters: %v", err)
			}

			// for each cluster with 'workload' in its purposes, create an AccessRequest to get a kubeconfig
			ars := map[string]*clustersv1alpha1.AccessRequest{}
			for _, cluster := range clusterList.Items {
				if !slices.Contains(cluster.Spec.Purposes, "workload") {
					continue
				}

				ar := &clustersv1alpha1.AccessRequest{
					Spec: clustersv1alpha1.AccessRequestSpec{
						ClusterRef: &commonapi.ObjectReference{
							Name:      cluster.Name,
							Namespace: cluster.Namespace,
						},
						Token: &clustersv1alpha1.TokenConfig{
							RoleRefs: []commonapi.RoleRef{
								{
									Name: "cluster-admin",
									Kind: "ClusterRole",
								},
							},
						},
					},
				}
				ar.SetName(cluster.Name + ".route")
				ar.SetNamespace(cluster.Namespace)

				if err := config.Client().Resources().Create(ctx, ar); err != nil {
					t.Errorf("failed to create AccessRequest for cluster %s: %v", cluster.Name, err)
				}

				ars[fmt.Sprintf("%s/%s", cluster.Namespace, cluster.Name)] = ar
				objectList = append(objectList, ar)
			}

			// wait for all AccessRequests to be granted
			if err := wait.For(func(ctx context.Context) (done bool, err error) {
				for _, ar := range ars {
					if getErr := config.Client().Resources().Get(ctx, ar.GetName(), ar.GetNamespace(), ar); getErr != nil {
						return false, getErr
					}

					if !ar.Status.IsGranted() {
						return false, fmt.Errorf("AccessRequest '%s/%s' is not granted yet", ar.Namespace, ar.Name)
					}
				}
				return true, nil
			}, wait.WithTimeout(1*time.Minute)); err != nil {
				t.Errorf("not all AccessRequests were granted: %v", err)
			}

			// access each workload cluster, inject the routing information into the coredns config and kill all coredns pods to restart them with the new configuration
			for clusterKey, ar := range ars {
				kubeconfigSecret := &corev1.Secret{}
				kubeconfigSecret.SetName(ar.Status.SecretRef.Name)
				kubeconfigSecret.SetNamespace(ar.GetNamespace())

				if err := config.Client().Resources().Get(ctx, kubeconfigSecret.GetName(), kubeconfigSecret.GetNamespace(), kubeconfigSecret); err != nil {
					t.Errorf("failed to get kubeconfig secret for AccessRequest '%s/%s': %v", ar.Namespace, ar.Name, err)
					continue
				}

				kubeconfigData, exists := kubeconfigSecret.Data[clustersv1alpha1.SecretKeyKubeconfig]
				if !exists {
					t.Errorf("kubeconfig secret for AccessRequest '%s/%s' does not contain kubeconfig data", ar.Namespace, ar.Name)
					continue
				}

				restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigData)
				if err != nil {
					t.Errorf("failed to create REST config from kubeconfig for AccessRequest '%s/%s': %v", ar.Namespace, ar.Name, err)
					continue
				}

				access := clusters.New(clusterKey).WithRESTConfig(restCfg)
				if err := access.InitializeClient(nil); err != nil {
					t.Errorf("failed to initialize client for cluster '%s': %v", clusterKey, err)
					continue
				}
				// In some environments (e.g. on Mac), docker uses a different network than the host machine, so the kubeconfigs from AccessRequest work only from within the kind clusters.
				// Therefore, we need to patch the cluster.
				access = localaccess.MustPatchClusterClient(ctx, ar, access)

				injectGatewayRoute(ctx, t, hostname, gatewayIP, access)
			}

			return ctx
		}).
		Assess("can read metrics from prometheus", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			localPort, stop, err := portForwardToPod(t, "prometheus-system", "prometheus-prometheus-0", 9090)
			if err != nil {
				t.Fatalf("failed to port-forward to Prometheus: %v", err)
			}
			defer stop()

			promAPI, err := newPrometheusAPI(localPort)
			if err != nil {
				t.Fatalf("failed to create Prometheus API client: %v", err)
			}

			assertMetricsAvailable(ctx, t, promAPI, []string{
				// metrics-operator custom metrics
				"co_kustomization",
				// metrics-operator custom federated metrics
				"co_controlplane",
				// controller-runtime workqueue metrics (scraped via annotation-based PodMonitor)
				"workqueue_depth",
				"controller_runtime_reconcile_errors_total",
			})

			return ctx
		}).
		Assess("can read logs from victoria logs", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			localPort, stop, err := portForwardToPod(t, "victoria-logs-system", "victoria-logs-0", 9428)
			if err != nil {
				t.Fatalf("failed to port-forward to Victoria Logs: %v", err)
			}
			defer stop()

			assertLogsAvailable(ctx, t, localPort)

			return ctx
		}).
		Assess("workload cluster logs are available via victoria logs", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			localPort, stop, err := portForwardToPod(t, "victoria-logs-system", "victoria-logs-0", 9428)
			if err != nil {
				t.Fatalf("failed to port-forward to Victoria Logs: %v", err)
			}
			defer stop()

			assertWorkloadLogsAvailable(ctx, t, localPort)

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			// Delete objects in reverse order to handle dependencies
			for i := len(objectList) - 1; i >= 0; i-- {
				obj := objectList[i]
				t.Logf("Deleting object [%v] %s/%s", obj.GetObjectKind().GroupVersionKind(), obj.GetNamespace(), obj.GetName())
				timeout := 5 * time.Minute
				if obj.GetObjectKind().GroupVersionKind().Group == "kro.run" {
					// leave more time for the ObservabilityStack and its parts to be deleted
					timeout = 10 * time.Minute
				}
				if err := resources.DeleteObject(ctx, config, obj, wait.WithTimeout(timeout)); err != nil {
					t.Errorf("failed to delete object: %v", err)
				}
			}

			for i := len(onboardingList.Items) - 1; i >= 0; i-- {
				obj := &onboardingList.Items[i]
				t.Logf("Deleting object on onboarding cluster [%v] %s/%s", obj.GetObjectKind().GroupVersionKind(), obj.GetNamespace(), obj.GetName())
				if err := resources.DeleteObject(ctx, onboardingConfig, obj, wait.WithTimeout(5*time.Minute)); err != nil {
					t.Errorf("failed to delete object on onboarding cluster: %v", err)
				}
			}

			return ctx
		})

	testenv.Test(t, obsStackTest.Feature())
}
