---
description: GCP cost summary - month-to-date vs previous month by service
---

# GCP Cost Summary

Generate a GCP cost summary comparing month-to-date spending with the previous month, broken down by service.

## Prerequisites Check

First, verify Google Cloud CLI is properly configured:

1. **Check gcloud CLI is installed and authenticated:**
   ```bash
   gcloud config get-value project && gcloud auth list --filter=status:ACTIVE --format="value(account)"
   ```

   If this fails or shows no active account, inform the user:
   - Run `gcloud auth login` to authenticate
   - Run `gcloud config set project PROJECT_ID` to set the project

   If the command is not found, inform the user to install the Google Cloud SDK.

2. **Check billing export is configured:**
   GCP requires BigQuery billing export to be set up. Query the user for:
   - Billing account ID (format: `billingAccounts/XXXXXX-XXXXXX-XXXXXX`)
   - BigQuery dataset location (e.g., `project.dataset.gcp_billing_export_v1_XXXXXX`)

   If not configured, direct user to: Console > Billing > Billing export > BigQuery export

## Cost Query Process

GCP billing data is accessed via BigQuery. There are two approaches:

### Option A: Using bq CLI (if billing export is configured)

#### 1. Get billing table name
```bash
# List tables in the billing dataset
bq ls --format=json PROJECT_ID:BILLING_DATASET
```

The export table is typically named `gcp_billing_export_v1_XXXXXX_XXXXXX_XXXXXX`.

#### 2. Query month-to-date costs by service
```bash
bq query --use_legacy_sql=false --format=json '
SELECT
  service.description AS service,
  SUM(cost) AS cost,
  currency
FROM `PROJECT_ID.BILLING_DATASET.gcp_billing_export_v1_XXXXXX`
WHERE
  DATE(_PARTITIONTIME) >= DATE_TRUNC(CURRENT_DATE(), MONTH)
  AND DATE(_PARTITIONTIME) <= CURRENT_DATE()
GROUP BY service.description, currency
ORDER BY cost DESC
'
```

#### 3. Query previous month costs by service
```bash
bq query --use_legacy_sql=false --format=json '
SELECT
  service.description AS service,
  SUM(cost) AS cost,
  currency
FROM `PROJECT_ID.BILLING_DATASET.gcp_billing_export_v1_XXXXXX`
WHERE
  DATE(_PARTITIONTIME) >= DATE_TRUNC(DATE_SUB(CURRENT_DATE(), INTERVAL 1 MONTH), MONTH)
  AND DATE(_PARTITIONTIME) < DATE_TRUNC(CURRENT_DATE(), MONTH)
GROUP BY service.description, currency
ORDER BY cost DESC
'
```

### Option B: Using gcloud billing CLI (limited data)

If BigQuery export is not available, use the billing budgets API for basic info:

```bash
# List billing accounts
gcloud billing accounts list --format=json

# Get billing account info
gcloud billing accounts describe BILLING_ACCOUNT_ID --format=json
```

Note: The gcloud CLI has limited cost reporting capabilities. BigQuery export is recommended for detailed service-level breakdown.

### Option C: Using Cloud Billing API directly

```bash
# Get cost breakdown (requires billing.accounts.getSpendingInformation permission)
gcloud alpha billing accounts spending describe BILLING_ACCOUNT_ID \
  --filter="service.description:*" \
  --format=json
```

## Output Format

Present the results as a markdown table with the following columns:

| Service | MTD Cost | Last Month | Change | % Change |
|---------|----------|------------|--------|----------|
| Compute Engine | $X.XX | $Y.YY | +$Z.ZZ | +N% |
| Cloud Storage | $X.XX | $Y.YY | -$Z.ZZ | -N% |
| BigQuery | $X.XX | $Y.YY | +$Z.ZZ | +N% |
| ... | | | | |
| **Total** | **$X.XX** | **$Y.YY** | **+/-$Z.ZZ** | **+/-N%** |

### Formatting Rules:
- Sort services by MTD cost descending (highest first)
- Show currency as detected (usually USD)
- Calculate percentage change: `((MTD - LastMonth) / LastMonth) * 100`
- Round costs to 2 decimal places
- Show "New" for services not present last month
- Show "Removed" for services present last month but not this month
- Include credits as negative costs if present

## Error Handling

If the cost query fails:
1. **BigQuery access denied**: User needs `bigquery.jobs.create` and table read permissions
2. **Table not found**: Billing export may not be configured or table name is wrong
3. **No data**: Billing export can take 24-48 hours to populate initially

## Setup Instructions (if billing export not configured)

If the user needs to set up billing export:

1. Go to Google Cloud Console > Billing > Billing export
2. Click "Edit settings" under BigQuery export
3. Select or create a BigQuery dataset
4. Choose "Detailed usage cost" export
5. Click "Save"
6. Wait 24-48 hours for data to populate

## Notes

- GCP billing data in BigQuery is typically delayed by 24 hours
- MTD costs are partial (month not complete) - note this in the output
- Credits and discounts appear as negative costs
- Sustained use discounts are applied automatically
- For organizations with multiple billing accounts, specify the correct billing account ID
