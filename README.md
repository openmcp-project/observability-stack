[![REUSE status](https://api.reuse.software/badge/github.com/openmcp-project/observability-stack)](https://api.reuse.software/info/github.com/openmcp-project/observability-stack)

# observability-stack

## About this project

A comprehensive observability stack for openMCP deployments, providing monitoring, metrics collection, log aggregation, and distributed tracing capabilities.

The stack deploys the following components:

| Component | Purpose |
| --- | --- |
| **cert-manager** | TLS certificate lifecycle management |
| **metrics-operator** | openMCP metrics collection |
| **OpenTelemetry Operator** | Manages OTel Collector instances |
| **OTel Collector (Deployment)** | Scrapes Kubernetes metrics → Prometheus |
| **OTel Collector (DaemonSet)** | Collects pod stdout/stderr logs → Victoria Logs |
| **Prometheus Operator** | Manages Prometheus instances |
| **Prometheus** | Metrics storage and query engine |
| **Victoria Logs** | Log storage and query engine |
| **Observability Gateway** | Shared Envoy Gateway providing HTTPS + mTLS access to Prometheus UI, Victoria Logs UI, and OTLP log ingestion |

## Requirements and Setup

### Prerequisites

Before setting up the observability stack, ensure you have the following:

- A Kubernetes cluster (v1.27+)
- `kubectl` configured to access your cluster
- `OCM Kubernetes Controllers` installed in your cluster (<https://github.com/open-component-model/open-component-model/tree/main/kubernetes/controller>)
- `Flux CD` installed in your cluster (<https://fluxcd.io/>)
- `kro` (Kubernetes Resource Orchestrator) installed in your cluster (<https://kro.run>)
- Access to GitHub Container Registry (ghcr.io)

For a local setup, you can use the `local-dev` script in [cluster-provider-kind](https://github.com/openmcp-project/cluster-provider-kind/blob/main/README.md#%E2%80%8D-development).

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
          - identities:
              - type: OCIRegistry
                hostname: ghcr.io
                path: openmcp-project/*
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
  victoriaLogs:
    namespace: victoria-logs-system
  observabilityGateway:
    namespace: observability-gateway-system
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
kubectl get pods -n victoria-logs-system
kubectl get pods -n observability-gateway-system

# Verify the log collector DaemonSet is running on all nodes
kubectl get daemonset -n open-telemetry-collector-system
```

#### 6. Access Observability Dashboards

Both Prometheus and Victoria Logs are exposed through a single shared Envoy Gateway in the `observability-gateway-system` namespace. The gateway uses HTTPS with mTLS client certificate authentication.

**Hostname Pattern:**

| Endpoint | URL | Purpose |
| --- | --- | --- |
| Prometheus UI | `https://metrics.<gateway-namespace>.<base-domain>:<port>` | Metrics query and dashboards |
| Victoria Logs UI | `https://logs.<gateway-namespace>.<base-domain>:<port>` | Log query and UI |
| OTLP log ingestion | `https://otlp-logs.<gateway-namespace>.<base-domain>:<port>` | Remote log ingestion (external clusters) |

The `<base-domain>` is derived from the openMCP Gateway's `dns.openmcp.cloud/base-domain` annotation. With the default configuration (`observabilityGateway.namespace: observability-gateway-system`), the hostnames look like:

- `metrics.observability-gateway-system.<base-domain>:8443`
- `logs.observability-gateway-system.<base-domain>:8443`
- `otlp-logs.observability-gateway-system.<base-domain>:8443`

**Get the Dashboard URLs:**

```bash
# Get the Prometheus hostname
kubectl get httproute prometheus -n prometheus-system -o jsonpath='{.spec.hostnames[0]}'

# Get the Victoria Logs hostname
kubectl get httproute victoria-logs -n victoria-logs-system -o jsonpath='{.spec.hostnames[0]}'

# Get the OTLP ingestion hostname
kubectl get httproute victoria-logs-otlp -n victoria-logs-system -o jsonpath='{.spec.hostnames[0]}'
```

**Extract mTLS Client Certificates:**

A single client certificate (`observability-client-cert`) is generated in the gateway namespace and can be used to authenticate against both Prometheus and Victoria Logs:

```bash
# Create a directory for the certificates
mkdir -p obs-certs
cd obs-certs

# Extract the client certificate (for mTLS authentication)
kubectl get secret observability-client-cert -n observability-gateway-system \
  -o jsonpath='{.data.tls\.crt}' | base64 -d > client.crt

# Extract the client private key
kubectl get secret observability-client-cert -n observability-gateway-system \
  -o jsonpath='{.data.tls\.key}' | base64 -d > client.key
```

**Use the Certificates with curl:**

```bash
export METRICS_HOST=$(kubectl get httproute prometheus -n prometheus-system -o jsonpath='{.spec.hostnames[0]}')
export LOGS_HOST=$(kubectl get httproute victoria-logs -n victoria-logs-system -o jsonpath='{.spec.hostnames[0]}')

# Query Prometheus (skip server cert verification — gateway uses a self-signed cert)
curl --cert client.crt --key client.key --insecure \
  "https://${METRICS_HOST}:8443/api/v1/query?query=up"

# Query Victoria Logs
curl --cert client.crt --key client.key --insecure \
  "https://${LOGS_HOST}:8443/select/logsql/query?query=*&limit=10"
```

**Use the Certificates with your Browser:**

1. Combine the client certificate and key into a PKCS#12 file:

   ```bash
   openssl pkcs12 -export -out observability-client.p12 \
     -inkey client.key \
     -in client.crt \
     -password pass:observability
   ```

2. Import the `observability-client.p12` file into your browser:
   - **Chrome/Edge**: Settings → Privacy and security → Security → Manage certificates → Your certificates → Import
   - **Firefox**: Settings → Privacy & Security → Certificates → View Certificates → Your Certificates → Import
   - **Safari**: Open Keychain Access → File → Import Items

3. When prompted for the client certificate password, use: `observability` (or the password you set in step 1)

4. Navigate to the dashboard URLs, accept the browser warning about the self-signed server certificate, and select the client certificate when prompted

#### 7. Configure Alerting (per landscape)

The observability stack deploys an Alertmanager instance as part of the base infrastructure, but the routing configuration — which notification channels receive alerts — must be provided separately for each landscape. This keeps credentials and routing decisions out of the shared stack.

The Prometheus Operator automatically loads a `Secret` named `alertmanager-alertmanager` from the Prometheus namespace. Create this Secret with your credentials and apply it:

```bash
kubectl apply -f alertmanager-config.yaml -n prometheus-system
```

Example `alertmanager-config.yaml` configuring a single receiver that sends every alert to both **Slack** and **VictorOps** simultaneously:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: alertmanager-alertmanager
  namespace: prometheus-system  # adjust if your Prometheus namespace differs
type: Opaque
stringData:
  alertmanager.yaml: |
    global:
      resolve_timeout: 5m
    route:
      group_by: ['alertname', 'severity', 'namespace']
      group_wait: 30s
      group_interval: 5m
      repeat_interval: 12h
      receiver: 'all'
      routes:
        - receiver: 'null'
          match:
            inhibit_alert: "true"
    receivers:
      - name: 'null'
      - name: 'all'
        slack_configs:
          - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
            channel: '#your-alerts-channel'
            send_resolved: true
            title: |-
              [{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}] {{ .CommonLabels.alertname }}
            text: >-
              {{ range .Alerts }}
              {{ .Annotations.description }}
              *Severity:* `{{ .Labels.severity }}`
              {{ end }}
        victorops_configs:
          - api_key: 'YOUR_VICTOROPS_API_KEY'
            routing_key: 'YOUR_VICTOROPS_ROUTING_KEY'
            send_resolved: true
    inhibit_rules:
      - source_match:
          severity: 'critical'
        target_match:
          severity: 'warning'
        equal: ['alertname', 'namespace']
```

| Field | Description |
| --- | --- |
| `slack_configs[].api_url` | Slack Incoming Webhook URL |
| `slack_configs[].channel` | Target Slack channel (e.g. `#alerts`) |
| `victorops_configs[].api_key` | VictorOps API key |
| `victorops_configs[].routing_key` | VictorOps routing key (maps to an escalation policy) |

If you only need one notification channel, remove the unused `_configs` block.
See <https://prometheus.io/docs/alerting/latest/configuration/> for prometheus alerting configuration.

> **Note:** Some alert rules in this stack carry the label `inhibit_alert: "true"` (currently the onboarding rules and controller-runtime rules). These alerts are intentionally visible in the Prometheus dashboard for status observability, but are suppressed in Alertmanager via the `null` receiver route above. The base route structure including the `null` receiver and its matcher **must** be present in your Alertmanager config for suppression to work.

**Verify Alertmanager is connected:**

```bash
# Check Alertmanager pod is running
kubectl get pods -n prometheus-system -l app.kubernetes.io/name=alertmanager

# Check Prometheus has discovered the Alertmanager (look for "1 active" under Alertmanagers)
kubectl get prometheus.monitoring.coreos.com prometheus -n prometheus-system -o jsonpath='{.status}'
```

The Prometheus dashboard also shows the connected Alertmanager count under **Status → Runtime & Build Info**.

#### 8. Log Collection and Cross-Cluster Ingestion

Pod logs (stdout/stderr from all containers on every node) are automatically collected by an OpenTelemetry Collector DaemonSet running in `open-telemetry-collector-system`. It reads from `/var/log/pods` on each node and ships logs to Victoria Logs via OTLP HTTP.

**Verify log ingestion:**

```bash
# Check the DaemonSet is running on all nodes
kubectl get daemonset logs -n open-telemetry-collector-system

# Port-forward to Victoria Logs and query recent logs
kubectl port-forward -n victoria-logs-system svc/victoria-logs 9428:9428 &

# Query any log from the last 15 minutes
curl "http://localhost:9428/select/logsql/query?query=*&limit=5&start=now-15m"
```

**Access the Victoria Logs UI:**

Once the port-forward is established (or using the HTTPS endpoint via the Observability Gateway), open the UI:

```
https://logs.<gateway-namespace>.<base-domain>:8443/select/vmui/
```

The UI provides a log query interface using [LogsQL](https://docs.victoriametrics.com/victorialogs/logsql/).

**Ingest logs from external Kubernetes clusters:**

The OTLP log ingestion endpoint (`otlp-logs.<gateway-ns>.<base-domain>:8443/insert/opentelemetry`) accepts logs from any OpenTelemetry Collector instance that presents a valid mTLS client certificate. To ship logs from another cluster:

1. Extract the client certificate from the central cluster:

   ```bash
   # On the central cluster
   kubectl get secret observability-client-cert -n observability-gateway-system \
     -o jsonpath='{.data.tls\.crt}' | base64 -d > client.crt
   kubectl get secret observability-client-cert -n observability-gateway-system \
     -o jsonpath='{.data.tls\.key}' | base64 -d > client.key
   kubectl get secret otlp-logs-cert -n observability-gateway-system \
     -o jsonpath='{.data.tls\.crt}' | base64 -d > otlp-logs-server.crt
   ```

2. Create a secret in the remote cluster's OTel Collector namespace:

   ```bash
   # On the remote cluster
   kubectl create secret generic observability-client-cert \
     --from-file=tls.crt=client.crt \
     --from-file=tls.key=client.key \
     --from-file=ca.crt=otlp-logs-server.crt \
     -n open-telemetry-collector-system
   ```

3. Configure the OTel Collector on the remote cluster to export logs via OTLP HTTP with mTLS:

   ```yaml
   config: |
     exporters:
       otlphttp/logs:
         endpoint: "https://otlp-logs.<gateway-namespace>.<base-domain>:8443/insert/opentelemetry"
         tls:
           cert_file: /etc/otel/certs/tls.crt
           key_file: /etc/otel/certs/tls.key
           ca_file: /etc/otel/certs/ca.crt
     service:
       pipelines:
         logs:
           exporters: [otlphttp/logs]
   volumeMounts:
   - name: client-certs
     mountPath: /etc/otel/certs
     readOnly: true
   volumes:
   - name: client-certs
     secret:
       secretName: observability-client-cert
   ```

### Configuration Options

The `ObservabilityStack` custom resource supports the following configuration options:

- **componentRef**: Reference to the OCM component containing all stack resources
- **imagePullSecretRef**: Secret for pulling container images
- **certManager**: Configuration for cert-manager (namespace)
- **metricsOperator**: Configuration for the metrics-operator (namespace)
- **metrics**: Configuration for openMCP metrics collection (namespace)
- **openTelemetryOperator**: Configuration for the OpenTelemetry Operator (namespace)
- **openTelemetryCollector**: Configuration for the OTel Collector Deployment that scrapes metrics (namespace)
- **prometheusOperator**: Configuration for the Prometheus Operator (namespace)
- **prometheus**: Configuration for the Prometheus instance (namespace)
- **victoriaLogs**: Configuration for the Victoria Logs instance (namespace)
- **observabilityGateway**: Configuration for the shared Envoy Gateway (namespace, port)

The `observabilityGateway.namespace` is used as the subdomain component for all three external hostnames:
`metrics.<namespace>.<base-domain>`, `logs.<namespace>.<base-domain>`, and `otlp-logs.<namespace>.<base-domain>`.

Adjust the namespace and configuration values in the `ObservabilityStack` resource according to your requirements.

## Contributing

For general contribution guidelines, see the [organization-wide CONTRIBUTING.md](https://github.com/openmcp-project/.github/blob/main/CONTRIBUTING.md).

## Security

If you find a security vulnerability, please follow our [security policy](https://github.com/openmcp-project/.github/blob/main/SECURITY.md). Do not create public GitHub issues for security concerns.

## Code of Conduct

All community members must abide by our [Code of Conduct](https://github.com/openmcp-project/.github/blob/main/CODE_OF_CONDUCT.md).

## Licensing

Copyright OpenControlPlane contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/openmcp-project/observability-stack).

---

<p align="center">
  <a href="https://apeirora.eu/content/projects/">
    <img alt="BMWK-EU funding logo" src="https://apeirora.eu/assets/img/BMWK-EU.png" width="300"/>
  </a>
</p>

<p align="center">
  OpenControlPlane is part of <a href="https://apeirora.eu/content/projects/">ApeiroRA</a>, an EU Important Project of Common European Interest (IPCEI-CIS).
</p>

<p align="center">
  Copyright Linux Foundation Europe. For web site terms of use, trademark policy and other project policies please see <a href="https://linuxfoundation.eu/en/policies">https://linuxfoundation.eu/en/policies</a>.
</p>
