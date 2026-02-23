package netcollector

import (
	"fmt"

	"proxtop/config"
	"proxtop/models"
)

func hostPrint(host *models.Host) []string {
	// Get raw float values for calculations
	receivedBytesFloat := host.GetMetricDiffUint64AsFloat("net_host_ReceivedBytes", true)
	receivedPacketsFloat := host.GetMetricDiffUint64AsFloat("net_host_ReceivedPackets", true)
	receivedErrs := host.GetMetricDiffUint64("net_host_ReceivedErrs", true)
	receivedDrop := host.GetMetricDiffUint64("net_host_ReceivedDrop", true)
	receivedFifo := host.GetMetricDiffUint64("net_host_ReceivedFifo", true)
	receivedFrame := host.GetMetricDiffUint64("net_host_ReceivedFrame", true)
	receivedCompressed := host.GetMetricDiffUint64("net_host_ReceivedCompressed", true)
	receivedMulticast := host.GetMetricDiffUint64("net_host_ReceivedMulticast", true)
	transmittedBytesFloat := host.GetMetricDiffUint64AsFloat("net_host_TransmittedBytes", true)
	transmittedPacketsFloat := host.GetMetricDiffUint64AsFloat("net_host_TransmittedPackets", true)
	transmittedErrs := host.GetMetricDiffUint64("net_host_TransmittedErrs", true)
	transmittedDrop := host.GetMetricDiffUint64("net_host_TransmittedDrop", true)
	transmittedFifo := host.GetMetricDiffUint64("net_host_TransmittedFifo", true)
	transmittedColls := host.GetMetricDiffUint64("net_host_TransmittedColls", true)
	transmittedCarrier := host.GetMetricDiffUint64("net_host_TransmittedCarrier", true)
	transmittedCompressed := host.GetMetricDiffUint64("net_host_TransmittedCompressed", true)

	speed, _ := host.GetMetricUint64("net_host_speed", 0)

	// esxtop style: MbRX/s, MbTX/s, PKTRX/s, PKTTX/s, speed
	mbRx := fmt.Sprintf("%.2f", receivedBytesFloat*8/1000000)      // bytes/s to Mb/s
	mbTx := fmt.Sprintf("%.2f", transmittedBytesFloat*8/1000000)   // bytes/s to Mb/s
	pktRx := fmt.Sprintf("%.0f", receivedPacketsFloat)
	pktTx := fmt.Sprintf("%.0f", transmittedPacketsFloat)

	result := append([]string{mbRx}, mbTx, pktRx, pktTx, speed)
	if config.Options.Verbose {
		result = append(result, receivedErrs, receivedDrop, receivedFifo, receivedFrame, receivedCompressed, receivedMulticast, transmittedErrs, transmittedDrop, transmittedFifo, transmittedColls, transmittedCarrier, transmittedCompressed)
	}

	return result
}
