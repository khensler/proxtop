package netcollector

import (
	"fmt"

	"proxtop/config"
	"proxtop/models"
	"proxtop/util"
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

// HostNetFields returns the field names for host physical network view
func HostNetFields() []string {
	fields := []string{
		"net_DEVICE",
		"net_RX-Bytes",
		"net_RX-Pkts",
		"net_RX-Errs",
		"net_RX-Drop",
		"net_TX-Bytes",
		"net_TX-Pkts",
		"net_TX-Errs",
		"net_TX-Drop",
	}
	if config.Options.Verbose {
		fields = append(fields,
			"net_RX-Fifo",
			"net_RX-Frame",
			"net_RX-Compressed",
			"net_RX-Multicast",
			"net_TX-Fifo",
			"net_TX-Colls",
			"net_TX-Carrier",
			"net_TX-Compressed",
		)
	}
	return fields
}

// HostPrintPerDevice returns per-device network stats for physical network view
// Returns a map of device name -> []string (field values in same order as HostNetFields)
func HostPrintPerDevice() map[string][]string {
	devices := util.GetPhysicalNetDevices()
	result := make(map[string][]string)

	for name, dev := range devices {
		values := []string{
			name,
			fmt.Sprintf("%d", dev.ReceivedBytes),
			fmt.Sprintf("%d", dev.ReceivedPackets),
			fmt.Sprintf("%d", dev.ReceivedErrs),
			fmt.Sprintf("%d", dev.ReceivedDrop),
			fmt.Sprintf("%d", dev.TransmittedBytes),
			fmt.Sprintf("%d", dev.TransmittedPackets),
			fmt.Sprintf("%d", dev.TransmittedErrs),
			fmt.Sprintf("%d", dev.TransmittedDrop),
		}

		if config.Options.Verbose {
			values = append(values,
				fmt.Sprintf("%d", dev.ReceivedFifo),
				fmt.Sprintf("%d", dev.ReceivedFrame),
				fmt.Sprintf("%d", dev.ReceivedCompressed),
				fmt.Sprintf("%d", dev.ReceivedMulticast),
				fmt.Sprintf("%d", dev.TransmittedFifo),
				fmt.Sprintf("%d", dev.TransmittedColls),
				fmt.Sprintf("%d", dev.TransmittedCarrier),
				fmt.Sprintf("%d", dev.TransmittedCompressed),
			)
		}

		result[name] = values
	}

	return result
}
