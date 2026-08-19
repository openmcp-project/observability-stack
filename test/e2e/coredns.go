package e2e

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// This regex matches a block from the coredns Corefile that looks like this:
//
//	kubernetes cluster.local in-addr.arpa ip6.arpa {
//	   pods insecure
//	   fallthrough in-addr.arpa ip6.arpa
//	   ttl 30
//	}
//
// Actually matched are only 'kubernetes cluster.local', followed by a '{' (with optional stuff in between), followed by multiple lines of content, followed by a line with only '}' (with optional whitespace).
// The capturing groups are:
// - whole: the entire matched block
// - space1: the whitespace before the "kubernetes" line
// - space2: the whitespace before the "pods" line
const corednsRegexString = `(?<whole>(?<space1>[^\S\n]*)kubernetes cluster\.local[^\{]*\{(?:\n(?<space2>\s*)\S[^\n]*)(?:\n\s*\S[^\n]*)*?\n\s*\})`

var corednsRegex = regexp.MustCompile(corednsRegexString)

// getHostname retrieves the first hostname defined in the HTTPRoute with the given name and namespace.
func getHostname(ctx context.Context, t *testing.T, config *envconf.Config, name, namespace string) string {
	t.Helper()

	httpRoute := &gatewayv1.HTTPRoute{}
	httpRoute.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})
	httpRoute.SetName(name)
	httpRoute.SetNamespace(namespace)
	if err := config.Client().Resources().Get(ctx, name, namespace, httpRoute); err != nil {
		t.Fatalf("failed to get HTTPRoute '%s/%s': %v", namespace, name, err)
	}
	if len(httpRoute.Spec.Hostnames) == 0 {
		t.Fatalf("HTTPRoute '%s/%s' does not have any hostnames defined", namespace, name)
	}
	return string(httpRoute.Spec.Hostnames[0])
}

// getGatewayIP retrieves the first IP address of the Gateway with the given name and namespace.
func getGatewayIP(ctx context.Context, t *testing.T, config *envconf.Config, name, namespace string) string {
	t.Helper()

	gateway := &gatewayv1.Gateway{}
	gateway.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	})
	gateway.SetName(name)
	gateway.SetNamespace(namespace)
	if err := config.Client().Resources().Get(ctx, name, namespace, gateway); err != nil {
		t.Fatalf("failed to get Gateway '%s/%s': %v", namespace, name, err)
	}
	for _, addr := range gateway.Status.Addresses {
		if addr.Type != nil && *addr.Type == gatewayv1.IPAddressType {
			return addr.Value
		}
	}
	t.Fatalf("Gateway '%s/%s' does not have any IP addresses exposed", namespace, name)
	return ""
}

// injectGatewayRoute modifies the coredns configmap to add a hosts entry for the given hostname and gateway IP, then restarts the coredns pods to apply the change.
func injectGatewayRoute(ctx context.Context, t *testing.T, hostname string, gatewayIP string, access *clusters.Cluster) {
	t.Helper()

	// fetch the coredns configmap
	cm := &corev1.ConfigMap{}
	cm.Name = "coredns"
	cm.Namespace = "kube-system"
	if err := access.Client().Get(ctx, client.ObjectKeyFromObject(cm), cm); err != nil {
		t.Fatalf("failed to get coredns configmap: %v", err)
	}
	coreData, exists := cm.Data["Corefile"]
	if !exists {
		t.Fatalf("coredns configmap does not contain Corefile")
	}

	match := corednsRegex.FindStringSubmatch(coreData)
	if match == nil {
		t.Fatalf("failed to find kubernetes block in coredns Corefile using regex: %s", corednsRegexString)
	}
	fullMatch := match[corednsRegex.SubexpIndex("whole")]
	outerIndent := match[corednsRegex.SubexpIndex("space1")]
	innerIndent := match[corednsRegex.SubexpIndex("space2")]

	replacement := fmt.Sprintf("%s\n%shosts {\n%s%s %s\n%sfallthrough\n%s}", fullMatch, outerIndent, innerIndent, gatewayIP, hostname, innerIndent, outerIndent)
	coreData = strings.Replace(coreData, fullMatch, replacement, 1)
	cm.Data["Corefile"] = coreData
	if err := access.Client().Update(ctx, cm); err != nil {
		t.Fatalf("failed to update coredns configmap: %v", err)
	}

	// kill the coredns pods to force them to reload the configmap
	if err := access.Client().DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace("kube-system"), client.MatchingLabels{"k8s-app": "kube-dns"}); err != nil {
		t.Fatalf("failed to delete coredns pods: %v", err)
	}

	// wait for the coredns pods to be ready again
	if err := waitFor(t, "CoreDNS pods Ready after configmap update", func(ctx context.Context) (done bool, err error) {
		podList := &corev1.PodList{}
		if err := access.Client().List(ctx, podList, client.InNamespace("kube-system"), client.MatchingLabels{"k8s-app": "kube-dns"}); err != nil {
			return false, fmt.Errorf("failed to list coredns pods: %v", err)
		}
		for _, pod := range podList.Items {
			if pod.Status.Phase != corev1.PodRunning {
				return false, nil
			}
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status != corev1.ConditionTrue {
					return false, nil
				}
			}
		}
		return true, nil
	}, wait.WithTimeout(2*time.Minute)); err != nil {
		t.Fatalf("%v", err)
	}
}
