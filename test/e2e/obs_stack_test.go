package e2e

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	ocmv1alpha1 "ocm.software/open-component-model/kubernetes/controller/api/v1alpha1"

	"github.com/openmcp-project/openmcp-testing/pkg/conditions"
	"github.com/openmcp-project/openmcp-testing/pkg/resources"
)

const (
	obsStackNamespace = "obs-stack"
)

func TestObsStack(t *testing.T) {
	var objectList []k8s.Object
	obsStackTest := features.New("observability-stack test").
		Setup(func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			// create the obs-stack namespace
			namespace := &corev1.Namespace{}
			namespace.SetName(obsStackNamespace)

			if err := c.Client().Resources().Create(ctx, namespace); err != nil {
				t.Errorf("failed to create namespace: %v", err)
			}

			objectList = append(objectList, namespace)

			// Process and apply onboarding YAML files with version injection (in-memory)
			files, _ := filepath.Glob("platform/*.yaml")
			for _, file := range files {
				content, err := processYAMLFile(file)
				if err != nil {
					t.Fatalf("failed to process %s: %v", file, err)
				}

				// Decode and create objects from the processed YAML content
				objs, err := decoder.DecodeAll(ctx, bytes.NewReader([]byte(content)))
				if err != nil {
					t.Fatalf("failed to decode %s: %v", file, err)
				}

				for _, obj := range objs {
					if err := c.Client().Resources().Create(ctx, obj); err != nil {
						t.Errorf("failed to create object from %s: %v", file, err)
					}
					objectList = append(objectList, obj)
				}
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
			component.SetNamespace(obsStackNamespace)

			deployer := &ocmv1alpha1.Deployer{}
			deployer.SetName("resource-graph-definition")
			deployer.SetName(obsStackNamespace)

			if err := wait.For(conditions.Match(repository, config, "Ready", corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("Repository not ready: %v", err)
			}

			if err := wait.For(conditions.Match(component, config, "Ready", corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("Component not ready: %v", err)
			}

			if err := wait.For(conditions.Match(resource, config, "Ready", corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("Resource not ready: %v", err)
			}

			if err := wait.For(conditions.Match(repository, config, "Ready", corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("Repository not ready: %v", err)
			}

			if err := wait.For(conditions.Match(deployer, config, "Ready", corev1.ConditionTrue), wait.WithTimeout(2*time.Minute)); err != nil {
				t.Errorf("Deployer not ready: %v", err)
			}

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			for _, obj := range objectList {
				if err := resources.DeleteObject(ctx, config, obj, wait.WithTimeout(time.Minute)); err != nil {
					t.Errorf("failed to delete object: %v", err)
				}
			}

			return ctx
		})

	testenv.Test(t, obsStackTest.Feature())
}
