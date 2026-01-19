package model

type Flags struct {
	// Common flags
	Provider string
	Trend    bool
	Waste    bool

	// AWS-specific flags
	Region  string
	Profile string

	// GCP-specific flags
	Project        string
	BillingAccount string

	// Azure-specific flags
	Subscription string
}
