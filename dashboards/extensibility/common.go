package extensibility

import (
	"fmt"
	"slices"

	"github.com/perses/perses/go-sdk/dashboard"
	"github.com/perses/perses/go-sdk/panel"
	panelgroup "github.com/perses/perses/go-sdk/panel-group"
	listVar "github.com/perses/perses/go-sdk/variable/list-variable"
	textvariable "github.com/perses/perses/go-sdk/variable/text-variable"
	promDs "github.com/perses/plugins/prometheus/sdk/go/datasource"
	"github.com/perses/plugins/prometheus/sdk/go/query"
	"github.com/perses/plugins/prometheus/sdk/go/variable/promql"
	timeSerieschart "github.com/perses/plugins/timeserieschart/sdk/go"
)

var (
	PrometheusDatasourceName = "prometheus"
	PrometheusEndpoint       = "http://prometheus.monitoring:9090"
	ProjectName              = "opencontrolplane"
)

// DASHBOARD PANELS

// RESOURCE PANEL FUNCS
func resourceUsagePanel() dashboard.Option {
	return dashboard.AddPanelGroup("Resource Usage",
		panelgroup.PanelsPerLine(2),
		memoryUsagePanel(),
		cpuUsagePanel(),
	)
}

func memoryUsagePanel() panelgroup.Option {
	return panelgroup.AddPanel("Memory Usage",
		timeSerieschart.Chart(timeSerieschart.WithLegend(timeSerieschart.Legend{
			Position: timeSerieschart.BottomPosition,
		})),
		panel.AddQuery(
			query.PromQL("container_memory_working_set_bytes{pod=~\"$provider\"}",
				query.SeriesNameFormat("{{pod}}"),
				query.Datasource(PrometheusDatasourceName)),
		),
	)
}

func cpuUsagePanel() panelgroup.Option {
	return panelgroup.AddPanel("CPU Usage (in cores)",
		timeSerieschart.Chart(timeSerieschart.WithLegend(timeSerieschart.Legend{
			Position: timeSerieschart.BottomPosition,
		})),
		panel.AddQuery(
			query.PromQL("rate(container_cpu_usage_seconds_total{pod=~\"$provider\"}[$duration])",
				query.SeriesNameFormat("{{pod}}"),
				query.Datasource(PrometheusDatasourceName)),
		),
	)
}

// PERFORMANCE PANEL FUNCS
func performancePanel(panelsPerLine int, opts ...panelgroup.Option) dashboard.Option {
	defaultOptions := []panelgroup.Option{
		panelgroup.PanelsPerLine(panelsPerLine),
		reconcileTimePanel(),
		reconcileCountPanel(),
		workQueuePanel(),
	}
	return dashboard.AddPanelGroup("Performance",
		slices.Concat(opts, defaultOptions)...,
	)
}

func reconcileTimePanel() panelgroup.Option {
	return panelgroup.AddPanel("Average Reconcile Time (seconds)",
		timeSerieschart.Chart(timeSerieschart.WithLegend(timeSerieschart.Legend{
			Position: timeSerieschart.BottomPosition,
		})),
		panel.AddQuery(
			query.PromQL("rate(controller_runtime_reconcile_time_seconds_sum{pod=~\"$provider\"}[$duration])"+
				"/rate(controller_runtime_reconcile_time_seconds_count{pod=~\"$provider\"}[$duration])",
				query.SeriesNameFormat("{{pod}}"),
				query.Datasource(PrometheusDatasourceName)),
		),
	)
}

func reconcileCountPanel() panelgroup.Option {
	return panelgroup.AddPanel("Reconcile Count",
		timeSerieschart.Chart(timeSerieschart.WithLegend(timeSerieschart.Legend{
			Position: timeSerieschart.BottomPosition,
		})),
		panel.AddQuery(
			query.PromQL("rate(controller_runtime_reconcile_total{pod=~\"$provider\"}[$duration])",
				query.SeriesNameFormat("result={{result}} ({{pod}})"),
				query.Datasource(PrometheusDatasourceName)),
		),
	)
}

func workQueuePanel() panelgroup.Option {
	return panelgroup.AddPanel("Workqueue",
		timeSerieschart.Chart(timeSerieschart.WithLegend(timeSerieschart.Legend{
			Position: timeSerieschart.BottomPosition,
		})),
		panel.AddQuery(
			query.PromQL("rate(workqueue_adds_total{pod=~\"$provider\"}[$duration])",
				query.SeriesNameFormat("items added ({{pod}})"),
				query.Datasource(PrometheusDatasourceName)),
			query.PromQL("rate(workqueue_queue_duration_seconds_sum{pod=~\"$provider\"}[$duration])"+
				"/rate(workqueue_queue_duration_seconds_count{pod=~\"$provider\"}[$duration])",
				query.SeriesNameFormat("average duration seconds ({{pod}})"),
				query.Datasource(PrometheusDatasourceName)),
			query.PromQL("rate(workqueue_depth{pod=~\"$provider\"}[$duration])",
				query.SeriesNameFormat("depth ({{pod}})"),
				query.Datasource(PrometheusDatasourceName)),
		),
	)
}

// DASHBOARD SETTINGS FUNCS

func prometheusDatasource() dashboard.Option {
	return dashboard.AddDatasource(PrometheusDatasourceName, promDs.Prometheus(promDs.HTTPProxy(PrometheusEndpoint)))
}

func providerSelectorVariable(podPrefix string) dashboard.Option {
	podRegex := fmt.Sprintf("%s.*", podPrefix)
	return dashboard.AddVariable("provider",
		listVar.List(
			promql.PrometheusPromQL(fmt.Sprintf("container_cpu_usage_seconds_total{pod=~%q}", podRegex),
				promql.LabelName("pod"),
				promql.Datasource(PrometheusDatasourceName),
			),
			listVar.DisplayName("Provider"),
			listVar.AllowMultiple(true),
			listVar.AllowAllValue(true),
		),
	)
}

func durationVariable() dashboard.Option {
	return dashboard.AddVariable("duration", textvariable.Text("1h", textvariable.DisplayName("Duration")))
}
