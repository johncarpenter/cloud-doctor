package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elC0mpa/aws-doctor/cmd/mcp/response"
	gcpbilling "github.com/elC0mpa/aws-doctor/service/gcp/billing"
	gcpcompute "github.com/elC0mpa/aws-doctor/service/gcp/compute"
	gcpidentity "github.com/elC0mpa/aws-doctor/service/gcp/identity"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterGCPTools registers all GCP tools with the MCP server
func RegisterGCPTools(s *server.MCPServer, projectID, billingAccount string) {
	// Project info
	s.AddTool(
		mcp.NewTool("gcp_get_project_info",
			mcp.WithDescription("Get GCP project identity information. Requires GCP_PROJECT_ID environment variable."),
		),
		makeGCPProjectInfoHandler(projectID),
	)

	// Current month costs
	s.AddTool(
		mcp.NewTool("gcp_get_current_month_costs",
			mcp.WithDescription("Get GCP costs for the current month, broken down by service. Requires GCP_PROJECT_ID and GCP_BILLING_ACCOUNT environment variables."),
		),
		makeGCPCurrentMonthCostsHandler(projectID, billingAccount),
	)

	// Cost comparison
	s.AddTool(
		mcp.NewTool("gcp_get_cost_comparison",
			mcp.WithDescription("Compare GCP costs between current month and last month (same period), showing difference and percent change. Requires GCP_PROJECT_ID and GCP_BILLING_ACCOUNT."),
		),
		makeGCPCostComparisonHandler(projectID, billingAccount),
	)

	// Cost trend
	s.AddTool(
		mcp.NewTool("gcp_get_cost_trend",
			mcp.WithDescription("Get GCP cost trend for the last 6 months with summary statistics. Requires GCP_PROJECT_ID and GCP_BILLING_ACCOUNT."),
		),
		makeGCPCostTrendHandler(projectID, billingAccount),
	)

	// Unused volumes
	s.AddTool(
		mcp.NewTool("gcp_get_unused_volumes",
			mcp.WithDescription("List persistent disks that are not attached to any VM instance. Requires GCP_PROJECT_ID."),
		),
		makeGCPUnusedVolumesHandler(projectID),
	)

	// Unused IPs
	s.AddTool(
		mcp.NewTool("gcp_get_unused_ips",
			mcp.WithDescription("List static external IP addresses that are not in use. Requires GCP_PROJECT_ID."),
		),
		makeGCPUnusedIPsHandler(projectID),
	)

	// Stopped instances
	s.AddTool(
		mcp.NewTool("gcp_get_stopped_instances",
			mcp.WithDescription("List VM instances in TERMINATED state. Requires GCP_PROJECT_ID."),
		),
		makeGCPStoppedInstancesHandler(projectID),
	)

	// Expiring reservations
	s.AddTool(
		mcp.NewTool("gcp_get_expiring_reservations",
			mcp.WithDescription("List Committed Use Discounts (CUDs) that are expiring within 30 days or have recently expired. Requires GCP_PROJECT_ID."),
		),
		makeGCPExpiringReservationsHandler(projectID),
	)

	// Waste summary
	s.AddTool(
		mcp.NewTool("gcp_get_waste_summary",
			mcp.WithDescription("Get a complete summary of all GCP waste detection: unused disks, unused IPs, stopped VMs, and expiring commitments. Requires GCP_PROJECT_ID."),
		),
		makeGCPWasteSummaryHandler(projectID),
	)
}

func makeGCPProjectInfoHandler(projectID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if projectID == "" {
			return mcp.NewToolResultError("GCP_PROJECT_ID environment variable is required"), nil
		}

		identitySvc, err := gcpidentity.NewService(ctx, projectID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GCP identity service: %v", err)), nil
		}

		info, err := identitySvc.GetAccountInfo(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get project info: %v", err)), nil
		}

		resp := response.ConvertAccountInfo(info)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeGCPCurrentMonthCostsHandler(projectID, billingAccount string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if projectID == "" {
			return mcp.NewToolResultError("GCP_PROJECT_ID environment variable is required"), nil
		}
		if billingAccount == "" {
			return mcp.NewToolResultError("GCP_BILLING_ACCOUNT environment variable is required for cost analysis"), nil
		}

		billingSvc, err := gcpbilling.NewService(ctx, projectID, billingAccount)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GCP billing service: %v", err)), nil
		}
		defer billingSvc.Close()

		costData, err := billingSvc.GetCurrentMonthCostsByService(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get costs: %v", err)), nil
		}

		resp := response.ConvertCostInfo(costData)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeGCPCostComparisonHandler(projectID, billingAccount string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if projectID == "" {
			return mcp.NewToolResultError("GCP_PROJECT_ID environment variable is required"), nil
		}
		if billingAccount == "" {
			return mcp.NewToolResultError("GCP_BILLING_ACCOUNT environment variable is required for cost analysis"), nil
		}

		billingSvc, err := gcpbilling.NewService(ctx, projectID, billingAccount)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GCP billing service: %v", err)), nil
		}
		defer billingSvc.Close()

		currentData, err := billingSvc.GetCurrentMonthCostsByService(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get current month costs: %v", err)), nil
		}

		lastData, err := billingSvc.GetLastMonthCostsByService(ctx)
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

func makeGCPCostTrendHandler(projectID, billingAccount string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if projectID == "" {
			return mcp.NewToolResultError("GCP_PROJECT_ID environment variable is required"), nil
		}
		if billingAccount == "" {
			return mcp.NewToolResultError("GCP_BILLING_ACCOUNT environment variable is required for cost analysis"), nil
		}

		billingSvc, err := gcpbilling.NewService(ctx, projectID, billingAccount)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GCP billing service: %v", err)), nil
		}
		defer billingSvc.Close()

		trendData, err := billingSvc.GetLastSixMonthsCosts(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get cost trend: %v", err)), nil
		}

		resp := response.ConvertTrendData(trendData)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeGCPUnusedVolumesHandler(projectID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if projectID == "" {
			return mcp.NewToolResultError("GCP_PROJECT_ID environment variable is required"), nil
		}

		computeSvc, err := gcpcompute.NewService(ctx, projectID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GCP compute service: %v", err)), nil
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

func makeGCPUnusedIPsHandler(projectID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if projectID == "" {
			return mcp.NewToolResultError("GCP_PROJECT_ID environment variable is required"), nil
		}

		computeSvc, err := gcpcompute.NewService(ctx, projectID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GCP compute service: %v", err)), nil
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

func makeGCPStoppedInstancesHandler(projectID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if projectID == "" {
			return mcp.NewToolResultError("GCP_PROJECT_ID environment variable is required"), nil
		}

		computeSvc, err := gcpcompute.NewService(ctx, projectID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GCP compute service: %v", err)), nil
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

func makeGCPExpiringReservationsHandler(projectID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if projectID == "" {
			return mcp.NewToolResultError("GCP_PROJECT_ID environment variable is required"), nil
		}

		computeSvc, err := gcpcompute.NewService(ctx, projectID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GCP compute service: %v", err)), nil
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

func makeGCPWasteSummaryHandler(projectID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if projectID == "" {
			return mcp.NewToolResultError("GCP_PROJECT_ID environment variable is required"), nil
		}

		identitySvc, err := gcpidentity.NewService(ctx, projectID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GCP identity service: %v", err)), nil
		}

		accountInfo, err := identitySvc.GetAccountInfo(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get project info: %v", err)), nil
		}

		computeSvc, err := gcpcompute.NewService(ctx, projectID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GCP compute service: %v", err)), nil
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
			Provider:             "gcp",
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
