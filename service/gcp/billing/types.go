package gcpbilling

import (
	"context"

	"cloud.google.com/go/bigquery"
	"github.com/elC0mpa/aws-doctor/model"
)

type service struct {
	projectID      string
	billingAccount string
	bqClient       *bigquery.Client
}

type BillingService interface {
	GetCurrentMonthCostsByService(ctx context.Context) (*model.CostInfo, error)
	GetLastMonthCostsByService(ctx context.Context) (*model.CostInfo, error)
	GetCurrentMonthTotalCosts(ctx context.Context) (*string, error)
	GetLastMonthTotalCosts(ctx context.Context) (*string, error)
	GetLastSixMonthsCosts(ctx context.Context) ([]model.CostInfo, error)
}
