package gcpconfig

import (
	"context"

	"golang.org/x/oauth2/google"
)

type service struct {
	projectID string
}

type ConfigService interface {
	GetCredentials(ctx context.Context) (*google.Credentials, error)
	GetProjectID() string
}
