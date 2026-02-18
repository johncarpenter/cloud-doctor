package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/elC0mpa/aws-doctor/cmd/mcp/response"
	azurecompute "github.com/elC0mpa/aws-doctor/service/azure/compute"
	azureconfig "github.com/elC0mpa/aws-doctor/service/azure/config"
	azurecostmanagement "github.com/elC0mpa/aws-doctor/service/azure/costmanagement"
	azureidentity "github.com/elC0mpa/aws-doctor/service/azure/identity"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterAzureTools registers all Azure tools with the MCP server
func RegisterAzureTools(s *server.MCPServer, subscriptionID string) {
	// List subscriptions (works without specific subscription ID)
	s.AddTool(
		mcp.NewTool("azure_list_subscriptions",
			mcp.WithDescription("List all Azure subscriptions the current credential has access to"),
		),
		makeAzureListSubscriptionsHandler(),
	)

	// Subscription info
	s.AddTool(
		mcp.NewTool("azure_get_subscription_info",
			mcp.WithDescription("Get Azure subscription details including ID, display name, and state. Requires AZURE_SUBSCRIPTION_ID environment variable."),
		),
		makeAzureSubscriptionInfoHandler(subscriptionID),
	)

	// Current month costs
	s.AddTool(
		mcp.NewTool("azure_get_current_month_costs",
			mcp.WithDescription("Get Azure costs for the current month, broken down by service. Requires AZURE_SUBSCRIPTION_ID."),
		),
		makeAzureCurrentMonthCostsHandler(subscriptionID),
	)

	// Cost comparison
	s.AddTool(
		mcp.NewTool("azure_get_cost_comparison",
			mcp.WithDescription("Compare Azure costs between current month and last month (same period), showing difference and percent change. Requires AZURE_SUBSCRIPTION_ID."),
		),
		makeAzureCostComparisonHandler(subscriptionID),
	)

	// Cost trend
	s.AddTool(
		mcp.NewTool("azure_get_cost_trend",
			mcp.WithDescription("Get Azure cost trend for the last 6 months with summary statistics. Requires AZURE_SUBSCRIPTION_ID."),
		),
		makeAzureCostTrendHandler(subscriptionID),
	)

	// Unused volumes
	s.AddTool(
		mcp.NewTool("azure_get_unused_volumes",
			mcp.WithDescription("List unattached Managed Disks. Requires AZURE_SUBSCRIPTION_ID."),
		),
		makeAzureUnusedVolumesHandler(subscriptionID),
	)

	// Unused IPs
	s.AddTool(
		mcp.NewTool("azure_get_unused_ips",
			mcp.WithDescription("List unassociated Public IP addresses. Requires AZURE_SUBSCRIPTION_ID."),
		),
		makeAzureUnusedIPsHandler(subscriptionID),
	)

	// Stopped instances
	s.AddTool(
		mcp.NewTool("azure_get_stopped_instances",
			mcp.WithDescription("List deallocated Virtual Machines with their attached disks. Requires AZURE_SUBSCRIPTION_ID."),
		),
		makeAzureStoppedInstancesHandler(subscriptionID),
	)

	// Expiring reservations
	s.AddTool(
		mcp.NewTool("azure_get_expiring_reservations",
			mcp.WithDescription("List Reserved VM Instances that are expiring within 30 days or have recently expired. Requires AZURE_SUBSCRIPTION_ID."),
		),
		makeAzureExpiringReservationsHandler(subscriptionID),
	)

	// Waste summary
	s.AddTool(
		mcp.NewTool("azure_get_waste_summary",
			mcp.WithDescription("Get a complete summary of all Azure waste detection: unattached disks, unused IPs, deallocated VMs, and expiring reservations. Requires AZURE_SUBSCRIPTION_ID."),
		),
		makeAzureWasteSummaryHandler(subscriptionID),
	)
}

func makeAzureListSubscriptionsHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		credential, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure credential: %v", err)), nil
		}

		client, err := armsubscriptions.NewClient(credential, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create subscriptions client: %v", err)), nil
		}

		var subscriptions []response.AzureSubscription
		pager := client.NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list subscriptions: %v", err)), nil
			}

			for _, sub := range page.Value {
				if sub.SubscriptionID == nil {
					continue
				}

				displayName := *sub.SubscriptionID
				if sub.DisplayName != nil {
					displayName = *sub.DisplayName
				}

				state := "Unknown"
				if sub.State != nil {
					state = string(*sub.State)
				}

				// Only include enabled subscriptions
				if state == "Enabled" {
					subscriptions = append(subscriptions, response.AzureSubscription{
						SubscriptionID: *sub.SubscriptionID,
						DisplayName:    displayName,
						State:          state,
					})
				}
			}
		}

		data, _ := json.MarshalIndent(subscriptions, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAzureSubscriptionInfoHandler(subscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if subscriptionID == "" {
			return mcp.NewToolResultError("AZURE_SUBSCRIPTION_ID environment variable is required"), nil
		}

		cfgSvc, err := azureconfig.NewService(subscriptionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure config: %v", err)), nil
		}

		identitySvc, err := azureidentity.NewService(subscriptionID, cfgSvc.GetCredential())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure identity service: %v", err)), nil
		}

		info, err := identitySvc.GetAccountInfo(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get subscription info: %v", err)), nil
		}

		resp := response.ConvertAccountInfo(info)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAzureCurrentMonthCostsHandler(subscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if subscriptionID == "" {
			return mcp.NewToolResultError("AZURE_SUBSCRIPTION_ID environment variable is required"), nil
		}

		cfgSvc, err := azureconfig.NewService(subscriptionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure config: %v", err)), nil
		}

		costSvc, err := azurecostmanagement.NewService(subscriptionID, cfgSvc.GetCredential())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure cost management service: %v", err)), nil
		}

		costData, err := costSvc.GetCurrentMonthCostsByService(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get costs: %v", err)), nil
		}

		resp := response.ConvertCostInfo(costData)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAzureCostComparisonHandler(subscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if subscriptionID == "" {
			return mcp.NewToolResultError("AZURE_SUBSCRIPTION_ID environment variable is required"), nil
		}

		cfgSvc, err := azureconfig.NewService(subscriptionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure config: %v", err)), nil
		}

		costSvc, err := azurecostmanagement.NewService(subscriptionID, cfgSvc.GetCredential())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure cost management service: %v", err)), nil
		}

		currentData, err := costSvc.GetCurrentMonthCostsByService(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get current month costs: %v", err)), nil
		}

		lastData, err := costSvc.GetLastMonthCostsByService(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get last month costs: %v", err)), nil
		}

		currentCosts := response.ConvertCostInfo(currentData)
		lastCosts := response.ConvertCostInfo(lastData)

		diff := currentCosts.Total - lastCosts.Total
		var percentChange float64
		if lastCosts.Total > 0 {
			percentChange = (diff / lastCosts.Total) * 100
		}

		resp := response.CostComparison{
			CurrentMonth:  *currentCosts,
			LastMonth:     *lastCosts,
			Difference:    diff,
			PercentChange: percentChange,
		}

		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAzureCostTrendHandler(subscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if subscriptionID == "" {
			return mcp.NewToolResultError("AZURE_SUBSCRIPTION_ID environment variable is required"), nil
		}

		cfgSvc, err := azureconfig.NewService(subscriptionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure config: %v", err)), nil
		}

		costSvc, err := azurecostmanagement.NewService(subscriptionID, cfgSvc.GetCredential())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure cost management service: %v", err)), nil
		}

		trendData, err := costSvc.GetLastSixMonthsCosts(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get cost trend: %v", err)), nil
		}

		resp := response.ConvertTrendData(trendData)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAzureUnusedVolumesHandler(subscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if subscriptionID == "" {
			return mcp.NewToolResultError("AZURE_SUBSCRIPTION_ID environment variable is required"), nil
		}

		cfgSvc, err := azureconfig.NewService(subscriptionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure config: %v", err)), nil
		}

		computeSvc, err := azurecompute.NewService(subscriptionID, cfgSvc.GetCredential())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure compute service: %v", err)), nil
		}

		volumes, err := computeSvc.GetUnusedVolumes(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get unused volumes: %v", err)), nil
		}

		resp := response.ConvertUnusedVolumes(volumes)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAzureUnusedIPsHandler(subscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if subscriptionID == "" {
			return mcp.NewToolResultError("AZURE_SUBSCRIPTION_ID environment variable is required"), nil
		}

		cfgSvc, err := azureconfig.NewService(subscriptionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure config: %v", err)), nil
		}

		computeSvc, err := azurecompute.NewService(subscriptionID, cfgSvc.GetCredential())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure compute service: %v", err)), nil
		}

		ips, err := computeSvc.GetUnusedIPs(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get unused IPs: %v", err)), nil
		}

		resp := response.ConvertUnusedIPs(ips)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAzureStoppedInstancesHandler(subscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if subscriptionID == "" {
			return mcp.NewToolResultError("AZURE_SUBSCRIPTION_ID environment variable is required"), nil
		}

		cfgSvc, err := azureconfig.NewService(subscriptionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure config: %v", err)), nil
		}

		computeSvc, err := azurecompute.NewService(subscriptionID, cfgSvc.GetCredential())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure compute service: %v", err)), nil
		}

		instances, attachedVolumes, err := computeSvc.GetStoppedInstances(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stopped instances: %v", err)), nil
		}

		resp := struct {
			Instances       []response.StoppedInstance `json:"stopped_instances"`
			AttachedVolumes []response.UnusedVolume    `json:"attached_volumes"`
		}{
			Instances:       response.ConvertStoppedInstances(instances),
			AttachedVolumes: response.ConvertUnusedVolumes(attachedVolumes),
		}

		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAzureExpiringReservationsHandler(subscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if subscriptionID == "" {
			return mcp.NewToolResultError("AZURE_SUBSCRIPTION_ID environment variable is required"), nil
		}

		cfgSvc, err := azureconfig.NewService(subscriptionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure config: %v", err)), nil
		}

		computeSvc, err := azurecompute.NewService(subscriptionID, cfgSvc.GetCredential())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure compute service: %v", err)), nil
		}

		reservations, err := computeSvc.GetExpiringReservations(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get expiring reservations: %v", err)), nil
		}

		resp := response.ConvertReservations(reservations)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAzureWasteSummaryHandler(subscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if subscriptionID == "" {
			return mcp.NewToolResultError("AZURE_SUBSCRIPTION_ID environment variable is required"), nil
		}

		cfgSvc, err := azureconfig.NewService(subscriptionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure config: %v", err)), nil
		}

		identitySvc, err := azureidentity.NewService(subscriptionID, cfgSvc.GetCredential())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure identity service: %v", err)), nil
		}

		accountInfo, err := identitySvc.GetAccountInfo(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get subscription info: %v", err)), nil
		}

		computeSvc, err := azurecompute.NewService(subscriptionID, cfgSvc.GetCredential())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create Azure compute service: %v", err)), nil
		}

		unusedVolumes, err := computeSvc.GetUnusedVolumes(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get unused volumes: %v", err)), nil
		}

		unusedIPs, err := computeSvc.GetUnusedIPs(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get unused IPs: %v", err)), nil
		}

		stoppedInstances, attachedVolumes, err := computeSvc.GetStoppedInstances(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stopped instances: %v", err)), nil
		}

		expiringReservations, err := computeSvc.GetExpiringReservations(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get expiring reservations: %v", err)), nil
		}

		resp := response.WasteSummary{
			Provider:             "azure",
			AccountID:            accountInfo.AccountID,
			UnusedVolumes:        response.ConvertUnusedVolumes(unusedVolumes),
			AttachedVolumes:      response.ConvertUnusedVolumes(attachedVolumes),
			UnusedIPs:            response.ConvertUnusedIPs(unusedIPs),
			StoppedInstances:     response.ConvertStoppedInstances(stoppedInstances),
			ExpiringReservations: response.ConvertReservations(expiringReservations),
		}

		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}
