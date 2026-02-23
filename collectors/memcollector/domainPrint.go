package memcollector

import (
	"fmt"
	"proxtop/config"
	"proxtop/models"
)

func domainPrint(domain *models.Domain) []string {
	// esxtop style: MEMSZ (configured memory), GRANT (used), RSS (resident)
	total, _ := domain.GetMetricUint64("ram_total", 0)
	used, _ := domain.GetMetricUint64("ram_used", 0)
	freeMem, _ := domain.GetMetricUint64("ram_free", 0)
	maxMem, _ := domain.GetMetricUint64("ram_max", 0)
	actualMem, _ := domain.GetMetricUint64("ram_actual", 0)

	vsize, _ := domain.GetMetricUint64("ram_vsize", 0)
	rss, _ := domain.GetMetricUint64("ram_rss", 0)

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

	// Default fields: MEMSZ, GRANT, FREE, %ACTV, RSS, MCTL, MINFLT, MAJFLT
	result := append([]string{total}, used, freeMem, activePct, rss, actualMem, minflt, majflt)
	if config.Options.Verbose {
		result = append(result, maxMem, vsize, swapIn, swapOut, cminflt, cmajflt)
	}

	return result
}
