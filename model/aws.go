package model

import "time"

// AWS-specific models

// ElasticIpInfo contains information about Elastic IP addresses
type ElasticIpInfo struct {
	UnusedElasticIpAddresses []string
	UsedElasticIpAddresses   []AttachedIpInfo
}

// AttachedIpInfo contains information about an attached IP address
type AttachedIpInfo struct {
	IpAddress    string
	AllocationId string
	ResourceType string
}

// RiExpirationInfo contains reserved instance expiration information
type RiExpirationInfo struct {
	ReservedInstanceId string
	InstanceType       string
	ExpirationDate     time.Time
	DaysUntilExpiry    int
	State              string
	Status             string
}
