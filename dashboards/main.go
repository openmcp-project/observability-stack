package main

import (
	"fmt"
	"os"

	"github.com/openmcp-project/observability-stack/dashboards/extensibility"
	"github.com/perses/perses/go-sdk"
	"github.com/perses/perses/go-sdk/dashboard"
)

type dashboardFunc func() (dashboard.Builder, error)

var dashboards = []dashboardFunc{
	extensibility.NewPlatformServiceDashboard,
	extensibility.NewServiceProviderDashboard,
}

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func main() {
	extensibility.PrometheusDatasourceName = envOrDefault("OCP_PROMETHEUS_DATASOURCE_NAME", extensibility.PrometheusDatasourceName)
	extensibility.PrometheusEndpoint = envOrDefault("OCP_PROMETHEUS_ENDPOINT", extensibility.PrometheusEndpoint)
	extensibility.ProjectName = envOrDefault("OCP_PROJECT_NAME", extensibility.ProjectName)

	exec := sdk.NewExec()
	// create one file that contains all dashboards
	for i, dFunc := range dashboards {
		exec.BuildDashboard(dFunc())
		if i == len(dashboards) - 1 {
			// skip trailing separtor
			continue
		}
		fmt.Println("---")
	}
}
