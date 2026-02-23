## What proxtop does
proxtop reads utilisation metrics about virtual machines running on a KVM or Proxmox VE hypervisor from the Linux proc filesystem, libvirt, and QMP (QEMU Machine Protocol).

*Why yet another monitoring tool for virtual machines?*

proxtop (formerly kvmtop) takes into account the difference between utilisation inside and
outside the virtual machine, which differs in cases of overprovisioning. proxtop collects utilisation values of the hypervisor for virtual machines, to measure the overhead needed to run a virtual machine. proxtop will help to identify resource shortcomings, leading
to the "noisy neighbour" effect.

proxtop supports both standard libvirt-based KVM hypervisors and Proxmox VE, with auto-detection of the hypervisor type.

The conceptual idea behind proxtop is scientifically published and described in "Reviewing Cloud Monitoring: Towards Cloud Resource Profiling."

```
@inproceedings{hauser2018reviewing,
  title={Reviewing Cloud Monitoring: Towards Cloud Resource Profiling},
  author={Hauser, Christopher B and Wesner, Stefan},
  booktitle={2018 IEEE 11th International Conference on Cloud Computing (CLOUD)},
  pages={678--685},
  year={2018},
  organization={IEEE}
}
```

*What does proxtop offer?*

The command line tool can be used by sysadmins, using a console ui (ncurses). Text or JSON output further allows to process the monitoring data. A built-in TCP output allows to send the data directly to a monitoring data sink, e.g. logstash.

## Installation

Build from source using Go:

```bash
go build -o proxtop ./cmd/proxtop
go build -o proxprofiler ./cmd/proxprofiler
```

## General Usage

```
Usage:
  proxtop [OPTIONS]

Monitor virtual machine experience from outside on KVM/Proxmox hypervisor level

Application Options:
  -v, --version        Show version
  -f, --frequency=     Frequency (in seconds) for collecting metrics (default: 1)
  -r, --runs=          Amount of collection runs (default: -1)
  -c, --connection=    connection uri to libvirt daemon (default: qemu:///system)
      --procfs=        path to the proc filesystem (default: /proc)
      --verbose        Verbose output, adds more detailed fields
      --cpu            enable cpu metrics
      --mem            enable memory metrics
      --disk           enable disk metrics
      --net            enable network metrics
      --io             enable io metrics (requires root)
      --pressure       enable pressure metrics (requires kernel 4.20+)
      --host           enable host metrics
  -p, --printer=       the output printer to use (valid printers: ncurses, text, json) (default: ncurses)
  -o, --output=        the output channel to send printer output (valid output: stdout, file, tcp, udp) (default: stdout)
      --target=        for output 'file' the location, for 'tcp' or 'udp' the url (host:port) to the server
      --netdev=        The network device used for the virtual traffic

Help Options:
  -h, --help           Show this help message

```

Exemplary output
```
UUID                                 name          cpu_cores cpu_total cpu_steal cpu_other_total cpu_other_steal
0dbe2ae8-1ee4-4b43-bdf3-b533dfe75486 ubuntu14.04-2 2         53        0         5               1
```

Please note: although the connection to libvirt may work remote (e.g. via ssh), proxtop requires access to the /proc file system of the hypervisor's operating system. You can use the `--connection` to connect to a remote libvirt, but need to mount the remote proc fs and specify the location with `--procfs`.

### Proxmox VE Support

proxtop auto-detects Proxmox VE environments and uses QMP (QEMU Machine Protocol) for fast metric collection. No additional configuration is needed - just run `proxtop` on your Proxmox host.

### Printers and Outputs

Printers define the representation of the monitoring data. This can be for humans in ncurses, or for further processing text (space separated) or json.

Outputs define the location where the printers send data to. Output works for text and json printers, yet not for ncurses. The output may be a file or a remote tcp server.

Example scenarios:

```
# write monitoring data to log file
proxtop --cpu --printer=text --output=file --target=/var/log/proxtop.log

# send monitoring data to tcp server (e.g. logstash with tcp input)
proxtop --cpu --printer=json --output=tcp --target=127.0.0.1:12345
```

## Collectors & Their Fields

| Collector | cli option | description |
| --- | --- | --- |
| CPU Collector | --cpu | CPU Stats (host and VMs) like cores, utilisation, frequency|
| Memory Collector | --mem | Memory stats (host and VMs)  like capacity, allocation, faults |
| Disk Collector | --disk | Disk stats (host and VMs) like capacity, utilisation, reads/writes, etc. |
| Network Collector | --net | Network stats (host and VMs) like transmitted and received bytes, packets, errors, etc. |
| I/O Collector | --io | Disk I/O stats (host and VMs) like reads/writes |
| PSI Collector | --psi | Pressure Stall Information (PSI) values (host only) |
| Host | --host | Host details (host only) |

## proxtop with InfluxDB

proxtop can be used as a monitoring agent to send data to an InfluxDB instance: proxtop transmits JSON data via TCP to logstash, while logstash writes to InfluxDB.

```
                  +-----------------------------------------------------+
                  |                                                     |
+------------     | +------------+     +------------+     +-----------+ |
|           |     | |            |     |            |     |           | |
|  proxtop  +---> | |  logstash  +---> |  influxdb  +---> |  grafana  | |
|           |     | |            |     |            |     |           | |
+------------     | +------------+     +------------+     +-----------+ |
                  |                                                     |
                  +-----------------------------------------------------+
```

# Development Guide

Install the golang binary and the required dependencies libvirt-dev and libncurses5-dev packages.

```bash
git clone <repository>
cd proxtop
go build -o proxtop ./cmd/proxtop
go build -o proxprofiler ./cmd/proxprofiler
```

Further reading: https://golang.org/doc/code.html
