// Package registry provides a local pull-through OCI registry that caches
// image pulls from a remote registry. It is intended to speed up e2e tests
// by avoiding repeated pulls of unchanged images into kind clusters.
//
// The registry runs as a Docker container on the host and is reachable from
// kind cluster containerd nodes via the host's docker bridge IP.
package registry

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

const (
	// ContainerName is the docker container name for the local registry.
	ContainerName = "openmcp-e2e-registry"
	// DefaultPort is the port the registry listens on on the host.
	DefaultPort = "5555"
)

// MirrorConfig holds the endpoint information callers need to configure
// containerd mirrors.
type MirrorConfig struct {
	// Endpoint is the mirror URL reachable from the kind node (e.g. http://172.17.0.1:5555).
	Endpoint string
	// ContainerEndpoint is the mirror URL reachable from the host (e.g. http://localhost:5555).
	ContainerEndpoint string
}

// StartPullThroughCache starts a local pull-through registry container that
// proxies pulls from remoteRegistry (e.g. "https://ghcr.io"). It returns the
// MirrorConfig and a stop function.
//
// The registry data is stored in a Docker volume named <ContainerName>-data so
// that it persists across test runs on the same host (or GH Actions runner with
// a warm cache). If the container is already running it is reused.
func StartPullThroughCache(ctx context.Context, remoteRegistry string) (*MirrorConfig, func(), error) {
	port := DefaultPort
	if p := os.Getenv("E2E_REGISTRY_PORT"); p != "" {
		port = p
	}

	// Idempotent: stop+remove stale containers with same name but different config.
	if err := ensureContainerStopped(ContainerName); err != nil {
		return nil, nil, fmt.Errorf("cleanup old registry: %w", err)
	}

	volumeName := ContainerName + "-data"
	cmd := exec.CommandContext(ctx, "docker", "run", "-d",
		"--name", ContainerName,
		"--restart", "unless-stopped",
		"-p", port+":5000",
		"-v", volumeName+":/var/lib/registry",
		"-e", "REGISTRY_PROXY_REMOTEURL="+remoteRegistry,
		"-e", "REGISTRY_STORAGE_DELETE_ENABLED=true",
		"registry:2",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Already running is fine — reuse it.
		if !strings.Contains(string(out), "already in use") {
			return nil, nil, fmt.Errorf("start registry container: %w\n%s", err, out)
		}
		klog.Infof("Reusing existing registry container %s", ContainerName)
	} else {
		klog.Infof("Started registry container %s on port %s -> %s", ContainerName, port, remoteRegistry)
	}

	// Discover the docker bridge IP so kind nodes (which are docker containers
	// themselves) can reach the registry on the host network.
	bridgeIP, err := dockerBridgeIP()
	if err != nil {
		return nil, nil, fmt.Errorf("detect docker bridge IP: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", bridgeIP, port)
	containerEndpoint := fmt.Sprintf("http://localhost:%s", port)

	if err := waitForRegistry(containerEndpoint, 30*time.Second); err != nil {
		return nil, nil, fmt.Errorf("registry did not become ready: %w", err)
	}

	stop := func() {
		// We intentionally do NOT remove the container or volume on stop so the
		// cache survives across test runs. On CI the runner is ephemeral anyway.
		klog.Infof("Registry container %s left running (cache preserved)", ContainerName)
	}

	return &MirrorConfig{
		Endpoint:          endpoint,
		ContainerEndpoint: containerEndpoint,
	}, stop, nil
}

// ensureContainerStopped removes a container if it exists but is not running
// (e.g. exited). Running containers are left as-is (reused).
func ensureContainerStopped(name string) error {
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", name).Output()
	if err != nil {
		// Container doesn't exist — nothing to do.
		return nil
	}
	status := strings.TrimSpace(string(out))
	if status == "running" {
		return nil // reuse
	}
	// Exited / dead / paused — remove so we can recreate with fresh config.
	return exec.Command("docker", "rm", "-f", name).Run()
}

// dockerBridgeIP returns the IP address of the docker0 bridge, which is
// reachable from inside kind nodes (they are docker containers on the same
// bridge network).
func dockerBridgeIP() (string, error) {
	// Ask docker for the bridge gateway IP.
	out, err := exec.Command("docker", "network", "inspect", "bridge",
		"--format", "{{range .IPAM.Config}}{{.Gateway}}{{end}}").Output()
	if err == nil {
		ip := strings.TrimSpace(string(out))
		if net.ParseIP(ip) != nil {
			return ip, nil
		}
	}
	// Fallback: use the well-known docker0 default.
	return "172.17.0.1", nil
}

// waitForRegistry polls the registry's /v2/ endpoint until it responds 200
// or the timeout expires.
func waitForRegistry(endpoint string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(endpoint + "/v2/")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("registry at %s not ready after %s", endpoint, timeout)
}
