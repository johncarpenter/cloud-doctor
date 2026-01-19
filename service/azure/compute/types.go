package azurecompute

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/reservations/armreservations"
	"github.com/elC0mpa/aws-doctor/model"
)

type service struct {
	subscriptionID     string
	disksClient        *armcompute.DisksClient
	vmClient           *armcompute.VirtualMachinesClient
	publicIPClient     *armnetwork.PublicIPAddressesClient
	reservationsClient *armreservations.ReservationOrderClient
}

type ComputeService interface {
	// Generic interface methods (implements service.ResourceService)
	GetUnusedVolumes(ctx context.Context) ([]model.UnusedVolume, error)
	GetUnusedIPs(ctx context.Context) ([]model.UnusedIP, error)
	GetStoppedInstances(ctx context.Context) ([]model.StoppedInstance, []model.UnusedVolume, error)
	GetExpiringReservations(ctx context.Context) ([]model.Reservation, error)

	// Azure-specific methods for detailed information
	GetUnattachedDisks(ctx context.Context) ([]*armcompute.Disk, error)
	GetDeallocatedVMs(ctx context.Context) ([]*armcompute.VirtualMachine, error)
	GetUnassociatedPublicIPs(ctx context.Context) ([]*armnetwork.PublicIPAddress, error)
	GetReservedInstances(ctx context.Context) ([]*armreservations.ReservationOrderResponse, error)
}

// Credential is passed to allow reuse across services
type Credential = azidentity.DefaultAzureCredential
