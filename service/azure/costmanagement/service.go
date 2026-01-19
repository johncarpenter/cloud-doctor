package azurecostmanagement

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement"
	"github.com/elC0mpa/aws-doctor/model"
)

func NewService(subscriptionID string, credential *Credential) (*service, error) {
	client, err := armcostmanagement.NewQueryClient(credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cost management client: %w", err)
	}

	return &service{
		subscriptionID: subscriptionID,
		client:         client,
	}, nil
}

// GetCurrentMonthCostsByService implements service.CostService
func (s *service) GetCurrentMonthCostsByService(ctx context.Context) (*model.CostInfo, error) {
	return s.getMonthCostsByService(ctx, time.Now())
}

// GetLastMonthCostsByService implements service.CostService
func (s *service) GetLastMonthCostsByService(ctx context.Context) (*model.CostInfo, error) {
	return s.getMonthCostsByService(ctx, time.Now().AddDate(0, -1, 0))
}

func (s *service) getMonthCostsByService(ctx context.Context, endDate time.Time) (*model.CostInfo, error) {
	startDate := s.getFirstDayOfMonth(endDate)
	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.Format("2006-01-02")

	scope := fmt.Sprintf("/subscriptions/%s", s.subscriptionID)

	// Query costs grouped by ServiceName
	queryDefinition := armcostmanagement.QueryDefinition{
		Type:      to.Ptr(armcostmanagement.ExportTypeActualCost),
		Timeframe: to.Ptr(armcostmanagement.TimeframeTypeCustom),
		TimePeriod: &armcostmanagement.QueryTimePeriod{
			From: to.Ptr(startDate),
			To:   to.Ptr(endDate),
		},
		Dataset: &armcostmanagement.QueryDataset{
			Granularity: to.Ptr(armcostmanagement.GranularityTypeDaily),
			Aggregation: map[string]*armcostmanagement.QueryAggregation{
				"totalCost": {
					Name:     to.Ptr("Cost"),
					Function: to.Ptr(armcostmanagement.FunctionTypeSum),
				},
			},
			Grouping: []*armcostmanagement.QueryGrouping{
				{
					Type: to.Ptr(armcostmanagement.QueryColumnTypeDimension),
					Name: to.Ptr("ServiceName"),
				},
			},
		},
	}

	resp, err := s.client.Usage(ctx, scope, queryDefinition, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query costs: %w", err)
	}

	costGroups := make(model.CostGroup)

	// Parse the response
	if resp.Properties != nil && resp.Properties.Rows != nil {
		for _, row := range resp.Properties.Rows {
			if len(row) >= 2 {
				// Row format: [cost, serviceName, ...]
				cost, ok := row[0].(float64)
				if !ok {
					continue
				}
				serviceName, ok := row[1].(string)
				if !ok {
					continue
				}

				if cost > 0 {
					// Aggregate costs by service (since we're querying daily granularity)
					existing := costGroups[serviceName]
					costGroups[serviceName] = struct {
						Amount float64
						Unit   string
					}{
						Amount: existing.Amount + cost,
						Unit:   "USD", // Azure Cost Management returns costs in USD by default
					}
				}
			}
		}
	}

	return &model.CostInfo{
		DateInterval: model.DateInterval{
			Start: &startDateStr,
			End:   &endDateStr,
		},
		CostGroup: costGroups,
	}, nil
}

// GetCurrentMonthTotalCosts implements service.CostService
func (s *service) GetCurrentMonthTotalCosts(ctx context.Context) (*string, error) {
	return s.getMonthTotalCosts(ctx, time.Now())
}

// GetLastMonthTotalCosts implements service.CostService
func (s *service) GetLastMonthTotalCosts(ctx context.Context) (*string, error) {
	return s.getMonthTotalCosts(ctx, time.Now().AddDate(0, -1, 0))
}

func (s *service) getMonthTotalCosts(ctx context.Context, endDate time.Time) (*string, error) {
	startDate := s.getFirstDayOfMonth(endDate)

	scope := fmt.Sprintf("/subscriptions/%s", s.subscriptionID)

	// Query total costs without grouping (use Daily granularity and aggregate in code)
	queryDefinition := armcostmanagement.QueryDefinition{
		Type:      to.Ptr(armcostmanagement.ExportTypeActualCost),
		Timeframe: to.Ptr(armcostmanagement.TimeframeTypeCustom),
		TimePeriod: &armcostmanagement.QueryTimePeriod{
			From: to.Ptr(startDate),
			To:   to.Ptr(endDate),
		},
		Dataset: &armcostmanagement.QueryDataset{
			Granularity: to.Ptr(armcostmanagement.GranularityTypeDaily),
			Aggregation: map[string]*armcostmanagement.QueryAggregation{
				"totalCost": {
					Name:     to.Ptr("Cost"),
					Function: to.Ptr(armcostmanagement.FunctionTypeSum),
				},
			},
		},
	}

	resp, err := s.client.Usage(ctx, scope, queryDefinition, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query total costs: %w", err)
	}

	var totalCost float64

	if resp.Properties != nil && resp.Properties.Rows != nil {
		for _, row := range resp.Properties.Rows {
			if len(row) >= 1 {
				if cost, ok := row[0].(float64); ok {
					totalCost += cost
				}
			}
		}
	}

	result := fmt.Sprintf("%.2f USD", totalCost)
	return &result, nil
}

// GetLastSixMonthsCosts implements service.CostService
func (s *service) GetLastSixMonthsCosts(ctx context.Context) ([]model.CostInfo, error) {
	var monthlyCosts []model.CostInfo

	// Query each month separately for the last 6 months
	for i := 6; i >= 1; i-- {
		monthDate := time.Now().AddDate(0, -i, 0)
		startDate := s.getFirstDayOfMonth(monthDate)
		endDate := s.getLastDayOfMonth(monthDate)

		scope := fmt.Sprintf("/subscriptions/%s", s.subscriptionID)

		queryDefinition := armcostmanagement.QueryDefinition{
			Type:      to.Ptr(armcostmanagement.ExportTypeActualCost),
			Timeframe: to.Ptr(armcostmanagement.TimeframeTypeCustom),
			TimePeriod: &armcostmanagement.QueryTimePeriod{
				From: to.Ptr(startDate),
				To:   to.Ptr(endDate),
			},
			Dataset: &armcostmanagement.QueryDataset{
				Granularity: to.Ptr(armcostmanagement.GranularityTypeDaily),
				Aggregation: map[string]*armcostmanagement.QueryAggregation{
					"totalCost": {
						Name:     to.Ptr("Cost"),
						Function: to.Ptr(armcostmanagement.FunctionTypeSum),
					},
				},
			},
		}

		resp, err := s.client.Usage(ctx, scope, queryDefinition, nil)
		if err != nil {
			// If we can't get data for a month, continue with zero
			continue
		}

		var totalCost float64
		if resp.Properties != nil && resp.Properties.Rows != nil {
			for _, row := range resp.Properties.Rows {
				if len(row) >= 1 {
					if cost, ok := row[0].(float64); ok {
						totalCost += cost
					}
				}
			}
		}

		startDateStr := startDate.Format("2006-01-02")
		endDateStr := endDate.Format("2006-01-02")

		costGroups := make(model.CostGroup)
		costGroups["Total"] = struct {
			Amount float64
			Unit   string
		}{
			Amount: totalCost,
			Unit:   "USD",
		}

		monthlyCosts = append(monthlyCosts, model.CostInfo{
			DateInterval: model.DateInterval{
				Start: &startDateStr,
				End:   &endDateStr,
			},
			CostGroup: costGroups,
		})
	}

	return monthlyCosts, nil
}

func (s *service) getFirstDayOfMonth(month time.Time) time.Time {
	return time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func (s *service) getLastDayOfMonth(month time.Time) time.Time {
	return time.Date(month.Year(), month.Month()+1, 0, 23, 59, 59, 0, time.UTC)
}
