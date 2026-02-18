package tools

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/elC0mpa/aws-doctor/cmd/mcp/response"
	awsconfig "github.com/elC0mpa/aws-doctor/service/aws/config"
	awscostexplorer "github.com/elC0mpa/aws-doctor/service/aws/costexplorer"
	awsec2 "github.com/elC0mpa/aws-doctor/service/aws/ec2"
	awssts "github.com/elC0mpa/aws-doctor/service/aws/sts"
	azurecompute "github.com/elC0mpa/aws-doctor/service/azure/compute"
	azureconfig "github.com/elC0mpa/aws-doctor/service/azure/config"
	azurecostmanagement "github.com/elC0mpa/aws-doctor/service/azure/costmanagement"
	azureidentity "github.com/elC0mpa/aws-doctor/service/azure/identity"
	gcpbilling "github.com/elC0mpa/aws-doctor/service/gcp/billing"
	gcpcompute "github.com/elC0mpa/aws-doctor/service/gcp/compute"
	gcpidentity "github.com/elC0mpa/aws-doctor/service/gcp/identity"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterMultiCloudTools registers multi-cloud aggregate tools with the MCP server
func RegisterMultiCloudTools(s *server.MCPServer, awsRegion, awsProfile, gcpProjectID, gcpBillingAccount, azureSubscriptionID string) {
	// Multi-cloud cost summary
	s.AddTool(
		mcp.NewTool("multicloud_get_cost_summary",
			mcp.WithDescription("Get cost summary across all configured cloud providers (AWS, GCP, Azure). Shows current month vs last month comparison for each provider."),
		),
		makeMultiCloudCostSummaryHandler(awsRegion, awsProfile, gcpProjectID, gcpBillingAccount, azureSubscriptionID),
	)

	// Multi-cloud waste summary
	s.AddTool(
		mcp.NewTool("multicloud_get_waste_summary",
			mcp.WithDescription("Get waste detection summary across all configured cloud providers (AWS, GCP, Azure). Shows unused resources for each provider."),
		),
		makeMultiCloudWasteSummaryHandler(awsRegion, awsProfile, gcpProjectID, azureSubscriptionID),
	)
}

func makeMultiCloudCostSummaryHandler(awsRegion, awsProfile, gcpProjectID, gcpBillingAccount, azureSubscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var results []response.ProviderCostSummary
		var mu sync.Mutex
		var wg sync.WaitGroup

		// AWS (always available via default credential chain)
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := collectAWSCostSummary(ctx, awsRegion, awsProfile)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()

		// GCP (only if configured)
		if gcpProjectID != "" && gcpBillingAccount != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result := collectGCPCostSummary(ctx, gcpProjectID, gcpBillingAccount)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}()
		}

		// Azure (only if configured)
		if azureSubscriptionID != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result := collectAzureCostSummary(ctx, azureSubscriptionID)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}()
		}

		wg.Wait()

		// Calculate totals
		var totalCurrent, totalLast float64
		currency := "USD"
		for _, r := range results {
			if r.Error == "" {
				totalCurrent += r.CurrentMonthCost
				totalLast += r.LastMonthCost
				if r.Currency != "" {
					currency = r.Currency
				}
			}
		}

		resp := response.MultiCloudCostSummary{
			Providers: results,
			Total:     totalCurrent,
			Currency:  currency,
		}

		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeMultiCloudWasteSummaryHandler(awsRegion, awsProfile, gcpProjectID, azureSubscriptionID string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var results []response.WasteSummary
		var mu sync.Mutex
		var wg sync.WaitGroup

		// AWS
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := collectAWSWasteSummary(ctx, awsRegion, awsProfile)
			if result != nil {
				mu.Lock()
				results = append(results, *result)
				mu.Unlock()
			}
		}()

		// GCP (only if configured)
		if gcpProjectID != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result := collectGCPWasteSummary(ctx, gcpProjectID)
				if result != nil {
					mu.Lock()
					results = append(results, *result)
					mu.Unlock()
				}
			}()
		}

		// Azure (only if configured)
		if azureSubscriptionID != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result := collectAzureWasteSummary(ctx, azureSubscriptionID)
				if result != nil {
					mu.Lock()
					results = append(results, *result)
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		resp := response.MultiCloudWasteSummary{
			Providers: results,
		}

		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

// AWS cost collection
func collectAWSCostSummary(ctx context.Context, region, profile string) response.ProviderCostSummary {
	result := response.ProviderCostSummary{
		Provider: "aws",
		Currency: "USD",
	}

	configSvc := awsconfig.NewService()
	awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	stsSvc := awssts.NewService(awsCfg)
	accountInfo, err := stsSvc.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.AccountID = accountInfo.AccountID

	costSvc := awscostexplorer.NewService(awsCfg)

	currentData, err := costSvc.GetCurrentMonthCostsByService(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	currentCosts := response.ConvertCostInfo(currentData)
	result.CurrentMonthCost = currentCosts.Total
	result.Currency = currentCosts.Currency

	lastData, err := costSvc.GetLastMonthCostsByService(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	lastCosts := response.ConvertCostInfo(lastData)
	result.LastMonthCost = lastCosts.Total

	result.Difference = result.CurrentMonthCost - result.LastMonthCost
	if result.LastMonthCost > 0 {
		result.PercentChange = (result.Difference / result.LastMonthCost) * 100
	}

	return result
}

// GCP cost collection
func collectGCPCostSummary(ctx context.Context, projectID, billingAccount string) response.ProviderCostSummary {
	result := response.ProviderCostSummary{
		Provider: "gcp",
		Currency: "USD",
	}

	identitySvc, err := gcpidentity.NewService(ctx, projectID)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	accountInfo, err := identitySvc.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.AccountID = accountInfo.AccountID

	billingSvc, err := gcpbilling.NewService(ctx, projectID, billingAccount)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer billingSvc.Close()

	currentData, err := billingSvc.GetCurrentMonthCostsByService(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	currentCosts := response.ConvertCostInfo(currentData)
	result.CurrentMonthCost = currentCosts.Total
	result.Currency = currentCosts.Currency

	lastData, err := billingSvc.GetLastMonthCostsByService(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	lastCosts := response.ConvertCostInfo(lastData)
	result.LastMonthCost = lastCosts.Total

	result.Difference = result.CurrentMonthCost - result.LastMonthCost
	if result.LastMonthCost > 0 {
		result.PercentChange = (result.Difference / result.LastMonthCost) * 100
	}

	return result
}

// Azure cost collection
func collectAzureCostSummary(ctx context.Context, subscriptionID string) response.ProviderCostSummary {
	result := response.ProviderCostSummary{
		Provider: "azure",
		Currency: "USD",
	}

	cfgSvc, err := azureconfig.NewService(subscriptionID)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	identitySvc, err := azureidentity.NewService(subscriptionID, cfgSvc.GetCredential())
	if err != nil {
		result.Error = err.Error()
		return result
	}

	accountInfo, err := identitySvc.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.AccountID = accountInfo.AccountID

	costSvc, err := azurecostmanagement.NewService(subscriptionID, cfgSvc.GetCredential())
	if err != nil {
		result.Error = err.Error()
		return result
	}

	currentData, err := costSvc.GetCurrentMonthCostsByService(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	currentCosts := response.ConvertCostInfo(currentData)
	result.CurrentMonthCost = currentCosts.Total
	result.Currency = currentCosts.Currency

	lastData, err := costSvc.GetLastMonthCostsByService(ctx)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	lastCosts := response.ConvertCostInfo(lastData)
	result.LastMonthCost = lastCosts.Total

	result.Difference = result.CurrentMonthCost - result.LastMonthCost
	if result.LastMonthCost > 0 {
		result.PercentChange = (result.Difference / result.LastMonthCost) * 100
	}

	return result
}

// AWS waste collection
func collectAWSWasteSummary(ctx context.Context, region, profile string) *response.WasteSummary {
	configSvc := awsconfig.NewService()
	awsCfg, err := configSvc.GetAWSCfg(ctx, region, profile)
	if err != nil {
		return nil
	}

	stsSvc := awssts.NewService(awsCfg)
	accountInfo, err := stsSvc.GetAccountInfo(ctx)
	if err != nil {
		return nil
	}

	ec2Svc := awsec2.NewService(awsCfg)

	unusedVolumes, _ := ec2Svc.GetUnusedVolumes(ctx)
	unusedIPs, _ := ec2Svc.GetUnusedIPs(ctx)
	stoppedInstances, attachedVolumes, _ := ec2Svc.GetStoppedInstances(ctx)
	expiringReservations, _ := ec2Svc.GetExpiringReservations(ctx)

	return &response.WasteSummary{
		Provider:             "aws",
		AccountID:            accountInfo.AccountID,
		UnusedVolumes:        response.ConvertUnusedVolumes(unusedVolumes),
		AttachedVolumes:      response.ConvertUnusedVolumes(attachedVolumes),
		UnusedIPs:            response.ConvertUnusedIPs(unusedIPs),
		StoppedInstances:     response.ConvertStoppedInstances(stoppedInstances),
		ExpiringReservations: response.ConvertReservations(expiringReservations),
	}
}

// GCP waste collection
func collectGCPWasteSummary(ctx context.Context, projectID string) *response.WasteSummary {
	identitySvc, err := gcpidentity.NewService(ctx, projectID)
	if err != nil {
		return nil
	}

	accountInfo, err := identitySvc.GetAccountInfo(ctx)
	if err != nil {
		return nil
	}

	computeSvc, err := gcpcompute.NewService(ctx, projectID)
	if err != nil {
		return nil
	}

	unusedVolumes, _ := computeSvc.GetUnusedVolumes(ctx)
	unusedIPs, _ := computeSvc.GetUnusedIPs(ctx)
	stoppedInstances, attachedVolumes, _ := computeSvc.GetStoppedInstances(ctx)
	expiringReservations, _ := computeSvc.GetExpiringReservations(ctx)

	return &response.WasteSummary{
		Provider:             "gcp",
		AccountID:            accountInfo.AccountID,
		UnusedVolumes:        response.ConvertUnusedVolumes(unusedVolumes),
		AttachedVolumes:      response.ConvertUnusedVolumes(attachedVolumes),
		UnusedIPs:            response.ConvertUnusedIPs(unusedIPs),
		StoppedInstances:     response.ConvertStoppedInstances(stoppedInstances),
		ExpiringReservations: response.ConvertReservations(expiringReservations),
	}
}

// Azure waste collection
func collectAzureWasteSummary(ctx context.Context, subscriptionID string) *response.WasteSummary {
	cfgSvc, err := azureconfig.NewService(subscriptionID)
	if err != nil {
		return nil
	}

	identitySvc, err := azureidentity.NewService(subscriptionID, cfgSvc.GetCredential())
	if err != nil {
		return nil
	}

	accountInfo, err := identitySvc.GetAccountInfo(ctx)
	if err != nil {
		return nil
	}

	computeSvc, err := azurecompute.NewService(subscriptionID, cfgSvc.GetCredential())
	if err != nil {
		return nil
	}

	unusedVolumes, _ := computeSvc.GetUnusedVolumes(ctx)
	unusedIPs, _ := computeSvc.GetUnusedIPs(ctx)
	stoppedInstances, attachedVolumes, _ := computeSvc.GetStoppedInstances(ctx)
	expiringReservations, _ := computeSvc.GetExpiringReservations(ctx)

	return &response.WasteSummary{
		Provider:             "azure",
		AccountID:            accountInfo.AccountID,
		UnusedVolumes:        response.ConvertUnusedVolumes(unusedVolumes),
		AttachedVolumes:      response.ConvertUnusedVolumes(attachedVolumes),
		UnusedIPs:            response.ConvertUnusedIPs(unusedIPs),
		StoppedInstances:     response.ConvertStoppedInstances(stoppedInstances),
		ExpiringReservations: response.ConvertReservations(expiringReservations),
	}
}
