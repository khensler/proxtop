package cpucollector

import (
	"path"
	"path/filepath"
	"regexp"
	"strconv"

	"fmt"

	"proxtop/config"
	"proxtop/connector"
	"proxtop/models"
	"proxtop/util"
	libvirt "github.com/libvirt/libvirt-go"
)

func cpuLookup(domain *models.Domain, libvirtDomain libvirt.Domain) {
	// get amount of cores
	vcpus, err := libvirtDomain.GetVcpus()
	if err != nil {
		return
	}
	cores := len(vcpus)
	newMeasurementCores := models.CreateMeasurement(uint64(cores))
	domain.AddMetricMeasurement("cpu_cores", newMeasurementCores)

	// cache old thread IDs for cleanup
	var oldThreadIds []int
	oldThreadIds = append(oldThreadIds, domain.GetMetricIntArray("cpu_threadIDs")...)
	oldThreadIds = append(oldThreadIds, domain.GetMetricIntArray("cpu_otherThreadIDs")...)

	// get core thread IDs
	vCPUThreads, err := libvirtDomain.QemuMonitorCommand("info cpus", libvirt.DOMAIN_QEMU_MONITOR_COMMAND_HMP)
	if err != nil {
		return
	}
	regThreadID := regexp.MustCompile("thread_id=([0-9]*)\\s")
	threadIDsRaw := regThreadID.FindAllStringSubmatch(vCPUThreads, -1)
	coreThreadIDs := make([]int, len(threadIDsRaw))
	for i, thread := range threadIDsRaw {
		threadID, _ := strconv.Atoi(thread[1])
		coreThreadIDs[i] = threadID
		oldThreadIds = removeFromArray(oldThreadIds, threadID)
	}
	newMeasurementThreads := models.CreateMeasurement(coreThreadIDs)
	domain.AddMetricMeasurement("cpu_threadIDs", newMeasurementThreads)

	// get thread IDs
	tasksFolder := fmt.Sprint(config.Options.ProcFS, "/", domain.PID, "/task/*")
	files, err := filepath.Glob(tasksFolder)
	if err != nil {
		return
	}
	otherThreadIDs := make([]int, 0)
	i := 0
	for _, f := range files {
		taskID, _ := strconv.Atoi(path.Base(f))
		found := false
		for _, n := range coreThreadIDs {
			if taskID == n {
				// taskID is for vCPU core. skip.
				found = true
				break
			}
		}
		if found {
			// taskID is for vCPU core. skip.
			continue
		}
		// taskID is not for a vCPU core
		otherThreadIDs = append(otherThreadIDs, taskID)
		oldThreadIds = removeFromArray(oldThreadIds, taskID)
		i++
	}
	domain.AddMetricMeasurement("cpu_otherThreadIDs", models.CreateMeasurement(otherThreadIDs))

	// remove cached but not existent thread IDs
	for _, id := range oldThreadIds {
		domain.DelMetricMeasurement(fmt.Sprint("cpu_times_", id))
		domain.DelMetricMeasurement(fmt.Sprint("cpu_runqueues_", id))
		domain.DelMetricMeasurement(fmt.Sprint("cpu_other_times_", id))
		domain.DelMetricMeasurement(fmt.Sprint("cpu_other_runqueues_", id))
	}
}

func cpuCollect(domain *models.Domain) {
	// PART A: stats for VCORES from threadIDs
	cpuCollectMeasurements(domain, "cpu_threadIDs", "cpu_")
	// PART B: stats for other threads (i/o or emulation)
	cpuCollectMeasurements(domain, "cpu_otherThreadIDs", "cpu_other_")
}

func cpuCollectMeasurements(domain *models.Domain, metricName string, measurementPrefix string) {
	threadIDs := domain.GetMetricIntArray(metricName)
	for _, threadID := range threadIDs {
		schedstat := util.GetProcPIDSchedStat(threadID)
		domain.AddMetricMeasurement(fmt.Sprint(measurementPrefix, "times_", threadID), models.CreateMeasurement(schedstat.Cputime))
		domain.AddMetricMeasurement(fmt.Sprint(measurementPrefix, "runqueues_", threadID), models.CreateMeasurement(schedstat.Runqueue))
	}
}

func cpuPrint(domain *models.Domain) []string {
	cores, _ := domain.GetMetricUint64("cpu_cores", 0)

	// cpu util for vcores (%USED in esxtop terms)
	cputimeAllCores := CpuPrintThreadMetric(domain, "cpu_threadIDs", "cpu_times")
	// queue time is similar to %RDY (ready/steal time)
	queuetimeAllCores := CpuPrintThreadMetric(domain, "cpu_threadIDs", "cpu_runqueues")

	// cpu util for other threads (i/o or emulation) - similar to %SYS in esxtop
	otherCputimeAllCores := CpuPrintThreadMetric(domain, "cpu_otherThreadIDs", "cpu_other_times")
	otherQueuetimeAllCores := CpuPrintThreadMetric(domain, "cpu_otherThreadIDs", "cpu_other_runqueues")

	// put results together - include %sys (other threads) by default (esxtop style)
	result := append([]string{cores}, cputimeAllCores, queuetimeAllCores, otherCputimeAllCores)
	if config.Options.Verbose {
		result = append(result, otherQueuetimeAllCores)
	}
	return result
}

func CpuPrintThreadMetric(domain *models.Domain, lookupMetric string, metric string) string {
	threadIDs := domain.GetMetricIntArray(lookupMetric)
	var measurementSum float64
	var measurementCount int
	for _, threadID := range threadIDs {
		metricName := fmt.Sprint(metric, "_", threadID)
		measurementStr := domain.GetMetricDiffUint64(metricName, true)
		if measurementStr == "" {
			continue
		}
		measurement, err := strconv.ParseUint(measurementStr, 10, 64)
		if err != nil {
			continue
		}
		measurementSeconds := float64(measurement) / 1000000000 // since counters are nanoseconds
		measurementSum += measurementSeconds
		measurementCount++
	}

	var avg float64
	if measurementCount > 0 {
		avg = float64(measurementSum) / float64(measurementCount)
	}
	percent := avg * 100
	return fmt.Sprintf("%.0f", percent)
}

func removeFromArray(s []int, r int) []int {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

// cpuLookupProxmox handles CPU lookup for Proxmox VMs
func cpuLookupProxmox(domain *models.Domain, vmInfo connector.VMInfo) {
	// Get cores from VM info or config
	var cores int
	if vmInfo.Cores > 0 {
		cores = vmInfo.Cores
	} else {
		// Try to get from Proxmox connector
		proxmoxConn, ok := connector.CurrentConnector.(*connector.ProxmoxConnector)
		if ok {
			threads, err := proxmoxConn.GetCPUThreads(vmInfo)
			if err == nil {
				cores = len(threads)
			}
		}
	}
	if cores == 0 {
		cores = 1 // Default to 1 core
	}
	newMeasurementCores := models.CreateMeasurement(uint64(cores))
	domain.AddMetricMeasurement("cpu_cores", newMeasurementCores)

	// cache old thread IDs for cleanup
	var oldThreadIds []int
	oldThreadIds = append(oldThreadIds, domain.GetMetricIntArray("cpu_threadIDs")...)
	oldThreadIds = append(oldThreadIds, domain.GetMetricIntArray("cpu_otherThreadIDs")...)

	// Get vCPU thread IDs from Proxmox
	var coreThreadIDs []int
	proxmoxConn, ok := connector.CurrentConnector.(*connector.ProxmoxConnector)
	if ok {
		threads, err := proxmoxConn.GetCPUThreads(vmInfo)
		if err == nil {
			coreThreadIDs = threads
		}
	}

	for _, threadID := range coreThreadIDs {
		oldThreadIds = removeFromArray(oldThreadIds, threadID)
	}
	newMeasurementThreads := models.CreateMeasurement(coreThreadIDs)
	domain.AddMetricMeasurement("cpu_threadIDs", newMeasurementThreads)

	// get other thread IDs from /proc/<pid>/task
	tasksFolder := fmt.Sprint(config.Options.ProcFS, "/", domain.PID, "/task/*")
	files, err := filepath.Glob(tasksFolder)
	if err != nil {
		return
	}
	otherThreadIDs := make([]int, 0)
	for _, f := range files {
		taskID, _ := strconv.Atoi(path.Base(f))
		found := false
		for _, n := range coreThreadIDs {
			if taskID == n {
				found = true
				break
			}
		}
		if found {
			continue
		}
		otherThreadIDs = append(otherThreadIDs, taskID)
		oldThreadIds = removeFromArray(oldThreadIds, taskID)
	}
	domain.AddMetricMeasurement("cpu_otherThreadIDs", models.CreateMeasurement(otherThreadIDs))

	// remove cached but not existent thread IDs
	for _, id := range oldThreadIds {
		domain.DelMetricMeasurement(fmt.Sprint("cpu_times_", id))
		domain.DelMetricMeasurement(fmt.Sprint("cpu_runqueues_", id))
		domain.DelMetricMeasurement(fmt.Sprint("cpu_other_times_", id))
		domain.DelMetricMeasurement(fmt.Sprint("cpu_other_runqueues_", id))
	}
}
