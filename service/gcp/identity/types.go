package gcpidentity

import (
	"context"

	"github.com/elC0mpa/aws-doctor/model"
	"google.golang.org/api/cloudresourcemanager/v1"
)

type service struct {
	projectID string
	client    *cloudresourcemanager.Service
}

type IdentityService interface {
	GetAccountInfo(ctx context.Context) (*model.AccountInfo, error)
	GetProjectInfo(ctx context.Context) (*cloudresourcemanager.Project, error)
}
