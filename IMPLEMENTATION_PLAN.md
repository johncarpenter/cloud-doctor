# Implementation Plan: Multi-Cloud Support (GCP & Azure)

## Overview

Extend the existing AWS Doctor application to support Google Cloud Platform (GCP) and Microsoft Azure, enabling unified cloud cost analysis and waste detection across all three major cloud providers.

## Current State

The application currently supports AWS with:
- Month-over-month cost comparison by service
- 6-month spending trend analysis
- Waste detection (unused EBS, stopped EC2s, unassociated IPs, expiring RIs)

## Target Architecture

```
cloud-doctor/
├── app.go                      # Entry point with provider routing
├── model/
│   ├── flag.go                 # Extended with --provider flag
│   ├── cost.go                 # Shared cost models (unchanged)
│   ├── resource.go             # Generic resource waste model (new)
│   ├── aws.go                  # AWS-specific models (renamed from ec2.go)
│   ├── gcp.go                  # GCP-specific models (new)
│   └── azure.go                # Azure-specific models (new)
├── service/
│   ├── flag/                   # Extended flag parsing
│   ├── orchestrator/           # Provider-aware orchestration
│   ├── aws/
│   │   ├── config/             # AWS SDK config
│   │   ├── sts/                # AWS identity
│   │   ├── costexplorer/       # AWS billing
│   │   └── ec2/                # AWS compute/storage
│   ├── gcp/
│   │   ├── config/             # GCP SDK config (new)
│   │   ├── identity/           # GCP project info (new)
│   │   ├── billing/            # GCP Cloud Billing (new)
│   │   └── compute/            # GCP Compute Engine (new)
│   └── azure/
│       ├── config/             # Azure SDK config (new)
│       ├── identity/           # Azure subscription info (new)
│       ├── costmanagement/     # Azure Cost Management (new)
│       └── compute/            # Azure VMs/Disks (new)
└── utils/                      # Shared display (minimal changes)
```

---

## Stage 1: Refactor for Multi-Provider Architecture

**Goal**: Restructure the existing codebase to support multiple cloud providers without breaking AWS functionality.

**Success Criteria**:
- All existing AWS functionality works unchanged
- New `--provider` flag defaults to "aws"
- AWS services moved to `service/aws/` subdirectory
- Shared interfaces defined for cost and resource services

**Tasks**:

1. Add `--provider` flag to `model/flag.go`
   - Values: "aws", "gcp", "azure"
   - Default: "aws"

2. Create shared service interfaces in `service/interfaces.go`:
   ```go
   type IdentityService interface {
       GetAccountInfo(ctx context.Context) (*model.AccountInfo, error)
   }

   type CostService interface {
       GetCurrentMonthCostsByService(ctx context.Context) (*model.CostInfo, error)
       GetLastMonthCostsByService(ctx context.Context) (*model.CostInfo, error)
       GetCurrentMonthTotalCosts(ctx context.Context) (*string, error)
       GetLastMonthTotalCosts(ctx context.Context) (*string, error)
       GetLastSixMonthsCosts(ctx context.Context) ([]model.CostInfo, error)
   }

   type ResourceService interface {
       GetUnusedVolumes(ctx context.Context) ([]model.UnusedVolume, error)
       GetStoppedInstances(ctx context.Context) ([]model.StoppedInstance, error)
       GetUnusedIPs(ctx context.Context) ([]model.UnusedIP, error)
       GetExpiringReservations(ctx context.Context) ([]model.Reservation, error)
   }
   ```

3. Create generic resource models in `model/resource.go`:
   ```go
   type AccountInfo struct {
       Provider    string
       AccountID   string
       AccountName string
   }

   type UnusedVolume struct {
       ID       string
       SizeGB   int
       Created  time.Time
   }

   type StoppedInstance struct {
       ID          string
       Name        string
       StoppedDays int
       VolumeIDs   []string
   }

   type UnusedIP struct {
       Address      string
       ResourceType string
   }

   type Reservation struct {
       ID           string
       Type         string
       Status       string // "expiring", "expired"
       DaysUntil    int    // negative if expired
   }
   ```

4. Reorganize AWS services into `service/aws/` directory:
   - Move `service/aws_config/` → `service/aws/config/`
   - Move `service/sts/` → `service/aws/sts/`
   - Move `service/costexplorer/` → `service/aws/costexplorer/`
   - Move `service/ec2/` → `service/aws/ec2/`

5. Update AWS services to implement shared interfaces

6. Update orchestrator to use interfaces and accept provider parameter

7. Update `app.go` to route based on `--provider` flag

**Tests**:
- [x] `go build` succeeds
- [ ] `./aws-doctor` (no flags) shows AWS cost comparison
- [ ] `./aws-doctor --trend` shows AWS trend
- [ ] `./aws-doctor --waste` shows AWS waste
- [ ] `./aws-doctor --provider aws` works identically to no flag

**Status**: Complete (build verified, runtime tests require AWS credentials)

---

## Stage 2: GCP Cost Analysis

**Goal**: Implement cost comparison and trend analysis for Google Cloud Platform.

**Success Criteria**:
- `./aws-doctor --provider gcp` shows GCP cost comparison
- `./aws-doctor --provider gcp --trend` shows 6-month GCP spending trend
- Cost data grouped by GCP service (Compute Engine, Cloud Storage, etc.)

**Tasks**:

1. Add GCP SDK dependencies:
   ```
   go get cloud.google.com/go/billing
   go get google.golang.org/api/cloudbilling/v1
   ```

2. Create `service/gcp/config/service.go`:
   - Load GCP credentials from environment or `--credentials` flag
   - Support `--project` flag for GCP project ID
   - Use Application Default Credentials when available

3. Create `service/gcp/identity/service.go`:
   - Implement `IdentityService` interface
   - Return project ID and project name

4. Create `service/gcp/billing/service.go`:
   - Implement `CostService` interface
   - Use Cloud Billing API to query costs
   - Group by service (SKU description)
   - Map GCP date formats to internal model

5. Update `model/flag.go`:
   - Add `--project` flag for GCP project ID
   - Add `--credentials` flag for GCP service account JSON path

6. Update orchestrator to create GCP services when provider is "gcp"

7. Update display utils to show "GCP" branding when appropriate

**API Mapping**:
| AWS API | GCP Equivalent |
|---------|----------------|
| Cost Explorer GetCostAndUsage | Cloud Billing budgets.budgets.list or BigQuery export |
| Dimension: SERVICE | Group by: service.description |
| Metric: UnblendedCost | Cost: cost.amount |

**Tests**:
- [x] `go build` succeeds
- [ ] `./cloud-doctor --provider gcp --project <id> --billing-account <id>` shows cost table
- [ ] `./cloud-doctor --provider gcp --project <id> --billing-account <id> --trend` shows bar chart
- [ ] Costs are grouped by GCP service name
- [ ] Date ranges match AWS logic (fair comparison)

**Status**: Complete (build verified, runtime tests require GCP credentials and BigQuery billing export)

---

## Stage 3: GCP Waste Detection

**Goal**: Implement waste detection for Google Cloud Platform resources.

**Success Criteria**:
- `./aws-doctor --provider gcp --waste` detects unused GCP resources
- Checks: unused disks, stopped VMs, unassigned IPs, expiring CUDs

**Tasks**:

1. Add GCP Compute SDK:
   ```
   go get cloud.google.com/go/compute
   ```

2. Create `service/gcp/compute/service.go`:
   - Implement `ResourceService` interface

3. Implement `GetUnusedVolumes()`:
   - List Persistent Disks with `users` field empty
   - Filter: `status == "READY"` AND no attached instances
   - Return disk ID, size, creation time

4. Implement `GetStoppedInstances()`:
   - List instances with `status == "TERMINATED"`
   - Parse `lastStopTimestamp` to calculate days stopped
   - Filter for > 30 days stopped
   - Collect attached disk names

5. Implement `GetUnusedIPs()`:
   - List external IP addresses (global and regional)
   - Filter where `users` field is empty
   - Identify resource type from address description

6. Implement `GetExpiringReservations()`:
   - List Committed Use Discounts
   - Filter by `status == "ACTIVE"`
   - Check `endTimestamp` against 30-day threshold
   - Return CUD ID, type, days until expiration

7. Create `model/gcp.go` for any GCP-specific model extensions

**GCP API Mapping**:
| AWS Check | GCP Equivalent |
|-----------|----------------|
| Unused EBS | Persistent Disks with empty `users` |
| Stopped EC2 > 30d | VMs with `status=TERMINATED`, check `lastStopTimestamp` |
| Unassociated EIP | External IPs with empty `users` |
| Expiring RI | Committed Use Discounts near `endTimestamp` |

**Tests**:
- [x] `go build` succeeds
- [ ] `./cloud-doctor --provider gcp --project <id> --waste` runs without error
- [ ] Detects unattached persistent disks
- [ ] Detects VMs stopped > 30 days
- [ ] Detects unassigned external IPs
- [ ] Detects CUDs expiring within 30 days
- [ ] Shows green checkmark if no waste found

**Status**: Complete (build verified, runtime tests require GCP credentials)

---

## Stage 4: Azure Cost Analysis

**Goal**: Implement cost comparison and trend analysis for Microsoft Azure.

**Success Criteria**:
- `./aws-doctor --provider azure` shows Azure cost comparison
- `./aws-doctor --provider azure --trend` shows 6-month Azure spending trend
- Cost data grouped by Azure service (Virtual Machines, Storage, etc.)

**Tasks**:

1. Add Azure SDK dependencies:
   ```
   go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
   go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement
   ```

2. Create `service/azure/config/service.go`:
   - Load Azure credentials (DefaultAzureCredential)
   - Support `--subscription` flag for Azure subscription ID
   - Support `--tenant` flag for Azure tenant ID (optional)

3. Create `service/azure/identity/service.go`:
   - Implement `IdentityService` interface
   - Return subscription ID and subscription name

4. Create `service/azure/costmanagement/service.go`:
   - Implement `CostService` interface
   - Use Cost Management Query API
   - Group by ServiceName dimension
   - Map Azure date formats to internal model

5. Update `model/flag.go`:
   - Add `--subscription` flag for Azure subscription ID
   - Add `--tenant` flag for Azure tenant ID

6. Update orchestrator to create Azure services when provider is "azure"

**API Mapping**:
| AWS API | Azure Equivalent |
|---------|------------------|
| Cost Explorer GetCostAndUsage | Cost Management Query API |
| Dimension: SERVICE | Dimension: ServiceName |
| Metric: UnblendedCost | Column: Cost (PreTaxCost) |

**Tests**:
- [x] `go build` succeeds
- [ ] `./cloud-doctor --provider azure --subscription <id>` shows cost table
- [ ] `./cloud-doctor --provider azure --subscription <id> --trend` shows bar chart
- [ ] Costs are grouped by Azure service name
- [ ] Date ranges match AWS logic (fair comparison)

**Status**: Complete (build verified, runtime tests require Azure credentials)

---

## Stage 5: Azure Waste Detection

**Goal**: Implement waste detection for Microsoft Azure resources.

**Success Criteria**:
- `./aws-doctor --provider azure --waste` detects unused Azure resources
- Checks: unattached disks, deallocated VMs, unassociated IPs, expiring RIs

**Tasks**:

1. Add Azure Compute SDK:
   ```
   go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute
   go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork
   ```

2. Create `service/azure/compute/service.go`:
   - Implement `ResourceService` interface

3. Implement `GetUnusedVolumes()`:
   - List Managed Disks
   - Filter: `diskState == "Unattached"`
   - Return disk ID, size, creation time

4. Implement `GetStoppedInstances()`:
   - List Virtual Machines
   - Check instance view for `PowerState/deallocated`
   - Calculate days since deallocation from activity logs or tags
   - Filter for > 30 days deallocated
   - Collect attached disk IDs

5. Implement `GetUnusedIPs()`:
   - List Public IP Addresses
   - Filter where `ipConfiguration == nil`
   - Return IP address and allocation method

6. Implement `GetExpiringReservations()`:
   - List Reserved VM Instances via Reservations API
   - Filter by `provisioningState == "Succeeded"`
   - Check `expiryDate` against 30-day threshold
   - Return reservation ID, VM size, days until expiration

7. Create `model/azure.go` for any Azure-specific model extensions

**Azure API Mapping**:
| AWS Check | Azure Equivalent |
|-----------|------------------|
| Unused EBS | Managed Disks with `diskState=Unattached` |
| Stopped EC2 > 30d | VMs with `PowerState/deallocated` |
| Unassociated EIP | Public IPs with `ipConfiguration=nil` |
| Expiring RI | Reserved VM Instances near `expiryDate` |

**Tests**:
- [x] `go build` succeeds
- [ ] `./cloud-doctor --provider azure --subscription <id> --waste` runs without error
- [ ] Detects unattached managed disks
- [ ] Detects VMs deallocated (reports all deallocated VMs)
- [ ] Detects unassociated public IPs
- [ ] Detects reserved instances expiring within 30 days
- [ ] Shows green checkmark if no waste found

**Note**: Azure doesn't store deallocation timestamp directly. The implementation reports all deallocated VMs with `StoppedDays: -1` to indicate unknown duration. For production use, Activity Logs could be queried to determine actual deallocation time.

**Status**: Complete (build verified, runtime tests require Azure credentials)

---

## Stage 6: Multi-Cloud Unified View (Optional Enhancement)

**Goal**: Provide a combined view across all configured cloud providers.

**Success Criteria**:
- `./aws-doctor --provider all` shows combined cost summary
- `./aws-doctor --provider all --waste` shows waste across all clouds

**Tasks**:

1. Add `--provider all` option to flag parsing

2. Create multi-provider orchestration:
   - Run AWS, GCP, Azure analyses in parallel
   - Aggregate results into unified view

3. Update display utils for multi-cloud tables:
   - Add "Provider" column to cost tables
   - Group waste by provider, then by resource type

4. Handle partial failures gracefully:
   - Show results for configured providers
   - Skip providers without credentials
   - Display warnings for failed providers

**Tests**:
- [x] `go build` succeeds
- [ ] `./cloud-doctor --provider all` shows costs from all configured clouds
- [ ] `./cloud-doctor --provider all --trend` shows trends from all configured clouds
- [ ] `./cloud-doctor --provider all --waste` shows waste from all clouds
- [ ] Gracefully handles missing credentials for some providers
- [ ] Summary tables aggregate totals across providers

**Implementation Notes**:
- Providers are queried in parallel using goroutines
- Results are sorted consistently (AWS, GCP, Azure)
- Failed providers show error in summary but don't block other providers
- AWS is always attempted (default credentials)
- GCP requires --project and --billing-account flags
- Azure requires --subscription flag

**Status**: Complete (build verified, runtime tests require cloud credentials)

---

## Dependencies Summary

### GCP
```
cloud.google.com/go/billing
cloud.google.com/go/compute
google.golang.org/api/cloudbilling/v1
google.golang.org/api/compute/v1
```

### Azure
```
github.com/Azure/azure-sdk-for-go/sdk/azidentity
github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement
github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute
github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork
github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/reservations/armreservations
```

---

## CLI Reference (Post-Implementation)

```bash
# AWS (existing)
./aws-doctor --provider aws --region us-east-1 --profile myprofile
./aws-doctor --provider aws --trend
./aws-doctor --provider aws --waste

# GCP (new)
./aws-doctor --provider gcp --project my-project-id
./aws-doctor --provider gcp --project my-project-id --credentials /path/to/sa.json
./aws-doctor --provider gcp --project my-project-id --trend
./aws-doctor --provider gcp --project my-project-id --waste

# Azure (new)
./aws-doctor --provider azure --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
./aws-doctor --provider azure --subscription <id> --tenant <tenant-id>
./aws-doctor --provider azure --subscription <id> --trend
./aws-doctor --provider azure --subscription <id> --waste

# Multi-cloud (optional)
./aws-doctor --provider all
./aws-doctor --provider all --waste
```

---

## Risk Considerations

1. **GCP Billing API Complexity**: GCP's billing data often requires BigQuery export for detailed analysis. May need to support both direct API and BigQuery approaches.

2. **Azure Deallocated VM Timing**: Azure doesn't store deallocation timestamp directly. May need to query Activity Logs or use resource tags.

3. **Rate Limiting**: All three clouds have API rate limits. Implement exponential backoff for production use.

4. **Cost Data Delay**: Cloud billing data typically has 24-48 hour delay. Document this limitation.

5. **Permission Requirements**: Document minimum IAM/RBAC permissions needed for each provider.
