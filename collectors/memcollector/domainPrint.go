package memcollector

import (
	"fmt"
	"proxtop/config"
	"proxtop/models"
	"proxtop/util"
)

// formatMemKB formats a memory value in KB, converting to bytes for human-readable format
func formatMemKB(valueKB uint64) string {
	if config.Options.HumanReadable {
		// Convert KB to bytes for formatting
		return util.FormatBytes(valueKB * 1024)
	}
	return fmt.Sprintf("%d", valueKB)
}

// formatMemBytes formats a memory value in bytes
func formatMemBytes(valueBytes uint64) string {
	if config.Options.HumanReadable {
		return util.FormatBytes(valueBytes)
	}
	return fmt.Sprintf("%d", valueBytes)
}

func domainPrint(domain *models.Domain) []string {
	// esxtop style: MEMSZ (configured memory), GRANT (used), RSS (resident)
	// Note: total, used, free, max, actual are in KB (from Proxmox/libvirt)
	// rss and vsize are in bytes (from /proc/[pid]/stat)
	totalKB, _ := domain.GetMetricUint64Raw("ram_total", 0)
	usedKB, _ := domain.GetMetricUint64Raw("ram_used", 0)
	freeMemKB, _ := domain.GetMetricUint64Raw("ram_free", 0)
	maxMemKB, _ := domain.GetMetricUint64Raw("ram_max", 0)
	actualMemKB, _ := domain.GetMetricUint64Raw("ram_actual", 0)

	// These are in bytes
	vsizeBytes, _ := domain.GetMetricUint64Raw("ram_vsize", 0)
	rssBytes, _ := domain.GetMetricUint64Raw("ram_rss", 0)

	// Active percentage (stored as float64, returned as formatted string)
	activePctStr := domain.GetMetricFloat64("ram_activepct", 0)
	// Parse and reformat to 1 decimal place
	var activePct string
	if activePctStr != "" {
		var pct float64
		fmt.Sscanf(activePctStr, "%f", &pct)
		activePct = fmt.Sprintf("%.1f", pct)
	} else {
		activePct = "0.0"
	}

	// Swap activity (cumulative)
	swapIn, _ := domain.GetMetricUint64("ram_swapin", 0)
	swapOut, _ := domain.GetMetricUint64("ram_swapout", 0)

	// Page faults per interval (minor = soft, major = hard page faults)
	minflt := domain.GetMetricDiffUint64("ram_minflt", false)
	cminflt := domain.GetMetricDiffUint64("ram_cminflt", false)
	majflt := domain.GetMetricDiffUint64("ram_majflt", false)
	cmajflt := domain.GetMetricDiffUint64("ram_cmajflt", false)

	// Format memory values with correct units
	totalFmt := formatMemKB(totalKB)
	usedFmt := formatMemKB(usedKB)
	freeMemFmt := formatMemKB(freeMemKB)
	rssFmt := formatMemBytes(rssBytes)
	actualMemFmt := formatMemKB(actualMemKB)

	// Default fields: MEMSZ, GRANT, FREE, %ACTV, RSS, MCTL, MINFLT, MAJFLT
	result := append([]string{totalFmt}, usedFmt, freeMemFmt, activePct, rssFmt, actualMemFmt, minflt, majflt)
	if config.Options.Verbose {
		maxMemFmt := formatMemKB(maxMemKB)
		vsizeFmt := formatMemBytes(vsizeBytes)
		result = append(result, maxMemFmt, vsizeFmt, swapIn, swapOut, cminflt, cmajflt)
	}

	return result
}
