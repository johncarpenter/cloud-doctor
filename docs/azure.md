# Getting Started with Azure

This guide walks you through setting up Cloud Doctor to analyze your Microsoft Azure subscription for cost analysis and waste detection.

## Prerequisites

- An Azure account with at least one subscription
- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) installed (recommended)
- Appropriate Azure RBAC permissions

## Quick Start

```bash
# Cost comparison (current month vs last month)
./cloud-doctor --provider azure --subscription YOUR_SUBSCRIPTION_ID

# 6-month trend analysis
./cloud-doctor --provider azure --subscription YOUR_SUBSCRIPTION_ID --trend

# Waste detection
./cloud-doctor --provider azure --subscription YOUR_SUBSCRIPTION_ID --waste
```

## Step 1: Find Your Subscription ID

### Using Azure CLI

```bash
# Login to Azure
az login

# List your subscriptions
az account list --output table

# Output example:
# Name                  CloudName    SubscriptionId                        State    IsDefault
# --------------------  -----------  ------------------------------------  -------  ----------
# Pay-As-You-Go         AzureCloud   xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx  Enabled  True
# Development           AzureCloud   yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy  Enabled  False
```

### Using Azure Portal

1. Go to [Azure Portal](https://portal.azure.com)
2. Search for "Subscriptions" in the top search bar
3. Click on your subscription
4. Copy the **Subscription ID** from the overview page

## Step 2: Authentication

Cloud Doctor uses Azure's `DefaultAzureCredential`, which supports multiple authentication methods automatically.

### Option A: Azure CLI (Recommended for Development)

```bash
# Login with your Azure account
az login

# Verify you're logged in
az account show

# Set your default subscription (optional)
az account set --subscription YOUR_SUBSCRIPTION_ID
```

### Option B: Service Principal (Production/CI)

Create a service principal for automated environments:

```bash
# Create a service principal with Reader role
az ad sp create-for-rbac \
  --name "cloud-doctor" \
  --role "Reader" \
  --scopes /subscriptions/YOUR_SUBSCRIPTION_ID

# Output:
# {
#   "appId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
#   "displayName": "cloud-doctor",
#   "password": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
#   "tenant": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
# }

# Set environment variables
export AZURE_CLIENT_ID="<appId>"
export AZURE_CLIENT_SECRET="<password>"
export AZURE_TENANT_ID="<tenant>"
```

### Option C: Managed Identity (Azure VMs/App Service)

If running on Azure infrastructure, use managed identity:

```bash
# Enable system-assigned managed identity on a VM
az vm identity assign \
  --resource-group myResourceGroup \
  --name myVM

# Grant the identity access to the subscription
az role assignment create \
  --assignee <principal-id> \
  --role "Reader" \
  --scope /subscriptions/YOUR_SUBSCRIPTION_ID
```

No environment variables needed—the SDK automatically uses the managed identity.

### Option D: Visual Studio Code / Azure PowerShell

If you're authenticated via VS Code Azure extension or Azure PowerShell, the SDK will use those credentials automatically.

## Step 3: Set Up RBAC Permissions

### Minimum Permissions for Cost Analysis

The authenticated identity needs these roles:

| Role | Scope | Purpose |
|------|-------|---------|
| `Cost Management Reader` | Subscription | Query cost data |
| `Reader` | Subscription | Read subscription metadata |

```bash
# Grant Cost Management Reader
az role assignment create \
  --assignee "user@example.com" \
  --role "Cost Management Reader" \
  --scope /subscriptions/YOUR_SUBSCRIPTION_ID

# Grant Reader (if not already assigned)
az role assignment create \
  --assignee "user@example.com" \
  --role "Reader" \
  --scope /subscriptions/YOUR_SUBSCRIPTION_ID
```

### Minimum Permissions for Waste Detection

| Role | Scope | Purpose |
|------|-------|---------|
| `Reader` | Subscription | List VMs, disks, IPs |
| `Reservations Reader` | Tenant (optional) | View reserved instances |

```bash
# Reader role covers compute resources
az role assignment create \
  --assignee "user@example.com" \
  --role "Reader" \
  --scope /subscriptions/YOUR_SUBSCRIPTION_ID
```

### Combined Permissions (All Features)

For full functionality:

```bash
# Cost Management Reader for billing data
az role assignment create \
  --assignee "user@example.com" \
  --role "Cost Management Reader" \
  --scope /subscriptions/YOUR_SUBSCRIPTION_ID

# Reader for resource inventory
az role assignment create \
  --assignee "user@example.com" \
  --role "Reader" \
  --scope /subscriptions/YOUR_SUBSCRIPTION_ID
```

### Service Principal Setup

```bash
# Create service principal with required roles
az ad sp create-for-rbac \
  --name "cloud-doctor" \
  --role "Cost Management Reader" \
  --scopes /subscriptions/YOUR_SUBSCRIPTION_ID

# Add Reader role
az role assignment create \
  --assignee <appId-from-above> \
  --role "Reader" \
  --scope /subscriptions/YOUR_SUBSCRIPTION_ID
```

## Step 4: Run Cloud Doctor

### Cost Analysis

Compare current month costs to last month:

```bash
./cloud-doctor --provider azure --subscription YOUR_SUBSCRIPTION_ID
```

Example output:
```
 💰 COST DIAGNOSIS
 Account/Project ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
 ------------------------------------------------
╭─────────────────────────────────┬────────────────┬────────────────┬────────────╮
│ Service                         │ Last Month     │ Current Month  │ Difference │
├─────────────────────────────────┼────────────────┼────────────────┼────────────┤
│ Total Costs                     │ 1,890.45 USD   │ 2,134.67 USD   │ 244.22 USD │
├─────────────────────────────────┼────────────────┼────────────────┼────────────┤
│ Virtual Machines                │ 950.00 USD     │ 1,100.00 USD   │ 150.00 USD │
│ Storage                         │ 400.00 USD     │ 450.00 USD     │ 50.00 USD  │
│ Azure SQL Database              │ 300.00 USD     │ 320.00 USD     │ 20.00 USD  │
│ Bandwidth                       │ 150.00 USD     │ 180.00 USD     │ 30.00 USD  │
│ Azure App Service               │ 90.45 USD      │ 84.67 USD      │ -5.78 USD  │
╰─────────────────────────────────┴────────────────┴────────────────┴────────────╯
```

### Trend Analysis

View 6-month spending trend:

```bash
./cloud-doctor --provider azure --subscription YOUR_SUBSCRIPTION_ID --trend
```

### Waste Detection

Find unused resources:

```bash
./cloud-doctor --provider azure --subscription YOUR_SUBSCRIPTION_ID --waste
```

This checks for:
- **Unattached Managed Disks**: Disks with state `Unattached`
- **Deallocated VMs**: VMs in `PowerState/deallocated` status
- **Unassociated Public IPs**: Public IPs not attached to any resource
- **Expiring Reservations**: Reserved VM Instances expiring within 30 days

Example output:
```
 🏥 CLOUD DOCTOR CHECKUP
 Account ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
 ------------------------------------------------

╭──────────────────────────────┬─────────────────────────┬────────────╮
│ Status                       │ Volume ID               │ Size (GiB) │
├──────────────────────────────┼─────────────────────────┼────────────┤
│ Available (Unattached)       │ my-unused-disk-01       │ 128 GiB    │
│                              │ old-data-disk           │ 256 GiB    │
╰──────────────────────────────┴─────────────────────────┴────────────╯

╭──────────────────────────────┬─────────────────────────┬────────────╮
│ Status                       │ IP Address              │ Allocation │
├──────────────────────────────┼─────────────────────────┼────────────┤
│ Unassociated                 │ 20.xx.xx.xx             │ my-pip-01  │
╰──────────────────────────────┴─────────────────────────┴────────────╯

╭──────────────────────────────┬─────────────────────────┬────────────╮
│ Status                       │ Instance ID             │ Time Info  │
├──────────────────────────────┼─────────────────────────┼────────────┤
│ Stopped Instance(> 30 Days)  │ my-stopped-vm           │ Unknown    │
╰──────────────────────────────┴─────────────────────────┴────────────╯
```

> **Note**: Azure doesn't store VM deallocation timestamps directly. Cloud Doctor reports all deallocated VMs but cannot determine exactly how long they've been stopped without querying Activity Logs.

## Analyzing Multiple Subscriptions

### Sequential Analysis

```bash
# List all subscriptions
az account list --query "[].id" -o tsv

# Analyze each subscription
for sub in $(az account list --query "[].id" -o tsv); do
  echo "=== Analyzing subscription: $sub ==="
  ./cloud-doctor --provider azure --subscription $sub
done
```

### Using Management Groups

If you have Azure Management Groups, you can analyze all subscriptions in a group:

```bash
# List subscriptions in a management group
az account management-group subscription show-sub-under-mg \
  --name "my-management-group" \
  --query "[].id" -o tsv

# Analyze each
for sub in $(az account management-group subscription show-sub-under-mg \
  --name "my-management-group" --query "[].id" -o tsv); do
  ./cloud-doctor --provider azure --subscription $sub
done
```

### Multi-Cloud Mode

Analyze Azure alongside AWS and GCP:

```bash
./cloud-doctor --provider all \
  --subscription YOUR_AZURE_SUBSCRIPTION_ID \
  --project my-gcp-project \
  --billing-account billingAccounts/0X0X0X-0X0X0X-0X0X0X
```

## Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--provider` | `aws` | Cloud provider (aws, gcp, azure, all) |
| `--subscription` | (required) | Azure subscription ID |
| `--trend` | `false` | Show 6-month spending trend |
| `--waste` | `false` | Show waste detection report |

## Troubleshooting

### Error: "failed to create Azure credential"

**Cause**: No Azure credentials found.

**Solution**:
```bash
# Verify you're logged in
az account show

# If not logged in
az login
```

### Error: "AuthenticationFailed"

**Cause**: Credentials exist but are invalid or expired.

**Solution**:
```bash
# Re-authenticate with Azure CLI
az login

# For service principals, verify environment variables
echo $AZURE_CLIENT_ID
echo $AZURE_TENANT_ID
# (don't echo the secret in production!)
```

### Error: "AuthorizationFailed" on Cost Management

**Cause**: Missing Cost Management Reader role.

**Solution**:
```bash
# Grant Cost Management Reader
az role assignment create \
  --assignee "your-user-or-sp" \
  --role "Cost Management Reader" \
  --scope /subscriptions/YOUR_SUBSCRIPTION_ID

# Verify the role assignment
az role assignment list \
  --assignee "your-user-or-sp" \
  --scope /subscriptions/YOUR_SUBSCRIPTION_ID \
  --output table
```

### Error: "SubscriptionNotFound"

**Cause**: Invalid subscription ID or no access to the subscription.

**Solution**:
```bash
# Verify the subscription exists and you have access
az account list --output table

# Ensure you're using the correct subscription ID format
# Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (UUID)
```

### No cost data returned

**Cause**: New subscription or cost data not yet available.

**Solution**: Azure Cost Management data is typically available within 24-48 hours. For new subscriptions, wait a few days for data to accumulate.

### Waste detection shows no deallocated VMs

**Cause**: No VMs are deallocated, or missing Reader role.

**Solution**:
```bash
# Verify you can list VMs
az vm list --subscription YOUR_SUBSCRIPTION_ID --output table

# Check for deallocated VMs manually
az vm list -d --query "[?powerState=='VM deallocated']" --output table
```

### Error accessing reservations

**Cause**: Reservations API requires tenant-level access.

**Solution**: Reservations are queried at the tenant level. If you don't have access, the feature will return empty results (not an error). To grant access:

```bash
# Grant Reservations Reader at the tenant root
az role assignment create \
  --assignee "your-user-or-sp" \
  --role "Reservations Reader" \
  --scope /providers/Microsoft.Capacity
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AZURE_CLIENT_ID` | Service principal application (client) ID |
| `AZURE_CLIENT_SECRET` | Service principal client secret |
| `AZURE_TENANT_ID` | Azure AD tenant ID |
| `AZURE_SUBSCRIPTION_ID` | Default subscription ID (optional) |

## Azure-Specific Notes

### Cost Data Granularity

Azure Cost Management API returns data with daily granularity. Cloud Doctor aggregates this to monthly totals for comparison.

### Currency

Azure costs are returned in the billing currency configured for your subscription (typically USD).

### Resource Regions

Unlike AWS, Azure waste detection scans all resources across all regions in the subscription automatically—no need to specify a region.

### Stopped vs Deallocated VMs

In Azure, there's a difference between "stopped" and "deallocated" VMs:
- **Stopped**: VM is powered off but still allocated (you're still billed for compute)
- **Deallocated**: VM is powered off and resources released (no compute charges, only storage)

Cloud Doctor detects **deallocated** VMs, which are truly idle resources.

## Next Steps

- [AWS Getting Started](aws.md) - Set up AWS cost analysis
- [GCP Getting Started](gcp.md) - Set up GCP cost analysis
- [Multi-Cloud Guide](multicloud.md) - Analyze all providers together
