package main

import (
	"fmt"
	"os"

	"github.com/elC0mpa/aws-doctor/cmd/mcp/tools"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	cfg := LoadConfig()

	s := server.NewMCPServer(
		"cloud-doctor-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register tools for each provider
	tools.RegisterAWSTools(s, cfg.AWSRegion, cfg.AWSProfile)
	tools.RegisterGCPTools(s, cfg.GCPProjectID, cfg.GCPBillingAccount)
	tools.RegisterAzureTools(s, cfg.AzureSubscriptionID)
	tools.RegisterMultiCloudTools(s, cfg.AWSRegion, cfg.AWSProfile, cfg.GCPProjectID, cfg.GCPBillingAccount, cfg.AzureSubscriptionID)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
