# Structural Changes for Multi-Cloud Support

This document tracks the structural changes made to support GCP and Azure alongside AWS.

## Stage 1: Refactor for Multi-Provider Architecture

### Directory Structure Changes

```
BEFORE:
service/
├── aws_config/          → MOVED to service/aws/config/
├── costexplorer/        → MOVED to service/aws/costexplorer/
├── ec2/                 → MOVED to service/aws/ec2/
├── elb/                 → MOVED to service/aws/elb/
├── sts/                 → MOVED to service/aws/sts/
├── flag/                → UNCHANGED
└── orchestrator/        → MODIFIED (uses interfaces)

AFTER:
service/
├── interfaces.go        → NEW: shared service interfaces
├── aws/
│   ├── config/          → AWS SDK configuration
│   ├── costexplorer/    → AWS Cost Explorer service
│   ├── ec2/             → AWS EC2/EBS service
│   ├── elb/             → AWS ELB service
│   └── sts/             → AWS STS service
├── gcp/                 → FUTURE: GCP services
├── azure/               → FUTURE: Azure services
├── flag/                → CLI flag parsing
└── orchestrator/        → Workflow orchestration
```

### Model Changes

```
model/
├── flag.go              → MODIFIED: added Provider field
├── cost.go              → MODIFIED: removed AWS SDK dependency
├── resource.go          → NEW: generic resource waste models
├── ec2.go               → RENAMED to aws.go (AWS-specific models)
├── gcp.go               → FUTURE: GCP-specific models
└── azure.go             → FUTURE: Azure-specific models
```

### New Shared Interfaces (service/interfaces.go)

```go
// IdentityService - cloud account/project identity
type IdentityService interface {
    GetAccountInfo(ctx context.Context) (*model.AccountInfo, error)
}

// CostService - billing and cost analysis
type CostService interface {
    GetCurrentMonthCostsByService(ctx context.Context) (*model.CostInfo, error)
    GetLastMonthCostsByService(ctx context.Context) (*model.CostInfo, error)
    GetCurrentMonthTotalCosts(ctx context.Context) (*string, error)
    GetLastMonthTotalCosts(ctx context.Context) (*string, error)
    GetLastSixMonthsCosts(ctx context.Context) ([]model.CostInfo, error)
}

// ResourceService - compute/storage waste detection
type ResourceService interface {
    GetUnusedVolumes(ctx context.Context) ([]model.UnusedVolume, error)
    GetUnusedIPs(ctx context.Context) ([]model.UnusedIP, error)
    GetStoppedInstances(ctx context.Context) ([]model.StoppedInstance, []model.UnusedVolume, error)
    GetExpiringReservations(ctx context.Context) ([]model.Reservation, error)
}
```

### New Generic Resource Models (model/resource.go)

```go
type AccountInfo struct {
    Provider    string
    AccountID   string
    AccountName string
}

type UnusedVolume struct {
    ID       string
    SizeGB   int32
    Status   string  // "available", "attached_stopped"
}

type StoppedInstance struct {
    ID          string
    Name        string
    StoppedDays int
}

type UnusedIP struct {
    Address      string
    AllocationID string
}

type Reservation struct {
    ID              string
    InstanceType    string
    Status          string  // "expiring", "expired"
    DaysUntilExpiry int
}
```

### Import Path Changes

All imports referencing AWS services need updating:

```go
// BEFORE
import awsconfig "github.com/elC0mpa/aws-doctor/service/aws_config"
import awscostexplorer "github.com/elC0mpa/aws-doctor/service/costexplorer"
import awsec2 "github.com/elC0mpa/aws-doctor/service/ec2"
import awssts "github.com/elC0mpa/aws-doctor/service/sts"

// AFTER
import awsconfig "github.com/elC0mpa/aws-doctor/service/aws/config"
import awscostexplorer "github.com/elC0mpa/aws-doctor/service/aws/costexplorer"
import awsec2 "github.com/elC0mpa/aws-doctor/service/aws/ec2"
import awssts "github.com/elC0mpa/aws-doctor/service/aws/sts"
```

### CLI Changes

```bash
# New flag added
--provider string    Cloud provider: aws, gcp, azure (default: "aws")

# Future flags (Stage 2+)
--project string     GCP project ID
--subscription string Azure subscription ID
```

### Breaking Changes

None - AWS functionality remains backward compatible:
- Default provider is "aws"
- All existing flags work unchanged
- Same output format

---

## Migration Checklist

- [x] Create CHANGES.md
- [x] Add Provider to model/flag.go
- [x] Create service/interfaces.go
- [x] Create model/resource.go
- [x] Refactor model/cost.go (remove AWS types.DateInterval)
- [x] Move service/aws_config → service/aws/config
- [x] Move service/costexplorer → service/aws/costexplorer
- [x] Move service/ec2 → service/aws/ec2
- [x] Move service/elb → service/aws/elb
- [x] Move service/sts → service/aws/sts
- [x] Update AWS services to implement shared interfaces
- [x] Update orchestrator to use interfaces
- [x] Update app.go for provider routing
- [x] Update utils to use generic models
- [x] Verify build passes
- [ ] Test AWS functionality unchanged (requires AWS credentials)

---

## Stage 2: GCP Cost Analysis

### New Directory Structure

```
service/gcp/
├── config/
│   ├── service.go       → GCP SDK configuration
│   └── types.go         → ConfigService interface
├── identity/
│   ├── service.go       → GCP project identity (implements IdentityService)
│   └── types.go         → IdentityService interface
└── billing/
    ├── service.go       → GCP billing via BigQuery (implements CostService)
    └── types.go         → BillingService interface
```

### New CLI Flags

```bash
--project string         GCP project ID (required for GCP)
--billing-account string GCP billing account ID (format: billingAccounts/XXXXXX-XXXXXX-XXXXXX)
```

### GCP Requirements

To use GCP cost analysis:
1. **Enable BigQuery billing export** in your GCP project
2. **Set up authentication** via one of:
   - `GOOGLE_APPLICATION_CREDENTIALS` environment variable
   - `gcloud auth application-default login`
   - Service account on GCE/Cloud Run/Cloud Functions
3. **Provide billing account ID**:
   ```bash
   gcloud billing accounts list
   ```

### Usage Examples

```bash
# GCP cost comparison
./cloud-doctor --provider gcp --project my-project-id --billing-account billingAccounts/XXXXXX-XXXXXX-XXXXXX

# GCP trend analysis
./cloud-doctor --provider gcp --project my-project-id --billing-account billingAccounts/XXXXXX-XXXXXX-XXXXXX --trend
```

### Multi-Cloud Branding Changes

- Banner changed from "AWS Doctor" to "Cloud Doctor"
- Cost table header changed from "AWS COST DIAGNOSIS" to "COST DIAGNOSIS"
- Trend chart header changed from "AWS DOCTOR TREND" to "COST TREND"
- Account label changed to "Account/Project ID"

### Migration Checklist

- [x] Add GCP SDK dependencies
- [x] Add --project and --billing-account flags
- [x] Create service/gcp/config
- [x] Create service/gcp/identity (implements IdentityService)
- [x] Create service/gcp/billing (implements CostService)
- [x] Update app.go with GCP routing
- [x] Update banner and display utilities
- [x] Verify build passes
- [ ] Test GCP functionality (requires GCP credentials and BigQuery export)

---

## Stage 3: GCP Waste Detection

### New Directory Structure

```
service/gcp/compute/
├── service.go       → GCP Compute Engine waste detection (implements ResourceService)
└── types.go         → ComputeService interface
```

### Waste Detection Checks

| Check | GCP Implementation |
|-------|-------------------|
| **Unused Volumes** | Persistent Disks with empty `users` field and status `READY` |
| **Stopped Instances** | VMs with status `TERMINATED` and `lastStopTimestamp` > 30 days ago |
| **Unused IPs** | External IPs (global and regional) with empty `users` and status `RESERVED` |
| **Expiring Reservations** | Committed Use Discounts with `endTimestamp` within 30 days |

### GCP API Calls

The compute service queries all zones and regions to find:
1. **Disks**: `compute.disks.list` in each zone
2. **Instances**: `compute.instances.list` with filter `status = TERMINATED`
3. **Addresses**: `compute.addresses.list` (regional) + `compute.globalAddresses.list`
4. **Commitments**: `compute.regionCommitments.list` in each region

### Usage Example

```bash
# GCP waste detection
./cloud-doctor --provider gcp --project my-project-id --waste
```

### Migration Checklist

- [x] Create service/gcp/compute/types.go
- [x] Implement GetUnusedVolumes (unattached Persistent Disks)
- [x] Implement GetStoppedInstances (TERMINATED VMs > 30 days)
- [x] Implement GetUnusedIPs (unassigned External IPs)
- [x] Implement GetExpiringReservations (Committed Use Discounts)
- [x] Update app.go to use GCP compute service for waste
- [x] Verify build passes
- [ ] Test GCP waste detection (requires GCP credentials)

---

## Stage 4: Azure Cost Analysis

### New Directory Structure

```
service/azure/
├── config/
│   ├── service.go       → Azure SDK configuration (DefaultAzureCredential)
│   └── types.go         → ConfigService interface
├── identity/
│   ├── service.go       → Azure subscription identity (implements IdentityService)
│   └── types.go         → IdentityService interface
└── costmanagement/
    ├── service.go       → Azure Cost Management API (implements CostService)
    └── types.go         → CostManagementService interface
```

### New CLI Flag

```bash
--subscription string    Azure subscription ID (required for Azure)
```

### Azure Authentication

The Azure implementation uses `DefaultAzureCredential` which supports:
- Environment variables (`AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_CLIENT_SECRET`)
- Managed Identity (on Azure VMs, App Service, etc.)
- Azure CLI (`az login`)
- Azure PowerShell
- Visual Studio Code

### Usage Examples

```bash
# Azure cost comparison
./cloud-doctor --provider azure --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# Azure trend analysis
./cloud-doctor --provider azure --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --trend
```

### Azure Cost Management API

The implementation uses the Azure Cost Management Query API:
- Queries costs grouped by `ServiceName` dimension
- Aggregates using `Cost` with `Sum` function
- Uses `Daily` granularity and aggregates in code for totals
- Supports custom time periods for fair comparison

### Migration Checklist

- [x] Add Azure SDK dependencies
- [x] Add --subscription flag
- [x] Create service/azure/config
- [x] Create service/azure/identity (implements IdentityService)
- [x] Create service/azure/costmanagement (implements CostService)
- [x] Update app.go with Azure routing
- [x] Verify build passes
- [ ] Test Azure functionality (requires Azure credentials)

---

## Stage 5: Azure Waste Detection

### New Directory Structure

```
service/azure/compute/
├── service.go       → Azure waste detection (implements ResourceService)
└── types.go         → ComputeService interface
```

### Waste Detection Checks

| Check | Azure Implementation |
|-------|---------------------|
| **Unused Volumes** | Managed Disks with `diskState = Unattached` |
| **Stopped Instances** | VMs with `PowerState/deallocated` status |
| **Unused IPs** | Public IP Addresses with `ipConfiguration = nil` |
| **Expiring Reservations** | Reserved VM Instances with `expiryDate` within 30 days |

### Azure API Calls

The compute service queries:
1. **Disks**: `armcompute.DisksClient.NewListPager()` - lists all Managed Disks
2. **Instances**: `armcompute.VirtualMachinesClient.NewListAllPager()` + `InstanceView()` for power state
3. **Public IPs**: `armnetwork.PublicIPAddressesClient.NewListAllPager()`
4. **Reservations**: `armreservations.ReservationOrderClient.NewListPager()`

### Azure SDK Dependencies Added

```
github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5
github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5
github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/reservations/armreservations
```

### Usage Example

```bash
# Azure waste detection
./cloud-doctor --provider azure --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --waste
```

### Known Limitations

- **Stopped Instance Duration**: Azure doesn't store the deallocation timestamp directly. To determine how long a VM has been stopped, you would need to query Activity Logs. The current implementation reports all deallocated VMs with `StoppedDays: -1` to indicate unknown duration.

### Migration Checklist

- [x] Add Azure Compute SDK dependencies
- [x] Create service/azure/compute/types.go
- [x] Implement GetUnusedVolumes (unattached Managed Disks)
- [x] Implement GetStoppedInstances (deallocated VMs)
- [x] Implement GetUnusedIPs (unassociated Public IPs)
- [x] Implement GetExpiringReservations (Reserved VM Instances)
- [x] Update app.go to use Azure compute service for waste
- [x] Verify build passes
- [ ] Test Azure waste detection (requires Azure credentials)

---

## Stage 6: Multi-Cloud Unified View

### New Features

The `--provider all` option enables simultaneous analysis across all configured cloud providers.

### New Models (model/resource.go)

```go
// ProviderCostResult - aggregates cost data for one provider
type ProviderCostResult struct {
    Provider         string
    AccountID        string
    CurrentMonthData *CostInfo
    LastMonthData    *CostInfo
    CurrentTotalCost string
    LastTotalCost    string
    TrendData        []CostInfo
    Error            error
}

// ProviderWasteResult - aggregates waste data for one provider
type ProviderWasteResult struct {
    Provider             string
    AccountID            string
    UnusedVolumes        []UnusedVolume
    AttachedVolumes      []UnusedVolume
    UnusedIPs            []UnusedIP
    StoppedInstances     []StoppedInstance
    ExpiringReservations []Reservation
    Error                error
}
```

### New Display Functions (utils/multicloud.go)

- `DrawMultiCloudCostTable()` - Summary table + per-provider cost details
- `DrawMultiCloudTrendChart()` - Per-provider trend charts
- `DrawMultiCloudWasteTable()` - Summary table + per-provider waste details
- `SortProviderCostResults()` - Consistent ordering (AWS, GCP, Azure)
- `SortProviderWasteResults()` - Consistent ordering (AWS, GCP, Azure)

### Usage Examples

```bash
# Multi-cloud cost comparison (requires credentials for each provider)
./cloud-doctor --provider all --project GCP_PROJECT --billing-account billingAccounts/XXX --subscription AZURE_SUBSCRIPTION

# Multi-cloud trend analysis
./cloud-doctor --provider all --project GCP_PROJECT --billing-account billingAccounts/XXX --subscription AZURE_SUBSCRIPTION --trend

# Multi-cloud waste detection
./cloud-doctor --provider all --project GCP_PROJECT --subscription AZURE_SUBSCRIPTION --waste
```

### Parallel Execution

The multi-cloud implementation uses goroutines to query all providers in parallel, reducing total execution time.

### Graceful Error Handling

- Providers without credentials are skipped automatically
- Failed providers show error message but don't stop other providers
- Summary tables indicate which providers failed

### Migration Checklist

- [x] Add --provider all option to flag parsing
- [x] Create ProviderCostResult and ProviderWasteResult models
- [x] Create utils/multicloud.go with multi-cloud display functions
- [x] Implement parallel execution with goroutines
- [x] Handle partial failures gracefully
- [x] Update app.go with runAll, runAllCosts, runAllTrend, runAllWaste
- [x] Verify build passes
- [ ] Test multi-cloud functionality (requires credentials for all providers)
