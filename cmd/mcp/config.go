package main

import "os"

// Config holds environment-based configuration for all cloud providers
type Config struct {
	// AWS configuration
	AWSRegion  string
	AWSProfile string

	// GCP configuration
	GCPProjectID      string
	GCPBillingAccount string

	// Azure configuration
	AzureSubscriptionID string
}

// LoadConfig reads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		AWSRegion:           getEnvOrDefault("AWS_REGION", "us-east-1"),
		AWSProfile:          os.Getenv("AWS_PROFILE"),
		GCPProjectID:        os.Getenv("GCP_PROJECT_ID"),
		GCPBillingAccount:   os.Getenv("GCP_BILLING_ACCOUNT"),
		AzureSubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
	}
}

// HasAWS returns true if AWS is available (always true - uses default credential chain)
func (c *Config) HasAWS() bool {
	return true
}

// HasGCP returns true if GCP project is configured
func (c *Config) HasGCP() bool {
	return c.GCPProjectID != ""
}

// HasGCPBilling returns true if GCP billing is configured for cost analysis
func (c *Config) HasGCPBilling() bool {
	return c.GCPProjectID != "" && c.GCPBillingAccount != ""
}

// HasAzure returns true if Azure subscription is configured
func (c *Config) HasAzure() bool {
	return c.AzureSubscriptionID != ""
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
