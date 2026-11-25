package extensibility

import (
	"github.com/perses/perses/go-sdk/dashboard"
	"github.com/perses/perses/go-sdk/panel"
	panelgroup "github.com/perses/perses/go-sdk/panel-group"
	"github.com/perses/plugins/prometheus/sdk/go/query"
	timeSerieschart "github.com/perses/plugins/timeserieschart/sdk/go"
)

// NewServiceProviderDashboard visualizes service providers separately with a dedicated opencontrolplane-runtime metrics panel
// and tenant resource status
func NewServiceProviderDashboard() (dashboard.Builder, error) {
	return dashboard.New("serviceproviders",
		dashboard.ProjectName(ProjectName),
		dashboard.Name("Service Providers"),
		resourceSelectorVariable("service_provider_status"),
		providerSelectorVariable("sp-"),
		durationVariable(),
		dashboard.AddPanelGroup("Health", resourceStatusPanel("Tenant Resources Status")),
		performancePanel(2, averageCreateAndUpdate()),
		resourceUsagePanel(),
		prometheusDatasource(),
	)
}

func averageCreateAndUpdate() panelgroup.Option {
	return panelgroup.AddPanel("Average Time to Create/Update",
		timeSerieschart.Chart(timeSerieschart.WithLegend(timeSerieschart.Legend{
			Position: timeSerieschart.BottomPosition,
		})),
		panel.AddQuery(
			query.PromQL("rate(opencontrolplane_runtime_resource_create_seconds_sum{pod=~\"$provider\"}[$duration])"+
				"/rate(opencontrolplane_runtime_resource_create_seconds_count{pod=~\"$provider\"}[$duration])",
				query.SeriesNameFormat("resource creation ({{pod}})"),
				query.Datasource(PrometheusDatasourceName)),
		),
		panel.AddQuery(
			query.PromQL("rate(opencontrolplane_runtime_resource_update_seconds_sum{pod=~\"$provider\"}[$duration])"+
				"/rate(opencontrolplane_runtime_resource_update_seconds_count{pod=~\"$provider\"}[$duration])",
				query.SeriesNameFormat("resource update ({{pod}})"),
				query.Datasource(PrometheusDatasourceName)),
		),
	)
}
