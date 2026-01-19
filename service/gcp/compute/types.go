package gcpcompute

import (
	"context"

	"github.com/elC0mpa/aws-doctor/model"
	"google.golang.org/api/compute/v1"
)

type service struct {
	projectID     string
	computeClient *compute.Service
}

type ComputeService interface {
	// Generic interface methods (implements service.ResourceService)
	GetUnusedVolumes(ctx context.Context) ([]model.UnusedVolume, error)
	GetUnusedIPs(ctx context.Context) ([]model.UnusedIP, error)
	GetStoppedInstances(ctx context.Context) ([]model.StoppedInstance, []model.UnusedVolume, error)
	GetExpiringReservations(ctx context.Context) ([]model.Reservation, error)

	// GCP-specific methods for detailed information
	GetUnattachedDisks(ctx context.Context) ([]*compute.Disk, error)
	GetTerminatedVMs(ctx context.Context) ([]*compute.Instance, error)
	GetUnassignedExternalIPs(ctx context.Context) ([]*compute.Address, error)
	GetCommittedUseDiscounts(ctx context.Context) ([]*compute.Commitment, error)
}
