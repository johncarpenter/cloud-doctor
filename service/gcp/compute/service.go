package gcpcompute

import (
	"context"
	"fmt"
	"time"

	"github.com/elC0mpa/aws-doctor/model"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

func NewService(ctx context.Context, projectID string) (*service, error) {
	computeClient, err := compute.NewService(ctx, option.WithScopes(
		compute.ComputeReadonlyScope,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create Compute client: %w", err)
	}

	return &service{
		projectID:     projectID,
		computeClient: computeClient,
	}, nil
}

// GetUnusedVolumes implements service.ResourceService
// Returns persistent disks that are not attached to any instance
func (s *service) GetUnusedVolumes(ctx context.Context) ([]model.UnusedVolume, error) {
	disks, err := s.GetUnattachedDisks(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.UnusedVolume, 0, len(disks))
	for _, disk := range disks {
		result = append(result, model.UnusedVolume{
			ID:     disk.Name,
			SizeGB: int32(disk.SizeGb),
			Status: "available",
		})
	}
	return result, nil
}

// GetUnattachedDisks returns all persistent disks that are not attached to any instance
func (s *service) GetUnattachedDisks(ctx context.Context) ([]*compute.Disk, error) {
	var unattachedDisks []*compute.Disk

	// List all zones in the project
	zonesResp, err := s.computeClient.Zones.List(s.projectID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	// Check disks in each zone
	for _, zone := range zonesResp.Items {
		disksResp, err := s.computeClient.Disks.List(s.projectID, zone.Name).Context(ctx).Do()
		if err != nil {
			// Skip zones with errors (might not have disks API enabled)
			continue
		}

		for _, disk := range disksResp.Items {
			// A disk is unattached if it has no users
			if len(disk.Users) == 0 && disk.Status == "READY" {
				unattachedDisks = append(unattachedDisks, disk)
			}
		}
	}

	return unattachedDisks, nil
}

// GetStoppedInstances implements service.ResourceService
// Returns VMs that have been stopped (TERMINATED) for more than 30 days
func (s *service) GetStoppedInstances(ctx context.Context) ([]model.StoppedInstance, []model.UnusedVolume, error) {
	instances, err := s.GetTerminatedVMs(ctx)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	thresholdTime := now.Add(-30 * 24 * time.Hour)

	var stoppedInstances []model.StoppedInstance
	var attachedVolumes []model.UnusedVolume

	for _, instance := range instances {
		// Parse the lastStopTimestamp
		var stoppedAt time.Time
		if instance.LastStopTimestamp != "" {
			stoppedAt, err = time.Parse(time.RFC3339, instance.LastStopTimestamp)
			if err != nil {
				// If we can't parse the timestamp, skip this instance
				continue
			}
		} else {
			// If no stop timestamp, use creation time as fallback
			stoppedAt, err = time.Parse(time.RFC3339, instance.CreationTimestamp)
			if err != nil {
				continue
			}
		}

		// Only include instances stopped for more than 30 days
		if stoppedAt.Before(thresholdTime) {
			days := int(now.Sub(stoppedAt).Hours() / 24)

			stoppedInstances = append(stoppedInstances, model.StoppedInstance{
				ID:          instance.Name,
				Name:        instance.Name,
				StoppedDays: days,
			})

			// Collect attached disks
			for _, disk := range instance.Disks {
				if disk.Source != "" {
					// Extract disk name from source URL
					diskName := extractResourceName(disk.Source)
					attachedVolumes = append(attachedVolumes, model.UnusedVolume{
						ID:     diskName,
						SizeGB: int32(disk.DiskSizeGb),
						Status: "attached_stopped",
					})
				}
			}
		}
	}

	return stoppedInstances, attachedVolumes, nil
}

// GetTerminatedVMs returns all VMs in TERMINATED state
func (s *service) GetTerminatedVMs(ctx context.Context) ([]*compute.Instance, error) {
	var terminatedVMs []*compute.Instance

	// List all zones in the project
	zonesResp, err := s.computeClient.Zones.List(s.projectID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	// Check instances in each zone
	for _, zone := range zonesResp.Items {
		// Filter for TERMINATED instances
		instancesResp, err := s.computeClient.Instances.List(s.projectID, zone.Name).
			Filter("status = TERMINATED").
			Context(ctx).Do()
		if err != nil {
			// Skip zones with errors
			continue
		}

		terminatedVMs = append(terminatedVMs, instancesResp.Items...)
	}

	return terminatedVMs, nil
}

// GetUnusedIPs implements service.ResourceService
// Returns external IP addresses that are not assigned to any resource
func (s *service) GetUnusedIPs(ctx context.Context) ([]model.UnusedIP, error) {
	addresses, err := s.GetUnassignedExternalIPs(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.UnusedIP, 0, len(addresses))
	for _, addr := range addresses {
		result = append(result, model.UnusedIP{
			Address:      addr.Address,
			AllocationID: addr.Name,
		})
	}
	return result, nil
}

// GetUnassignedExternalIPs returns all external IPs not assigned to any resource
func (s *service) GetUnassignedExternalIPs(ctx context.Context) ([]*compute.Address, error) {
	var unassignedIPs []*compute.Address

	// Check global addresses
	globalResp, err := s.computeClient.GlobalAddresses.List(s.projectID).Context(ctx).Do()
	if err == nil {
		for _, addr := range globalResp.Items {
			// An address is unassigned if it has no users and status is RESERVED
			if len(addr.Users) == 0 && addr.Status == "RESERVED" {
				unassignedIPs = append(unassignedIPs, addr)
			}
		}
	}

	// Check regional addresses
	regionsResp, err := s.computeClient.Regions.List(s.projectID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	for _, region := range regionsResp.Items {
		addressesResp, err := s.computeClient.Addresses.List(s.projectID, region.Name).Context(ctx).Do()
		if err != nil {
			// Skip regions with errors
			continue
		}

		for _, addr := range addressesResp.Items {
			// An address is unassigned if it has no users and status is RESERVED
			if len(addr.Users) == 0 && addr.Status == "RESERVED" {
				unassignedIPs = append(unassignedIPs, addr)
			}
		}
	}

	return unassignedIPs, nil
}

// GetExpiringReservations implements service.ResourceService
// Returns Committed Use Discounts (CUDs) that are expiring soon or recently expired
func (s *service) GetExpiringReservations(ctx context.Context) ([]model.Reservation, error) {
	commitments, err := s.GetCommittedUseDiscounts(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	next30Days := now.Add(30 * 24 * time.Hour)
	prev30Days := now.Add(-30 * 24 * time.Hour)

	var result []model.Reservation

	for _, commitment := range commitments {
		// Parse end time
		endTime, err := time.Parse(time.RFC3339, commitment.EndTimestamp)
		if err != nil {
			continue
		}

		daysDiff := int(endTime.Sub(now).Hours() / 24)

		// Check if expiring within 30 days
		if commitment.Status == "ACTIVE" && endTime.Before(next30Days) && endTime.After(now) {
			result = append(result, model.Reservation{
				ID:              commitment.Name,
				InstanceType:    commitment.Type,
				Status:          "expiring",
				DaysUntilExpiry: daysDiff,
			})
		}

		// Check if recently expired (within last 30 days)
		if endTime.After(prev30Days) && endTime.Before(now) {
			result = append(result, model.Reservation{
				ID:              commitment.Name,
				InstanceType:    commitment.Type,
				Status:          "expired",
				DaysUntilExpiry: daysDiff,
			})
		}
	}

	return result, nil
}

// GetCommittedUseDiscounts returns all Committed Use Discounts in the project
func (s *service) GetCommittedUseDiscounts(ctx context.Context) ([]*compute.Commitment, error) {
	var allCommitments []*compute.Commitment

	// List all regions
	regionsResp, err := s.computeClient.Regions.List(s.projectID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	// Check commitments in each region
	for _, region := range regionsResp.Items {
		commitmentsResp, err := s.computeClient.RegionCommitments.List(s.projectID, region.Name).Context(ctx).Do()
		if err != nil {
			// Skip regions with errors (might not have commitments)
			continue
		}

		allCommitments = append(allCommitments, commitmentsResp.Items...)
	}

	return allCommitments, nil
}

// extractResourceName extracts the resource name from a GCP resource URL
// e.g., "https://compute.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/disks/my-disk"
// returns "my-disk"
func extractResourceName(resourceURL string) string {
	// Find the last "/" and return everything after it
	for i := len(resourceURL) - 1; i >= 0; i-- {
		if resourceURL[i] == '/' {
			return resourceURL[i+1:]
		}
	}
	return resourceURL
}
