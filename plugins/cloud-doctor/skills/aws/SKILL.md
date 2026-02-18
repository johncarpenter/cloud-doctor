---
description: AWS cost summary - month-to-date vs previous month by service
---

# AWS Cost Summary

Generate an AWS cost summary comparing month-to-date spending with the previous month, broken down by service.

## Prerequisites Check

First, verify AWS CLI is properly configured:

1. **Check AWS CLI is installed and configured:**
   ```bash
   aws sts get-caller-identity --query "{Account:Account, Arn:Arn}" --output json
   ```

   If this fails with a credentials error, inform the user:
   - Run `aws configure` to set up credentials
   - Or set `AWS_PROFILE` environment variable
   - Or configure IAM role if running on EC2/ECS

   If the command is not found, inform the user to install the AWS CLI.

2. **Verify Cost Explorer access:**
   The user needs `ce:GetCostAndUsage` permission. Cost Explorer must also be enabled in the AWS account (it's enabled by default but costs $0.01 per request).

## Cost Query Process

Once authenticated, query costs using the AWS CLI:

### 1. Calculate date ranges

For macOS:
```bash
# Current month: 1st of month to today
CURRENT_MONTH_START=$(date -v1d +%Y-%m-%d)
TODAY=$(date +%Y-%m-%d)

# Previous month: 1st to last day of previous month
PREV_MONTH_START=$(date -v1d -v-1m +%Y-%m-%d)
PREV_MONTH_END=$(date -v1d -v-1d +%Y-%m-%d)
```

For Linux:
```bash
CURRENT_MONTH_START=$(date -d "$(date +%Y-%m-01)" +%Y-%m-%d)
TODAY=$(date +%Y-%m-%d)
PREV_MONTH_START=$(date -d "$(date +%Y-%m-01) -1 month" +%Y-%m-%d)
PREV_MONTH_END=$(date -d "$(date +%Y-%m-01) -1 day" +%Y-%m-%d)
```

### 2. Query month-to-date costs by service
```bash
aws ce get-cost-and-usage \
  --time-period Start=$CURRENT_MONTH_START,End=$TODAY \
  --granularity MONTHLY \
  --metrics "UnblendedCost" \
  --group-by Type=DIMENSION,Key=SERVICE \
  --output json
```

### 3. Query previous month costs by service
```bash
aws ce get-cost-and-usage \
  --time-period Start=$PREV_MONTH_START,End=$PREV_MONTH_END \
  --granularity MONTHLY \
  --metrics "UnblendedCost" \
  --group-by Type=DIMENSION,Key=SERVICE \
  --output json
```

## Response Parsing

The AWS CE response format is:
```json
{
  "ResultsByTime": [{
    "Groups": [{
      "Keys": ["Amazon EC2"],
      "Metrics": {
        "UnblendedCost": {
          "Amount": "123.45",
          "Unit": "USD"
        }
      }
    }]
  }]
}
```

Extract service names from `Keys[0]` and costs from `Metrics.UnblendedCost.Amount`.

## Output Format

Present the results as a markdown table with the following columns:

| Service | MTD Cost | Last Month | Change | % Change |
|---------|----------|------------|--------|----------|
| Amazon EC2 | $X.XX | $Y.YY | +$Z.ZZ | +N% |
| Amazon S3 | $X.XX | $Y.YY | -$Z.ZZ | -N% |
| AWS Lambda | $X.XX | $Y.YY | +$Z.ZZ | +N% |
| ... | | | | |
| **Total** | **$X.XX** | **$Y.YY** | **+/-$Z.ZZ** | **+/-N%** |

### Formatting Rules:
- Sort services by MTD cost descending (highest first)
- Show currency as USD (AWS Cost Explorer default)
- Calculate percentage change: `((MTD - LastMonth) / LastMonth) * 100`
- Round costs to 2 decimal places
- Show "New" for services not present last month
- Show "Removed" for services present last month but not this month
- Exclude services with $0.00 in both periods

## Error Handling

If the cost query fails:
1. **AccessDeniedException**: User needs `ce:GetCostAndUsage` permission
2. **DataUnavailableException**: Cost Explorer may not be enabled - direct user to AWS Console > Cost Explorer to enable
3. **InvalidParameterException**: Check date format (must be YYYY-MM-DD)

## Notes

- AWS costs are typically delayed by 24 hours
- MTD costs are partial (month not complete) - note this in the output
- UnblendedCost shows actual charges; use BlendedCost for consolidated billing views
- For Organizations, this shows the linked account's costs unless using management account
