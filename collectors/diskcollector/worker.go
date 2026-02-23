package diskcollector

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"proxtop/config"
	"proxtop/connector"
	"proxtop/models"
	"proxtop/util"
	libvirt "github.com/libvirt/libvirt-go"
	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

type diskstats struct {
	RdBytesSet         bool
	RdBytes            int64
	RdReqSet           bool
	RdReq              int64
	RdTotalTimesSet    bool
	RdTotalTimes       int64
	WrBytesSet         bool
	WrBytes            int64
	WrReqSet           bool
	WrReq              int64
	WrTotalTimesSet    bool
	WrTotalTimes       int64
	FlushReqSet        bool
	FlushReq           int64
	FlushTotalTimesSet bool
	FlushTotalTimes    int64
	// ErrsSet            bool
	// Errs               int64
	Capacity   uint64
	Allocation uint64
	Physical   uint64
}

func diskLookup(domain *models.Domain, libvirtDomain libvirt.Domain) {

	xmldoc, _ := libvirtDomain.GetXMLDesc(0)
	domcfg := &libvirtxml.Domain{}
	domcfg.Unmarshal(xmldoc)

	// generate list of virtual disks
	// var disks []string
	// for _, disk := range domcfg.Devices.Disks {
	//	disks = append(disks, disk.Target.Dev)
	// }
	// newMeasurementDisks := models.CreateMeasurement(disks)
	// domain.AddMetricMeasurement("disk_disks", newMeasurementDisks)

	// sum up stats from virtual disks
	var sums diskstats
	disksources := ""
	for _, disk := range domcfg.Devices.Disks {
		if disk.Target == nil || disk.Target.Dev == "" {
			// skip if disk specs are invalid
			continue
		}
		dev := disk.Target.Dev
		ioStats, err := libvirtDomain.BlockStats(dev)

		if ioStats != nil && err == nil {
			// ioStats.ErrsSet - works only for xen
			/*if ioStats.ErrsSet {
				sums.ErrsSet = true
				sums.Errs += ioStats.Errs
			}*/
			// ioStats.FlushReq
			if ioStats.FlushReqSet {
				sums.FlushReqSet = true
				sums.FlushReq += ioStats.FlushReq
			}
			// ioStats.FlushTotalTimes
			if ioStats.FlushTotalTimesSet {
				sums.FlushTotalTimesSet = true
				sums.FlushTotalTimes += ioStats.FlushTotalTimes
			}
			// ioStats.RdBytes
			if ioStats.RdBytesSet {
				sums.RdBytesSet = true
				sums.RdBytes += ioStats.RdBytes
			}
			// ioStats.RdReq
			if ioStats.RdReqSet {
				sums.RdReqSet = true
				sums.RdReq += ioStats.RdReq
			}
			// ioStats.RdTotalTimes
			if ioStats.RdTotalTimesSet {
				sums.RdTotalTimesSet = true
				sums.RdTotalTimes += ioStats.RdTotalTimes
			}
			// ioStats.WrBytes
			if ioStats.WrBytesSet {
				sums.WrBytesSet = true
				sums.WrBytes += ioStats.WrBytes
			}
			// ioStats.WrReq
			if ioStats.WrReqSet {
				sums.WrReqSet = true
				sums.WrReq += ioStats.WrReq
			}
			// ioStats.WrTotalTimes
			if ioStats.WrTotalTimesSet {
				sums.WrTotalTimesSet = true
				sums.WrTotalTimes += ioStats.WrTotalTimes
			}
		}

		sizeStats, err := libvirtDomain.GetBlockInfo(dev, 0)
		// sizes
		if sizeStats != nil && err == nil {
			sums.Capacity += sizeStats.Capacity
			sums.Allocation += sizeStats.Allocation
			sums.Physical += sizeStats.Physical
		}

		// find source path
		if disk.Source != nil && disk.Source.File != nil {
			// only consider file based disks
			sourcefile := disk.Source.File
			sourcedir := filepath.Dir(sourcefile.File)
			if !strings.Contains(disksources, sourcedir) {
				if disksources != "" {
					disksources += ","
				}
				disksources += sourcedir
			}
		}

	}

	// sizes
	domain.AddMetricMeasurement("disk_size_capacity", models.CreateMeasurement(uint64(sums.Capacity)))
	domain.AddMetricMeasurement("disk_size_allocation", models.CreateMeasurement(uint64(sums.Allocation)))
	domain.AddMetricMeasurement("disk_size_physical", models.CreateMeasurement(uint64(sums.Physical)))
	// IOs
	// domain.AddMetricMeasurement("disk_stats_errs", models.CreateMeasurement(uint64(sums.Errs)))
	domain.AddMetricMeasurement("disk_stats_flushreq", models.CreateMeasurement(uint64(sums.FlushReq)))
	domain.AddMetricMeasurement("disk_stats_flushtotaltimes", models.CreateMeasurement(uint64(sums.FlushTotalTimes)))
	domain.AddMetricMeasurement("disk_stats_rdbytes", models.CreateMeasurement(uint64(sums.RdBytes)))
	domain.AddMetricMeasurement("disk_stats_rdreq", models.CreateMeasurement(uint64(sums.RdReq)))
	domain.AddMetricMeasurement("disk_stats_rdtotaltimes", models.CreateMeasurement(uint64(sums.RdTotalTimes)))
	domain.AddMetricMeasurement("disk_stats_wrbytes", models.CreateMeasurement(uint64(sums.WrBytes)))
	domain.AddMetricMeasurement("disk_stats_wrreq", models.CreateMeasurement(uint64(sums.WrReq)))
	domain.AddMetricMeasurement("disk_stats_wrtotaltimes", models.CreateMeasurement(uint64(sums.WrTotalTimes)))
	// information
	domain.AddMetricMeasurement("disk_sources", models.CreateMeasurement(disksources))
}

func diskCollect(domain *models.Domain, host *models.Host) {
	pid := domain.PID
	stats := util.GetProcPIDStat(pid)
	domain.AddMetricMeasurement("disk_delayblkio", models.CreateMeasurement(uint64(stats.DelayacctBlkioTicks)))

	// calculate ioutil as estimation
	domainIOUtil := estimateIOUtil(domain, host)
	domain.AddMetricMeasurement("disk_ioutil", models.CreateMeasurement(domainIOUtil))
}

// formatDiskSize formats a disk size value, optionally in human-readable format
func formatDiskSize(value string) string {
	if config.Options.HumanReadable {
		return util.FormatBytesFromString(value)
	}
	return value
}

func diskPrint(domain *models.Domain) []string {
	capacity, _ := domain.GetMetricUint64("disk_size_capacity", 0)
	allocation, _ := domain.GetMetricUint64("disk_size_allocation", 0)
	physical, _ := domain.GetMetricUint64("disk_size_physical", 0)

	// Get IO stats as floats for calculation
	flushreqFloat := domain.GetMetricDiffUint64AsFloat("disk_stats_flushreq", true)
	flushtotaltimesFloat := domain.GetMetricDiffUint64AsFloat("disk_stats_flushtotaltimes", true)
	rdbytesFloat := domain.GetMetricDiffUint64AsFloat("disk_stats_rdbytes", true)
	rdreqFloat := domain.GetMetricDiffUint64AsFloat("disk_stats_rdreq", true)
	rdtotaltimesFloat := domain.GetMetricDiffUint64AsFloat("disk_stats_rdtotaltimes", true)
	wrbytesFloat := domain.GetMetricDiffUint64AsFloat("disk_stats_wrbytes", true)
	wrreqFloat := domain.GetMetricDiffUint64AsFloat("disk_stats_wrreq", true)
	wrtotaltimesFloat := domain.GetMetricDiffUint64AsFloat("disk_stats_wrtotaltimes", true)

	delayblkio := domain.GetMetricDiffUint64("disk_delayblkio", true)

	ioutil := domain.GetMetricString("disk_ioutil", 0)

	// esxtop style: READS/s, WRITES/s, MBRD/s, MBWR/s, LAT/rd (ms), LAT/wr (ms)
	rdreq := fmt.Sprintf("%.0f", rdreqFloat)
	wrreq := fmt.Sprintf("%.0f", wrreqFloat)
	mbRead := fmt.Sprintf("%.2f", rdbytesFloat/1024/1024)
	mbWrite := fmt.Sprintf("%.2f", wrbytesFloat/1024/1024)

	// Calculate average latency in ms (total time in ns / requests)
	var latRd, latWr, latFl, latAvg string
	if rdreqFloat > 0 {
		latRd = fmt.Sprintf("%.2f", rdtotaltimesFloat/rdreqFloat/1000000) // ns to ms
	} else {
		latRd = "0.00"
	}
	if wrreqFloat > 0 {
		latWr = fmt.Sprintf("%.2f", wrtotaltimesFloat/wrreqFloat/1000000) // ns to ms
	} else {
		latWr = "0.00"
	}
	if flushreqFloat > 0 {
		latFl = fmt.Sprintf("%.2f", flushtotaltimesFloat/flushreqFloat/1000000) // ns to ms
	} else {
		latFl = "0.00"
	}

	// Calculate overall average latency (combined rd+wr+flush)
	totalOps := rdreqFloat + wrreqFloat + flushreqFloat
	totalTime := rdtotaltimesFloat + wrtotaltimesFloat + flushtotaltimesFloat
	if totalOps > 0 {
		latAvg = fmt.Sprintf("%.2f", totalTime/totalOps/1000000)
	} else {
		latAvg = "0.00"
	}

	// Format disk sizes (human-readable if enabled)
	capacityFmt := formatDiskSize(capacity)
	allocationFmt := formatDiskSize(allocation)

	// Default: SIZE, ALLOC, %UTIL, READS/s, WRITES/s, MBRD/s, MBWR/s, LAT/rd, LAT/wr, LAT/fl, LAT/avg
	result := append([]string{capacityFmt}, allocationFmt, ioutil, rdreq, wrreq, mbRead, mbWrite, latRd, latWr, latFl, latAvg)
	if config.Options.Verbose {
		flushreq := fmt.Sprintf("%.0f", flushreqFloat)
		// Total time breakdown (in ms)
		rdTotalMs := fmt.Sprintf("%.0f", rdtotaltimesFloat/1000000)
		wrTotalMs := fmt.Sprintf("%.0f", wrtotaltimesFloat/1000000)
		flTotalMs := fmt.Sprintf("%.0f", flushtotaltimesFloat/1000000)
		physicalFmt := formatDiskSize(physical)
		result = append(result, physicalFmt, flushreq, rdTotalMs, wrTotalMs, flTotalMs, delayblkio)
	}
	return result
}

func estimateIOUtil(domain *models.Domain, host *models.Host) string {
	hostIOUtilstr := host.GetMetricString("disk_device_ioutil", 0)
	hostIOUtil, errc := strconv.Atoi(hostIOUtilstr)
	if errc != nil {
		return ""
	}
	hostReads := host.GetMetricDiffUint64AsFloat("disk_device_sectorsread", true)
	hostWrites := host.GetMetricDiffUint64AsFloat("disk_device_sectorswritten", true)
	hostReadBytes := hostReads * 512
	hostWritesBytes := hostWrites * 512

	domainReadBytes := domain.GetMetricDiffUint64AsFloat("disk_stats_rdbytes", true)
	domainWrittenBytes := domain.GetMetricDiffUint64AsFloat("disk_stats_wrbytes", true)

	hostLoad := hostReadBytes + hostWritesBytes
	domainLoad := domainReadBytes + domainWrittenBytes

	var ratio float64
	if hostLoad > 0 {
		ratio = domainLoad / hostLoad
	}
	if ratio > 1 {
		ratio = 1
	}
	domainIOUtil := ratio * float64(hostIOUtil)
	domainIOUtilStr := fmt.Sprintf("%.0f", domainIOUtil)

	//fmt.Printf("\thost: %.0f MB/s domain: %.0f MB/s - hostio: %d%% domainio: %s%%\n", hostLoad/1024/1024, domainLoad/1024/1024, hostIOUtil, domainIOUtilStr)

	return domainIOUtilStr
}

// diskLookupProxmox handles disk lookup for Proxmox VMs
func diskLookupProxmox(domain *models.Domain, vmInfo connector.VMInfo) {
	// Get disk stats from Proxmox connector
	proxmoxConn, ok := connector.CurrentConnector.(*connector.ProxmoxConnector)
	if !ok {
		return
	}

	// Get totals
	stats, err := proxmoxConn.GetDiskStats(vmInfo)
	if err != nil {
		return
	}

	// Get per-disk stats
	perDiskStats, diskNames, perErr := proxmoxConn.GetPerDiskStats(vmInfo)
	if perErr == nil && len(diskNames) > 0 {
		// Store disk device list
		domain.AddMetricMeasurement("disk_devices", models.CreateMeasurement(diskNames))

		// Store per-disk stats
		for diskName, diskStats := range perDiskStats {
			domain.AddMetricMeasurement(fmt.Sprintf("disk_stats_rdbytes_%s", diskName), models.CreateMeasurement(uint64(diskStats.RdBytes)))
			domain.AddMetricMeasurement(fmt.Sprintf("disk_stats_wrbytes_%s", diskName), models.CreateMeasurement(uint64(diskStats.WrBytes)))
			domain.AddMetricMeasurement(fmt.Sprintf("disk_stats_rdreq_%s", diskName), models.CreateMeasurement(uint64(diskStats.RdReq)))
			domain.AddMetricMeasurement(fmt.Sprintf("disk_stats_wrreq_%s", diskName), models.CreateMeasurement(uint64(diskStats.WrReq)))
			domain.AddMetricMeasurement(fmt.Sprintf("disk_stats_flushreq_%s", diskName), models.CreateMeasurement(uint64(diskStats.FlushReq)))
			domain.AddMetricMeasurement(fmt.Sprintf("disk_stats_rdtotaltimes_%s", diskName), models.CreateMeasurement(uint64(diskStats.RdTotalTimes)))
			domain.AddMetricMeasurement(fmt.Sprintf("disk_stats_wrtotaltimes_%s", diskName), models.CreateMeasurement(uint64(diskStats.WrTotalTimes)))
			domain.AddMetricMeasurement(fmt.Sprintf("disk_stats_flushtotaltimes_%s", diskName), models.CreateMeasurement(uint64(diskStats.FlushTotalTimes)))
		}
	}

	// sizes (totals)
	domain.AddMetricMeasurement("disk_size_capacity", models.CreateMeasurement(stats.Capacity))
	domain.AddMetricMeasurement("disk_size_allocation", models.CreateMeasurement(stats.Capacity)) // Use capacity as allocation
	domain.AddMetricMeasurement("disk_size_physical", models.CreateMeasurement(stats.Physical))

	// IOs (totals)
	domain.AddMetricMeasurement("disk_stats_flushreq", models.CreateMeasurement(uint64(stats.FlushReq)))
	domain.AddMetricMeasurement("disk_stats_flushtotaltimes", models.CreateMeasurement(uint64(stats.FlushTotalTimes)))
	domain.AddMetricMeasurement("disk_stats_rdbytes", models.CreateMeasurement(uint64(stats.RdBytes)))
	domain.AddMetricMeasurement("disk_stats_rdreq", models.CreateMeasurement(uint64(stats.RdReq)))
	domain.AddMetricMeasurement("disk_stats_rdtotaltimes", models.CreateMeasurement(uint64(stats.RdTotalTimes)))
	domain.AddMetricMeasurement("disk_stats_wrbytes", models.CreateMeasurement(uint64(stats.WrBytes)))
	domain.AddMetricMeasurement("disk_stats_wrreq", models.CreateMeasurement(uint64(stats.WrReq)))
	domain.AddMetricMeasurement("disk_stats_wrtotaltimes", models.CreateMeasurement(uint64(stats.WrTotalTimes)))

	// disk sources - use Proxmox storage path
	domain.AddMetricMeasurement("disk_sources", models.CreateMeasurement(fmt.Sprintf("/var/lib/vz/images/%s", vmInfo.VMID)))
}

// DiskPrintPerDevice returns per-disk stats for a domain
// Returns a map of disk device name -> []string (same format as diskPrint but per-device)
func DiskPrintPerDevice(domain *models.Domain) map[string][]string {
	diskDevices := domain.GetMetricStringArray("disk_devices")
	result := make(map[string][]string)

	for _, devname := range diskDevices {
		// Get per-disk stats
		rdBytesFloat := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("disk_stats_rdbytes_%s", devname), true)
		wrBytesFloat := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("disk_stats_wrbytes_%s", devname), true)
		rdReqFloat := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("disk_stats_rdreq_%s", devname), true)
		wrReqFloat := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("disk_stats_wrreq_%s", devname), true)
		flReqFloat := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("disk_stats_flushreq_%s", devname), true)
		rdTotalTimesFloat := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("disk_stats_rdtotaltimes_%s", devname), true)
		wrTotalTimesFloat := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("disk_stats_wrtotaltimes_%s", devname), true)
		flTotalTimesFloat := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("disk_stats_flushtotaltimes_%s", devname), true)

		// Calculate stats same as diskPrint
		rdReq := fmt.Sprintf("%.0f", rdReqFloat)
		wrReq := fmt.Sprintf("%.0f", wrReqFloat)
		mbRead := fmt.Sprintf("%.2f", rdBytesFloat/1024/1024)
		mbWrite := fmt.Sprintf("%.2f", wrBytesFloat/1024/1024)

		var latRd, latWr, latFl, latAvg string
		if rdReqFloat > 0 {
			latRd = fmt.Sprintf("%.2f", rdTotalTimesFloat/rdReqFloat/1000000)
		} else {
			latRd = "0.00"
		}
		if wrReqFloat > 0 {
			latWr = fmt.Sprintf("%.2f", wrTotalTimesFloat/wrReqFloat/1000000)
		} else {
			latWr = "0.00"
		}
		if flReqFloat > 0 {
			latFl = fmt.Sprintf("%.2f", flTotalTimesFloat/flReqFloat/1000000)
		} else {
			latFl = "0.00"
		}

		// Calculate overall average latency
		totalOps := rdReqFloat + wrReqFloat + flReqFloat
		totalTime := rdTotalTimesFloat + wrTotalTimesFloat + flTotalTimesFloat
		if totalOps > 0 {
			latAvg = fmt.Sprintf("%.2f", totalTime/totalOps/1000000)
		} else {
			latAvg = "0.00"
		}

		// READS/s, WRITES/s, MBRD/s, MBWR/s, LAT/rd, LAT/wr, LAT/fl, LAT/avg
		result[devname] = []string{rdReq, wrReq, mbRead, mbWrite, latRd, latWr, latFl, latAvg}
	}

	return result
}
