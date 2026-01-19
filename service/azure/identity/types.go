package azureidentity

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/elC0mpa/aws-doctor/model"
)

type service struct {
	subscriptionID string
	client         *armsubscriptions.Client
}

type IdentityService interface {
	GetAccountInfo(ctx context.Context) (*model.AccountInfo, error)
	GetSubscriptionInfo(ctx context.Context) (*armsubscriptions.Subscription, error)
}

// Credential is passed to allow reuse across services
type Credential = azidentity.DefaultAzureCredential
