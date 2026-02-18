package response

// AccountInfo represents cloud account/project identity
type AccountInfo struct {
	Provider    string `json:"provider"`
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
}

// ServiceCost represents cost for a single service
type ServiceCost struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

// CostInfo represents cost data for a time period
type CostInfo struct {
	StartDate string        `json:"start_date"`
	EndDate   string        `json:"end_date"`
	Services  []ServiceCost `json:"services"`
	Total     float64       `json:"total"`
	Currency  string        `json:"currency"`
}

// CostComparison represents cost comparison between two periods
type CostComparison struct {
	CurrentMonth  CostInfo `json:"current_month"`
	LastMonth     CostInfo `json:"last_month"`
	Difference    float64  `json:"difference"`
	PercentChange float64  `json:"percent_change"`
}

// TrendSummary provides summary statistics for cost trend
type TrendSummary struct {
	TotalSpend     float64 `json:"total_spend_6_months"`
	AverageMonthly float64 `json:"average_monthly"`
	HighestMonth   string  `json:"highest_month"`
	HighestAmount  float64 `json:"highest_amount"`
	LowestMonth    string  `json:"lowest_month"`
	LowestAmount   float64 `json:"lowest_amount"`
}

// CostTrend represents 6-month cost trend with summary
type CostTrend struct {
	Months  []CostInfo   `json:"months"`
	Summary TrendSummary `json:"summary"`
}

// UnusedVolume represents an unused storage volume
type UnusedVolume struct {
	ID     string `json:"id"`
	SizeGB int32  `json:"size_gb"`
	Status string `json:"status"`
}

// StoppedInstance represents a stopped compute instance
type StoppedInstance struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	StoppedDays int    `json:"stopped_days"`
}

// UnusedIP represents an unassociated IP address
type UnusedIP struct {
	Address      string `json:"address"`
	AllocationID string `json:"allocation_id"`
}

// Reservation represents a reserved instance/commitment
type Reservation struct {
	ID              string `json:"id"`
	InstanceType    string `json:"instance_type"`
	Status          string `json:"status"`
	DaysUntilExpiry int    `json:"days_until_expiry"`
}

// WasteSummary aggregates all waste detection results
type WasteSummary struct {
	Provider             string            `json:"provider"`
	AccountID            string            `json:"account_id"`
	UnusedVolumes        []UnusedVolume    `json:"unused_volumes"`
	AttachedVolumes      []UnusedVolume    `json:"volumes_attached_to_stopped_instances"`
	UnusedIPs            []UnusedIP        `json:"unused_ips"`
	StoppedInstances     []StoppedInstance `json:"stopped_instances"`
	ExpiringReservations []Reservation     `json:"expiring_reservations"`
}

// AzureSubscription represents Azure subscription details
type AzureSubscription struct {
	SubscriptionID string `json:"subscription_id"`
	DisplayName    string `json:"display_name"`
	State          string `json:"state"`
}

// MultiCloudCostSummary represents costs across all providers
type MultiCloudCostSummary struct {
	Providers []ProviderCostSummary `json:"providers"`
	Total     float64               `json:"total"`
	Currency  string                `json:"currency"`
}

// ProviderCostSummary represents cost summary for a single provider
type ProviderCostSummary struct {
	Provider         string   `json:"provider"`
	AccountID        string   `json:"account_id"`
	CurrentMonthCost float64  `json:"current_month_cost"`
	LastMonthCost    float64  `json:"last_month_cost"`
	Difference       float64  `json:"difference"`
	PercentChange    float64  `json:"percent_change"`
	Currency         string   `json:"currency"`
	Error            string   `json:"error,omitempty"`
}

// MultiCloudWasteSummary represents waste across all providers
type MultiCloudWasteSummary struct {
	Providers []WasteSummary `json:"providers"`
}
