package response

import (
	"sort"
	"strconv"
	"strings"

	"github.com/elC0mpa/aws-doctor/model"
)

// ConvertAccountInfo converts model.AccountInfo to response.AccountInfo
func ConvertAccountInfo(info *model.AccountInfo) *AccountInfo {
	if info == nil {
		return nil
	}
	return &AccountInfo{
		Provider:    info.Provider,
		AccountID:   info.AccountID,
		AccountName: info.AccountName,
	}
}

// ConvertCostInfo converts model.CostInfo to response.CostInfo
func ConvertCostInfo(info *model.CostInfo) *CostInfo {
	if info == nil {
		return nil
	}

	var services []ServiceCost
	var total float64
	var currency string

	for name, cost := range info.CostGroup {
		services = append(services, ServiceCost{
			Name:   name,
			Amount: cost.Amount,
			Unit:   cost.Unit,
		})
		total += cost.Amount
		if currency == "" {
			currency = cost.Unit
		}
	}

	// Sort by amount descending
	sort.Slice(services, func(i, j int) bool {
		return services[i].Amount > services[j].Amount
	})

	startDate := ""
	if info.Start != nil {
		startDate = *info.Start
	}
	endDate := ""
	if info.End != nil {
		endDate = *info.End
	}

	if currency == "" {
		currency = "USD"
	}

	return &CostInfo{
		StartDate: startDate,
		EndDate:   endDate,
		Services:  services,
		Total:     total,
		Currency:  currency,
	}
}

// ParseTotalCostString parses "123.45 USD" format from existing services
func ParseTotalCostString(totalStr string) (float64, string) {
	parts := strings.Fields(totalStr)
	if len(parts) >= 2 {
		amount, _ := strconv.ParseFloat(parts[0], 64)
		return amount, parts[1]
	}
	if len(parts) == 1 {
		amount, _ := strconv.ParseFloat(parts[0], 64)
		return amount, "USD"
	}
	return 0, "USD"
}

// ConvertTrendData converts []model.CostInfo to CostTrend with summary
func ConvertTrendData(data []model.CostInfo) *CostTrend {
	if len(data) == 0 {
		return &CostTrend{
			Months:  []CostInfo{},
			Summary: TrendSummary{},
		}
	}

	var months []CostInfo
	var totalSpend float64
	var highestAmount, lowestAmount float64
	var highestMonth, lowestMonth string
	first := true

	for _, monthData := range data {
		costInfo := ConvertCostInfo(&monthData)
		if costInfo == nil {
			continue
		}

		// Extract total from the "Total" key in CostGroup
		monthTotal := 0.0
		for name, cost := range monthData.CostGroup {
			if name == "Total" {
				monthTotal = cost.Amount
				break
			}
		}
		costInfo.Total = monthTotal

		months = append(months, *costInfo)
		totalSpend += monthTotal

		monthLabel := ""
		if costInfo.StartDate != "" && len(costInfo.StartDate) >= 7 {
			monthLabel = costInfo.StartDate[:7] // YYYY-MM
		}

		if first || monthTotal > highestAmount {
			highestAmount = monthTotal
			highestMonth = monthLabel
		}
		if first || monthTotal < lowestAmount {
			lowestAmount = monthTotal
			lowestMonth = monthLabel
		}
		first = false
	}

	avgMonthly := 0.0
	if len(months) > 0 {
		avgMonthly = totalSpend / float64(len(months))
	}

	return &CostTrend{
		Months: months,
		Summary: TrendSummary{
			TotalSpend:     totalSpend,
			AverageMonthly: avgMonthly,
			HighestMonth:   highestMonth,
			HighestAmount:  highestAmount,
			LowestMonth:    lowestMonth,
			LowestAmount:   lowestAmount,
		},
	}
}

// ConvertUnusedVolumes converts []model.UnusedVolume to response format
func ConvertUnusedVolumes(volumes []model.UnusedVolume) []UnusedVolume {
	result := make([]UnusedVolume, 0, len(volumes))
	for _, v := range volumes {
		result = append(result, UnusedVolume{
			ID:     v.ID,
			SizeGB: v.SizeGB,
			Status: v.Status,
		})
	}
	return result
}

// ConvertStoppedInstances converts []model.StoppedInstance to response format
func ConvertStoppedInstances(instances []model.StoppedInstance) []StoppedInstance {
	result := make([]StoppedInstance, 0, len(instances))
	for _, i := range instances {
		result = append(result, StoppedInstance{
			ID:          i.ID,
			Name:        i.Name,
			StoppedDays: i.StoppedDays,
		})
	}
	return result
}

// ConvertUnusedIPs converts []model.UnusedIP to response format
func ConvertUnusedIPs(ips []model.UnusedIP) []UnusedIP {
	result := make([]UnusedIP, 0, len(ips))
	for _, ip := range ips {
		result = append(result, UnusedIP{
			Address:      ip.Address,
			AllocationID: ip.AllocationID,
		})
	}
	return result
}

// ConvertReservations converts []model.Reservation to response format
func ConvertReservations(reservations []model.Reservation) []Reservation {
	result := make([]Reservation, 0, len(reservations))
	for _, r := range reservations {
		result = append(result, Reservation{
			ID:              r.ID,
			InstanceType:    r.InstanceType,
			Status:          r.Status,
			DaysUntilExpiry: r.DaysUntilExpiry,
		})
	}
	return result
}
