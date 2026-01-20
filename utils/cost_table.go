package utils

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/elC0mpa/aws-doctor/model"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func DrawCostTable(accountId string, lastTotalCost, currenttotalCost string, lastMonthGroups, currentMonthGroups *model.CostInfo, costAggregation string) {
	fmt.Printf("\n%s\n", text.FgHiWhite.Sprint(" 💰 COST DIAGNOSIS"))
	fmt.Printf(" Account/Project ID: %s\n", text.FgBlue.Sprint(accountId))
	fmt.Println(text.FgHiBlue.Sprint(" ------------------------------------------------"))

	currentMonthHeader := fmt.Sprintf("Current Month\n(%s\n%s)", *currentMonthGroups.Start, *currentMonthGroups.End)
	lastMonthHeader := fmt.Sprintf("Last Month\n(%s\n%s)", *lastMonthGroups.Start, *lastMonthGroups.End)

	rowHeader := table.Row{
		"Service",
		lastMonthHeader,
		currentMonthHeader,
		"Difference",
	}

	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.AppendHeader(rowHeader)

	var rows []table.Row

	rows = append(rows, populateFirstRow(lastTotalCost, currenttotalCost))

	// Merge services from both months to show complete picture
	mergedServicesCosts := mergeCostServices(&lastMonthGroups.CostGroup, &currentMonthGroups.CostGroup)

	for _, currentMonthService := range mergedServicesCosts {
		rows = append(rows, populateRowWithBothMonths(*lastMonthGroups, *currentMonthGroups, currentMonthService.Name))
	}

	tw.AppendRows(rows)
	tw.SetStyle(table.StyleRounded)
	
	tw.SetColumnConfigs([]table.ColumnConfig{
		{
			Number:       1,
			VAlignHeader: text.VAlignMiddle,
		},
		{
			Number: 2,
			Align:  text.AlignRight,
		},
		{
			Number: 3,
			Align:  text.AlignRight,
		},
		{
			Number:       4,
			Align:        text.AlignRight,
			VAlignHeader: text.VAlignMiddle,
		},
	})
	tw.Render()
}

func orderCostServices(costGroups *model.CostGroup) []model.ServiceCost {
	sortedServices := make([]model.ServiceCost, 0, len(*costGroups))
	for key, group := range *costGroups {
		sortedServices = append(sortedServices, model.ServiceCost{
			Name:   key,
			Amount: group.Amount,
			Unit:   group.Unit,
		})
	}

	sort.Slice(sortedServices, func(i, j int) bool {
		return sortedServices[i].Amount > sortedServices[j].Amount
	})

	return sortedServices
}

// mergeCostServices combines services from both months, sorting by max cost across either month
func mergeCostServices(lastMonthGroups, currentMonthGroups *model.CostGroup) []model.ServiceCost {
	// Use a map to collect unique service names
	serviceMap := make(map[string]model.ServiceCost)

	// Add all services from current month
	for key, group := range *currentMonthGroups {
		serviceMap[key] = model.ServiceCost{
			Name:   key,
			Amount: group.Amount,
			Unit:   group.Unit,
		}
	}

	// Add services from last month that aren't already in the map
	// or update the unit if current month didn't have this service
	for key, group := range *lastMonthGroups {
		if existing, exists := serviceMap[key]; exists {
			// Service exists in both months, keep current month's data
			// but ensure we have a unit (in case current month amount is 0)
			if existing.Unit == "" && group.Unit != "" {
				existing.Unit = group.Unit
				serviceMap[key] = existing
			}
		} else {
			// Service only exists in last month
			serviceMap[key] = model.ServiceCost{
				Name:   key,
				Amount: 0, // Current month amount is 0
				Unit:   group.Unit,
			}
		}
	}

	// Convert map to slice
	sortedServices := make([]model.ServiceCost, 0, len(serviceMap))
	for _, service := range serviceMap {
		sortedServices = append(sortedServices, service)
	}

	// Sort by max cost across both months (descending)
	sort.Slice(sortedServices, func(i, j int) bool {
		// Get last month costs for comparison
		lastI := (*lastMonthGroups)[sortedServices[i].Name].Amount
		lastJ := (*lastMonthGroups)[sortedServices[j].Name].Amount

		// Use max of current and last month for sorting
		maxI := sortedServices[i].Amount
		if lastI > maxI {
			maxI = lastI
		}
		maxJ := sortedServices[j].Amount
		if lastJ > maxJ {
			maxJ = lastJ
		}

		return maxI > maxJ
	})

	return sortedServices
}

func populateFirstRow(lastTotalCost, currentTotalCost string) table.Row {
	currentTotalSplitted := strings.Split(currentTotalCost, " ")
	lastTotalSplitted := strings.Split(lastTotalCost, " ")

	currentTotalAmount, err := strconv.ParseFloat(currentTotalSplitted[0], 64)
	if err != nil {
		panic("Error parsing current month total cost")
	}

	lastTotalAmount, err := strconv.ParseFloat(lastTotalSplitted[0], 64)
	if err != nil {
		panic("Error parsing last month total cost")
	}

	difference := currentTotalAmount - lastTotalAmount

	row := make(table.Row, 4)
	row[0] = text.FgHiGreen.Sprint("Total Costs")
	row[1] = text.FgHiYellow.Sprintf("%s", lastTotalCost)
	row[2] = text.FgHiGreen.Sprintf("%s", currentTotalCost)
	row[3] = text.FgHiGreen.Sprintf("%.2f %s", difference, currentTotalSplitted[1])

	if difference > 0 {
		row[2] = text.FgHiRed.Sprintf("%s", currentTotalCost)
		row[0] = text.FgHiRed.Sprintf("Total Costs")
		row[3] = text.FgHiRed.Sprintf("%.2f %s", difference, currentTotalSplitted[1])
	}

	return row
}

func populateRow(lastMonthGroups model.CostInfo, currentMonthGroup model.ServiceCost) table.Row {
	row := make(table.Row, 4)

	serviceName := currentMonthGroup.Name
	lastMonthGroup := lastMonthGroups.CostGroup[serviceName]

	currentServiceCost := fmt.Sprintf("%.2f %s", currentMonthGroup.Amount, currentMonthGroup.Unit)
	lastServiceCost := fmt.Sprintf("%.2f %s", lastMonthGroup.Amount, lastMonthGroup.Unit)

	difference := currentMonthGroup.Amount - lastMonthGroup.Amount

	row[0] = text.FgGreen.Sprintf("%s", serviceName)
	row[1] = text.FgYellow.Sprintf("%s", lastServiceCost)
	row[2] = text.FgGreen.Sprintf("%s", currentServiceCost)
	row[3] = text.FgGreen.Sprintf("%.2f %s", difference, currentMonthGroup.Unit)

	if difference > 0 {
		row[0] = text.FgRed.Sprintf("%s", serviceName)
		row[2] = text.FgRed.Sprintf("%s", currentServiceCost)
		row[3] = text.FgRed.Sprintf("%.2f %s", difference, currentMonthGroup.Unit)
	}

	return row
}

func populateRowWithBothMonths(lastMonthGroups, currentMonthGroups model.CostInfo, serviceName string) table.Row {
	row := make(table.Row, 4)

	lastMonthGroup := lastMonthGroups.CostGroup[serviceName]
	currentMonthGroup := currentMonthGroups.CostGroup[serviceName]

	// Determine the unit to use (prefer current month, fallback to last month)
	unit := currentMonthGroup.Unit
	if unit == "" {
		unit = lastMonthGroup.Unit
	}

	currentServiceCost := fmt.Sprintf("%.2f %s", currentMonthGroup.Amount, unit)
	lastServiceCost := fmt.Sprintf("%.2f %s", lastMonthGroup.Amount, unit)

	difference := currentMonthGroup.Amount - lastMonthGroup.Amount

	row[0] = text.FgGreen.Sprintf("%s", serviceName)
	row[1] = text.FgYellow.Sprintf("%s", lastServiceCost)
	row[2] = text.FgGreen.Sprintf("%s", currentServiceCost)
	row[3] = text.FgGreen.Sprintf("%.2f %s", difference, unit)

	if difference > 0 {
		row[0] = text.FgRed.Sprintf("%s", serviceName)
		row[2] = text.FgRed.Sprintf("%s", currentServiceCost)
		row[3] = text.FgRed.Sprintf("%.2f %s", difference, unit)
	}

	return row
}
