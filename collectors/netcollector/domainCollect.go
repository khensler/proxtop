package netcollector

import (
	"fmt"

	"proxtop/models"
	"proxtop/util"
)

func domainCollect(domain *models.Domain) {
	/*
		// get stats from netstat
		stats := util.GetProcNetstat(domain.PID)
		domain.AddMetricMeasurement("net_ipextinoctets", models.CreateMeasurement(uint64(stats.IPExtInOctets)))
		domain.AddMetricMeasurement("net_ipextoutoctets", models.CreateMeasurement(uint64(stats.IPExtOutOctets)))
	*/

	// get stats from net/dev for domain interfaces
	ifs := domain.GetMetricStringArray("net_interfaces")
	statsSum := util.ProcPIDNetDev{}
	for _, devname := range ifs {
		devStats := util.GetProcPIDNetDev(domain.PID, devname)

		// Store per-interface stats with device name suffix
		domain.AddMetricMeasurement(fmt.Sprintf("net_ReceivedBytes_%s", devname), models.CreateMeasurement(uint64(devStats.ReceivedBytes)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_ReceivedPackets_%s", devname), models.CreateMeasurement(uint64(devStats.ReceivedPackets)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_ReceivedErrs_%s", devname), models.CreateMeasurement(uint64(devStats.ReceivedErrs)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_ReceivedDrop_%s", devname), models.CreateMeasurement(uint64(devStats.ReceivedDrop)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_ReceivedFifo_%s", devname), models.CreateMeasurement(uint64(devStats.ReceivedFifo)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_ReceivedFrame_%s", devname), models.CreateMeasurement(uint64(devStats.ReceivedFrame)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_ReceivedCompressed_%s", devname), models.CreateMeasurement(uint64(devStats.ReceivedCompressed)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_ReceivedMulticast_%s", devname), models.CreateMeasurement(uint64(devStats.ReceivedMulticast)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_TransmittedBytes_%s", devname), models.CreateMeasurement(uint64(devStats.TransmittedBytes)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_TransmittedPackets_%s", devname), models.CreateMeasurement(uint64(devStats.TransmittedPackets)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_TransmittedErrs_%s", devname), models.CreateMeasurement(uint64(devStats.TransmittedErrs)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_TransmittedDrop_%s", devname), models.CreateMeasurement(uint64(devStats.TransmittedDrop)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_TransmittedFifo_%s", devname), models.CreateMeasurement(uint64(devStats.TransmittedFifo)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_TransmittedColls_%s", devname), models.CreateMeasurement(uint64(devStats.TransmittedColls)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_TransmittedCarrier_%s", devname), models.CreateMeasurement(uint64(devStats.TransmittedCarrier)))
		domain.AddMetricMeasurement(fmt.Sprintf("net_TransmittedCompressed_%s", devname), models.CreateMeasurement(uint64(devStats.TransmittedCompressed)))

		// Sum for totals
		statsSum.ReceivedBytes += devStats.ReceivedBytes
		statsSum.ReceivedPackets += devStats.ReceivedPackets
		statsSum.ReceivedErrs += devStats.ReceivedErrs
		statsSum.ReceivedDrop += devStats.ReceivedDrop
		statsSum.ReceivedFifo += devStats.ReceivedFifo
		statsSum.ReceivedFrame += devStats.ReceivedFrame
		statsSum.ReceivedCompressed += devStats.ReceivedCompressed
		statsSum.ReceivedMulticast += devStats.ReceivedMulticast

		statsSum.TransmittedBytes += devStats.TransmittedBytes
		statsSum.TransmittedPackets += devStats.TransmittedPackets
		statsSum.TransmittedErrs += devStats.TransmittedErrs
		statsSum.TransmittedDrop += devStats.TransmittedDrop
		statsSum.TransmittedFifo += devStats.TransmittedFifo
		statsSum.TransmittedColls += devStats.TransmittedColls
		statsSum.TransmittedCarrier += devStats.TransmittedCarrier
		statsSum.TransmittedCompressed += devStats.TransmittedCompressed
	}
	// Store totals (for backward compatibility)
	domain.AddMetricMeasurement("net_ReceivedBytes", models.CreateMeasurement(uint64(statsSum.ReceivedBytes)))
	domain.AddMetricMeasurement("net_ReceivedPackets", models.CreateMeasurement(uint64(statsSum.ReceivedPackets)))
	domain.AddMetricMeasurement("net_ReceivedErrs", models.CreateMeasurement(uint64(statsSum.ReceivedErrs)))
	domain.AddMetricMeasurement("net_ReceivedDrop", models.CreateMeasurement(uint64(statsSum.ReceivedDrop)))
	domain.AddMetricMeasurement("net_ReceivedFifo", models.CreateMeasurement(uint64(statsSum.ReceivedFifo)))
	domain.AddMetricMeasurement("net_ReceivedFrame", models.CreateMeasurement(uint64(statsSum.ReceivedFrame)))
	domain.AddMetricMeasurement("net_ReceivedCompressed", models.CreateMeasurement(uint64(statsSum.ReceivedCompressed)))
	domain.AddMetricMeasurement("net_ReceivedMulticast", models.CreateMeasurement(uint64(statsSum.ReceivedMulticast)))
	domain.AddMetricMeasurement("net_TransmittedBytes", models.CreateMeasurement(uint64(statsSum.TransmittedBytes)))
	domain.AddMetricMeasurement("net_TransmittedPackets", models.CreateMeasurement(uint64(statsSum.TransmittedPackets)))
	domain.AddMetricMeasurement("net_TransmittedErrs", models.CreateMeasurement(uint64(statsSum.TransmittedErrs)))
	domain.AddMetricMeasurement("net_TransmittedDrop", models.CreateMeasurement(uint64(statsSum.TransmittedDrop)))
	domain.AddMetricMeasurement("net_TransmittedFifo", models.CreateMeasurement(uint64(statsSum.TransmittedFifo)))
	domain.AddMetricMeasurement("net_TransmittedColls", models.CreateMeasurement(uint64(statsSum.TransmittedColls)))
	domain.AddMetricMeasurement("net_TransmittedCarrier", models.CreateMeasurement(uint64(statsSum.TransmittedCarrier)))
	domain.AddMetricMeasurement("net_TransmittedCompressed", models.CreateMeasurement(uint64(statsSum.TransmittedCompressed)))
}
