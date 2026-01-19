package model

// AccountInfo represents cloud account/project identity
type AccountInfo struct {
	Provider    string
	AccountID   string
	AccountName string
}

// UnusedVolume represents an unused storage volume
type UnusedVolume struct {
	ID     string
	SizeGB int32
	Status string // "available", "attached_stopped"
}

// StoppedInstance represents a stopped compute instance
type StoppedInstance struct {
	ID          string
	Name        string
	StoppedDays int
}

// UnusedIP represents an unassociated IP address
type UnusedIP struct {
	Address      string
	AllocationID string
}

// Reservation represents a reserved instance/commitment
type Reservation struct {
	ID              string
	InstanceType    string
	Status          string // "expiring", "expired"
	DaysUntilExpiry int
}

// ProviderCostResult represents cost analysis results for a single provider
type ProviderCostResult struct {
	Provider         string
	AccountID        string
	CurrentMonthData *CostInfo
	LastMonthData    *CostInfo
	CurrentTotalCost string
	LastTotalCost    string
	TrendData        []CostInfo
	Error            error
}

// ProviderWasteResult represents waste detection results for a single provider
type ProviderWasteResult struct {
	Provider             string
	AccountID            string
	UnusedVolumes        []UnusedVolume
	AttachedVolumes      []UnusedVolume
	UnusedIPs            []UnusedIP
	StoppedInstances     []StoppedInstance
	ExpiringReservations []Reservation
	Error                error
}
