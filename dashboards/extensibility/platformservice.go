package extensibility

import (
	"github.com/perses/perses/go-sdk/dashboard"
)

// NewPlatformServiceDashboard visualizes platform services and cluster providers as they share the same metrics
func NewPlatformServiceDashboard() (dashboard.Builder, error) {
	return dashboard.New("platformservices",
		dashboard.ProjectName(ProjectName),
		dashboard.Name("Platform Services and Cluster Providers"),
		providerSelectorVariable("cp-", "ps-"),
		durationVariable(),
		performancePanel(3),
		resourceUsagePanel(),
		prometheusDatasource(),
	)
}
