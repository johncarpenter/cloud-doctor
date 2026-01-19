package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/elC0mpa/aws-doctor/model"
	awsconfig "github.com/elC0mpa/aws-doctor/service/aws/config"
	awscostexplorer "github.com/elC0mpa/aws-doctor/service/aws/costexplorer"
	awsec2 "github.com/elC0mpa/aws-doctor/service/aws/ec2"
	awssts "github.com/elC0mpa/aws-doctor/service/aws/sts"
	azurecompute "github.com/elC0mpa/aws-doctor/service/azure/compute"
	azureconfig "github.com/elC0mpa/aws-doctor/service/azure/config"
	azurecostmanagement "github.com/elC0mpa/aws-doctor/service/azure/costmanagement"
	azureidentity "github.com/elC0mpa/aws-doctor/service/azure/identity"
	"github.com/elC0mpa/aws-doctor/service/flag"
	gcpbilling "github.com/elC0mpa/aws-doctor/service/gcp/billing"
	gcpcompute "github.com/elC0mpa/aws-doctor/service/gcp/compute"
	gcpidentity "github.com/elC0mpa/aws-doctor/service/gcp/identity"
	"github.com/elC0mpa/aws-doctor/service/orchestrator"
	"github.com/elC0mpa/aws-doctor/utils"
)

func main() {
	utils.DrawBanner()
	utils.StartSpinner()

	flagService := flag.NewService()
	flags, err := flagService.GetParsedFlags()
	if err != nil {
		panic(err)
	}

	switch flags.Provider {
	case "aws":
		err = runAWS(flags)
	case "gcp":
		err = runGCP(flags)
	case "azure":
		err = runAzure(flags)
	case "all":
		err = runAll(flags)
	default:
		utils.StopSpinner()
		fmt.Printf("Unknown provider: %s. Supported providers: aws, gcp, azure, all\n", flags.Provider)
		os.Exit(1)
	}

	if err != nil {
		utils.StopSpinner()
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func runAWS(flags model.Flags) error {
	cfgService := awsconfig.NewService()
	awsCfg, err := cfgService.GetAWSCfg(context.Background(), flags.Region, flags.Profile)
	if err != nil {
		return err
	}

	costService := awscostexplorer.NewService(awsCfg)
	stsService := awssts.NewService(awsCfg)
	ec2Service := awsec2.NewService(awsCfg)

	orchestratorService := orchestrator.NewService(stsService, costService, ec2Service)

	return orchestratorService.Orchestrate(flags)
}

func runGCP(flags model.Flags) error {
	ctx := context.Background()

	// Validate required GCP flags
	if flags.Project == "" {
		utils.StopSpinner()
		return fmt.Errorf("--project flag is required for GCP provider")
	}

	if flags.BillingAccount == "" && !flags.Waste {
		utils.StopSpinner()
		return fmt.Errorf("--billing-account flag is required for GCP cost analysis\n\nTo find your billing account ID:\n  gcloud billing accounts list\n\nUsage:\n  cloud-doctor --provider gcp --project PROJECT_ID --billing-account billingAccounts/XXXXXX-XXXXXX-XXXXXX")
	}

	// Create GCP identity service
	identityService, err := gcpidentity.NewService(ctx, flags.Project)
	if err != nil {
		return fmt.Errorf("failed to create GCP identity service: %w", err)
	}

	// Handle waste detection
	if flags.Waste {
		// Create GCP compute service for waste detection
		computeService, err := gcpcompute.NewService(ctx, flags.Project)
		if err != nil {
			return fmt.Errorf("failed to create GCP compute service: %w", err)
		}

		// Create orchestrator with identity and compute services (no billing needed)
		orchestratorService := orchestrator.NewService(identityService, nil, computeService)
		return orchestratorService.Orchestrate(flags)
	}

	// Handle cost analysis (default and trend)
	billingService, err := gcpbilling.NewService(ctx, flags.Project, flags.BillingAccount)
	if err != nil {
		return fmt.Errorf("failed to create GCP billing service: %w", err)
	}
	defer billingService.Close()

	// Create orchestrator with GCP services
	// Note: For cost analysis, we pass nil for resource service since it's not needed
	orchestratorService := orchestrator.NewService(identityService, billingService, nil)

	return orchestratorService.Orchestrate(flags)
}

func runAzure(flags model.Flags) error {
	// Validate required Azure flags
	if flags.Subscription == "" {
		utils.StopSpinner()
		return fmt.Errorf("--subscription flag is required for Azure provider\n\nTo find your subscription ID:\n  az account list --output table\n\nUsage:\n  cloud-doctor --provider azure --subscription SUBSCRIPTION_ID")
	}

	// Create Azure config service (handles authentication)
	cfgService, err := azureconfig.NewService(flags.Subscription)
	if err != nil {
		return fmt.Errorf("failed to create Azure config: %w", err)
	}

	// Create Azure identity service
	identityService, err := azureidentity.NewService(flags.Subscription, cfgService.GetCredential())
	if err != nil {
		return fmt.Errorf("failed to create Azure identity service: %w", err)
	}

	// Handle waste detection
	if flags.Waste {
		// Create Azure compute service for waste detection
		computeService, err := azurecompute.NewService(flags.Subscription, cfgService.GetCredential())
		if err != nil {
			return fmt.Errorf("failed to create Azure compute service: %w", err)
		}

		// Create orchestrator with identity and compute services (no cost service needed)
		orchestratorService := orchestrator.NewService(identityService, nil, computeService)
		return orchestratorService.Orchestrate(flags)
	}

	// Handle cost analysis (default and trend)
	costService, err := azurecostmanagement.NewService(flags.Subscription, cfgService.GetCredential())
	if err != nil {
		return fmt.Errorf("failed to create Azure cost management service: %w", err)
	}

	// Create orchestrator with Azure services
	// Note: For cost analysis, we pass nil for resource service since it's not needed
	orchestratorService := orchestrator.NewService(identityService, costService, nil)

	return orchestratorService.Orchestrate(flags)
}

func runAll(flags model.Flags) error {
	ctx := context.Background()

	if flags.Waste {
		return runAllWaste(ctx, flags)
	}

	if flags.Trend {
		return runAllTrend(ctx, flags)
	}

	return runAllCosts(ctx, flags)
}

func runAllCosts(ctx context.Context, flags model.Flags) error {
	var results []model.ProviderCostResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Run AWS
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := collectAWSCosts(ctx, flags)
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	}()

	// Run GCP (only if project and billing account are provided)
	if flags.Project != "" && flags.BillingAccount != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := collectGCPCosts(ctx, flags)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}

	// Run Azure (only if subscription is provided)
	if flags.Subscription != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := collectAzureCosts(ctx, flags)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}

	wg.Wait()
	utils.StopSpinner()

	if len(results) == 0 {
		return fmt.Errorf("no providers configured. Use --region/--profile for AWS, --project/--billing-account for GCP, --subscription for Azure")
	}

	utils.SortProviderCostResults(results)
	utils.DrawMultiCloudCostTable(results)

	return nil
}

func runAllTrend(ctx context.Context, flags model.Flags) error {
	var results []model.ProviderCostResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Run AWS
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := collectAWSTrend(ctx, flags)
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	}()

	// Run GCP (only if project and billing account are provided)
	if flags.Project != "" && flags.BillingAccount != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := collectGCPTrend(ctx, flags)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}

	// Run Azure (only if subscription is provided)
	if flags.Subscription != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := collectAzureTrend(ctx, flags)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}

	wg.Wait()
	utils.StopSpinner()

	if len(results) == 0 {
		return fmt.Errorf("no providers configured. Use --region/--profile for AWS, --project/--billing-account for GCP, --subscription for Azure")
	}

	utils.SortProviderCostResults(results)
	utils.DrawMultiCloudTrendChart(results)

	return nil
}

func runAllWaste(ctx context.Context, flags model.Flags) error {
	var results []model.ProviderWasteResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Run AWS
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := collectAWSWaste(ctx, flags)
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	}()

	// Run GCP (only if project is provided)
	if flags.Project != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := collectGCPWaste(ctx, flags)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}

	// Run Azure (only if subscription is provided)
	if flags.Subscription != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := collectAzureWaste(ctx, flags)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}

	wg.Wait()
	utils.StopSpinner()

	if len(results) == 0 {
		return fmt.Errorf("no providers configured. Use --region/--profile for AWS, --project for GCP, --subscription for Azure")
	}

	utils.SortProviderWasteResults(results)
	utils.DrawMultiCloudWasteTable(results)

	return nil
}

// AWS cost collectors
func collectAWSCosts(ctx context.Context, flags model.Flags) model.ProviderCostResult {
	result := model.ProviderCostResult{Provider: "aws"}

	cfgService := awsconfig.NewService()
	awsCfg, err := cfgService.GetAWSCfg(ctx, flags.Region, flags.Profile)
	if err != nil {
		result.Error = err
		return result
	}

	costService := awscostexplorer.NewService(awsCfg)
	stsService := awssts.NewService(awsCfg)

	accountInfo, err := stsService.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.AccountID = accountInfo.AccountID

	currentMonthData, err := costService.GetCurrentMonthCostsByService(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.CurrentMonthData = currentMonthData

	lastMonthData, err := costService.GetLastMonthCostsByService(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.LastMonthData = lastMonthData

	currentTotalCost, err := costService.GetCurrentMonthTotalCosts(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.CurrentTotalCost = *currentTotalCost

	lastTotalCost, err := costService.GetLastMonthTotalCosts(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.LastTotalCost = *lastTotalCost

	return result
}

func collectAWSTrend(ctx context.Context, flags model.Flags) model.ProviderCostResult {
	result := model.ProviderCostResult{Provider: "aws"}

	cfgService := awsconfig.NewService()
	awsCfg, err := cfgService.GetAWSCfg(ctx, flags.Region, flags.Profile)
	if err != nil {
		result.Error = err
		return result
	}

	costService := awscostexplorer.NewService(awsCfg)
	stsService := awssts.NewService(awsCfg)

	accountInfo, err := stsService.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.AccountID = accountInfo.AccountID

	trendData, err := costService.GetLastSixMonthsCosts(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.TrendData = trendData

	return result
}

func collectAWSWaste(ctx context.Context, flags model.Flags) model.ProviderWasteResult {
	result := model.ProviderWasteResult{Provider: "aws"}

	cfgService := awsconfig.NewService()
	awsCfg, err := cfgService.GetAWSCfg(ctx, flags.Region, flags.Profile)
	if err != nil {
		result.Error = err
		return result
	}

	stsService := awssts.NewService(awsCfg)
	ec2Service := awsec2.NewService(awsCfg)

	accountInfo, err := stsService.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.AccountID = accountInfo.AccountID

	unusedIPs, err := ec2Service.GetUnusedIPs(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.UnusedIPs = unusedIPs

	unusedVolumes, err := ec2Service.GetUnusedVolumes(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.UnusedVolumes = unusedVolumes

	stoppedInstances, attachedVolumes, err := ec2Service.GetStoppedInstances(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.StoppedInstances = stoppedInstances
	result.AttachedVolumes = attachedVolumes

	expiringReservations, err := ec2Service.GetExpiringReservations(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.ExpiringReservations = expiringReservations

	return result
}

// GCP cost collectors
func collectGCPCosts(ctx context.Context, flags model.Flags) model.ProviderCostResult {
	result := model.ProviderCostResult{Provider: "gcp"}

	identityService, err := gcpidentity.NewService(ctx, flags.Project)
	if err != nil {
		result.Error = err
		return result
	}

	billingService, err := gcpbilling.NewService(ctx, flags.Project, flags.BillingAccount)
	if err != nil {
		result.Error = err
		return result
	}
	defer billingService.Close()

	accountInfo, err := identityService.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.AccountID = accountInfo.AccountID

	currentMonthData, err := billingService.GetCurrentMonthCostsByService(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.CurrentMonthData = currentMonthData

	lastMonthData, err := billingService.GetLastMonthCostsByService(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.LastMonthData = lastMonthData

	currentTotalCost, err := billingService.GetCurrentMonthTotalCosts(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.CurrentTotalCost = *currentTotalCost

	lastTotalCost, err := billingService.GetLastMonthTotalCosts(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.LastTotalCost = *lastTotalCost

	return result
}

func collectGCPTrend(ctx context.Context, flags model.Flags) model.ProviderCostResult {
	result := model.ProviderCostResult{Provider: "gcp"}

	identityService, err := gcpidentity.NewService(ctx, flags.Project)
	if err != nil {
		result.Error = err
		return result
	}

	billingService, err := gcpbilling.NewService(ctx, flags.Project, flags.BillingAccount)
	if err != nil {
		result.Error = err
		return result
	}
	defer billingService.Close()

	accountInfo, err := identityService.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.AccountID = accountInfo.AccountID

	trendData, err := billingService.GetLastSixMonthsCosts(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.TrendData = trendData

	return result
}

func collectGCPWaste(ctx context.Context, flags model.Flags) model.ProviderWasteResult {
	result := model.ProviderWasteResult{Provider: "gcp"}

	identityService, err := gcpidentity.NewService(ctx, flags.Project)
	if err != nil {
		result.Error = err
		return result
	}

	computeService, err := gcpcompute.NewService(ctx, flags.Project)
	if err != nil {
		result.Error = err
		return result
	}

	accountInfo, err := identityService.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.AccountID = accountInfo.AccountID

	unusedIPs, err := computeService.GetUnusedIPs(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.UnusedIPs = unusedIPs

	unusedVolumes, err := computeService.GetUnusedVolumes(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.UnusedVolumes = unusedVolumes

	stoppedInstances, attachedVolumes, err := computeService.GetStoppedInstances(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.StoppedInstances = stoppedInstances
	result.AttachedVolumes = attachedVolumes

	expiringReservations, err := computeService.GetExpiringReservations(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.ExpiringReservations = expiringReservations

	return result
}

// Azure cost collectors
func collectAzureCosts(ctx context.Context, flags model.Flags) model.ProviderCostResult {
	result := model.ProviderCostResult{Provider: "azure"}

	cfgService, err := azureconfig.NewService(flags.Subscription)
	if err != nil {
		result.Error = err
		return result
	}

	identityService, err := azureidentity.NewService(flags.Subscription, cfgService.GetCredential())
	if err != nil {
		result.Error = err
		return result
	}

	costService, err := azurecostmanagement.NewService(flags.Subscription, cfgService.GetCredential())
	if err != nil {
		result.Error = err
		return result
	}

	accountInfo, err := identityService.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.AccountID = accountInfo.AccountID

	currentMonthData, err := costService.GetCurrentMonthCostsByService(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.CurrentMonthData = currentMonthData

	lastMonthData, err := costService.GetLastMonthCostsByService(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.LastMonthData = lastMonthData

	currentTotalCost, err := costService.GetCurrentMonthTotalCosts(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.CurrentTotalCost = *currentTotalCost

	lastTotalCost, err := costService.GetLastMonthTotalCosts(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.LastTotalCost = *lastTotalCost

	return result
}

func collectAzureTrend(ctx context.Context, flags model.Flags) model.ProviderCostResult {
	result := model.ProviderCostResult{Provider: "azure"}

	cfgService, err := azureconfig.NewService(flags.Subscription)
	if err != nil {
		result.Error = err
		return result
	}

	identityService, err := azureidentity.NewService(flags.Subscription, cfgService.GetCredential())
	if err != nil {
		result.Error = err
		return result
	}

	costService, err := azurecostmanagement.NewService(flags.Subscription, cfgService.GetCredential())
	if err != nil {
		result.Error = err
		return result
	}

	accountInfo, err := identityService.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.AccountID = accountInfo.AccountID

	trendData, err := costService.GetLastSixMonthsCosts(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.TrendData = trendData

	return result
}

func collectAzureWaste(ctx context.Context, flags model.Flags) model.ProviderWasteResult {
	result := model.ProviderWasteResult{Provider: "azure"}

	cfgService, err := azureconfig.NewService(flags.Subscription)
	if err != nil {
		result.Error = err
		return result
	}

	identityService, err := azureidentity.NewService(flags.Subscription, cfgService.GetCredential())
	if err != nil {
		result.Error = err
		return result
	}

	computeService, err := azurecompute.NewService(flags.Subscription, cfgService.GetCredential())
	if err != nil {
		result.Error = err
		return result
	}

	accountInfo, err := identityService.GetAccountInfo(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.AccountID = accountInfo.AccountID

	unusedIPs, err := computeService.GetUnusedIPs(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.UnusedIPs = unusedIPs

	unusedVolumes, err := computeService.GetUnusedVolumes(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.UnusedVolumes = unusedVolumes

	stoppedInstances, attachedVolumes, err := computeService.GetStoppedInstances(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.StoppedInstances = stoppedInstances
	result.AttachedVolumes = attachedVolumes

	expiringReservations, err := computeService.GetExpiringReservations(ctx)
	if err != nil {
		result.Error = err
		return result
	}
	result.ExpiringReservations = expiringReservations

	return result
}
