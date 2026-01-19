package service

import (
	"context"

	"github.com/elC0mpa/aws-doctor/model"
)

// IdentityService provides cloud account/project identity information
type IdentityService interface {
	GetAccountInfo(ctx context.Context) (*model.AccountInfo, error)
}

// CostService provides billing and cost analysis
type CostService interface {
	GetCurrentMonthCostsByService(ctx context.Context) (*model.CostInfo, error)
	GetLastMonthCostsByService(ctx context.Context) (*model.CostInfo, error)
	GetCurrentMonthTotalCosts(ctx context.Context) (*string, error)
	GetLastMonthTotalCosts(ctx context.Context) (*string, error)
	GetLastSixMonthsCosts(ctx context.Context) ([]model.CostInfo, error)
}

// ResourceService provides compute/storage waste detection
type ResourceService interface {
	GetUnusedVolumes(ctx context.Context) ([]model.UnusedVolume, error)
	GetUnusedIPs(ctx context.Context) ([]model.UnusedIP, error)
	GetStoppedInstances(ctx context.Context) ([]model.StoppedInstance, []model.UnusedVolume, error)
	GetExpiringReservations(ctx context.Context) ([]model.Reservation, error)
}
