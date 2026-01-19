package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/elC0mpa/aws-doctor/model"
)

type service struct {
	client *ec2.Client
}

type EC2Service interface {
	// Legacy AWS-specific methods (for backward compatibility)
	GetElasticIpAddressesInfo(ctx context.Context) (*model.ElasticIpInfo, error)
	GetUnusedElasticIpAddressesInfo(ctx context.Context) ([]types.Address, error)
	GetUnusedEBSVolumes(ctx context.Context) ([]types.Volume, error)
	GetStoppedInstancesInfo(ctx context.Context) ([]types.Instance, []types.Volume, error)
	GetReservedInstanceExpiringOrExpired30DaysWaste(ctx context.Context) ([]model.RiExpirationInfo, error)

	// Generic interface methods (for multi-cloud support)
	GetUnusedVolumes(ctx context.Context) ([]model.UnusedVolume, error)
	GetUnusedIPs(ctx context.Context) ([]model.UnusedIP, error)
	GetStoppedInstances(ctx context.Context) ([]model.StoppedInstance, []model.UnusedVolume, error)
	GetExpiringReservations(ctx context.Context) ([]model.Reservation, error)
}
