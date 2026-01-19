package gcpconfig

import (
	"context"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
)

func NewService(projectID string) *service {
	return &service{
		projectID: projectID,
	}
}

func (s *service) GetCredentials(ctx context.Context) (*google.Credentials, error) {
	// Use Application Default Credentials
	// This supports:
	// - GOOGLE_APPLICATION_CREDENTIALS environment variable
	// - gcloud auth application-default login
	// - Service account on GCE/Cloud Run/Cloud Functions
	return google.FindDefaultCredentials(ctx,
		cloudbilling.CloudBillingScope,
		cloudresourcemanager.CloudPlatformReadOnlyScope,
		compute.ComputeReadonlyScope,
	)
}

func (s *service) GetProjectID() string {
	return s.projectID
}
