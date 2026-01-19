package azurecompute

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/reservations/armreservations"
	"github.com/elC0mpa/aws-doctor/model"
)

func NewService(subscriptionID string, credential *Credential) (*service, error) {
	disksClient, err := armcompute.NewDisksClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create disks client: %w", err)
	}

	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM client: %w", err)
	}

	publicIPClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create public IP client: %w", err)
	}

	reservationsClient, err := armreservations.NewReservationOrderClient(credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create reservations client: %w", err)
	}

	return &service{
		subscriptionID:     subscriptionID,
		disksClient:        disksClient,
		vmClient:           vmClient,
		publicIPClient:     publicIPClient,
		reservationsClient: reservationsClient,
	}, nil
}

// GetUnusedVolumes implements service.ResourceService
// Returns Managed Disks that are not attached to any VM
func (s *service) GetUnusedVolumes(ctx context.Context) ([]model.UnusedVolume, error) {
	disks, err := s.GetUnattachedDisks(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.UnusedVolume, 0, len(disks))
	for _, disk := range disks {
		var sizeGB int32
		if disk.Properties != nil && disk.Properties.DiskSizeGB != nil {
			sizeGB = *disk.Properties.DiskSizeGB
		}

		name := ""
		if disk.Name != nil {
			name = *disk.Name
		}

		result = append(result, model.UnusedVolume{
			ID:     name,
			SizeGB: sizeGB,
			Status: "available",
		})
	}
	return result, nil
}

// GetUnattachedDisks returns all Managed Disks that are unattached
func (s *service) GetUnattachedDisks(ctx context.Context) ([]*armcompute.Disk, error) {
	var unattachedDisks []*armcompute.Disk

	pager := s.disksClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list disks: %w", err)
		}

		for _, disk := range page.Value {
			// A disk is unattached if DiskState is "Unattached"
			if disk.Properties != nil && disk.Properties.DiskState != nil {
				if *disk.Properties.DiskState == armcompute.DiskStateUnattached {
					unattachedDisks = append(unattachedDisks, disk)
				}
			}
		}
	}

	return unattachedDisks, nil
}

// GetStoppedInstances implements service.ResourceService
// Returns VMs that are deallocated
// Note: Azure doesn't store deallocation timestamp directly, so we report all deallocated VMs
// In a production implementation, you might query Activity Logs for the actual deallocation time
func (s *service) GetStoppedInstances(ctx context.Context) ([]model.StoppedInstance, []model.UnusedVolume, error) {
	vms, err := s.GetDeallocatedVMs(ctx)
	if err != nil {
		return nil, nil, err
	}

	var stoppedInstances []model.StoppedInstance
	var attachedVolumes []model.UnusedVolume

	for _, vm := range vms {
		name := ""
		if vm.Name != nil {
			name = *vm.Name
		}

		// For deallocated VMs, we report days as -1 since Azure doesn't provide timestamp
		// without querying Activity Logs
		stoppedInstances = append(stoppedInstances, model.StoppedInstance{
			ID:          name,
			Name:        name,
			StoppedDays: -1, // Unknown - would need Activity Log query
		})

		// Collect attached disks
		if vm.Properties != nil && vm.Properties.StorageProfile != nil {
			// OS Disk
			if vm.Properties.StorageProfile.OSDisk != nil &&
				vm.Properties.StorageProfile.OSDisk.ManagedDisk != nil &&
				vm.Properties.StorageProfile.OSDisk.ManagedDisk.ID != nil {
				diskName := extractResourceName(*vm.Properties.StorageProfile.OSDisk.ManagedDisk.ID)
				var sizeGB int32
				if vm.Properties.StorageProfile.OSDisk.DiskSizeGB != nil {
					sizeGB = *vm.Properties.StorageProfile.OSDisk.DiskSizeGB
				}
				attachedVolumes = append(attachedVolumes, model.UnusedVolume{
					ID:     diskName,
					SizeGB: sizeGB,
					Status: "attached_stopped",
				})
			}

			// Data Disks
			for _, dataDisk := range vm.Properties.StorageProfile.DataDisks {
				if dataDisk.ManagedDisk != nil && dataDisk.ManagedDisk.ID != nil {
					diskName := extractResourceName(*dataDisk.ManagedDisk.ID)
					var sizeGB int32
					if dataDisk.DiskSizeGB != nil {
						sizeGB = *dataDisk.DiskSizeGB
					}
					attachedVolumes = append(attachedVolumes, model.UnusedVolume{
						ID:     diskName,
						SizeGB: sizeGB,
						Status: "attached_stopped",
					})
				}
			}
		}
	}

	return stoppedInstances, attachedVolumes, nil
}

// GetDeallocatedVMs returns all VMs in deallocated state
func (s *service) GetDeallocatedVMs(ctx context.Context) ([]*armcompute.VirtualMachine, error) {
	var deallocatedVMs []*armcompute.VirtualMachine

	// List all VMs with instance view to get power state
	pager := s.vmClient.NewListAllPager(&armcompute.VirtualMachinesClientListAllOptions{
		StatusOnly: nil,
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list VMs: %w", err)
		}

		for _, vm := range page.Value {
			if vm.ID == nil {
				continue
			}

			// Extract resource group from VM ID
			resourceGroup := extractResourceGroup(*vm.ID)
			vmName := ""
			if vm.Name != nil {
				vmName = *vm.Name
			}

			// Get instance view to check power state
			instanceView, err := s.vmClient.InstanceView(ctx, resourceGroup, vmName, nil)
			if err != nil {
				// Skip VMs we can't get instance view for
				continue
			}

			// Check if VM is deallocated
			isDeallocated := false
			if instanceView.Statuses != nil {
				for _, status := range instanceView.Statuses {
					if status.Code != nil && strings.HasPrefix(*status.Code, "PowerState/deallocated") {
						isDeallocated = true
						break
					}
				}
			}

			if isDeallocated {
				deallocatedVMs = append(deallocatedVMs, vm)
			}
		}
	}

	return deallocatedVMs, nil
}

// GetUnusedIPs implements service.ResourceService
// Returns Public IP addresses that are not associated with any resource
func (s *service) GetUnusedIPs(ctx context.Context) ([]model.UnusedIP, error) {
	ips, err := s.GetUnassociatedPublicIPs(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.UnusedIP, 0, len(ips))
	for _, ip := range ips {
		address := ""
		if ip.Properties != nil && ip.Properties.IPAddress != nil {
			address = *ip.Properties.IPAddress
		}

		name := ""
		if ip.Name != nil {
			name = *ip.Name
		}

		result = append(result, model.UnusedIP{
			Address:      address,
			AllocationID: name,
		})
	}
	return result, nil
}

// GetUnassociatedPublicIPs returns all Public IPs not associated with any resource
func (s *service) GetUnassociatedPublicIPs(ctx context.Context) ([]*armnetwork.PublicIPAddress, error) {
	var unassociatedIPs []*armnetwork.PublicIPAddress

	pager := s.publicIPClient.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list public IPs: %w", err)
		}

		for _, ip := range page.Value {
			// A Public IP is unassociated if IPConfiguration is nil
			if ip.Properties != nil && ip.Properties.IPConfiguration == nil {
				unassociatedIPs = append(unassociatedIPs, ip)
			}
		}
	}

	return unassociatedIPs, nil
}

// GetExpiringReservations implements service.ResourceService
// Returns Reserved VM Instances that are expiring soon or recently expired
func (s *service) GetExpiringReservations(ctx context.Context) ([]model.Reservation, error) {
	reservationOrders, err := s.GetReservedInstances(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	next30Days := now.Add(30 * 24 * time.Hour)
	prev30Days := now.Add(-30 * 24 * time.Hour)

	var result []model.Reservation

	for _, order := range reservationOrders {
		if order.Properties == nil {
			continue
		}

		name := ""
		if order.Name != nil {
			name = *order.Name
		}

		displayName := ""
		if order.Properties.DisplayName != nil {
			displayName = *order.Properties.DisplayName
		}

		// Check expiry date
		if order.Properties.ExpiryDate != nil {
			expiryTime := *order.Properties.ExpiryDate
			daysDiff := int(expiryTime.Sub(now).Hours() / 24)

			// Check if reservation is expiring within 30 days
			if order.Properties.ProvisioningState != nil &&
				*order.Properties.ProvisioningState == armreservations.ProvisioningStateSucceeded &&
				expiryTime.Before(next30Days) && expiryTime.After(now) {
				result = append(result, model.Reservation{
					ID:              name,
					InstanceType:    displayName,
					Status:          "expiring",
					DaysUntilExpiry: daysDiff,
				})
			}

			// Check if recently expired (within last 30 days)
			if expiryTime.After(prev30Days) && expiryTime.Before(now) {
				result = append(result, model.Reservation{
					ID:              name,
					InstanceType:    displayName,
					Status:          "expired",
					DaysUntilExpiry: daysDiff,
				})
			}
		}
	}

	return result, nil
}

// GetReservedInstances returns all Reserved VM Instances for the subscription
func (s *service) GetReservedInstances(ctx context.Context) ([]*armreservations.ReservationOrderResponse, error) {
	var allReservations []*armreservations.ReservationOrderResponse

	// List all reservation orders
	pager := s.reservationsClient.NewListPager(nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			// Reservations API might not be available or have permissions
			// Return empty list instead of error
			return allReservations, nil
		}

		allReservations = append(allReservations, page.Value...)
	}

	return allReservations, nil
}

// extractResourceName extracts the resource name from an Azure resource ID
// e.g., "/subscriptions/.../resourceGroups/.../providers/Microsoft.Compute/disks/my-disk"
// returns "my-disk"
func extractResourceName(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return resourceID
}

// extractResourceGroup extracts the resource group from an Azure resource ID
func extractResourceGroup(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "resourceGroups") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
