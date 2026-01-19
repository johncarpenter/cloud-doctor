# Getting Started with GCP

This guide walks you through setting up Cloud Doctor to analyze your Google Cloud Platform projects for cost analysis and waste detection.

## Prerequisites

- A GCP account with at least one project
- [Google Cloud SDK (gcloud CLI)](https://cloud.google.com/sdk/docs/install) installed
- Billing export to BigQuery enabled (for cost analysis)
- Appropriate IAM permissions

## Quick Start

```bash
# Cost comparison (current month vs last month)
./cloud-doctor --provider gcp \
  --project my-project-id \
  --billing-account billingAccounts/XXXXXX-XXXXXX-XXXXXX

# 6-month trend analysis
./cloud-doctor --provider gcp \
  --project my-project-id \
  --billing-account billingAccounts/XXXXXX-XXXXXX-XXXXXX \
  --trend

# Waste detection (no billing account needed)
./cloud-doctor --provider gcp \
  --project my-project-id \
  --waste
```

## Step 1: Authentication

Cloud Doctor uses Google Cloud's Application Default Credentials (ADC). Set up authentication using one of these methods:

### Option A: User Account (Development)

```bash
# Login with your Google account
gcloud auth application-default login

# Set your default project
gcloud config set project YOUR_PROJECT_ID
```

### Option B: Service Account (Production/CI)

```bash
# Create a service account
gcloud iam service-accounts create cloud-doctor \
  --display-name="Cloud Doctor Service Account"

# Grant required roles (see IAM Permissions section)
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="serviceAccount:cloud-doctor@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/bigquery.dataViewer"

# Download the key file
gcloud iam service-accounts keys create ~/cloud-doctor-key.json \
  --iam-account=cloud-doctor@YOUR_PROJECT_ID.iam.gserviceaccount.com

# Set the credentials environment variable
export GOOGLE_APPLICATION_CREDENTIALS=~/cloud-doctor-key.json
```

## Step 2: Enable BigQuery Billing Export (For Cost Analysis)

Cost analysis requires billing data to be exported to BigQuery. This is a one-time setup per billing account.

### 2.1 Enable the BigQuery API

```bash
gcloud services enable bigquery.googleapis.com --project=YOUR_PROJECT_ID
```

### 2.2 Create a Dataset for Billing Export

```bash
# Create a dataset named 'billing_export' in your project
bq mk --dataset YOUR_PROJECT_ID:billing_export
```

### 2.3 Enable Billing Export in Cloud Console

1. Go to [Billing Export](https://console.cloud.google.com/billing/export) in Cloud Console
2. Select your billing account
3. Click on **BigQuery Export**
4. Click **Edit Settings**
5. Select your project and the `billing_export` dataset
6. Choose **Standard usage cost** (recommended) or **Detailed usage cost**
7. Click **Save**

> **Note**: Billing data will start appearing in BigQuery within 24-48 hours. Historical data for the current month will be backfilled.

### 2.4 Find Your Billing Account ID

```bash
# List your billing accounts
gcloud billing accounts list

# Output example:
# ACCOUNT_ID            NAME                 OPEN  MASTER_ACCOUNT_ID
# 0X0X0X-0X0X0X-0X0X0X  My Billing Account   True
```

The billing account ID format for Cloud Doctor is: `billingAccounts/0X0X0X-0X0X0X-0X0X0X`

### 2.5 Verify BigQuery Table Exists

After enabling export, verify the table was created:

```bash
# List tables in the billing_export dataset
bq ls YOUR_PROJECT_ID:billing_export

# You should see a table like:
# gcp_billing_export_v1_0X0X0X_0X0X0X_0X0X0X
```

## Step 3: Set Up IAM Permissions

### Minimum Permissions for Cost Analysis

The authenticated user/service account needs:

| Role | Purpose |
|------|---------|
| `roles/bigquery.dataViewer` | Read billing data from BigQuery |
| `roles/bigquery.jobUser` | Run BigQuery queries |

```bash
# Grant BigQuery permissions
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="user:your-email@example.com" \
  --role="roles/bigquery.dataViewer"

gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="user:your-email@example.com" \
  --role="roles/bigquery.jobUser"
```

### Minimum Permissions for Waste Detection

| Role | Purpose |
|------|---------|
| `roles/compute.viewer` | List VMs, disks, and IPs |
| `roles/resourcemanager.projectViewer` | View project metadata |

```bash
# Grant Compute Engine permissions
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="user:your-email@example.com" \
  --role="roles/compute.viewer"
```

### Combined Permissions (All Features)

For full functionality, grant these roles:

```bash
# All-in-one: Viewer role covers most read operations
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="user:your-email@example.com" \
  --role="roles/viewer"

# Plus BigQuery job execution
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="user:your-email@example.com" \
  --role="roles/bigquery.jobUser"
```

## Step 4: Run Cloud Doctor

### Cost Analysis

Compare current month costs to last month, grouped by service:

```bash
./cloud-doctor --provider gcp \
  --project my-project-id \
  --billing-account billingAccounts/0X0X0X-0X0X0X-0X0X0X
```

Example output:
```
 💰 COST DIAGNOSIS
 Account/Project ID: my-project-id
 ------------------------------------------------
╭─────────────────────────┬────────────────┬────────────────┬────────────╮
│ Service                 │ Last Month     │ Current Month  │ Difference │
├─────────────────────────┼────────────────┼────────────────┼────────────┤
│ Total Costs             │ 1,234.56 USD   │ 1,456.78 USD   │ 222.22 USD │
├─────────────────────────┼────────────────┼────────────────┼────────────┤
│ Compute Engine          │ 800.00 USD     │ 950.00 USD     │ 150.00 USD │
│ Cloud Storage           │ 200.00 USD     │ 250.00 USD     │ 50.00 USD  │
│ BigQuery                │ 150.00 USD     │ 180.00 USD     │ 30.00 USD  │
│ Cloud SQL               │ 84.56 USD      │ 76.78 USD      │ -7.78 USD  │
╰─────────────────────────┴────────────────┴────────────────┴────────────╯
```

### Trend Analysis

View 6-month spending trend:

```bash
./cloud-doctor --provider gcp \
  --project my-project-id \
  --billing-account billingAccounts/0X0X0X-0X0X0X-0X0X0X \
  --trend
```

### Waste Detection

Find unused resources:

```bash
./cloud-doctor --provider gcp \
  --project my-project-id \
  --waste
```

This checks for:
- **Unused Persistent Disks**: Disks not attached to any VM
- **Stopped VMs**: VMs in TERMINATED state for over 30 days
- **Unassigned External IPs**: Reserved IPs not attached to any resource
- **Expiring Committed Use Discounts**: CUDs expiring within 30 days

## Analyzing Multiple Projects

### Sequential Analysis

Run Cloud Doctor for each project:

```bash
for project in project-1 project-2 project-3; do
  echo "=== Analyzing $project ==="
  ./cloud-doctor --provider gcp --project $project --waste
done
```

### Multi-Cloud Mode

If you have AWS and/or Azure configured, use `--provider all`:

```bash
./cloud-doctor --provider all \
  --project my-gcp-project \
  --billing-account billingAccounts/0X0X0X-0X0X0X-0X0X0X \
  --subscription my-azure-subscription-id
```

## Troubleshooting

### Error: "failed to create BigQuery client"

**Cause**: Authentication not configured.

**Solution**:
```bash
# Verify you're authenticated
gcloud auth application-default print-access-token

# If that fails, re-authenticate
gcloud auth application-default login
```

### Error: "failed to execute BigQuery query"

**Cause**: Billing export table doesn't exist or permissions issue.

**Solution**:
1. Verify billing export is enabled:
   ```bash
   bq ls YOUR_PROJECT_ID:billing_export
   ```

2. Check the table name matches your billing account:
   ```bash
   # Your billing account: 0X0X0X-0X0X0X-0X0X0X
   # Expected table: gcp_billing_export_v1_0X0X0X_0X0X0X_0X0X0X
   ```

3. Verify BigQuery permissions:
   ```bash
   bq query --use_legacy_sql=false \
     'SELECT 1 FROM `YOUR_PROJECT.billing_export.gcp_billing_export_v1_XXX` LIMIT 1'
   ```

### Error: "Permission denied on resource project"

**Cause**: Missing IAM permissions.

**Solution**:
```bash
# Check your current permissions
gcloud projects get-iam-policy YOUR_PROJECT_ID \
  --filter="bindings.members:your-email@example.com" \
  --format="table(bindings.role)"

# Grant missing roles as needed
```

### No billing data showing

**Cause**: Billing export was just enabled.

**Solution**: Wait 24-48 hours for data to appear. Billing export is not retroactive beyond the current month.

### Waste detection shows no results

**Cause**: Either no waste exists, or missing Compute Engine permissions.

**Solution**:
```bash
# Verify you can list VMs
gcloud compute instances list --project=YOUR_PROJECT_ID

# Verify you can list disks
gcloud compute disks list --project=YOUR_PROJECT_ID
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to service account JSON key file |
| `GOOGLE_CLOUD_PROJECT` | Default project ID (optional) |

## Next Steps

- [AWS Getting Started](aws.md) - Set up AWS cost analysis
- [Azure Getting Started](azure.md) - Set up Azure cost analysis
- [Multi-Cloud Guide](multicloud.md) - Analyze all providers together
