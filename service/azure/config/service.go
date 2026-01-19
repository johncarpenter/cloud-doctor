package azureconfig

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func NewService(subscriptionID string) (*service, error) {
	// Use DefaultAzureCredential which supports:
	// - Environment variables (AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_CLIENT_SECRET)
	// - Managed Identity (on Azure VMs, App Service, etc.)
	// - Azure CLI (az login)
	// - Azure PowerShell
	// - Visual Studio Code
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	return &service{
		subscriptionID: subscriptionID,
		credential:     credential,
	}, nil
}

func (s *service) GetCredential() *azidentity.DefaultAzureCredential {
	return s.credential
}

func (s *service) GetSubscriptionID() string {
	return s.subscriptionID
}
