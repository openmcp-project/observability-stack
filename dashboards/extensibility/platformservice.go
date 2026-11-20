package extensibility

import (
	"github.com/perses/perses/go-sdk/dashboard"
)

func NewPlatformServiceDashboard() (dashboard.Builder, error) {
	return dashboard.New("platformservices",
		dashboard.ProjectName(projectName),
		dashboard.Name("Platform Services"),
		providerSelectorVariable("ps-"),
		durationVariable(),
		performancePanel(3),
		resourceUsagePanel(),
		prometheusDatasource(),
	)
}
