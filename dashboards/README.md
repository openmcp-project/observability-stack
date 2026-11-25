# Dashboards

The `dashboards` module contains a set of [Dashboard-as-Code](https://perses.dev/docs/perses/latest/concepts/dashboard-as-code/) definitions for OpenControlPlane.

## Configuration

The following environment variables allow customizing the dashboard build:

- OCP_PROMETHEUS_DATASOURCE_NAME (default: "prometheus")
- OCP_PROMETHEUS_ENDPOINT (default: "http://prometheus.monitoring:9090")
- OCP_PROJECT_NAME (default: "opencontrolplane")

## Observability Stack Integration

See [CI/CD setup](https://perses.dev/docs/perses/latest/dac/getting-started/#cicd-setup) for integration with [percli](https://perses.dev/docs/perses/latest/cli/) and GitHub Actions.

## Developer Workflow

Prerequisites:

- Running [Perses server](https://perses.dev/docs/perses/latest/getting-started/#install-perses), e.g. running in [kind](https://kind.sigs.k8s.io/).
- [Perses CLI](https://perses.dev/docs/perses/latest/cli/)

0. (Login to server with e.g. `percli login http://localhost:8080`)
1. Build dashboards with `percli dac build -f main.go`
2. Deploy dashboards with `percli apply -f built/main_output.yaml`

Note that only yaml output is supported (see [main.go](main.go) that adds the yaml separator to the output file).

## Current Assumptions

The health panel of the service provider dashboard expects a metric named `service_provider_status` based on [metrics-operator](https://github.com/openmcp-project/metrics-operator) `Metrics` resources provided by each service provider, e.g.:

```yaml
apiVersion: metrics.open-control-plane.io/v1alpha1
kind: Metric
metadata:
  name: service-provider-crossplane
spec:
  name: service-provider-status
  description: "crossplane resources status"
  target:
    kind: Crossplane
    group: crossplane.services.open-control-plane.io
    version: v1alpha1
  interval: "1m"
  projections:
    - name: phase
      fieldPath: "status.phase"
      default: Unknown
---
apiVersion: metrics.open-control-plane.io/v1alpha1
kind: Metric
metadata:
  name: service-provider-flux
spec:
  name: service-provider-status
  description: "flux resources status"
  target:
    kind: Flux
    group: flux.services.open-control-plane.io
    version: v1alpha1
  interval: "1m"
  projections:
    - name: phase
      fieldPath: "status.phase"
      default: Unknown
```
