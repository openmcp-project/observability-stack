package e2e

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	krov1alpha1 "github.com/kubernetes-sigs/kro/api/v1alpha1"
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

			// Process and apply the platform ocm manifests
			objs, err := applyYAMLFilesFromGlob(ctx, t, c, "platform/ocm/*.yaml")
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
			}, wait.WithTimeout(5*time.Minute)); err != nil {
				t.Errorf("ObservabilityStack not ready: %v", err)
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
				"co_managedcontrolplanev2",
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
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			// Delete objects in reverse order to handle dependencies
			for i := len(objectList) - 1; i >= 0; i-- {
				if err := resources.DeleteObject(ctx, config, objectList[i], wait.WithTimeout(5*time.Minute)); err != nil {
					t.Errorf("failed to delete object: %v", err)
				}
			}

			for i := len(onboardingList.Items) - 1; i >= 0; i-- {
				if err := resources.DeleteObject(ctx, onboardingConfig, &onboardingList.Items[i], wait.WithTimeout(5*time.Minute)); err != nil {
					t.Errorf("failed to delete object on onboarding cluster: %v", err)
				}
			}

			return ctx
		})

	testenv.Test(t, obsStackTest.Feature())
}
