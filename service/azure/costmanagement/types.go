package azurecostmanagement

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement"
	"github.com/elC0mpa/aws-doctor/model"
)

type service struct {
	subscriptionID string
	client         *armcostmanagement.QueryClient
}

type CostManagementService interface {
	GetCurrentMonthCostsByService(ctx context.Context) (*model.CostInfo, error)
	GetLastMonthCostsByService(ctx context.Context) (*model.CostInfo, error)
	GetCurrentMonthTotalCosts(ctx context.Context) (*string, error)
	GetLastMonthTotalCosts(ctx context.Context) (*string, error)
	GetLastSixMonthsCosts(ctx context.Context) ([]model.CostInfo, error)
}

// Credential is passed to allow reuse across services
type Credential = azidentity.DefaultAzureCredential
