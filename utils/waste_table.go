package utils

import (
	"fmt"
	"os"

	"github.com/elC0mpa/aws-doctor/model"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func DrawWasteTable(accountId string, unusedIPs []model.UnusedIP, unusedVolumes []model.UnusedVolume, attachedToStoppedVolumes []model.UnusedVolume, expiringReservations []model.Reservation, stoppedInstances []model.StoppedInstance) {
	fmt.Printf("\n%s\n", text.FgHiWhite.Sprint(" 🏥 CLOUD DOCTOR CHECKUP"))
	fmt.Printf(" Account ID: %s\n", text.FgBlue.Sprint(accountId))
	fmt.Println(text.FgHiBlue.Sprint(" ------------------------------------------------"))

	hasWaste := len(unusedIPs) > 0 ||
		len(unusedVolumes) > 0 ||
		len(attachedToStoppedVolumes) > 0 ||
		len(stoppedInstances) > 0 ||
		len(expiringReservations) > 0

	if !hasWaste {
		fmt.Println("\n" + text.FgHiGreen.Sprint(" ✅  Your account is healthy! No waste found."))
		return
	}

	if len(unusedVolumes) > 0 || len(attachedToStoppedVolumes) > 0 {
		drawVolumeTable(unusedVolumes, attachedToStoppedVolumes)
	}

	if len(unusedIPs) > 0 {
		drawIPTable(unusedIPs)
	}

	if len(stoppedInstances) > 0 || len(expiringReservations) > 0 {
		drawInstanceTable(stoppedInstances, expiringReservations)
	}
}

func drawVolumeTable(unusedVolumes []model.UnusedVolume, attachedToStoppedVolumes []model.UnusedVolume) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.SetTitle("Volume Waste")

	t.AppendHeader(table.Row{"Status", "Volume ID", "Size (GiB)"})

	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Number: 3,
			Align:  text.AlignRight,
		},
	})

	statusAvailable := "Available (Unattached)"
	rows := populateVolumeRows(unusedVolumes)

	if len(rows) > 0 {
		halfRow := len(rows) / 2
		rows[halfRow][0] = text.FgHiRed.Sprint(statusAvailable)
	}

	t.AppendRows(rows)

	rows = []table.Row{}

	if len(unusedVolumes) > 0 && len(attachedToStoppedVolumes) > 0 {
		t.AppendSeparator()
	}

	statusStopped := "Attached to Stopped Instance"
	rows = populateVolumeRows(attachedToStoppedVolumes)

	if len(rows) > 0 {
		halfRow := len(rows) / 2
		rows[halfRow][0] = text.FgHiRed.Sprint(statusStopped)
	}

	t.AppendRows(rows)

	if t.Length() > 0 {
		t.Render()
		fmt.Println()
	}
}

func drawInstanceTable(instances []model.StoppedInstance, reservations []model.Reservation) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.SetTitle("Instance & Reserved Instance Waste")

	t.AppendHeader(table.Row{"Status", "Instance ID", "Time Info"})

	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 3, Align: text.AlignRight},
	})

	var hasPreviousRows bool

	if len(instances) > 0 {
		statusLabel := "Stopped Instance(> 30 Days)"
		rows := populateStoppedInstanceRows(instances)

		halfRow := len(rows) / 2
		rows[halfRow][0] = text.FgHiRed.Sprint(statusLabel)

		t.AppendRows(rows)
		hasPreviousRows = true
	}

	if len(reservations) > 0 {
		var expiring, expired []model.Reservation
		for _, r := range reservations {
			if r.Status == "expiring" {
				expiring = append(expiring, r)
			} else {
				expired = append(expired, r)
			}
		}

		if len(expiring) > 0 {
			if hasPreviousRows {
				t.AppendSeparator()
			}
			statusLabel := "Reserved Instance\n(Expiring Soon)"
			rows := populateReservationRows(expiring)

			halfRow := len(rows) / 2
			rows[halfRow][0] = text.FgHiYellow.Sprint(statusLabel)

			t.AppendRows(rows)
			hasPreviousRows = true
		}

		if len(expired) > 0 {
			if hasPreviousRows {
				t.AppendSeparator()
			}
			statusLabel := "Reserved Instance\n(Recently Expired)"
			rows := populateReservationRows(expired)

			halfRow := len(rows) / 2
			rows[halfRow][0] = text.FgHiRed.Sprint(statusLabel)

			t.AppendRows(rows)
		}
	}

	t.Render()
	fmt.Println()
}

func drawIPTable(unusedIPs []model.UnusedIP) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.SetTitle("IP Address Waste")

	t.AppendHeader(table.Row{"Status", "IP Address", "Allocation ID"})

	statusUnused := "Unassociated"
	rows := populateIPRows(unusedIPs)

	if len(rows) > 0 {
		halfRow := len(rows) / 2
		rows[halfRow][0] = text.FgHiRed.Sprint(statusUnused)
	}

	t.AppendRows(rows)
	t.Render()
	fmt.Println()
}

func populateVolumeRows(volumes []model.UnusedVolume) []table.Row {
	var rows []table.Row

	for _, vol := range volumes {
		rows = append(rows, table.Row{
			"",
			vol.ID,
			fmt.Sprintf("%d GiB", vol.SizeGB),
		})
	}

	return rows
}

func populateIPRows(ips []model.UnusedIP) []table.Row {
	var rows []table.Row

	for _, ip := range ips {
		rows = append(rows, table.Row{
			"",
			ip.Address,
			ip.AllocationID,
		})
	}

	return rows
}

func populateStoppedInstanceRows(instances []model.StoppedInstance) []table.Row {
	var rows []table.Row

	for _, instance := range instances {
		timeInfo := fmt.Sprintf("%d days ago", instance.StoppedDays)

		rows = append(rows, table.Row{
			"",
			instance.ID,
			timeInfo,
		})
	}
	return rows
}

func populateReservationRows(reservations []model.Reservation) []table.Row {
	var rows []table.Row
	for _, r := range reservations {
		timeInfo := ""
		if r.DaysUntilExpiry >= 0 {
			timeInfo = fmt.Sprintf("In %d days", r.DaysUntilExpiry)
		} else {
			timeInfo = fmt.Sprintf("%d days ago", -r.DaysUntilExpiry)
		}

		rows = append(rows, table.Row{
			"",
			r.ID,
			timeInfo,
		})
	}
	return rows
}
