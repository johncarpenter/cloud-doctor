# Getting Started with AWS

This guide walks you through setting up Cloud Doctor to analyze your Amazon Web Services account for cost analysis and waste detection.

## Prerequisites

- An AWS account
- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) installed (recommended)
- AWS credentials configured
- Appropriate IAM permissions

## Quick Start

```bash
# Cost comparison (current month vs last month)
./cloud-doctor --provider aws

# With specific profile and region
./cloud-doctor --provider aws --profile myprofile --region us-west-2

# 6-month trend analysis
./cloud-doctor --provider aws --trend

# Waste detection
./cloud-doctor --provider aws --waste
```

## Step 1: Authentication

Cloud Doctor uses the AWS SDK's default credential chain. Configure credentials using one of these methods:

### Option A: AWS CLI Configuration (Recommended)

```bash
# Configure default credentials
aws configure

# You'll be prompted for:
# AWS Access Key ID: AKIAIOSFODNN7EXAMPLE
# AWS Secret Access Key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
# Default region name: us-east-1
# Default output format: json
```

### Option B: Named Profiles

For multiple accounts, create named profiles:

```bash
# Configure a named profile
aws configure --profile production

# Use the profile with Cloud Doctor
./cloud-doctor --provider aws --profile production
```

Your credentials file (`~/.aws/credentials`) will look like:

```ini
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[production]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY
```

### Option C: Environment Variables

```bash
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export AWS_REGION=us-east-1

./cloud-doctor --provider aws
```

### Option D: IAM Role (EC2/ECS/Lambda)

If running on AWS infrastructure, use an IAM role attached to your instance/task/function. No credential configuration needed—the SDK automatically uses the instance metadata service.

### Option E: AWS SSO

```bash
# Configure SSO
aws configure sso

# Login to SSO
aws sso login --profile my-sso-profile

# Use with Cloud Doctor
./cloud-doctor --provider aws --profile my-sso-profile
```

## Step 2: Set Up IAM Permissions

### Minimum Permissions for Cost Analysis

Create an IAM policy with these permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "CostExplorerAccess",
            "Effect": "Allow",
            "Action": [
                "ce:GetCostAndUsage",
                "ce:GetCostForecast"
            ],
            "Resource": "*"
        },
        {
            "Sid": "STSAccess",
            "Effect": "Allow",
            "Action": [
                "sts:GetCallerIdentity"
            ],
            "Resource": "*"
        }
    ]
}
```

### Minimum Permissions for Waste Detection

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "EC2ReadAccess",
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeInstances",
                "ec2:DescribeVolumes",
                "ec2:DescribeAddresses",
                "ec2:DescribeReservedInstances"
            ],
            "Resource": "*"
        },
        {
            "Sid": "STSAccess",
            "Effect": "Allow",
            "Action": [
                "sts:GetCallerIdentity"
            ],
            "Resource": "*"
        }
    ]
}
```

### Combined Policy (All Features)

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "CloudDoctorFullAccess",
            "Effect": "Allow",
            "Action": [
                "ce:GetCostAndUsage",
                "ce:GetCostForecast",
                "ec2:DescribeInstances",
                "ec2:DescribeVolumes",
                "ec2:DescribeAddresses",
                "ec2:DescribeReservedInstances",
                "sts:GetCallerIdentity"
            ],
            "Resource": "*"
        }
    ]
}
```

### Attach Policy via AWS CLI

```bash
# Create the policy
aws iam create-policy \
  --policy-name CloudDoctorPolicy \
  --policy-document file://cloud-doctor-policy.json

# Attach to a user
aws iam attach-user-policy \
  --user-name your-username \
  --policy-arn arn:aws:iam::123456789012:policy/CloudDoctorPolicy

# Or attach to a role
aws iam attach-role-policy \
  --role-name your-role-name \
  --policy-arn arn:aws:iam::123456789012:policy/CloudDoctorPolicy
```

## Step 3: Enable Cost Explorer

Cost Explorer must be enabled in your AWS account (it's enabled by default for most accounts created after 2019).

### Verify Cost Explorer is Enabled

1. Go to [AWS Cost Explorer](https://console.aws.amazon.com/cost-management/home#/cost-explorer)
2. If prompted, click **Enable Cost Explorer**
3. Wait up to 24 hours for historical data to populate

### Enable via AWS CLI

```bash
aws ce get-cost-and-usage \
  --time-period Start=2024-01-01,End=2024-01-02 \
  --granularity DAILY \
  --metrics "UnblendedCost"
```

If you get an error about Cost Explorer not being enabled, enable it in the console.

## Step 4: Run Cloud Doctor

### Cost Analysis

Compare current month costs to last month:

```bash
./cloud-doctor --provider aws
```

With a specific region and profile:

```bash
./cloud-doctor --provider aws --region us-west-2 --profile production
```

Example output:
```
 💰 COST DIAGNOSIS
 Account/Project ID: 123456789012
 ------------------------------------------------
╭─────────────────────────────────┬────────────────┬────────────────┬────────────╮
│ Service                         │ Last Month     │ Current Month  │ Difference │
├─────────────────────────────────┼────────────────┼────────────────┼────────────┤
│ Total Costs                     │ 2,345.67 USD   │ 2,567.89 USD   │ 222.22 USD │
├─────────────────────────────────┼────────────────┼────────────────┼────────────┤
│ Amazon Elastic Compute Cloud    │ 1,200.00 USD   │ 1,350.00 USD   │ 150.00 USD │
│ Amazon Simple Storage Service   │ 500.00 USD     │ 550.00 USD     │ 50.00 USD  │
│ Amazon Relational Database      │ 400.00 USD     │ 420.00 USD     │ 20.00 USD  │
│ AWS Lambda                      │ 150.00 USD     │ 160.00 USD     │ 10.00 USD  │
│ Amazon CloudWatch               │ 95.67 USD      │ 87.89 USD      │ -7.78 USD  │
╰─────────────────────────────────┴────────────────┴────────────────┴────────────╯
```

### Trend Analysis

View 6-month spending trend:

```bash
./cloud-doctor --provider aws --trend
```

### Waste Detection

Find unused resources:

```bash
./cloud-doctor --provider aws --waste
```

This checks for:
- **Unused EBS Volumes**: Volumes in `available` state (not attached)
- **Stopped EC2 Instances**: Instances stopped for over 30 days
- **Unassociated Elastic IPs**: EIPs not attached to any instance
- **Expiring Reserved Instances**: RIs expiring within 30 days or recently expired

Example output:
```
 🏥 CLOUD DOCTOR CHECKUP
 Account ID: 123456789012
 ------------------------------------------------

╭──────────────────────────────┬─────────────────────┬────────────╮
│ Status                       │ Volume ID           │ Size (GiB) │
├──────────────────────────────┼─────────────────────┼────────────┤
│ Available (Unattached)       │ vol-0abc123def456   │ 100 GiB    │
│                              │ vol-0def456abc789   │ 50 GiB     │
├──────────────────────────────┼─────────────────────┼────────────┤
│ Attached to Stopped Instance │ vol-0xyz789abc123   │ 200 GiB    │
╰──────────────────────────────┴─────────────────────┴────────────╯

╭──────────────────────────────┬─────────────────────┬────────────╮
│ Status                       │ Instance ID         │ Time Info  │
├──────────────────────────────┼─────────────────────┼────────────┤
│ Stopped Instance(> 30 Days)  │ i-0abc123def456789  │ 45 days ago│
╰──────────────────────────────┴─────────────────────┴────────────╯
```

## Analyzing Multiple Accounts

### Using Named Profiles

```bash
# Analyze each account
for profile in dev staging production; do
  echo "=== Analyzing $profile ==="
  ./cloud-doctor --provider aws --profile $profile
done
```

### Using AWS Organizations

If you have AWS Organizations set up, you can assume roles in member accounts:

```bash
# Configure cross-account role in ~/.aws/config
[profile member-account]
role_arn = arn:aws:iam::111111111111:role/CloudDoctorRole
source_profile = default

# Run Cloud Doctor
./cloud-doctor --provider aws --profile member-account
```

### Multi-Cloud Mode

Analyze AWS alongside GCP and Azure:

```bash
./cloud-doctor --provider all \
  --project my-gcp-project \
  --billing-account billingAccounts/0X0X0X-0X0X0X-0X0X0X \
  --subscription my-azure-subscription-id
```

## Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--provider` | `aws` | Cloud provider (aws, gcp, azure, all) |
| `--region` | `us-east-1` | AWS region for API calls |
| `--profile` | (default) | AWS credential profile name |
| `--trend` | `false` | Show 6-month spending trend |
| `--waste` | `false` | Show waste detection report |

## Troubleshooting

### Error: "NoCredentialProviders"

**Cause**: No AWS credentials found.

**Solution**:
```bash
# Verify credentials are configured
aws sts get-caller-identity

# If that fails, configure credentials
aws configure
```

### Error: "ExpiredToken"

**Cause**: Temporary credentials have expired.

**Solution**:
```bash
# For SSO profiles
aws sso login --profile your-profile

# For assumed roles, re-assume the role
```

### Error: "AccessDenied" on Cost Explorer

**Cause**: Missing `ce:GetCostAndUsage` permission or Cost Explorer not enabled.

**Solution**:
1. Verify Cost Explorer is enabled in the console
2. Check IAM permissions include `ce:GetCostAndUsage`
3. Note: Cost Explorer permissions are global—region doesn't matter

### Error: "AccessDenied" on EC2 operations

**Cause**: Missing EC2 read permissions.

**Solution**:
```bash
# Verify you can describe instances
aws ec2 describe-instances --region us-east-1

# If that fails, add EC2 read permissions to your IAM policy
```

### No cost data returned

**Cause**: Cost Explorer was just enabled or account is new.

**Solution**: Wait 24 hours for Cost Explorer data to populate. Historical data for up to 12 months will be available.

### Region-specific resources not showing

**Cause**: Cloud Doctor queries the region specified by `--region`.

**Solution**: Run Cloud Doctor for each region with resources:
```bash
for region in us-east-1 us-west-2 eu-west-1; do
  echo "=== Region: $region ==="
  ./cloud-doctor --provider aws --region $region --waste
done
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |
| `AWS_SESSION_TOKEN` | Session token (for temporary credentials) |
| `AWS_REGION` | Default region |
| `AWS_PROFILE` | Default profile name |
| `AWS_CONFIG_FILE` | Path to config file (default: `~/.aws/config`) |
| `AWS_SHARED_CREDENTIALS_FILE` | Path to credentials file (default: `~/.aws/credentials`) |

## Next Steps

- [GCP Getting Started](gcp.md) - Set up GCP cost analysis
- [Azure Getting Started](azure.md) - Set up Azure cost analysis
- [Multi-Cloud Guide](multicloud.md) - Analyze all providers together
