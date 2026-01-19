package gcpidentity

import (
	"context"

	"github.com/elC0mpa/aws-doctor/model"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
)

func NewService(ctx context.Context, projectID string) (*service, error) {
	client, err := cloudresourcemanager.NewService(ctx, option.WithScopes(
		cloudresourcemanager.CloudPlatformReadOnlyScope,
	))
	if err != nil {
		return nil, err
	}

	return &service{
		projectID: projectID,
		client:    client,
	}, nil
}

// GetAccountInfo implements service.IdentityService
func (s *service) GetAccountInfo(ctx context.Context) (*model.AccountInfo, error) {
	project, err := s.GetProjectInfo(ctx)
	if err != nil {
		return nil, err
	}

	return &model.AccountInfo{
		Provider:    "gcp",
		AccountID:   s.projectID,
		AccountName: project.Name,
	}, nil
}

// GetProjectInfo returns detailed GCP project information
func (s *service) GetProjectInfo(ctx context.Context) (*cloudresourcemanager.Project, error) {
	return s.client.Projects.Get(s.projectID).Context(ctx).Do()
}
