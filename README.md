# Cloud Doctor

A terminal-based tool that acts as a comprehensive health check for your cloud accounts across **AWS**, **Google Cloud Platform (GCP)**, and **Microsoft Azure**. Built with Golang, **Cloud Doctor** diagnoses cost anomalies, detects idle resources, and provides proactive analysis of your cloud infrastructure—giving you insights similar to AWS Trusted Advisor, GCP Recommender, and Azure Advisor without requiring premium support plans.

> **Fork Notice**: This project is a fork of [aws-doctor](https://github.com/elC0mpa/aws-doctor) by [@elC0mpa](https://github.com/elC0mpa). The original project provided AWS-only functionality. This fork extends it to support multi-cloud environments. Thank you to elC0mpa for the excellent foundation!

## Features

- **Multi-Cloud Support**: Analyze AWS, GCP, and Azure from a single tool
- **Unified View**: Run `--provider all` to see costs across all configured clouds in one report
- **Cost Comparison**: Compare costs between the current and previous month for the same period
- **Trend Analysis**: Visualize cost history over the last 6 months to spot anomalies
- **Waste Detection**: Scan for "zombie" resources silently inflating your bill
- **Parallel Execution**: Multi-cloud queries run simultaneously for fast results
- **Graceful Degradation**: Missing credentials for one provider won't block others

## Quick Start

```bash
# Build from source
go build -o cloud-doctor

# AWS cost analysis (default)
./cloud-doctor

# GCP cost analysis
./cloud-doctor --provider gcp \
  --project my-project-id \
  --billing-account billingAccounts/XXXXXX-XXXXXX-XXXXXX

# Azure cost analysis
./cloud-doctor --provider azure \
  --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# Multi-cloud comparison
./cloud-doctor --provider all \
  --project my-gcp-project \
  --billing-account billingAccounts/XXXXXX-XXXXXX-XXXXXX \
  --subscription xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

## Installation

### From Source

```bash
git clone https://github.com/johncarpenter/cloud-doctor.git
cd cloud-doctor
go build -o cloud-doctor
```

### Go Install

```bash
go install github.com/johncarpenter/cloud-doctor@latest
```

## Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--provider` | `aws` | Cloud provider: `aws`, `gcp`, `azure`, `all` |
| `--profile` | (default) | AWS credential profile name |
| `--region` | `us-east-1` | AWS region for API calls |
| `--project` | (required for GCP) | GCP project ID |
| `--billing-account` | (required for GCP costs) | GCP billing account ID |
| `--subscription` | (required for Azure) | Azure subscription ID |
| `--trend` | `false` | Show 6-month spending trend |
| `--waste` | `false` | Show waste detection report |

## Analysis Modes

### Cost Comparison (Default)

Compares costs between the current and previous month for the exact same period (e.g., Jan 1–15 vs Feb 1–15) to give a fair assessment of spending velocity.

```bash
./cloud-doctor --provider aws
./cloud-doctor --provider gcp --project my-project --billing-account billingAccounts/XXX
./cloud-doctor --provider azure --subscription xxx-xxx-xxx
```

### Trend Analysis

Visualizes cost history over the last 6 months to spot long-term anomalies.

```bash
./cloud-doctor --provider aws --trend
./cloud-doctor --provider gcp --project my-project --billing-account billingAccounts/XXX --trend
./cloud-doctor --provider azure --subscription xxx-xxx-xxx --trend
```

### Waste Detection

Scans your account for unused resources that are silently inflating your bill.

```bash
./cloud-doctor --provider aws --waste
./cloud-doctor --provider gcp --project my-project --waste
./cloud-doctor --provider azure --subscription xxx-xxx-xxx --waste
```

**Detected Resources by Provider:**

| Check | AWS | GCP | Azure |
|-------|-----|-----|-------|
| Unused Volumes | EBS (unattached) | Persistent Disks (no users) | Managed Disks (Unattached) |
| Stopped Instances (>30 days) | EC2 (stopped) | VMs (TERMINATED) | VMs (deallocated) |
| Unused IPs | Elastic IPs | External IPs | Public IPs |
| Expiring Reservations | Reserved Instances | Committed Use Discounts | Reserved VM Instances |

## Multi-Cloud Mode

Analyze all your cloud providers in a single command:

```bash
# Cost comparison across all clouds
./cloud-doctor --provider all \
  --project my-gcp-project \
  --billing-account billingAccounts/XXX \
  --subscription xxx-xxx-xxx

# Waste detection across all clouds
./cloud-doctor --provider all \
  --project my-gcp-project \
  --subscription xxx-xxx-xxx \
  --waste

# 6-month trend across all clouds
./cloud-doctor --provider all \
  --project my-gcp-project \
  --billing-account billingAccounts/XXX \
  --subscription xxx-xxx-xxx \
  --trend
```

Multi-cloud mode:
- Queries all providers in parallel for faster results
- Shows a summary table with totals across all providers
- Continues with available providers if some credentials are missing
- Shows detailed per-provider breakdown after the summary

## Documentation

Detailed setup guides for each provider:

- [AWS Getting Started](docs/aws.md) - IAM permissions, Cost Explorer setup, credential configuration
- [GCP Getting Started](docs/gcp.md) - BigQuery billing export, IAM roles, Application Default Credentials
- [Azure Getting Started](docs/azure.md) - RBAC roles, Cost Management Reader, subscription setup
- [Multi-Cloud Guide](docs/multicloud.md) - Unified analysis, best practices, troubleshooting

## Authentication

### AWS
Uses the standard AWS credential chain:
- Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- Shared credentials file (`~/.aws/credentials`)
- IAM roles (when running on AWS infrastructure)
- SSO profiles

### GCP
Uses Application Default Credentials:
- `gcloud auth application-default login` for development
- Service account JSON key via `GOOGLE_APPLICATION_CREDENTIALS`
- Metadata service on GCP infrastructure

### Azure
Uses DefaultAzureCredential:
- Azure CLI (`az login`)
- Environment variables (`AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`)
- Managed Identity on Azure infrastructure

## Example Output

### Multi-Cloud Cost Summary

```
 MULTI-CLOUD COST DIAGNOSIS
 ------------------------------------------------

+-----------+--------------------------------------+----------------+----------------+-------------+
| Provider  | Account/Project ID                   | Last Month     | Current Month  | Difference  |
+-----------+--------------------------------------+----------------+----------------+-------------+
| AWS       | 123456789012                         | 2,345.67 USD   | 2,567.89 USD   | +222.22 USD |
| GCP       | my-project-id                        | 1,234.56 USD   | 1,456.78 USD   | +222.22 USD |
| AZURE     | xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx | 1,890.45 USD   | 2,134.67 USD   | +244.22 USD |
+-----------+--------------------------------------+----------------+----------------+-------------+
| TOTAL     |                                      | 5,470.68 USD   | 6,159.34 USD   | +688.66 USD |
+-----------+--------------------------------------+----------------+----------------+-------------+
```

### Multi-Cloud Waste Summary

```
 MULTI-CLOUD DOCTOR CHECKUP
 ------------------------------------------------

+-----------+------------------+----------------+------------+-------------------+--------------+--------------+
| Provider  | Account/Project  | Unused Volumes | Unused IPs | Stopped Instances | Expiring RIs | Status       |
+-----------+------------------+----------------+------------+-------------------+--------------+--------------+
| AWS       | 123456789012     | 3              | 2          | 1                 | 0            | Warning      |
| GCP       | my-project-id    | 1              | 0          | 0                 | 0            | Warning      |
| AZURE     | xxxxxxxx-xxxx... | 2              | 1          | 2                 | 1            | Warning      |
+-----------+------------------+----------------+------------+-------------------+--------------+--------------+
| TOTAL     |                  | 6              | 3          | 3                 | 1            | Action Needed|
+-----------+------------------+----------------+------------+-------------------+--------------+--------------+
```

## Roadmap

- [x] AWS cost comparison and trend analysis
- [x] AWS waste detection
- [x] GCP cost comparison and trend analysis
- [x] GCP waste detection
- [x] Azure cost comparison and trend analysis
- [x] Azure waste detection
- [x] Multi-cloud unified view
- [ ] Export reports to CSV and PDF formats
- [ ] Slack/Teams webhook notifications
- [ ] Cost anomaly alerts with thresholds
- [ ] Additional waste checks (Load Balancers, NAT Gateways, etc.)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Original [aws-doctor](https://github.com/elC0mpa/aws-doctor) by [@elC0mpa](https://github.com/elC0mpa) - the foundation this project is built upon
- AWS SDK for Go v2
- Google Cloud Go SDK
- Azure SDK for Go
