---
description: Azure cost summary - month-to-date vs previous month by service
---

# Azure Cost Summary

Generate an Azure cost summary comparing month-to-date spending with the previous month, broken down by service.

## Prerequisites Check

First, verify Azure CLI is properly configured:

1. **Check az CLI is installed and logged in:**
   ```bash
   az account show --query "{subscription:name, subscriptionId:id, tenantId:tenantId}" -o json
   ```

   If this fails with an authentication error, inform the user:
   - Run `az login` to authenticate
   - Or run `az login --use-device-code` for device code flow

   If the command is not found, inform the user to install the Azure CLI.

2. **Verify Cost Management access:**
   The user needs at least **Cost Management Reader** role on the subscription.

## Cost Query Process

Once authenticated, query costs using the Azure CLI:

### 1. Get current subscription ID
```bash
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
echo "Subscription: $SUBSCRIPTION_ID"
```

### 2. Calculate date ranges
```bash
# Current month: 1st of month to today
CURRENT_MONTH_START=$(date -v1d +%Y-%m-%d)
TODAY=$(date +%Y-%m-%d)

# Previous month: 1st to last day of previous month
PREV_MONTH_START=$(date -v1d -v-1m +%Y-%m-%d)
PREV_MONTH_END=$(date -v1d -v-1d +%Y-%m-%d)
```

### 3. Query month-to-date costs by service
```bash
az cost management query \
  --type ActualCost \
  --scope "/subscriptions/$SUBSCRIPTION_ID" \
  --timeframe Custom \
  --time-period from=$CURRENT_MONTH_START to=$TODAY \
  --dataset-aggregation '{"totalCost":{"name":"Cost","function":"Sum"}}' \
  --dataset-grouping name=ServiceName type=Dimension \
  -o json
```

### 4. Query previous month costs by service
```bash
az cost management query \
  --type ActualCost \
  --scope "/subscriptions/$SUBSCRIPTION_ID" \
  --timeframe Custom \
  --time-period from=$PREV_MONTH_START to=$PREV_MONTH_END \
  --dataset-aggregation '{"totalCost":{"name":"Cost","function":"Sum"}}' \
  --dataset-grouping name=ServiceName type=Dimension \
  -o json
```

## Output Format

Present the results as a markdown table with the following columns:

| Service | MTD Cost | Last Month | Change | % Change |
|---------|----------|------------|--------|----------|
| Virtual Machines | $X.XX | $Y.YY | +$Z.ZZ | +N% |
| Storage | $X.XX | $Y.YY | -$Z.ZZ | -N% |
| ... | | | | |
| **Total** | **$X.XX** | **$Y.YY** | **+/-$Z.ZZ** | **+/-N%** |

### Formatting Rules:
- Sort services by MTD cost descending (highest first)
- Show currency as detected from response (usually USD or AUD)
- Calculate percentage change: `((MTD - LastMonth) / LastMonth) * 100`
- Use green for cost decreases, red for increases (if terminal supports it)
- Round costs to 2 decimal places
- Show "New" for services not present last month
- Show "Removed" for services present last month but not this month

## Error Handling

If the cost query fails:
1. Check if the subscription has Cost Management enabled
2. Verify the user has the required permissions
3. Suggest checking if the subscription is a free tier (limited cost data)
4. For Enterprise Agreement subscriptions, note that EA portal may be required

## Notes

- Costs are typically delayed by 24-48 hours
- MTD costs are partial (month not complete) - note this in the output
- For accurate month-over-month comparison, note the number of days in each period
