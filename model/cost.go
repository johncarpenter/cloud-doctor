package model

// DateInterval represents a time period for cost analysis
type DateInterval struct {
	Start *string
	End   *string
}

// CostInfo contains cost data for a time period
type CostInfo struct {
	DateInterval
	CostGroup
}

// CostGroup maps service names to their cost data
type CostGroup map[string]struct {
	Amount float64
	Unit   string
}

// ServiceCost represents cost for a single service
type ServiceCost struct {
	Name   string
	Amount float64
	Unit   string
}
