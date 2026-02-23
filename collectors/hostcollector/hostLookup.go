package hostcollector

import (
	"os"

	"proxtop/config"
	"proxtop/models"
	"proxtop/util"
)

func hostLookup(host *models.Host) {
	name, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	host.AddMetricMeasurement("host_name", models.CreateMeasurement(name))

	if config.Options.Verbose {
		uuid := util.GetSysDmiUUID()
		host.AddMetricMeasurement("host_uuid", models.CreateMeasurement(uuid.Value))
	}
}
