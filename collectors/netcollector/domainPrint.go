package netcollector

import (
	"fmt"
	"strings"

	"proxtop/config"
	"proxtop/models"
)

func domainPrint(domain *models.Domain) []string {
	// Get raw float values for calculations
	receivedBytesFloat := domain.GetMetricDiffUint64AsFloat("net_ReceivedBytes", true)
	receivedPacketsFloat := domain.GetMetricDiffUint64AsFloat("net_ReceivedPackets", true)
	receivedErrs := domain.GetMetricDiffUint64("net_ReceivedErrs", true)
	receivedDropFloat := domain.GetMetricDiffUint64AsFloat("net_ReceivedDrop", true)
	receivedFifo := domain.GetMetricDiffUint64("net_ReceivedFifo", true)
	receivedFrame := domain.GetMetricDiffUint64("net_ReceivedFrame", true)
	receivedCompressed := domain.GetMetricDiffUint64("net_ReceivedCompressed", true)
	receivedMulticast := domain.GetMetricDiffUint64("net_ReceivedMulticast", true)
	transmittedBytesFloat := domain.GetMetricDiffUint64AsFloat("net_TransmittedBytes", true)
	transmittedPacketsFloat := domain.GetMetricDiffUint64AsFloat("net_TransmittedPackets", true)
	transmittedErrs := domain.GetMetricDiffUint64("net_TransmittedErrs", true)
	transmittedDropFloat := domain.GetMetricDiffUint64AsFloat("net_TransmittedDrop", true)
	transmittedFifo := domain.GetMetricDiffUint64("net_TransmittedFifo", true)
	transmittedColls := domain.GetMetricDiffUint64("net_TransmittedColls", true)
	transmittedCarrier := domain.GetMetricDiffUint64("net_TransmittedCarrier", true)
	transmittedCompressed := domain.GetMetricDiffUint64("net_TransmittedCompressed", true)

	ifsRaw := domain.GetMetricStringArray("net_interfaces")
	interfaces := strings.Join(ifsRaw, ";")

	// esxtop style: MbRX/s, MbTX/s, PKTRX/s, PKTTX/s, %DRPRX, %DRPTX
	mbRx := fmt.Sprintf("%.2f", receivedBytesFloat*8/1000000)      // bytes/s to Mb/s
	mbTx := fmt.Sprintf("%.2f", transmittedBytesFloat*8/1000000)   // bytes/s to Mb/s
	pktRx := fmt.Sprintf("%.0f", receivedPacketsFloat)
	pktTx := fmt.Sprintf("%.0f", transmittedPacketsFloat)

	// Calculate drop percentage
	var dropRxPct, dropTxPct string
	if receivedPacketsFloat > 0 {
		dropRxPct = fmt.Sprintf("%.2f", (receivedDropFloat/receivedPacketsFloat)*100)
	} else {
		dropRxPct = "0.00"
	}
	if transmittedPacketsFloat > 0 {
		dropTxPct = fmt.Sprintf("%.2f", (transmittedDropFloat/transmittedPacketsFloat)*100)
	} else {
		dropTxPct = "0.00"
	}

	// Default: MbRX/s, MbTX/s, PKTRX/s, PKTTX/s, %DRPRX, %DRPTX
	result := append([]string{mbRx}, mbTx, pktRx, pktTx, dropRxPct, dropTxPct)
	if config.Options.Verbose {
		result = append(result, receivedErrs, receivedFifo, receivedFrame, receivedCompressed, receivedMulticast, transmittedErrs, transmittedFifo, transmittedColls, transmittedCarrier, transmittedCompressed, interfaces)
	}
	return result
}

// DomainPrintPerInterface returns per-interface stats for a domain
// Returns a map of interface name -> []string (same format as domainPrint)
func DomainPrintPerInterface(domain *models.Domain) map[string][]string {
	ifsRaw := domain.GetMetricStringArray("net_interfaces")
	result := make(map[string][]string)

	for _, devname := range ifsRaw {
		// Get per-interface stats
		rxBytes := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("net_ReceivedBytes_%s", devname), true)
		rxPackets := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("net_ReceivedPackets_%s", devname), true)
		rxDrop := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("net_ReceivedDrop_%s", devname), true)
		txBytes := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("net_TransmittedBytes_%s", devname), true)
		txPackets := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("net_TransmittedPackets_%s", devname), true)
		txDrop := domain.GetMetricDiffUint64AsFloat(fmt.Sprintf("net_TransmittedDrop_%s", devname), true)

		// Calculate stats same as domainPrint
		mbRx := fmt.Sprintf("%.2f", rxBytes*8/1000000)
		mbTx := fmt.Sprintf("%.2f", txBytes*8/1000000)
		pktRx := fmt.Sprintf("%.0f", rxPackets)
		pktTx := fmt.Sprintf("%.0f", txPackets)

		var dropRxPct, dropTxPct string
		if rxPackets > 0 {
			dropRxPct = fmt.Sprintf("%.2f", (rxDrop/rxPackets)*100)
		} else {
			dropRxPct = "0.00"
		}
		if txPackets > 0 {
			dropTxPct = fmt.Sprintf("%.2f", (txDrop/txPackets)*100)
		} else {
			dropTxPct = "0.00"
		}

		// Default values: MbRX/s, MbTX/s, PKTRX/s, PKTTX/s, %DRPRX, %DRPTX
		row := []string{mbRx, mbTx, pktRx, pktTx, dropRxPct, dropTxPct}

		// Add verbose fields when verbose mode is enabled
		if config.Options.Verbose {
			rxErrs := domain.GetMetricDiffUint64(fmt.Sprintf("net_ReceivedErrs_%s", devname), true)
			rxFifo := domain.GetMetricDiffUint64(fmt.Sprintf("net_ReceivedFifo_%s", devname), true)
			rxFrame := domain.GetMetricDiffUint64(fmt.Sprintf("net_ReceivedFrame_%s", devname), true)
			rxComp := domain.GetMetricDiffUint64(fmt.Sprintf("net_ReceivedCompressed_%s", devname), true)
			rxMcast := domain.GetMetricDiffUint64(fmt.Sprintf("net_ReceivedMulticast_%s", devname), true)
			txErrs := domain.GetMetricDiffUint64(fmt.Sprintf("net_TransmittedErrs_%s", devname), true)
			txFifo := domain.GetMetricDiffUint64(fmt.Sprintf("net_TransmittedFifo_%s", devname), true)
			txColls := domain.GetMetricDiffUint64(fmt.Sprintf("net_TransmittedColls_%s", devname), true)
			txCarrier := domain.GetMetricDiffUint64(fmt.Sprintf("net_TransmittedCarrier_%s", devname), true)
			txComp := domain.GetMetricDiffUint64(fmt.Sprintf("net_TransmittedCompressed_%s", devname), true)
			row = append(row, rxErrs, rxFifo, rxFrame, rxComp, rxMcast, txErrs, txFifo, txColls, txCarrier, txComp, devname)
		}

		result[devname] = row
	}

	return result
}
