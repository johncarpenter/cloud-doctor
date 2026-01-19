package gcpbilling

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/elC0mpa/aws-doctor/model"
	"google.golang.org/api/iterator"
)

func NewService(ctx context.Context, projectID, billingAccount string) (*service, error) {
	bqClient, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client: %w", err)
	}

	return &service{
		projectID:      projectID,
		billingAccount: billingAccount,
		bqClient:       bqClient,
	}, nil
}

// Close closes the BigQuery client
func (s *service) Close() error {
	return s.bqClient.Close()
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

	// Query BigQuery billing export table
	// The table name format is: project.dataset.gcp_billing_export_v1_BILLING_ACCOUNT_ID
	// User needs to provide the billing account ID and have billing export enabled
	billingAccountID := strings.ReplaceAll(s.billingAccount, "billingAccounts/", "")
	billingAccountID = strings.ReplaceAll(billingAccountID, "-", "_")

	query := fmt.Sprintf(`
		SELECT
			service.description AS service_name,
			SUM(cost) AS total_cost,
			currency
		FROM %s.%s.gcp_billing_export_v1_%s
		WHERE
			project.id = @projectID
			AND DATE(usage_start_time) >= @startDate
			AND DATE(usage_start_time) < @endDate
		GROUP BY service.description, currency
		HAVING SUM(cost) > 0
		ORDER BY total_cost DESC
	`, s.projectID, "billing_export", billingAccountID)

	q := s.bqClient.Query(query)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "projectID", Value: s.projectID},
		{Name: "startDate", Value: startDateStr},
		{Name: "endDate", Value: endDateStr},
	}

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute BigQuery query: %w", err)
	}

	costGroups := make(model.CostGroup)

	for {
		var row struct {
			ServiceName string  `bigquery:"service_name"`
			TotalCost   float64 `bigquery:"total_cost"`
			Currency    string  `bigquery:"currency"`
		}

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read BigQuery row: %w", err)
		}

		costGroups[row.ServiceName] = struct {
			Amount float64
			Unit   string
		}{
			Amount: row.TotalCost,
			Unit:   row.Currency,
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
	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.Format("2006-01-02")

	billingAccountID := strings.ReplaceAll(s.billingAccount, "billingAccounts/", "")
	billingAccountID = strings.ReplaceAll(billingAccountID, "-", "_")

	query := fmt.Sprintf(`
		SELECT
			SUM(cost) AS total_cost,
			currency
		FROM %s.%s.gcp_billing_export_v1_%s
		WHERE
			project.id = @projectID
			AND DATE(usage_start_time) >= @startDate
			AND DATE(usage_start_time) < @endDate
		GROUP BY currency
	`, s.projectID, "billing_export", billingAccountID)

	q := s.bqClient.Query(query)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "projectID", Value: s.projectID},
		{Name: "startDate", Value: startDateStr},
		{Name: "endDate", Value: endDateStr},
	}

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute BigQuery query: %w", err)
	}

	var totalCost float64
	var currency string

	for {
		var row struct {
			TotalCost float64 `bigquery:"total_cost"`
			Currency  string  `bigquery:"currency"`
		}

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read BigQuery row: %w", err)
		}

		totalCost += row.TotalCost
		currency = row.Currency
	}

	result := fmt.Sprintf("%.2f %s", totalCost, currency)
	return &result, nil
}

// GetLastSixMonthsCosts implements service.CostService
func (s *service) GetLastSixMonthsCosts(ctx context.Context) ([]model.CostInfo, error) {
	billingAccountID := strings.ReplaceAll(s.billingAccount, "billingAccounts/", "")
	billingAccountID = strings.ReplaceAll(billingAccountID, "-", "_")

	startDate := s.getFirstDayOfMonth(time.Now().AddDate(0, -6, 0))
	endDate := s.getFirstDayOfMonth(time.Now())

	query := fmt.Sprintf(`
		SELECT
			FORMAT_DATE('%%Y-%%m-01', DATE(usage_start_time)) AS month_start,
			FORMAT_DATE('%%Y-%%m-%%d', DATE_ADD(DATE_TRUNC(DATE(usage_start_time), MONTH), INTERVAL 1 MONTH)) AS month_end,
			SUM(cost) AS total_cost,
			currency
		FROM %s.%s.gcp_billing_export_v1_%s
		WHERE
			project.id = @projectID
			AND DATE(usage_start_time) >= @startDate
			AND DATE(usage_start_time) < @endDate
		GROUP BY month_start, month_end, currency
		ORDER BY month_start
	`, s.projectID, "billing_export", billingAccountID)

	q := s.bqClient.Query(query)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "projectID", Value: s.projectID},
		{Name: "startDate", Value: startDate.Format("2006-01-02")},
		{Name: "endDate", Value: endDate.Format("2006-01-02")},
	}

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute BigQuery query: %w", err)
	}

	var monthlyCosts []model.CostInfo

	for {
		var row struct {
			MonthStart string  `bigquery:"month_start"`
			MonthEnd   string  `bigquery:"month_end"`
			TotalCost  float64 `bigquery:"total_cost"`
			Currency   string  `bigquery:"currency"`
		}

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read BigQuery row: %w", err)
		}

		costGroups := make(model.CostGroup)
		costGroups["Total"] = struct {
			Amount float64
			Unit   string
		}{
			Amount: row.TotalCost,
			Unit:   row.Currency,
		}

		monthlyCosts = append(monthlyCosts, model.CostInfo{
			DateInterval: model.DateInterval{
				Start: &row.MonthStart,
				End:   &row.MonthEnd,
			},
			CostGroup: costGroups,
		})
	}

	return monthlyCosts, nil
}

func (s *service) getFirstDayOfMonth(month time.Time) time.Time {
	return time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
}
