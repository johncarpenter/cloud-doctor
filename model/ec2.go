package model

import "time"

type ElasticIpInfo struct {
	UnusedElasticIpAddresses []string
	UsedElasticIpAddresses   []AttachedIpInfo
}

type AttachedIpInfo struct {
	IpAddress    string
	AllocationId string
	ResourceType string
}

type RiExpirationInfo struct {
	ReservedInstanceId string
	InstanceType       string
	ExpirationDate     time.Time
	DaysUntilExpiry    int
	State              string
	Status             string
}
