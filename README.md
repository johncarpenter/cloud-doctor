# aws-billing

A terminal-based AWS cost and billing tool built with Golang. It provides a costs comparison between current and previous month but in the same period of time. For example, if today is 15th of the month, it will compare costs from 1st to 15th of the current month with costs from 1st to 15th of the previous month.

## Demo

### Basic usage

![](https://github.com/elC0mpa/aws-cost-billing/blob/main/demo/basic.gif)

### Trend

![](https://github.com/elC0mpa/aws-cost-billing/blob/main/demo/trend.gif)

### Waste

> The idea with this flag is to perform the cost optimization checks performed by AWS Trusted Advisor but without needing a Business or Enterprise support plan.

- [ ] Export report to CSV and PDF formats.
- [ ] Distribute the CLI in fedora, ubuntu and macOS repositories.

## Motivation

As a Cloud Architect, I often need to check AWS costs and billing information. Even though AWS provides this information through the console, I usually executed the same steps to get the summary I needed, and basically this is why I created this tool. Besides saving time, it provides a table with all information you need to compare costs between current and previous month for the same period of time.

## Flags

- `--profile`: Specify the AWS profile to use (default is "").
- `--region`: Specify the AWS region to use (default is "us-east-1").
- `--trend`: Shows a trend analysis for the last 6 months.
- `--waste`: Makes an analysis of possible money waste you have in your AWS Account.
    - [x] Unused EBS Volumes (not attached to any instance).
    - [x] EBS Volumes attached to stopped EC2 instances.
    - [x] Unassociated Elastic IPs.
    - [x] EC2 reserved instance that are scheduled to expire in the next 30 days or have expired in the preceding 30 days.
    - [x] EC2 instance stopped for more than 30 days.
    - [x] Load Balancers with no attached target groups.
    - [] EC2 instances stopped for more than 30 days.
    - [] Inactive VPC interface endpoints.
    - [] Inactive NAT Gateways.
    - [] Idle Load Balancers.
    - [] RDS Idle DB Instances.


## Pending features

- [x] Add monthly trend analysis.
- [x] Add waste analysis.
- [ ] Export report to CSV and PDF formats.
- [ ] Distribute the CLI in fedora, ubuntu and macOS repositories.
