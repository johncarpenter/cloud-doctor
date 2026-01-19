package azureidentity

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/elC0mpa/aws-doctor/model"
)

func NewService(subscriptionID string, credential *Credential) (*service, error) {
	client, err := armsubscriptions.NewClient(credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriptions client: %w", err)
	}

	return &service{
		subscriptionID: subscriptionID,
		client:         client,
	}, nil
}

// GetAccountInfo implements service.IdentityService
func (s *service) GetAccountInfo(ctx context.Context) (*model.AccountInfo, error) {
	subscription, err := s.GetSubscriptionInfo(ctx)
	if err != nil {
		return nil, err
	}

	displayName := s.subscriptionID
	if subscription.DisplayName != nil {
		displayName = *subscription.DisplayName
	}

	return &model.AccountInfo{
		Provider:    "azure",
		AccountID:   s.subscriptionID,
		AccountName: displayName,
	}, nil
}

// GetSubscriptionInfo returns detailed Azure subscription information
func (s *service) GetSubscriptionInfo(ctx context.Context) (*armsubscriptions.Subscription, error) {
	resp, err := s.client.Get(ctx, s.subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription info: %w", err)
	}

	return &resp.Subscription, nil
}
