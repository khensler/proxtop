package hostcollector

import (
	"os"

	"proxtop/models"
	libvirt "github.com/libvirt/libvirt-go"
)

func domainLookup(domain *models.Domain, libvirtDomain libvirt.Domain) {
	name, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	domain.AddMetricMeasurement("host_name", models.CreateMeasurement(name))
}
