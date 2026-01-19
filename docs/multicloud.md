# Multi-Cloud Analysis Guide

This guide explains how to use Cloud Doctor to analyze costs and waste across multiple cloud providers simultaneously.

## Overview

Cloud Doctor can analyze AWS, GCP, and Azure in parallel, providing:
- **Combined cost summary** with totals across all providers
- **Per-provider cost details** with service-level breakdowns
- **Unified waste detection** showing unused resources across clouds
- **Graceful error handling** when some providers aren't configured

## Prerequisites

Before using multi-cloud mode, set up authentication for each provider you want to analyze:

- [AWS Setup](aws.md) - Configure AWS credentials
- [GCP Setup](gcp.md) - Configure GCP authentication and billing export
- [Azure Setup](azure.md) - Configure Azure authentication

## Quick Start

```bash
# Multi-cloud cost comparison
./cloud-doctor --provider all \
  --project my-gcp-project \
  --billing-account billingAccounts/0X0X0X-0X0X0X-0X0X0X \
  --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# Multi-cloud trend analysis
./cloud-doctor --provider all \
  --project my-gcp-project \
  --billing-account billingAccounts/0X0X0X-0X0X0X-0X0X0X \
  --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --trend

# Multi-cloud waste detection
./cloud-doctor --provider all \
  --project my-gcp-project \
  --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --waste
```

## Command Line Flags

| Flag | Required For | Description |
|------|--------------|-------------|
| `--provider all` | Always | Enable multi-cloud mode |
| `--region` | AWS (optional) | AWS region (default: us-east-1) |
| `--profile` | AWS (optional) | AWS credential profile |
| `--project` | GCP | GCP project ID |
| `--billing-account` | GCP cost analysis | GCP billing account ID |
| `--subscription` | Azure | Azure subscription ID |
| `--trend` | Optional | Show 6-month trend instead of cost comparison |
| `--waste` | Optional | Show waste detection instead of cost analysis |

## Provider Configuration

### Minimum Configuration

AWS is always included (uses default credentials). Add flags for other providers:

```bash
# AWS only (always included)
./cloud-doctor --provider all

# AWS + GCP
./cloud-doctor --provider all \
  --project my-gcp-project \
  --billing-account billingAccounts/XXX

# AWS + Azure
./cloud-doctor --provider all \
  --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# All three providers
./cloud-doctor --provider all \
  --project my-gcp-project \
  --billing-account billingAccounts/XXX \
  --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

### Waste Detection

For waste detection, billing account is not required for GCP:

```bash
# AWS + GCP + Azure waste detection
./cloud-doctor --provider all \
  --project my-gcp-project \
  --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --waste
```

## Output Examples

### Multi-Cloud Cost Summary

```
 💰 MULTI-CLOUD COST DIAGNOSIS
 ------------------------------------------------

╭──────────┬──────────────────────────────────────┬────────────────┬────────────────┬─────────────╮
│ Provider │ Account/Project ID                   │ Last Month     │ Current Month  │ Difference  │
├──────────┼──────────────────────────────────────┼────────────────┼────────────────┼─────────────┤
│ AWS      │ 123456789012                         │ 2,345.67 USD   │ 2,567.89 USD   │ +222.22 USD │
│ GCP      │ my-project-id                        │ 1,234.56 USD   │ 1,456.78 USD   │ +222.22 USD │
│ AZURE    │ xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx │ 1,890.45 USD   │ 2,134.67 USD   │ +244.22 USD │
├──────────┼──────────────────────────────────────┼────────────────┼────────────────┼─────────────┤
│ TOTAL    │                                      │ 5,470.68 USD   │ 6,159.34 USD   │ +688.66 USD │
╰──────────┴──────────────────────────────────────┴────────────────┴────────────────┴─────────────╯

 📊 AWS Details
 [Detailed AWS cost table...]

 📊 GCP Details
 [Detailed GCP cost table...]

 📊 AZURE Details
 [Detailed Azure cost table...]
```

### Multi-Cloud Waste Summary

```
 🏥 MULTI-CLOUD DOCTOR CHECKUP
 ------------------------------------------------

╭──────────┬──────────────────────────────────────┬────────────────┬────────────┬───────────────────┬──────────────┬──────────────╮
│ Provider │ Account/Project ID                   │ Unused Volumes │ Unused IPs │ Stopped Instances │ Expiring RIs │ Status       │
├──────────┼──────────────────────────────────────┼────────────────┼────────────┼───────────────────┼──────────────┼──────────────┤
│ AWS      │ 123456789012                         │ 3              │ 2          │ 1                 │ 0            │ ⚠ Waste Found│
│ GCP      │ my-project-id                        │ 1              │ 0          │ 0                 │ 0            │ ⚠ Waste Found│
│ AZURE    │ xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx │ 2              │ 1          │ 2                 │ 1            │ ⚠ Waste Found│
├──────────┼──────────────────────────────────────┼────────────────┼────────────┼───────────────────┼──────────────┼──────────────┤
│ TOTAL    │                                      │ 6              │ 3          │ 3                 │ 1            │ ⚠ Action Needed│
╰──────────┴──────────────────────────────────────┴────────────────┴────────────┴───────────────────┴──────────────┴──────────────╯

 🔍 AWS Details
 [Detailed AWS waste tables...]

 🔍 GCP Details
 [Detailed GCP waste tables...]

 🔍 AZURE Details
 [Detailed Azure waste tables...]
```

## Error Handling

Multi-cloud mode handles provider failures gracefully:

### Missing Credentials

If a provider's credentials aren't configured, it shows an error but continues with other providers:

```
╭──────────┬──────────────────┬────────────────┬────────────────┬─────────────────────╮
│ Provider │ Account/Project  │ Last Month     │ Current Month  │ Difference          │
├──────────┼──────────────────┼────────────────┼────────────────┼─────────────────────┤
│ AWS      │ 123456789012     │ 2,345.67 USD   │ 2,567.89 USD   │ +222.22 USD         │
│ GCP      │ Error            │ -              │ -              │ Failed to retrieve  │
│ AZURE    │ xxxxxxxx-xxxx... │ 1,890.45 USD   │ 2,134.67 USD   │ +244.22 USD         │
╰──────────┴──────────────────┴────────────────┴────────────────┴─────────────────────╯

 ⚠ GCP: failed to create GCP identity service: google: could not find default credentials
```

### Partial Success

Results from successful providers are always shown, even if others fail.

## Parallel Execution

Multi-cloud mode queries all providers simultaneously using goroutines, reducing total execution time. A query that takes:
- AWS: 3 seconds
- GCP: 4 seconds
- Azure: 3 seconds

Will complete in approximately 4 seconds (not 10 seconds sequentially).

## Best Practices

### 1. Use Environment Variables for CI/CD

```bash
# AWS
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=xxx

# GCP
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json

# Azure
export AZURE_CLIENT_ID=xxx
export AZURE_CLIENT_SECRET=xxx
export AZURE_TENANT_ID=xxx

# Run multi-cloud analysis
./cloud-doctor --provider all \
  --project my-project \
  --billing-account billingAccounts/XXX \
  --subscription xxx-xxx-xxx
```

### 2. Script for Multiple Accounts

```bash
#!/bin/bash

# Define your accounts
AWS_PROFILES=("prod" "staging")
GCP_PROJECTS=("prod-project" "staging-project")
AZURE_SUBS=("prod-sub-id" "staging-sub-id")

# Analyze production
./cloud-doctor --provider all \
  --profile "${AWS_PROFILES[0]}" \
  --project "${GCP_PROJECTS[0]}" \
  --billing-account billingAccounts/XXX \
  --subscription "${AZURE_SUBS[0]}"

# Analyze staging
./cloud-doctor --provider all \
  --profile "${AWS_PROFILES[1]}" \
  --project "${GCP_PROJECTS[1]}" \
  --billing-account billingAccounts/XXX \
  --subscription "${AZURE_SUBS[1]}"
```

### 3. Schedule Regular Reports

Add to crontab for daily reports:

```bash
# Run at 8am daily
0 8 * * * /path/to/cloud-doctor --provider all \
  --project my-project \
  --billing-account billingAccounts/XXX \
  --subscription xxx >> /var/log/cloud-doctor.log 2>&1
```

## Troubleshooting

### "no providers configured"

**Cause**: Running `--provider all` but no provider credentials are available.

**Solution**: Ensure at least AWS default credentials are configured:
```bash
aws configure
```

### One provider always fails

**Cause**: Missing credentials or permissions for that provider.

**Solution**: Follow the individual provider setup guides:
- [AWS Setup](aws.md)
- [GCP Setup](gcp.md)
- [Azure Setup](azure.md)

### Timeout errors

**Cause**: Slow network or large amount of data.

**Solution**: Try running providers individually to identify the slow one:
```bash
./cloud-doctor --provider aws
./cloud-doctor --provider gcp --project xxx --billing-account xxx
./cloud-doctor --provider azure --subscription xxx
```

## Limitations

1. **Currency**: Each provider may use different currencies. The summary assumes USD for totals.
2. **Date Ranges**: All providers use the same date logic (current month, last month, 6 months).
3. **Rate Limits**: Running against many accounts may hit cloud provider API rate limits.

## Next Steps

- [AWS Getting Started](aws.md) - Detailed AWS setup
- [GCP Getting Started](gcp.md) - Detailed GCP setup
- [Azure Getting Started](azure.md) - Detailed Azure setup
