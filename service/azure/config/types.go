package azureconfig

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type service struct {
	subscriptionID string
	credential     *azidentity.DefaultAzureCredential
}

type ConfigService interface {
	GetCredential() *azidentity.DefaultAzureCredential
	GetSubscriptionID() string
}
