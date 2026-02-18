package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elC0mpa/aws-doctor/cmd/mcp/response"
	awsconfig "github.com/elC0mpa/aws-doctor/service/aws/config"
	awscostexplorer "github.com/elC0mpa/aws-doctor/service/aws/costexplorer"
	awsec2 "github.com/elC0mpa/aws-doctor/service/aws/ec2"
	awssts "github.com/elC0mpa/aws-doctor/service/aws/sts"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterAWSTools registers all AWS tools with the MCP server
func RegisterAWSTools(s *server.MCPServer, region, profile string) {
	// Account info
	s.AddTool(
		mcp.NewTool("aws_get_account_info",
			mcp.WithDescription("Get AWS account identity information including account ID and ARN"),
		),
		makeAWSAccountInfoHandler(region, profile),
	)

	// Current month costs
	s.AddTool(
		mcp.NewTool("aws_get_current_month_costs",
			mcp.WithDescription("Get AWS costs for the current month, broken down by service"),
		),
		makeAWSCurrentMonthCostsHandler(region, profile),
	)

	// Cost comparison
	s.AddTool(
		mcp.NewTool("aws_get_cost_comparison",
			mcp.WithDescription("Compare AWS costs between current month and last month (same period), showing difference and percent change"),
		),
		makeAWSCostComparisonHandler(region, profile),
	)

	// Cost trend
	s.AddTool(
		mcp.NewTool("aws_get_cost_trend",
			mcp.WithDescription("Get AWS cost trend for the last 6 months with summary statistics"),
		),
		makeAWSCostTrendHandler(region, profile),
	)

	// Unused volumes
	s.AddTool(
		mcp.NewTool("aws_get_unused_volumes",
			mcp.WithDescription("List EBS volumes that are not attached to any EC2 instance"),
		),
		makeAWSUnusedVolumesHandler(region, profile),
	)

	// Unused IPs
	s.AddTool(
		mcp.NewTool("aws_get_unused_ips",
			mcp.WithDescription("List Elastic IP addresses that are not associated with any resource"),
		),
		makeAWSUnusedIPsHandler(region, profile),
	)

	// Stopped instances
	s.AddTool(
		mcp.NewTool("aws_get_stopped_instances",
			mcp.WithDescription("List EC2 instances that have been stopped for more than 30 days, along with their attached volumes"),
		),
		makeAWSStoppedInstancesHandler(region, profile),
	)

	// Expiring reservations
	s.AddTool(
		mcp.NewTool("aws_get_expiring_reservations",
			mcp.WithDescription("List Reserved Instances that are expiring within 30 days or have recently expired"),
		),
		makeAWSExpiringReservationsHandler(region, profile),
	)

	// Waste summary
	s.AddTool(
		mcp.NewTool("aws_get_waste_summary",
			mcp.WithDescription("Get a complete summary of all AWS waste detection: unused volumes, unused IPs, stopped instances, and expiring reservations"),
		),
		makeAWSWasteSummaryHandler(region, profile),
	)
}

func makeAWSAccountInfoHandler(region, profile string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		configSvc := awsconfig.NewService()
		awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to configure AWS: %v", err)), nil
		}

		stsSvc := awssts.NewService(awsCfg)
		info, err := stsSvc.GetAccountInfo(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get account info: %v", err)), nil
		}

		resp := response.ConvertAccountInfo(info)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAWSCurrentMonthCostsHandler(region, profile string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		configSvc := awsconfig.NewService()
		awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to configure AWS: %v", err)), nil
		}

		costSvc := awscostexplorer.NewService(awsCfg)
		costData, err := costSvc.GetCurrentMonthCostsByService(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get costs: %v", err)), nil
		}

		resp := response.ConvertCostInfo(costData)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAWSCostComparisonHandler(region, profile string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		configSvc := awsconfig.NewService()
		awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to configure AWS: %v", err)), nil
		}

		costSvc := awscostexplorer.NewService(awsCfg)

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

func makeAWSCostTrendHandler(region, profile string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		configSvc := awsconfig.NewService()
		awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to configure AWS: %v", err)), nil
		}

		costSvc := awscostexplorer.NewService(awsCfg)
		trendData, err := costSvc.GetLastSixMonthsCosts(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get cost trend: %v", err)), nil
		}

		resp := response.ConvertTrendData(trendData)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAWSUnusedVolumesHandler(region, profile string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		configSvc := awsconfig.NewService()
		awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to configure AWS: %v", err)), nil
		}

		ec2Svc := awsec2.NewService(awsCfg)
		volumes, err := ec2Svc.GetUnusedVolumes(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get unused volumes: %v", err)), nil
		}

		resp := response.ConvertUnusedVolumes(volumes)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAWSUnusedIPsHandler(region, profile string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		configSvc := awsconfig.NewService()
		awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to configure AWS: %v", err)), nil
		}

		ec2Svc := awsec2.NewService(awsCfg)
		ips, err := ec2Svc.GetUnusedIPs(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get unused IPs: %v", err)), nil
		}

		resp := response.ConvertUnusedIPs(ips)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAWSStoppedInstancesHandler(region, profile string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		configSvc := awsconfig.NewService()
		awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to configure AWS: %v", err)), nil
		}

		ec2Svc := awsec2.NewService(awsCfg)
		instances, attachedVolumes, err := ec2Svc.GetStoppedInstances(ctx)
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

func makeAWSExpiringReservationsHandler(region, profile string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		configSvc := awsconfig.NewService()
		awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to configure AWS: %v", err)), nil
		}

		ec2Svc := awsec2.NewService(awsCfg)
		reservations, err := ec2Svc.GetExpiringReservations(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get expiring reservations: %v", err)), nil
		}

		resp := response.ConvertReservations(reservations)
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeAWSWasteSummaryHandler(region, profile string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		configSvc := awsconfig.NewService()
		awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to configure AWS: %v", err)), nil
		}

		stsSvc := awssts.NewService(awsCfg)
		accountInfo, err := stsSvc.GetAccountInfo(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get account info: %v", err)), nil
		}

		ec2Svc := awsec2.NewService(awsCfg)

		unusedVolumes, err := ec2Svc.GetUnusedVolumes(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get unused volumes: %v", err)), nil
		}

		unusedIPs, err := ec2Svc.GetUnusedIPs(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get unused IPs: %v", err)), nil
		}

		stoppedInstances, attachedVolumes, err := ec2Svc.GetStoppedInstances(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stopped instances: %v", err)), nil
		}

		expiringReservations, err := ec2Svc.GetExpiringReservations(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get expiring reservations: %v", err)), nil
		}

		resp := response.WasteSummary{
			Provider:             "aws",
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
