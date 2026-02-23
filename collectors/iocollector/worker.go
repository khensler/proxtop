package iocollector

import (
	"fmt"

	"proxtop/config"
	"proxtop/models"
	"proxtop/util"

	libvirt "github.com/libvirt/libvirt-go"
)

func ioLookup(domain *models.Domain, libvirtDomain libvirt.Domain) {
	// nothing to do
}

func ioCollect(domain *models.Domain) {
	stats := util.GetProcPIDIO(domain.PID)
	domain.AddMetricMeasurement("io_rchar", models.CreateMeasurement(uint64(stats.Rchar)))
	domain.AddMetricMeasurement("io_wchar", models.CreateMeasurement(uint64(stats.Wchar)))
	domain.AddMetricMeasurement("io_syscr", models.CreateMeasurement(uint64(stats.Syscr)))
	domain.AddMetricMeasurement("io_syscw", models.CreateMeasurement(uint64(stats.Syscw)))
	domain.AddMetricMeasurement("io_read_bytes", models.CreateMeasurement(uint64(stats.Read_bytes)))
	domain.AddMetricMeasurement("io_write_bytes", models.CreateMeasurement(uint64(stats.Write_bytes)))
	domain.AddMetricMeasurement("io_cancelled_write_bytes", models.CreateMeasurement(uint64(stats.Cancelled_write_bytes)))
}

func ioPrint(domain *models.Domain) []string {
	// Get raw float values for calculations
	rcharFloat := domain.GetMetricDiffUint64AsFloat("io_rchar", true)
	wcharFloat := domain.GetMetricDiffUint64AsFloat("io_wchar", true)
	syscrFloat := domain.GetMetricDiffUint64AsFloat("io_syscr", true)
	syscwFloat := domain.GetMetricDiffUint64AsFloat("io_syscw", true)
	readBytesFloat := domain.GetMetricDiffUint64AsFloat("io_read_bytes", true)
	writeBytesFloat := domain.GetMetricDiffUint64AsFloat("io_write_bytes", true)
	cancelledWriteBytes := domain.GetMetricDiffUint64("io_cancelled_write_bytes", true)

	// esxtop style: MBRD/s, MBWR/s, RDOPS (syscalls), WROPS (syscalls)
	mbRead := fmt.Sprintf("%.2f", readBytesFloat/1024/1024)
	mbWrite := fmt.Sprintf("%.2f", writeBytesFloat/1024/1024)
	rdOps := fmt.Sprintf("%.0f", syscrFloat)
	wrOps := fmt.Sprintf("%.0f", syscwFloat)
	rchar := fmt.Sprintf("%.0f", rcharFloat)
	wchar := fmt.Sprintf("%.0f", wcharFloat)

	result := append([]string{mbRead}, mbWrite, rdOps, wrOps)
	if config.Options.Verbose {
		result = append(result, rchar, wchar, cancelledWriteBytes)
	}
	return result
}
