package flag

import (
	"flag"

	"github.com/elC0mpa/aws-doctor/model"
)

func NewService() *service {
	return &service{}
}

func (s *service) GetParsedFlags() (model.Flags, error) {
	// Common flags
	provider := flag.String("provider", "aws", "Cloud provider: aws, gcp, azure, all")
	trend := flag.Bool("trend", false, "Display a trend report for the last 6 months")
	waste := flag.Bool("waste", false, "Display waste report")

	// AWS-specific flags
	region := flag.String("region", "us-east-1", "AWS region")
	profile := flag.String("profile", "", "AWS profile configuration")

	// GCP-specific flags
	project := flag.String("project", "", "GCP project ID")
	billingAccount := flag.String("billing-account", "", "GCP billing account ID (format: billingAccounts/XXXXXX-XXXXXX-XXXXXX)")

	// Azure-specific flags
	subscription := flag.String("subscription", "", "Azure subscription ID")

	flag.Parse()

	return model.Flags{
		Provider:       *provider,
		Trend:          *trend,
		Waste:          *waste,
		Region:         *region,
		Profile:        *profile,
		Project:        *project,
		BillingAccount: *billingAccount,
		Subscription:   *subscription,
	}, nil
}
