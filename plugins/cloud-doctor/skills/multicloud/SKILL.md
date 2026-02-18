---
description: Multi-cloud cost summary across AWS, GCP, and Azure
---

# Multi-Cloud Cost Summary

Generate a unified cost summary comparing month-to-date spending with the previous month across all configured cloud providers (AWS, GCP, Azure).

## Overview

This skill aggregates cost data from multiple cloud providers into a single view. It will:
1. Detect which cloud providers are configured
2. Query each provider for cost data
3. Normalize and aggregate the results
4. Present a unified comparison

## Prerequisites Detection

Check which cloud providers are configured by running these commands:

### AWS
```bash
aws sts get-caller-identity --output json 2>/dev/null && echo "AWS: Configured" || echo "AWS: Not configured"
```

### GCP
```bash
gcloud config get-value project 2>/dev/null && echo "GCP: Configured" || echo "GCP: Not configured"
```

### Azure
```bash
az account show --output json 2>/dev/null && echo "Azure: Configured" || echo "Azure: Not configured"
```

Report which providers are available before proceeding. If none are configured, provide setup instructions for each.

## Cost Query Process

For each configured provider, query costs in parallel where possible.

### AWS Costs
```bash
# Date calculations (macOS)
CURRENT_MONTH_START=$(date -v1d +%Y-%m-%d)
TODAY=$(date +%Y-%m-%d)
PREV_MONTH_START=$(date -v1d -v-1m +%Y-%m-%d)
PREV_MONTH_END=$(date -v1d -v-1d +%Y-%m-%d)

# Current month
aws ce get-cost-and-usage \
  --time-period Start=$CURRENT_MONTH_START,End=$TODAY \
  --granularity MONTHLY \
  --metrics "UnblendedCost" \
  --group-by Type=DIMENSION,Key=SERVICE \
  --output json

# Previous month
aws ce get-cost-and-usage \
  --time-period Start=$PREV_MONTH_START,End=$PREV_MONTH_END \
  --granularity MONTHLY \
  --metrics "UnblendedCost" \
  --group-by Type=DIMENSION,Key=SERVICE \
  --output json
```

### Azure Costs
```bash
SUBSCRIPTION_ID=$(az account show --query id -o tsv)

# Current month
az cost management query \
  --type ActualCost \
  --scope "/subscriptions/$SUBSCRIPTION_ID" \
  --timeframe MonthToDate \
  --dataset-aggregation '{"totalCost":{"name":"Cost","function":"Sum"}}' \
  --dataset-grouping name=ServiceName type=Dimension \
  -o json

# Previous month
az cost management query \
  --type ActualCost \
  --scope "/subscriptions/$SUBSCRIPTION_ID" \
  --timeframe TheLastMonth \
  --dataset-aggregation '{"totalCost":{"name":"Cost","function":"Sum"}}' \
  --dataset-grouping name=ServiceName type=Dimension \
  -o json
```

### GCP Costs

If BigQuery billing export is configured, query via bq CLI. Otherwise, note that GCP detailed costs require BigQuery export setup.

## Output Format

### 1. Provider Summary Table

| Provider | MTD Cost | Last Month | Change | % Change | Status |
|----------|----------|------------|--------|----------|--------|
| AWS | $X,XXX.XX | $Y,YYY.YY | +$Z.ZZ | +N% | Active |
| Azure | $X,XXX.XX | $Y,YYY.YY | -$Z.ZZ | -N% | Active |
| GCP | $X,XXX.XX | $Y,YYY.YY | +$Z.ZZ | +N% | Active |
| **Total** | **$X,XXX.XX** | **$Y,YYY.YY** | **+/-$Z.ZZ** | **+/-N%** | |

### 2. Top Services Across All Providers

| Rank | Provider | Service | MTD Cost | Change |
|------|----------|---------|----------|--------|
| 1 | AWS | Amazon EC2 | $X,XXX.XX | +$Z.ZZ |
| 2 | Azure | Virtual Machines | $X,XXX.XX | +$Z.ZZ |
| 3 | GCP | Compute Engine | $X,XXX.XX | -$Z.ZZ |
| 4 | AWS | Amazon RDS | $XXX.XX | +$Z.ZZ |
| 5 | Azure | Storage | $XXX.XX | +$Z.ZZ |
| ... | | | | |

Show top 10 services by MTD cost across all providers.

### 3. Per-Provider Breakdown (Optional Detail)

If the user requests detailed breakdown, show the full service breakdown for each provider separately (use the individual AWS, GCP, Azure skills format).

## Currency Handling

- AWS: Always USD
- Azure: Varies by subscription (USD, EUR, AUD, etc.)
- GCP: Varies by billing account

For the unified view:
1. Display each provider's costs in their native currency
2. Note if currencies differ: "Note: Totals are in mixed currencies"
3. If all providers use the same currency, show a true total

## Cost Normalization Notes

Different providers categorize services differently. Common mappings:
- **Compute**: AWS EC2 ≈ Azure VMs ≈ GCP Compute Engine
- **Storage**: AWS S3 ≈ Azure Blob ≈ GCP Cloud Storage
- **Database**: AWS RDS ≈ Azure SQL ≈ Cloud SQL
- **Serverless**: AWS Lambda ≈ Azure Functions ≈ Cloud Functions

Do not attempt to merge these categories - show them separately with provider prefix for clarity.

## Error Handling

For each provider that fails:
1. Note the failure in the output (e.g., "AWS: Failed to query costs - access denied")
2. Continue with other providers
3. Show partial results with clear indication of what's missing

## Recommendations Section

After the cost summary, provide brief recommendations:

1. **Highest growth**: Flag services with >20% MTD increase
2. **New services**: Note any services that appeared this month
3. **Provider concentration**: If >80% of costs are in one provider, note it
4. **Optimization opportunities**: Suggest reviewing top 3 cost drivers

## Example Output

```
## Multi-Cloud Cost Summary (Feb 1-18, 2026)

### Provider Overview
| Provider | MTD Cost | Last Month | Change | % Change |
|----------|----------|------------|--------|----------|
| AWS | $4,521.34 | $5,892.10 | -$1,370.76 | -23% |
| Azure | $2,847.92 | $2,654.18 | +$193.74 | +7% |
| GCP | $1,203.45 | $1,156.22 | +$47.23 | +4% |
| **Total** | **$8,572.71** | **$9,702.50** | **-$1,129.79** | **-12%** |

### Top 10 Services (All Providers)
| # | Provider | Service | MTD Cost | vs Last Month |
|---|----------|---------|----------|---------------|
| 1 | AWS | Amazon EC2 | $2,145.67 | -$892.33 (-29%) |
| 2 | Azure | Virtual Machines | $1,567.23 | +$123.45 (+9%) |
| 3 | AWS | Amazon RDS | $987.45 | -$234.56 (-19%) |
...

### Recommendations
- AWS EC2 costs dropped 29% - verify this is expected (shutdown? reservations?)
- Azure VMs increased 9% - review new deployments
- Consider Reserved Instances for stable AWS EC2 workloads
```

## Notes

- Run during business hours for most current data (overnight batches may not be complete)
- Cost data is typically delayed 24-48 hours across all providers
- For accurate month-over-month comparison, note the number of days in each period
