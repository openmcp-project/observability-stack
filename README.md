[![REUSE status](https://api.reuse.software/badge/github.com/openmcp-project/observability-stack)](https://api.reuse.software/info/github.com/openmcp-project/observability-stack)

# observability-stack

## About this project

A comprehensive observability stack for openMCP deployments, providing monitoring, metrics collection, and distributed tracing capabilities.

## Requirements and Setup

### Prerequisites

Before setting up the observability stack, ensure you have the following:

- A Kubernetes cluster (v1.27+)
- `kubectl` configured to access your cluster
- `OCM Kubernetes Controllers` installed in your cluster (<https://github.com/open-component-model/open-component-model/tree/main/kubernetes/controller>)
- `kro` (Kubernetes Resource Orchestrator) installed in your cluster (<https://kro.run>)
- Access to GitHub Container Registry (ghcr.io)

### Installation Steps

#### 1. Create the Target Namespace

```bash
kubectl create namespace obs-stack
```

#### 2. Configure OCM Registry Credentials

Create an OCM configuration file for accessing the component registry:

```bash
# Create the OCM config with your GitHub credentials
cat <<EOF > .ocmconfig
type: generic.config.ocm.software/v1
configurations:
  - type: credentials.config.ocm.software
    consumers:
      - identity:
          type: OCIRepository
          scheme: https
          hostname: ghcr.io
        credentials:
          - type: Credentials
            properties:
              username: <your-github-username>
              password: <your-github-token>
      - identity:
          type: OCIRepository
          hostname: ghcr.io
        credentials:
          - type: Credentials
            properties:
              username: <your-github-username>
              password: <your-github-token>
EOF
```

Create a Kubernetes secret from this configuration:

```bash
kubectl create secret generic ocm-config \
  --from-file=.ocmconfig \
  --namespace=obs-stack
```

#### 3. Create Image Pull Secret

Create a secret for pulling container images from the registry:

```bash
kubectl create secret docker-registry regcred \
  --docker-server=ghcr.io \
  --docker-username=<your-github-username> \
  --docker-password=<your-github-token> \
  --namespace=obs-stack
```

#### 4. Deploy the Observability Stack

Apply the deployment manifests:

```bash
kubectl apply -f - <<EOF
apiVersion: delivery.ocm.software/v1alpha1
kind: Repository
metadata:
  name: obs-stack-repository
  namespace: obs-stack
spec:
  repositorySpec:
    baseUrl: ghcr.io/openmcp-project/components
    type: OCIRegistry
  interval: 1m
  ocmConfig:
    - kind: Secret
      name: ocm-config
---
apiVersion: delivery.ocm.software/v1alpha1
kind: Component
metadata:
  name: obs-stack-component
  namespace: obs-stack
spec:
  component: github.com/openmcp-project/observability-stack
  repositoryRef:
    name: obs-stack-repository
  semver: ">=0.0.1"
  interval: 1m
---
apiVersion: delivery.ocm.software/v1alpha1
kind: Resource
metadata:
  name: resource-graph-definition
  namespace: obs-stack
spec:
  componentRef:
    name: obs-stack-component
  resource:
    byReference:
      resource:
        name: resource-graph-definition
  interval: 1m
---
apiVersion: delivery.ocm.software/v1alpha1
kind: Deployer
metadata:
  name: resource-graph-definition
spec:
  resourceRef:
    name: resource-graph-definition
    namespace: obs-stack
---
apiVersion: kro.run/v1alpha1
kind: ObservabilityStack
metadata:
  name: stack
  namespace: obs-stack
spec:
  componentRef:
    name: obs-stack-component
  imagePullSecretRef:
    name: regcred
    namespace: obs-stack
  certManager:
    namespace: cert-manager-system
  metricsOperator:
    namespace: metrics-operator-system
  metrics:
    namespace: metrics-operator-system
  openTelemetryOperator:
    namespace: open-telemetry-operator-system
  openTelemetryCollector:
    namespace: open-telemetry-collector-system
  prometheusOperator:
    namespace: prometheus-operator-system
  prometheus:
    namespace: prometheus-system
    dashboard:
      port: 8443
EOF
```

#### 5. Verify the Deployment

Monitor the deployment progress:

```bash
# Check component status
kubectl get component -n obs-stack

# Check resource status
kubectl get resource -n obs-stack

# Check deployer status
kubectl get deployer -n obs-stack

# Check the observability stack instance
kubectl get observabilitystack -n obs-stack

# Verify all components are running
kubectl get pods -n cert-manager-system
kubectl get pods -n metrics-operator-system
kubectl get pods -n open-telemetry-operator-system
kubectl get pods -n open-telemetry-collector-system
kubectl get pods -n prometheus-operator-system
kubectl get pods -n prometheus-system
```

#### 6. Access Prometheus Dashboard

The Prometheus deployment automatically creates a Gateway and HTTPRoute for external access. The dashboard is accessible via HTTPS using a dynamically generated hostname based on the openMCP Gateway configuration.

**Get the Dashboard URL:**

```bash
# Get the hostname from the HTTPRoute
kubectl get httproute prometheus -n prometheus-system -o jsonpath='{.spec.hostnames[0]}'
```

The hostname follows the pattern: `prometheus.<namespace>.<base-domain>` where the base domain is derived from the openMCP Gateway's `dns.openmcp.cloud/base-domain` annotation.

**Access the Dashboard:**

```bash
# Get the complete URL
export HOSTNAME=$(kubectl get httproute prometheus -n prometheus-system -o jsonpath='{.spec.hostnames[0]}')
echo "Prometheus Dashboard: https://${HOSTNAME}:8443"
```

Open the URL in your browser. The dashboard uses:

- **HTTPS** with TLS termination at the Gateway
- **mTLS** (mutual TLS) with client certificate validation
- **Port** configured in the ObservabilityStack spec (default: 8443)

**Extract mTLS Client Certificates:**

To authenticate with the Prometheus dashboard, you need to extract the client certificates that are automatically generated during deployment:

```bash
# Create a directory for the certificates
mkdir -p prometheus-certs
cd prometheus-certs

# Extract the client certificate (for mTLS authentication)
kubectl get secret prometheus-client-cert -n prometheus-system -o jsonpath='{.data.tls\.crt}' | base64 -d > client.crt

# Extract the client private key
kubectl get secret prometheus-client-cert -n prometheus-system -o jsonpath='{.data.tls\.key}' | base64 -d > client.key

# Extract the server certificate (for verifying the gateway's identity)
kubectl get secret prometheus-cert -n prometheus-system -o jsonpath='{.data.tls\.crt}' | base64 -d > server.crt
```

**Use the Certificates with curl:**

```bash
# Using the server certificate for verification
curl --cert client.crt --key client.key --cacert server.crt "https://${HOSTNAME}:8443/api/v1/query?query=up"

# Or skip certificate verification (not recommended for production)
curl --cert client.crt --key client.key --insecure "https://${HOSTNAME}:8443/api/v1/query?query=up"
```

**Use the Certificates with your Browser:**

1. Combine the client certificate and key into a PKCS#12 file:

   ```bash
   openssl pkcs12 -export -out prometheus-client.p12 \
     -inkey client.key \
     -in client.crt \
     -password pass:prometheus
   ```

2. Import the `prometheus-client.p12` file into your browser:
   - **Chrome/Edge**: Settings → Privacy and security → Security → Manage certificates → Your certificates → Import
   - **Firefox**: Settings → Privacy & Security → Certificates → View Certificates → Your Certificates → Import
   - **Safari**: Open Keychain Access → File → Import Items

3. Import the server certificate as a trusted CA (to avoid browser warnings about self-signed certificates):
   - **Chrome/Edge**: Settings → Privacy and security → Security → Manage certificates → Authorities → Import `server.crt`
   - **Firefox**: Settings → Privacy & Security → Certificates → View Certificates → Authorities → Import `server.crt`
   - **Safari**: Open Keychain Access → File → Import Items (select `server.crt`), then double-click the certificate and set "Always Trust"

4. When prompted for the client certificate password, use: `prometheus` (or the password you set in step 1)

5. Navigate to the Prometheus dashboard URL and select the client certificate when prompted

### Configuration Options

The `ObservabilityStack` custom resource supports various configuration options:

- **componentRef**: Reference to the OCM component containing all stack resources
- **imagePullSecretRef**: Secret for pulling container images
- **certManager**: Configuration for cert-manager deployment
- **metricsOperator**: Configuration for metrics-operator deployment
- **metrics**: Configuration for metrics collection
- **openTelemetryOperator**: Configuration for OpenTelemetry operator
- **openTelemetryCollector**: Configuration for OpenTelemetry collector
- **prometheusOperator**: Configuration for Prometheus operator
- **prometheus**: Configuration for Prometheus, including dashboard port

Adjust the namespace and configuration values in the `ObservabilityStack` resource according to your requirements.

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/openmcp-project/observability-stack/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/openmcp-project/observability-stack/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2026 SAP SE or an SAP affiliate company and observability-stack contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/openmcp-project/observability-stack).
