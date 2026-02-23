## What proxtop does

**proxtop** is a real-time VM monitoring tool for KVM and Proxmox VE hypervisors. It reads resource utilization metrics from the Linux proc filesystem, libvirt, and QMP (QEMU Machine Protocol) to give you visibility into what your virtual machines are actually consuming from the hypervisor's perspective.

**Think of it as `esxtop` for KVM and Proxmox VE.**

### Why use proxtop?

- **Hypervisor-level visibility**: See what VMs are actually consuming, not just what they report internally
- **Detect CPU steal time**: Identify when VMs are waiting for CPU due to overprovisioning
- **Find noisy neighbors**: Discover which VMs are causing resource contention
- **Monitor storage latency**: Track disk IOPS, throughput, queue depth, and latency per-VM
- **Real-time troubleshooting**: Interactive ncurses UI with multiple view modes and sorting
- **Integration-ready**: Stream JSON metrics to InfluxDB, Prometheus, or any monitoring system via TCP

### When to use proxtop

- **Performance troubleshooting**: A VM is slow - is it CPU steal? Memory pressure? Disk latency?
- **Capacity planning**: Are your hosts overprovisioned? Which VMs are heaviest?
- **VMware migration**: Moving from ESXi to Proxmox? proxtop gives you the same visibility as esxtop
- **Noisy neighbor detection**: Which VM is consuming all the disk I/O or network bandwidth?
- **SLA monitoring**: Stream metrics to your monitoring stack for alerting and dashboards

### proxtop vs esxtop

| Feature | esxtop (VMware) | proxtop (KVM/Proxmox) |
|---------|-----------------|----------------------|
| Hypervisor | ESXi | KVM, Proxmox VE |
| Metrics source | VMkernel | /proc, libvirt, QMP |
| CPU steal/ready time | ✅ %RDY, %CSTP | ✅ cpu_steal, %rdy |
| Memory overhead | ✅ MCTLSZ, SWCUR | ✅ RSS, MCTL, page faults |
| Disk latency | ✅ GAVG, DAVG, KAVG | ✅ LAT/rd, LAT/wr, AWAIT, SVCTM |
| Disk queue metrics | ✅ ACTV, QUED | ✅ QDEPTH, QLEN, %UTIL |
| Network stats | ✅ MbRX/TX | ✅ bytes/packets, Mb/s |
| Interactive UI | ✅ ncurses | ✅ ncurses with field selector |
| Batch/script mode | ✅ CSV export | ✅ text, JSON output |
| Streaming to TSDB | ❌ requires vROps | ✅ built-in TCP to Logstash/InfluxDB |
| Human-readable units | ✅ | ✅ press 'u' or use -H flag |
| Sort direction toggle | ✅ | ✅ press 'r' for asc/desc |
| Physical device views | ✅ | ✅ press 'p' (net) or 's' (disk) |

If you're migrating from VMware to Proxmox or KVM, proxtop provides the same hypervisor-level visibility you're used to with esxtop.

### How it works

proxtop (formerly kvmtop) measures resource utilization from outside the VM at the hypervisor level. This captures the real resource consumption including virtualization overhead - the difference between what VMs think they're using vs. what they actually consume.

proxtop auto-detects whether you're running standard libvirt-based KVM or Proxmox VE and uses the appropriate connector (libvirt API or QMP sockets).

*What does proxtop offer?*

The command line tool can be used by sysadmins, using a console ui (ncurses). Text or JSON output further allows to process the monitoring data. A built-in TCP output allows to send the data directly to a monitoring data sink, e.g. logstash.

## Documentation

For detailed information, see the following documentation:

| Document | Description |
|----------|-------------|
| [DOCUMENTATION.md](DOCUMENTATION.md) | Comprehensive guide covering installation, configuration, collectors, output formats, deployment, and troubleshooting |
| [METRICS_REFERENCE.md](METRICS_REFERENCE.md) | Complete reference of all collected metrics with sources, descriptions, and calculation formulas |
| [docs/README.md](docs/README.md) | Technical metric collector documentation |

## Installation

### Download Pre-built Packages

Download the latest `.deb` package for Proxmox/Debian from the [Releases](../../releases) page.

```bash
# Install on Proxmox/Debian
dpkg -i proxtop_*.deb

# Configure target server
nano /etc/proxtop.conf

# Enable and start service
systemctl enable proxtop
systemctl start proxtop
```

### Build from Source

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
