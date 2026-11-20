# Dashboards

The `dashboards` module contains a set of [Dashboard-as-Code](https://perses.dev/docs/perses/latest/concepts/dashboard-as-code/) definitions for OpenControlPlane.

## Observability Stack Integration

See [CI/CD setup](https://perses.dev/docs/perses/latest/dac/getting-started/#cicd-setup) for integration with [percli](https://perses.dev/docs/perses/latest/cli/) and GitHub Actions.

## Developer Worklflow

Prerequisites:

- Running [Perses server](https://perses.dev/docs/perses/latest/getting-started/#install-perses), e.g. running in [kind](https://kind.sigs.k8s.io/).
- [Perses CLI](https://perses.dev/docs/perses/latest/cli/)

0. (Login to server with e.g. `percli login http://localhost:8080`)
1. Build dashboards with `percli dac build -f main.go`
2. Deploy dashbaords with `percli apply -f built/main_output.yaml`

Note that only yaml output is supported (see [main.go](main.go) that adds the yaml separator to the output file).
