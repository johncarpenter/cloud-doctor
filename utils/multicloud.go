package utils

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/elC0mpa/aws-doctor/model"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// DrawMultiCloudCostTable displays cost comparison across multiple providers
func DrawMultiCloudCostTable(results []model.ProviderCostResult) {
	fmt.Printf("\n%s\n", text.FgHiWhite.Sprint(" 💰 MULTI-CLOUD COST DIAGNOSIS"))
	fmt.Println(text.FgHiBlue.Sprint(" ------------------------------------------------"))

	// Show summary table first
	drawCostSummaryTable(results)

	// Then show per-provider details
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("\n %s %s: %s\n",
				text.FgHiRed.Sprint("⚠"),
				text.FgHiYellow.Sprint(strings.ToUpper(result.Provider)),
				text.FgRed.Sprint(result.Error.Error()))
			continue
		}

		if result.CurrentMonthData != nil && result.LastMonthData != nil {
			fmt.Printf("\n %s\n", text.FgHiCyan.Sprintf("📊 %s Details", strings.ToUpper(result.Provider)))
			DrawCostTable(result.AccountID, result.LastTotalCost, result.CurrentTotalCost, result.LastMonthData, result.CurrentMonthData, "UnblendedCost")
		}
	}
}

func drawCostSummaryTable(results []model.ProviderCostResult) {
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetTitle("Cost Summary by Provider")
	tw.AppendHeader(table.Row{"Provider", "Account/Project ID", "Last Month", "Current Month", "Difference"})
	tw.SetStyle(table.StyleRounded)

	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 3, Align: text.AlignRight},
		{Number: 4, Align: text.AlignRight},
		{Number: 5, Align: text.AlignRight},
	})

	var totalLast, totalCurrent float64
	var currency string

	for _, result := range results {
		if result.Error != nil {
			tw.AppendRow(table.Row{
				text.FgHiYellow.Sprint(strings.ToUpper(result.Provider)),
				text.FgRed.Sprint("Error"),
				"-",
				"-",
				text.FgRed.Sprint("Failed to retrieve"),
			})
			continue
		}

		lastCost := parseCost(result.LastTotalCost)
		currentCost := parseCost(result.CurrentTotalCost)
		diff := currentCost - lastCost

		totalLast += lastCost
		totalCurrent += currentCost

		// Extract currency
		if currency == "" {
			parts := strings.Split(result.CurrentTotalCost, " ")
			if len(parts) > 1 {
				currency = parts[1]
			}
		}

		diffStr := fmt.Sprintf("%.2f %s", diff, currency)
		currentStr := result.CurrentTotalCost
		providerColor := text.FgGreen

		if diff > 0 {
			diffStr = text.FgHiRed.Sprintf("+%.2f %s", diff, currency)
			currentStr = text.FgHiRed.Sprint(result.CurrentTotalCost)
			providerColor = text.FgRed
		} else if diff < 0 {
			diffStr = text.FgHiGreen.Sprintf("%.2f %s", diff, currency)
			currentStr = text.FgHiGreen.Sprint(result.CurrentTotalCost)
		}

		tw.AppendRow(table.Row{
			providerColor.Sprint(strings.ToUpper(result.Provider)),
			result.AccountID,
			result.LastTotalCost,
			currentStr,
			diffStr,
		})
	}

	// Add total row
	if len(results) > 1 {
		tw.AppendSeparator()
		totalDiff := totalCurrent - totalLast
		totalDiffStr := fmt.Sprintf("%.2f %s", totalDiff, currency)
		totalCurrentStr := fmt.Sprintf("%.2f %s", totalCurrent, currency)

		if totalDiff > 0 {
			totalDiffStr = text.FgHiRed.Sprintf("+%.2f %s", totalDiff, currency)
			totalCurrentStr = text.FgHiRed.Sprintf("%.2f %s", totalCurrent, currency)
		} else if totalDiff < 0 {
			totalDiffStr = text.FgHiGreen.Sprintf("%.2f %s", totalDiff, currency)
			totalCurrentStr = text.FgHiGreen.Sprintf("%.2f %s", totalCurrent, currency)
		}

		tw.AppendRow(table.Row{
			text.FgHiWhite.Sprint("TOTAL"),
			"",
			fmt.Sprintf("%.2f %s", totalLast, currency),
			totalCurrentStr,
			totalDiffStr,
		})
	}

	tw.Render()
}

// DrawMultiCloudTrendChart displays trend analysis across multiple providers
func DrawMultiCloudTrendChart(results []model.ProviderCostResult) {
	fmt.Printf("\n%s\n", text.FgHiWhite.Sprint(" 📈 MULTI-CLOUD COST TREND"))
	fmt.Println(text.FgHiBlue.Sprint(" ------------------------------------------------"))

	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("\n %s %s: %s\n",
				text.FgHiRed.Sprint("⚠"),
				text.FgHiYellow.Sprint(strings.ToUpper(result.Provider)),
				text.FgRed.Sprint(result.Error.Error()))
			continue
		}

		if len(result.TrendData) > 0 {
			fmt.Printf("\n %s\n", text.FgHiCyan.Sprintf("📊 %s Trend (Account: %s)", strings.ToUpper(result.Provider), result.AccountID))
			DrawTrendChart(result.AccountID, result.TrendData)
		}
	}
}

// DrawMultiCloudWasteTable displays waste detection across multiple providers
func DrawMultiCloudWasteTable(results []model.ProviderWasteResult) {
	fmt.Printf("\n%s\n", text.FgHiWhite.Sprint(" 🏥 MULTI-CLOUD DOCTOR CHECKUP"))
	fmt.Println(text.FgHiBlue.Sprint(" ------------------------------------------------"))

	// Draw summary table
	drawWasteSummaryTable(results)

	// Draw per-provider details
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("\n %s %s: %s\n",
				text.FgHiRed.Sprint("⚠"),
				text.FgHiYellow.Sprint(strings.ToUpper(result.Provider)),
				text.FgRed.Sprint(result.Error.Error()))
			continue
		}

		hasWaste := len(result.UnusedVolumes) > 0 ||
			len(result.AttachedVolumes) > 0 ||
			len(result.UnusedIPs) > 0 ||
			len(result.StoppedInstances) > 0 ||
			len(result.ExpiringReservations) > 0

		if hasWaste {
			fmt.Printf("\n %s\n", text.FgHiCyan.Sprintf("🔍 %s Details (Account: %s)", strings.ToUpper(result.Provider), result.AccountID))
			DrawWasteTable(result.AccountID, result.UnusedIPs, result.UnusedVolumes, result.AttachedVolumes, result.ExpiringReservations, result.StoppedInstances)
		}
	}
}

func drawWasteSummaryTable(results []model.ProviderWasteResult) {
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetTitle("Waste Summary by Provider")
	tw.AppendHeader(table.Row{"Provider", "Account/Project ID", "Unused Volumes", "Unused IPs", "Stopped Instances", "Expiring RIs", "Status"})
	tw.SetStyle(table.StyleRounded)

	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 3, Align: text.AlignCenter},
		{Number: 4, Align: text.AlignCenter},
		{Number: 5, Align: text.AlignCenter},
		{Number: 6, Align: text.AlignCenter},
		{Number: 7, Align: text.AlignCenter},
	})

	totalVolumes := 0
	totalIPs := 0
	totalInstances := 0
	totalRIs := 0

	for _, result := range results {
		if result.Error != nil {
			tw.AppendRow(table.Row{
				text.FgHiYellow.Sprint(strings.ToUpper(result.Provider)),
				text.FgRed.Sprint("Error"),
				"-",
				"-",
				"-",
				"-",
				text.FgRed.Sprint("⚠ Failed"),
			})
			continue
		}

		volumes := len(result.UnusedVolumes) + len(result.AttachedVolumes)
		ips := len(result.UnusedIPs)
		instances := len(result.StoppedInstances)
		ris := len(result.ExpiringReservations)

		totalVolumes += volumes
		totalIPs += ips
		totalInstances += instances
		totalRIs += ris

		status := text.FgHiGreen.Sprint("✅ Healthy")
		if volumes > 0 || ips > 0 || instances > 0 || ris > 0 {
			status = text.FgHiRed.Sprint("⚠ Waste Found")
		}

		tw.AppendRow(table.Row{
			text.FgHiCyan.Sprint(strings.ToUpper(result.Provider)),
			result.AccountID,
			formatWasteCount(volumes),
			formatWasteCount(ips),
			formatWasteCount(instances),
			formatWasteCount(ris),
			status,
		})
	}

	// Add total row
	if len(results) > 1 {
		tw.AppendSeparator()
		totalStatus := text.FgHiGreen.Sprint("✅ All Healthy")
		if totalVolumes > 0 || totalIPs > 0 || totalInstances > 0 || totalRIs > 0 {
			totalStatus = text.FgHiRed.Sprint("⚠ Action Needed")
		}

		tw.AppendRow(table.Row{
			text.FgHiWhite.Sprint("TOTAL"),
			"",
			formatWasteCount(totalVolumes),
			formatWasteCount(totalIPs),
			formatWasteCount(totalInstances),
			formatWasteCount(totalRIs),
			totalStatus,
		})
	}

	tw.Render()
}

func formatWasteCount(count int) string {
	if count == 0 {
		return text.FgGreen.Sprint("0")
	}
	return text.FgHiRed.Sprintf("%d", count)
}

func parseCost(costStr string) float64 {
	parts := strings.Split(costStr, " ")
	if len(parts) > 0 {
		var cost float64
		fmt.Sscanf(parts[0], "%f", &cost)
		return cost
	}
	return 0
}

// SortProviderResults sorts results by provider name for consistent display
func SortProviderCostResults(results []model.ProviderCostResult) {
	providerOrder := map[string]int{"aws": 1, "gcp": 2, "azure": 3}
	sort.Slice(results, func(i, j int) bool {
		return providerOrder[results[i].Provider] < providerOrder[results[j].Provider]
	})
}

func SortProviderWasteResults(results []model.ProviderWasteResult) {
	providerOrder := map[string]int{"aws": 1, "gcp": 2, "azure": 3}
	sort.Slice(results, func(i, j int) bool {
		return providerOrder[results[i].Provider] < providerOrder[results[j].Provider]
	})
}
