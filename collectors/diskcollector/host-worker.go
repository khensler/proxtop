package diskcollector

// cf. https://www.percona.com/doc/percona-toolkit/LATEST/pt-diskstats.html#description

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"

	"proxtop/config"
	"proxtop/util"

	"proxtop/models"
)

func diskHostLookup(host *models.Host) {

	/*
		// find relevant devices
		devices := []string{}
		mounts := util.GetProcMounts()
		diskSources := strings.Split(host.GetMetricString("disk_sources", 0), ",")
		for _, source := range diskSources {
			// find best matching mountpoint
			var bestMount util.ProcMount
			for _, mount := range mounts {
				// matches at all?
				if !strings.HasPrefix(source, mount.Mountpoint) {
					continue
				}
				// matches better than already found one?
				if len(bestMount.Mountpoint) < len(mount.Mountpoint) {
					bestMount = mount
				}
			}
			// add bestMount to devices, if not contained
			found := false
			for _, device := range devices {
				if device == bestMount.Device {
					found = true
					break
				}
			}
			if !found {
				device := filepath.Base(bestMount.Device)
				devices = append(devices, device)
			}
		}
	*/

	devices := []string{}
	if config.Options.StorageDevice != "" {
		devices = strings.Split(config.Options.StorageDevice, ",")
	}

	// lookup diskstats for relevant devices
	diskstats := util.GetProcDiskstats()
	combinedDiskstat := util.ProcDiskstat{}
	combinedDiskstatCounts := uint64(0)
	if len(devices) > 0 {
		// consider only relevant devices
		for _, device := range devices {
			if stats, ok := diskstats[device]; ok {
				combinedDiskstat.Reads += stats.Reads
				combinedDiskstat.ReadsMerged += stats.ReadsMerged
				combinedDiskstat.SectorsRead += stats.SectorsRead
				combinedDiskstat.TimeReading += stats.TimeReading
				combinedDiskstat.Writes += stats.Writes
				combinedDiskstat.WritesMerged += stats.WritesMerged
				combinedDiskstat.SectorsWritten += stats.SectorsWritten
				combinedDiskstat.TimeWriting += stats.TimeWriting
				combinedDiskstat.CurrentOps += stats.CurrentOps
				combinedDiskstat.TimeForOps += stats.TimeForOps
				combinedDiskstat.WeightedTimeForOps += stats.WeightedTimeForOps
				combinedDiskstatCounts++
			}
		}
	} else {

		// consider all available devices (clean duplicates like sda and sda1)
		diskstats = clearDuplicateDevices(diskstats)
		for _, stats := range diskstats {
			combinedDiskstat.Reads += stats.Reads
			combinedDiskstat.ReadsMerged += stats.ReadsMerged
			combinedDiskstat.SectorsRead += stats.SectorsRead
			combinedDiskstat.TimeReading += stats.TimeReading
			combinedDiskstat.Writes += stats.Writes
			combinedDiskstat.WritesMerged += stats.WritesMerged
			combinedDiskstat.SectorsWritten += stats.SectorsWritten
			combinedDiskstat.TimeWriting += stats.TimeWriting
			combinedDiskstat.CurrentOps += stats.CurrentOps
			combinedDiskstat.TimeForOps += stats.TimeForOps
			combinedDiskstat.WeightedTimeForOps += stats.WeightedTimeForOps
			combinedDiskstatCounts++
		}
	}

	host.AddMetricMeasurement("disk_device_reads", models.CreateMeasurement(combinedDiskstat.Reads))
	host.AddMetricMeasurement("disk_device_readsmerged", models.CreateMeasurement(combinedDiskstat.ReadsMerged))
	host.AddMetricMeasurement("disk_device_sectorsread", models.CreateMeasurement(combinedDiskstat.SectorsRead))
	host.AddMetricMeasurement("disk_device_timereading", models.CreateMeasurement(combinedDiskstat.TimeReading))
	host.AddMetricMeasurement("disk_device_writes", models.CreateMeasurement(combinedDiskstat.Writes))
	host.AddMetricMeasurement("disk_device_writesmerged", models.CreateMeasurement(combinedDiskstat.WritesMerged))
	host.AddMetricMeasurement("disk_device_sectorswritten", models.CreateMeasurement(combinedDiskstat.SectorsWritten))
	host.AddMetricMeasurement("disk_device_timewriting", models.CreateMeasurement(combinedDiskstat.TimeWriting))
	host.AddMetricMeasurement("disk_device_currentops", models.CreateMeasurement(combinedDiskstat.CurrentOps))
	host.AddMetricMeasurement("disk_device_timeforops", models.CreateMeasurement(combinedDiskstat.TimeForOps))
	host.AddMetricMeasurement("disk_device_weightedtimeforops", models.CreateMeasurement(combinedDiskstat.WeightedTimeForOps))
	host.AddMetricMeasurement("disk_device_count", models.CreateMeasurement(combinedDiskstatCounts))

	// Store per-device stats for physical disk view (enables rate calculations)
	allDiskstats := util.GetProcDiskstats()
	for name, dev := range allDiskstats {
		// Filter out loop, ram, dm- devices
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "dm-") {
			continue
		}
		prefix := fmt.Sprintf("disk_physdev_%s_", name)
		host.AddMetricMeasurement(prefix+"reads", models.CreateMeasurement(dev.Reads))
		host.AddMetricMeasurement(prefix+"writes", models.CreateMeasurement(dev.Writes))
		host.AddMetricMeasurement(prefix+"sectorsread", models.CreateMeasurement(dev.SectorsRead))
		host.AddMetricMeasurement(prefix+"sectorswritten", models.CreateMeasurement(dev.SectorsWritten))
		host.AddMetricMeasurement(prefix+"timeforops", models.CreateMeasurement(dev.TimeForOps))
		host.AddMetricMeasurement(prefix+"weightedtimeforops", models.CreateMeasurement(dev.WeightedTimeForOps))
	}
}

func diskHostCollect(host *models.Host) {
	// %UTIL: total time during which I/Os were in progress, divided by the
	// sampling interval (device busy percentage)
	ioutil := diffInMilliseconds(host, "disk_device_timeforops", true)

	// QLEN (queue length/size): weighted number of milliseconds spent doing I/Os divided by
	// the milliseconds elapsed - represents average queue length
	queuelen := diffInMilliseconds(host, "disk_device_weightedtimeforops", false)

	// Calculate AWAIT (average wait time) and SVCTM (average service time)
	await, servicetime := getTimes(host)

	// Calculate MB/s for reads and writes
	mbReadPerSec, mbWritePerSec := getMBPerSec(host)

	host.AddMetricMeasurement("disk_device_ioutil", models.CreateMeasurement(ioutil))
	host.AddMetricMeasurement("disk_device_queuelen", models.CreateMeasurement(queuelen))
	host.AddMetricMeasurement("disk_device_await", models.CreateMeasurement(await))
	host.AddMetricMeasurement("disk_device_servicetime", models.CreateMeasurement(servicetime))
	host.AddMetricMeasurement("disk_device_mbread", models.CreateMeasurement(mbReadPerSec))
	host.AddMetricMeasurement("disk_device_mbwrite", models.CreateMeasurement(mbWritePerSec))
}

// getMBPerSec calculates MB/s for reads and writes
func getMBPerSec(host *models.Host) (string, string) {
	// Sectors are 512 bytes each
	sectorsRead := host.GetMetricDiffUint64AsFloat("disk_device_sectorsread", true)
	sectorsWritten := host.GetMetricDiffUint64AsFloat("disk_device_sectorswritten", true)

	mbRead := (sectorsRead * 512) / (1024 * 1024)
	mbWrite := (sectorsWritten * 512) / (1024 * 1024)

	return fmt.Sprintf("%.2f", mbRead), fmt.Sprintf("%.2f", mbWrite)
}

func diskPrintHost(host *models.Host) []string {
	// I/O rates
	diskDeviceReads := host.GetMetricDiffUint64("disk_device_reads", true)
	diskDeviceWrites := host.GetMetricDiffUint64("disk_device_writes", true)

	// Throughput (MB/s)
	mbRead := host.GetMetricString("disk_device_mbread", 0)
	mbWrite := host.GetMetricString("disk_device_mbwrite", 0)

	// Device busy percentage
	ioutil := host.GetMetricString("disk_device_ioutil", 0)

	// Queue depth (current I/Os in progress) - instantaneous value
	qdepthRaw, _ := host.GetMetricUint64("disk_device_currentops", 0)

	// Queue length (average queue size)
	queuelen := host.GetMetricString("disk_device_queuelen", 0)

	// Service time and wait time (latency metrics)
	servicetime := host.GetMetricString("disk_device_servicetime", 0)
	await := host.GetMetricString("disk_device_await", 0)

	// Default: READS/s, WRITES/s, MBRD/s, MBWR/s, %UTIL, QDEPTH, QLEN, SVCTM, AWAIT
	result := []string{diskDeviceReads, diskDeviceWrites, mbRead, mbWrite, ioutil, qdepthRaw, queuelen, servicetime, await}

	if config.Options.Verbose {
		// Additional detailed stats for verbose mode
		diskDeviceReadsmerged := host.GetMetricDiffUint64("disk_device_readsmerged", true)
		diskDeviceSectorsread := host.GetMetricDiffUint64("disk_device_sectorsread", true)
		diskDeviceTimereading := host.GetMetricDiffUint64("disk_device_timereading", true)
		diskDeviceWritesmerged := host.GetMetricDiffUint64("disk_device_writesmerged", true)
		diskDeviceSectorswritten := host.GetMetricDiffUint64("disk_device_sectorswritten", true)
		diskDeviceTimewriting := host.GetMetricDiffUint64("disk_device_timewriting", true)
		diskDeviceTimeforops := host.GetMetricDiffUint64("disk_device_timeforops", true)
		diskDeviceWeightedtimeforops := host.GetMetricDiffUint64("disk_device_weightedtimeforops", true)
		diskDeviceCountStr, _ := host.GetMetricUint64("disk_device_count", 0)

		result = append(result, diskDeviceReadsmerged, diskDeviceSectorsread, diskDeviceTimereading)
		result = append(result, diskDeviceWritesmerged, diskDeviceSectorswritten, diskDeviceTimewriting)
		result = append(result, diskDeviceTimeforops, diskDeviceWeightedtimeforops, diskDeviceCountStr)
	}

	return result
}

func diffInMilliseconds(host *models.Host, metricName string, inPercent bool) string {
	var output string
	var percent float64
	if metric, ok := host.GetMetric(metricName); ok {
		if len(metric.Values) >= 2 {
			// get first value
			byteValue1 := metric.Values[0].Value
			reader1 := bytes.NewReader(byteValue1)
			decoder1 := gob.NewDecoder(reader1)
			var value1 uint64
			decoder1.Decode(&value1)

			// get second value
			byteValue2 := metric.Values[1].Value
			reader2 := bytes.NewReader(byteValue2)
			decoder2 := gob.NewDecoder(reader2)
			var value2 uint64
			decoder2.Decode(&value2)

			// calculate value diff per time
			value := float64(value1 - value2)

			// get time diff
			ts1 := metric.Values[0].Timestamp
			ts2 := metric.Values[1].Timestamp
			diffSeconds := ts1.Sub(ts2).Seconds()
			valuePerSecond := value / 1000 // since value is in ms
			ratio := valuePerSecond / diffSeconds

			if inPercent {
				percent = ratio * 100 // compute it as percent
				output = fmt.Sprintf("%.0f", percent)
			} else {
				output = fmt.Sprintf("%.0f", ratio)
			}
		}
	}
	return output
}

func getTimes(host *models.Host) (string, string) {
	queueTime := ""
	serviceTime := ""

	reads := host.GetMetricDiffUint64AsFloat("disk_device_reads", true)
	readsMerged := host.GetMetricDiffUint64AsFloat("disk_device_readsmerged", true)
	writes := host.GetMetricDiffUint64AsFloat("disk_device_writes", true)
	writesMerged := host.GetMetricDiffUint64AsFloat("disk_device_writesmerged", true)
	timeForOps := host.GetMetricDiffUint64AsFloat("disk_device_timeforops", true)
	currentOps := host.GetMetricDiffUint64AsFloat("disk_device_currentops", true)
	weightedTimeForOps := host.GetMetricDiffUint64AsFloat("disk_device_weightedtimeforops", true)

	// serviceTime:
	// delta[field10] / delta[field1, 2, 5, 6]
	// => TimeForOps / (Reads, ReadsMerged, Writes, WritesMerged)
	sum1 := (reads + readsMerged + writes + writesMerged)
	var stime float64
	if sum1 > 0 {
		stime = timeForOps / sum1
	}

	// queueTime:
	// delta[field11] / (delta[field1, 2, 5, 6] + delta[field9])
	// - serviceTime
	// => WeightedTimeForOps / (Reads, ReadsMerged, Writes, WritesMerged + CurrentOps)
	sum2 := sum1 + currentOps
	var qtime float64
	if sum2 > 0 {
		qtime = (weightedTimeForOps / sum2) - stime
	}

	if stime < 0 {
		stime = 0
	}
	if qtime < 0 {
		qtime = 0
	}

	serviceTime = fmt.Sprintf("%.0f", stime)
	queueTime = fmt.Sprintf("%.0f", qtime)

	return queueTime, serviceTime
}

func clearDuplicateDevices(diskstats map[string]util.ProcDiskstat) map[string]util.ProcDiskstat {
	result := make(map[string]util.ProcDiskstat)
	keys := make([]string, 0, len(diskstats))
	for k := range diskstats {
		keys = append(keys, k)
	}

	// remove duplicates like sda and sda1 - only consider sda1
	for key, stats := range diskstats {
		// is there a key in keys which is longer?
		considerDisk := true
		for _, k := range keys {
			if strings.HasPrefix(k, key) && len(k) > len(key) {
				// found more detailed device name
				considerDisk = false
				break
			}
		}
		if considerDisk {
			result[key] = stats
		}
	}
	return result

}

// HostDiskFields returns the field names for host physical disk view
func HostDiskFields() []string {
	fields := []string{
		"dsk_DEVICE",
		"dsk_READS/s",
		"dsk_WRITES/s",
		"dsk_MBRD/s",
		"dsk_MBWR/s",
		"dsk_%UTIL",
		"dsk_QDEPTH",
		"dsk_SVCTM",
		"dsk_AWAIT",
	}
	if config.Options.Verbose {
		fields = append(fields,
			"dsk_rdmerged",
			"dsk_sectorsrd",
			"dsk_timerd",
			"dsk_wrmerged",
			"dsk_sectorswr",
			"dsk_timewr",
			"dsk_timeforops",
			"dsk_weightedtime",
		)
	}
	return fields
}

// HostPrintPerDevice returns per-device disk stats for physical disk view
// Returns a map of device name -> []string (field values in same order as HostDiskFields)
func HostPrintPerDevice() map[string][]string {
	diskstats := util.GetProcDiskstats()
	result := make(map[string][]string)
	host := &models.Collection.Host

	// Filter out loop, ram, dm- devices
	for name, dev := range diskstats {
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "dm-") {
			continue
		}

		prefix := fmt.Sprintf("disk_physdev_%s_", name)

		// Calculate per-second rates using stored metrics
		readsRate := host.GetMetricDiffUint64AsFloat(prefix+"reads", true)
		writesRate := host.GetMetricDiffUint64AsFloat(prefix+"writes", true)
		reads := fmt.Sprintf("%.0f", readsRate)
		writes := fmt.Sprintf("%.0f", writesRate)

		// Calculate MB/s from sectors (512 bytes each)
		sectorsReadRate := host.GetMetricDiffUint64AsFloat(prefix+"sectorsread", true)
		sectorsWrittenRate := host.GetMetricDiffUint64AsFloat(prefix+"sectorswritten", true)
		mbRead := fmt.Sprintf("%.2f", sectorsReadRate*512/1024/1024)
		mbWrite := fmt.Sprintf("%.2f", sectorsWrittenRate*512/1024/1024)

		// %UTIL: time spent doing I/O (ms/s), divided by 1000ms = percentage
		// TimeForOps is cumulative ms spent doing I/O, diff gives ms/s
		timeForOpsRate := host.GetMetricDiffUint64AsFloat(prefix+"timeforops", true)
		utilPct := timeForOpsRate / 10 // ms/s to percentage (1000ms = 100%)
		if utilPct > 100 {
			utilPct = 100 // Cap at 100%
		}
		utilStr := fmt.Sprintf("%.0f", utilPct)

		// Queue depth (current ops in flight) - instantaneous value
		qDepth := fmt.Sprintf("%d", dev.CurrentOps)

		// Calculate SVCTM and AWAIT using rate values
		// SVCTM = time spent on completed I/O / number of completed I/Os
		// AWAIT = weighted time / number of completed I/Os
		totalOpsRate := readsRate + writesRate
		var svcTm, awaitStr string
		if totalOpsRate > 0.1 { // Avoid division by very small numbers
			// SVCTM: average service time per I/O
			svcTm = fmt.Sprintf("%.2f", timeForOpsRate/totalOpsRate)
			// AWAIT: average wait time per I/O (includes queue time)
			weightedTimeRate := host.GetMetricDiffUint64AsFloat(prefix+"weightedtimeforops", true)
			awaitStr = fmt.Sprintf("%.2f", weightedTimeRate/totalOpsRate)
		} else {
			svcTm = "0.00"
			awaitStr = "0.00"
		}

		values := []string{name, reads, writes, mbRead, mbWrite, utilStr, qDepth, svcTm, awaitStr}

		if config.Options.Verbose {
			values = append(values,
				fmt.Sprintf("%d", dev.ReadsMerged),
				fmt.Sprintf("%d", dev.SectorsRead),
				fmt.Sprintf("%d", dev.TimeReading),
				fmt.Sprintf("%d", dev.WritesMerged),
				fmt.Sprintf("%d", dev.SectorsWritten),
				fmt.Sprintf("%d", dev.TimeWriting),
				fmt.Sprintf("%d", dev.TimeForOps),
				fmt.Sprintf("%d", dev.WeightedTimeForOps),
			)
		}

		result[name] = values
	}

	return result
}
