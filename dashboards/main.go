package main

import (
	"flag"
	"fmt"

	"github.com/openmcp-project/observability-stack/dashboards/extensibility"
	"github.com/perses/perses/go-sdk"
	"github.com/perses/perses/go-sdk/dashboard"
)

var dashboards = []dashboardFunc{
	extensibility.NewPlatformServiceDashboard,
	extensibility.NewServiceProviderDashboard,
}

type dashboardFunc func() (dashboard.Builder, error)

func main() {
	flag.Parse()
	exec := sdk.NewExec()
	// create one file that contains all dashboards
	for _, dFunc := range dashboards {
		exec.BuildDashboard(dFunc())
		fmt.Println("---")
	}
}
