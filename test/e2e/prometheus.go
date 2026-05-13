package e2e

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	prometheusapi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/e2e-framework/klient/wait"
)

// portForwardToPod opens a port-forward tunnel to a pod and returns the local port and a stop function.
// Kubeconfig is loaded via the KUBECONFIG env var or ~/.kube/config, matching kubectl and the
// e2e-framework Bootstrap behaviour (which always sets KUBECONFIG when creating a kind cluster).
func portForwardToPod(t *testing.T, namespace, podName string, remotePort int) (int, func(), error) {
	t.Helper()

	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to find free local port: %w", err)
	}
	localPort := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	roundTripper, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create SPDY round tripper: %w", err)
	}

	serverURL, err := url.Parse(fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/portforward",
		restConfig.Host, namespace, podName))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse port-forward URL: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, serverURL)

	stopCh := make(chan struct{})
	readyCh := make(chan struct{})

	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, remotePort)}, stopCh, readyCh, os.Stdout, os.Stderr)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create port-forwarder: %w", err)
	}

	go func() {
		if fwErr := forwarder.ForwardPorts(); fwErr != nil {
			t.Logf("port-forward error: %v", fwErr)
		}
	}()

	select {
	case <-readyCh:
	case <-time.After(30 * time.Second):
		close(stopCh)
		return 0, nil, fmt.Errorf("timed out waiting for port-forward to become ready")
	}

	// Wait until the application is actually accepting connections on the forwarded port.
	if err := wait.For(func(context.Context) (bool, error) {
		conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", localPort), time.Second)
		if dialErr != nil {
			return false, nil
		}
		conn.Close()
		return true, nil
	}, wait.WithTimeout(30*time.Second)); err != nil {
		close(stopCh)
		return 0, nil, fmt.Errorf("timed out waiting for port %d to accept connections: %w", localPort, err)
	}

	return localPort, func() { close(stopCh) }, nil
}

// newPrometheusAPI creates a Prometheus v1 API client pointing at the given local port.
func newPrometheusAPI(localPort int) (prometheusv1.API, error) {
	client, err := prometheusapi.NewClient(prometheusapi.Config{
		Address: fmt.Sprintf("http://localhost:%d", localPort),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus HTTP client: %w", err)
	}
	return prometheusv1.NewAPI(client), nil
}

// assertMetricsAvailable waits until every metric name in the list has at least one
// sample in Prometheus, failing the test if any metric is missing after the timeout.
func assertMetricsAvailable(ctx context.Context, t *testing.T, promAPI prometheusv1.API, metrics []string) {
	t.Helper()
	for _, metricName := range metrics {
		name := metricName
		if err := wait.For(func(ctx context.Context) (bool, error) {
			result, _, err := promAPI.Query(ctx, name, time.Now())
			if err != nil {
				return false, nil // treat as not-ready, will retry
			}
			vector, ok := result.(model.Vector)
			return ok && len(vector) > 0, nil
		}, wait.WithTimeout(5*time.Minute), wait.WithContext(ctx)); err != nil {
			t.Errorf("metric %q not found in Prometheus after timeout: %v", name, err)
		}
	}
}
