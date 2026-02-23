# proxtop - Comprehensive Documentation

## Table of Contents

1. [Overview](#overview)
2. [Key Concepts](#key-concepts)
3. [Installation](#installation)
4. [Command Line Reference](#command-line-reference)
5. [Collectors](#collectors)
6. [Output Formats](#output-formats)
7. [Hypervisor Connectors](#hypervisor-connectors)
8. [Interactive ncurses Interface](#interactive-ncurses-interface)
9. [proxprofiler - Statistical Profiling](#proxprofiler---statistical-profiling)
10. [Deployment](#deployment)
11. [Integration with Monitoring Systems](#integration-with-monitoring-systems)
12. [Architecture](#architecture)
13. [Troubleshooting](#troubleshooting)

---

## Overview

**proxtop** (formerly kvmtop) is a command-line monitoring tool for virtual machines running on KVM or Proxmox VE hypervisors. It measures VM resource utilization from the hypervisor level, capturing the real resource consumption including overhead.

### Why proxtop?

Traditional VM monitoring tools measure resources from inside the guest OS. proxtop takes a different approach:

- **Hypervisor-level metrics**: Measures actual resource consumption from the host perspective
- **Overhead detection**: Captures the difference between what VMs think they're using vs. what they actually consume
- **Noisy neighbor detection**: Identifies resource shortcomings in overprovisioned environments
- **CPU steal time**: Measures how much CPU time is "stolen" from VMs due to contention

### Scientific Background

The conceptual approach is documented in:

```bibtex
@inproceedings{hauser2018reviewing,
  title={Reviewing Cloud Monitoring: Towards Cloud Resource Profiling},
  author={Hauser, Christopher B and Wesner, Stefan},
  booktitle={2018 IEEE 11th International Conference on Cloud Computing (CLOUD)},
  pages={678--685},
  year={2018},
  organization={IEEE}
}
```

---

## Key Concepts

### Utilization Inside vs. Outside VMs

When a VM reports 50% CPU usage internally, the hypervisor might show:
- **Higher usage**: Due to virtualization overhead (I/O emulation, device drivers)
- **Lower effective usage**: Due to CPU steal time when the host is overprovisioned

proxtop reveals this discrepancy by measuring from the hypervisor level.

### Metrics Collection Cycle

proxtop operates in two phases:
1. **Lookup**: Discovers VMs, their PIDs, thread IDs, network interfaces, disk devices
2. **Collect**: Reads current metric values from proc filesystem, libvirt, or QMP

### Data Sources

| Source | Type | Usage |
|--------|------|-------|
| `/proc` filesystem | Host-level | CPU stats, memory info, network stats, I/O stats |
| libvirt API | VM management | VM discovery, vCPU mapping, disk/network config |
| QMP (QEMU Machine Protocol) | Direct QEMU | Fast metric collection on Proxmox VE |
| `/sys` filesystem | Device info | CPU frequencies, network speeds |

---

## Installation

### Prerequisites

- Linux host running KVM or Proxmox VE
- Go 1.18+ (for building from source)
- libvirt development libraries: `libvirt-dev`
- ncurses library: `libncurses5-dev`

### Build from Source

```bash
git clone <repository-url>
cd proxtop

# Build main monitoring tool
go build -o proxtop ./cmd/proxtop

# Build statistical profiler
go build -o proxprofiler ./cmd/proxprofiler
```

### Install System-Wide

```bash
sudo cp proxtop /usr/bin/
sudo cp proxprofiler /usr/bin/
```

---

## Command Line Reference

### proxtop Options

```
Usage:
  proxtop [OPTIONS]

Application Options:
  -v, --version        Show version
  -f, --frequency=     Frequency (in seconds) for collecting metrics (default: 2)
  -r, --runs=          Amount of collection runs (default: -1, infinite)
  -c, --connection=    Connection URI to libvirt daemon (default: qemu:///system)
      --procfs=        Path to the proc filesystem (default: /proc)
      --verbose        Enable verbose output with additional fields

Hypervisor Selection:
      --proxmox        Force Proxmox VE connector (auto-detected by default)
      --libvirt        Force libvirt connector (auto-detected by default)

Collectors:
      --cpu            Enable CPU metrics
      --mem            Enable memory metrics
      --disk           Enable disk metrics
      --net            Enable network metrics
      --io             Enable I/O metrics (requires root)
      --pressure       Enable PSI metrics (requires kernel 4.20+)
      --host           Enable host identification metrics

Output:
  -p, --printer=       Output format: ncurses, text, json (default: ncurses)
  -o, --output=        Output destination: stdout, file, tcp, udp (default: stdout)
      --target=        For 'file': path; for 'tcp'/'udp': host:port
      --netdev=        Network device for virtual traffic monitoring
      --storedev=      Storage device for virtual disk monitoring

Help Options:
  -h, --help           Show this help message
```

### Examples

```bash
# Interactive monitoring with all default collectors
proxtop

# Monitor only CPU and memory
proxtop --cpu --mem

# Verbose output with all metrics
proxtop --cpu --mem --disk --net --io --verbose

# JSON output to file
proxtop --cpu --mem --printer=json --output=file --target=/var/log/proxtop.log

# Stream to TCP server (e.g., Logstash)
proxtop --cpu --mem --printer=json --output=tcp --target=192.168.1.100:5000

# Text output with 5-second intervals
proxtop --cpu --printer=text --frequency=5

# Run for exactly 10 collection cycles
proxtop --cpu --runs=10

# Force libvirt connector on Proxmox
proxtop --libvirt
```

---

## Collectors

### CPU Collector (`--cpu`)

Monitors CPU utilization for host and VMs.

#### Host Metrics

| Metric | Source | Description |
|--------|--------|-------------|
| `cpu_cores` | /proc/cpuinfo | Total number of CPU cores |
| `cpu_curfreq` | /sys/devices/system/cpu | Current mean frequency (MHz) |
| `cpu_user` | /proc/stat | % time in user space |
| `cpu_system` | /proc/stat | % time in kernel space |
| `cpu_idle` | /proc/stat | % time idle |
| `cpu_steal` | /proc/stat | % time stolen by hypervisor |

**Verbose mode adds:** `cpu_minfreq`, `cpu_maxfreq`, `cpu_nice`, `cpu_iowait`, `cpu_irq`, `cpu_softirq`, `cpu_guest`, `cpu_guestnice`

#### VM Metrics

| Metric | Source | Description |
|--------|--------|-------------|
| `cpu_cores` | libvirt/QMP | Number of virtual CPU cores |
| `cpu_total` | schedstat | % utilization across all vCPUs |
| `cpu_steal` | schedstat | % CPU stolen due to host contention |

**Verbose mode adds:** `cpu_other_total`, `cpu_other_steal` (overhead threads)

### Memory Collector (`--mem`)

Monitors memory allocation and usage.

#### Host Metrics

| Metric | Source | Description |
|--------|--------|-------------|
| `ram_Total` | /proc/meminfo | Total physical RAM (KB) |
| `ram_Free` | /proc/meminfo | Free RAM (KB) |
| `ram_Available` | /proc/meminfo | Available RAM (KB) |

**Verbose mode adds:** 30+ additional metrics including buffers, cache, swap, huge pages

#### VM Metrics

| Metric | Source | Description |
|--------|--------|-------------|
| `ram_total` | libvirt/QMP | Maximum VM memory (KB) |
| `ram_used` | libvirt/QMP | Currently used memory (KB) |

**Verbose mode adds:** `ram_vsize`, `ram_rss`, `ram_minflt`, `ram_majflt`

### Disk Collector (`--disk`)

Monitors disk I/O and storage.

#### Host Metrics

| Metric | Source | Description |
|--------|--------|-------------|
| `disk_device_reads` | /proc/diskstats | Completed read operations |
| `disk_device_writes` | /proc/diskstats | Completed write operations |
| `disk_device_ioutil` | calculated | I/O saturation % |

**Verbose mode adds:** sectors, timing, queue metrics

#### VM Metrics

| Metric | Source | Description |
|--------|--------|-------------|
| `disk_size_capacity` | libvirt/QMP | Total virtual disk capacity |
| `disk_size_allocation` | libvirt/QMP | Allocated disk space |
| `disk_ioutil` | calculated | Estimated I/O utilization % |

### Network Collector (`--net`)

Monitors network traffic.

#### Host Metrics

| Metric | Source | Description |
|--------|--------|-------------|
| `net_host_receivedBytes` | /proc/net/dev | Bytes received |
| `net_host_transmittedBytes` | /proc/net/dev | Bytes transmitted |
| `net_host_speed` | /sys/class/net | Link speed (Mbps) |

#### VM Metrics

| Metric | Source | Description |
|--------|--------|-------------|
| `net_receivedBytes` | /proc/PID/net/dev | Bytes received by VM |
| `net_transmittedBytes` | /proc/PID/net/dev | Bytes transmitted by VM |

### I/O Collector (`--io`)

Extended I/O metrics from process-level data. **Requires root access.**

| Metric | Source | Description |
|--------|--------|-------------|
| `io_MBRD/s` | /proc/PID/io | MB/s read |
| `io_MBWR/s` | /proc/PID/io | MB/s written |
| `io_RDOPS` | /proc/PID/io | Read operations/sec |
| `io_WROPS` | /proc/PID/io | Write operations/sec |

### PSI Collector (`--pressure`)

Pressure Stall Information metrics. **Requires kernel 4.20+.**

| Metric | Source | Description |
|--------|--------|-------------|
| `psi_some_cpu_avg60` | /proc/pressure/cpu | % time tasks delayed (CPU) |
| `psi_some_io_avg60` | /proc/pressure/io | % time tasks delayed (I/O) |
| `psi_full_io_avg60` | /proc/pressure/io | % time ALL tasks delayed (I/O) |
| `psi_some_mem_avg60` | /proc/pressure/memory | % time tasks delayed (memory) |
| `psi_full_mem_avg60` | /proc/pressure/memory | % time ALL tasks delayed (memory) |

### Host Collector (`--host`)

Adds host identification to metrics.

| Metric | Description |
|--------|-------------|
| `host_name` | Hostname of the hypervisor |
| `host_uuid` | DMI UUID (verbose mode, requires root) |

---

## Output Formats

### ncurses (Interactive)

Default output mode with a live-updating terminal interface.

**Features:**
- Status bar showing refresh interval and view mode
- Host metrics panel at top
- VM metrics table with sorting
- Multiple view modes
- Field selection

### Text

Tab-separated output suitable for processing with `awk`, `cut`, etc.

```
UUID    name    cpu_cores    cpu_total    cpu_steal
abc123  vm1     2            45           2
def456  vm2     4            78           5
```

### JSON

Machine-readable JSON output, one object per collection cycle.

```json
{
  "host": {
    "cpu_cores": 16,
    "cpu_user": 25.5,
    "ram_Total": 65536000
  },
  "domains": [
    {
      "UUID": "abc-123",
      "name": "webserver",
      "cpu_total": 45,
      "ram_used": 2048000
    }
  ]
}
```

---

## Hypervisor Connectors

### Auto-Detection

proxtop automatically detects the hypervisor type:
1. Checks for `/etc/pve` directory (Proxmox VE)
2. Checks for `qm` command (Proxmox VE)
3. Falls back to libvirt

### Libvirt Connector

Used for standard KVM installations with libvirt.

**Configuration:**
```bash
# Default connection
proxtop --connection=qemu:///system

# Remote connection (requires proc filesystem mount)
proxtop --connection=qemu+ssh://user@host/system --procfs=/mnt/remote-proc
```

**Data Sources:**
- VM discovery via libvirt API
- vCPU thread mapping via QEMU Monitor Protocol (HMP)
- Disk/network configuration from domain XML

### Proxmox VE Connector

Optimized for Proxmox VE environments using QMP directly.

**Features:**
- No libvirt dependency
- Fast metric collection via Unix socket QMP
- Automatic VM discovery from `/var/run/qemu-server/`
- Direct config parsing from `/etc/pve/qemu-server/`

**QMP Commands Used:**
- `query-balloon`: Memory statistics
- `query-blockstats`: Disk I/O statistics

---

## Interactive ncurses Interface

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `q` / `Q` / `Ctrl+C` | Quit proxtop |
| `h` / `H` / `?` | Toggle help overlay |
| `f` / `F` | Toggle field selector (show/hide columns, filtered by current view) |
| `u` / `U` | Toggle human-readable units (KB/MB/GB) - status shows `[H]` |
| `a` / `A` | Show all metrics |
| `c` / `C` | CPU metrics only |
| `m` / `M` | Memory metrics only |
| `d` / `D` | Disk metrics only |
| `n` / `N` | Network metrics only |
| `i` / `I` | I/O metrics only |
| `p` / `P` | Physical network interfaces |
| `s` / `S` | Physical storage devices |
| `<` / `>` | Change sort column |
| `r` / `R` | Reverse sort direction (ascending/descending) |
| `+` / `-` | Increase/decrease refresh interval |

### Sorting

- Use `<` / `>` to change which column is used for sorting
- Use `r` to toggle between ascending (^) and descending (v) sort order
- The sorted column header shows the direction indicator
- Numeric columns are sorted numerically (e.g., 100 > 20, not "100" < "20")
- Settings changes (sort, units) take effect immediately

### Field Selector

Press `f` to open the field selector overlay:
- Use `↑`/`↓` or `j`/`k` to navigate
- Press `Space` or `Enter` to toggle field visibility
- Press `f` or `Escape` to close
- Verbose fields are hidden by default but can be enabled here
- Field list is filtered to the current view mode

### Screen Layout

```
┌─────────────────────────────────────────────────────────────┐
│ proxtop | Refresh: 2.0s | View: ALL | VMs: 5 | Press ? help │  <- Status bar
├─────────────────────────────────────────────────────────────┤
│ HOST METRICS                                                 │
│ cpu_cores: 16    ram_Total: 64GB    net_speed: 10Gbps       │  <- Host panel
├─────────────────────────────────────────────────────────────┤
│ UUID         NAME      CPU%   MEM%   DISK    NET            │
│ abc-123      web-01    45     60     10MB/s  5MB/s          │  <- VM table
│ def-456      db-01     78     85     50MB/s  2MB/s          │
│ ghi-789      app-01    23     40     5MB/s   1MB/s          │
└─────────────────────────────────────────────────────────────┘
```

---

## proxprofiler - Statistical Profiling

The `proxprofiler` tool creates statistical profiles from monitoring data.

### Usage

```bash
proxprofiler [OPTIONS]

Profiler Options:
      --states=        Number of discrete states (default: 4)
      --buffersize=    Buffer size for profiling (default: 10)
      --history=       History depth (default: 1)
      --filterstddevs= Filter outliers by std deviations (default: -1)
      --fixedbound     Use fixed bounds for states
      --periodsize=    Comma-separated list of period sizes
      --outputFreq=    Output frequency (default: 60s)
```

### Example

```bash
proxprofiler --cpu --net --io \
  --printer=json --output=tcp --target=192.168.1.100:5000 \
  --states=4 --history=1 --outputFreq=60s
```

### Profile Output

Profiles capture resource usage patterns as transition matrices, useful for:
- Capacity planning
- Anomaly detection
- Workload characterization

---

## Deployment

### Systemd Service

**proxtop.service:**
```ini
[Unit]
Description=Monitor virtual machine experience from outside on KVM/Proxmox hypervisor level
After=libvirtd.service

[Service]
Type=simple
Restart=always
RestartSec=3
EnvironmentFile=/etc/proxtop.conf
ExecStart=/usr/bin/proxtop --printer=json --output=tcp --target=${PROXTOP_TARGET} --cpu --net --mem --io --disk --host --verbose

[Install]
WantedBy=multi-user.target
```

**Configuration file `/etc/proxtop.conf`:**
```bash
PROXTOP_TARGET=192.168.50.230:12345
```

**Installation:**
```bash
sudo cp proxtop /usr/bin/
sudo cp proxtop.service /etc/systemd/system/
sudo cp proxtop.conf /etc/
sudo systemctl daemon-reload
sudo systemctl enable proxtop
sudo systemctl start proxtop
```

### Docker

**Dockerfile example:**
```dockerfile
FROM alpine:latest
RUN apk add libvirt-client ncurses5-libs
COPY proxtop /bin/proxtop
ENV PARAMS "-c qemu:///system --printer=text --cpu --mem --net --disk"
CMD ["/bin/sh", "-c", "proxtop $PARAMS"]
```

**Run:**
```bash
docker run -v /proc:/host/proc:ro \
           -v /var/run/libvirt:/var/run/libvirt \
           proxtop --procfs=/host/proc
```

---

## Integration with Monitoring Systems

### InfluxDB + Grafana Pipeline

```
┌──────────┐    TCP/JSON    ┌───────────┐    HTTP    ┌──────────┐    Query    ┌─────────┐
│ proxtop  │ ────────────▶ │ Logstash  │ ────────▶ │ InfluxDB │ ◀────────── │ Grafana │
└──────────┘               └───────────┘           └──────────┘            └─────────┘
```

**Logstash Configuration:**
```ruby
input {
  tcp {
    port => 12345
    codec => json
  }
}

output {
  influxdb {
    host => "influxdb.local"
    db => "proxtop"
    measurement => "vm_metrics"
  }
}
```

### Prometheus

Export metrics to file, then use node_exporter's textfile collector:
```bash
proxtop --cpu --mem --printer=text --output=file \
  --target=/var/lib/node_exporter/proxtop.prom
```

---

## Architecture

### Directory Structure

```
proxtop/
├── cmd/
│   ├── proxtop/          # Main monitoring tool
│   └── proxprofiler/     # Statistical profiler
├── collectors/
│   ├── cpucollector/     # CPU metrics
│   ├── memcollector/     # Memory metrics
│   ├── diskcollector/    # Disk metrics
│   ├── netcollector/     # Network metrics
│   ├── iocollector/      # I/O metrics
│   ├── psicollector/     # PSI metrics
│   └── hostcollector/    # Host identification
├── connector/
│   ├── libvirt.go        # libvirt connector
│   └── proxmox.go        # Proxmox VE connector
├── printers/
│   ├── ncurses.go        # Interactive UI
│   ├── textprint.go      # Text output
│   └── json.go           # JSON output
├── config/               # Configuration handling
├── models/               # Data structures
├── runners/              # Collection orchestration
├── util/                 # Proc/sys file parsing
└── profiler/             # Statistical profiling
```

### Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                         Main Loop                                │
│  ┌──────────┐    ┌───────────┐    ┌──────────┐    ┌──────────┐  │
│  │  Lookup  │───▶│  Collect  │───▶│  Print   │───▶│  Output  │  │
│  │  Runner  │    │  Runner   │    │  Runner  │    │          │  │
│  └──────────┘    └───────────┘    └──────────┘    └──────────┘  │
│       │               │                │               │         │
│       ▼               ▼                ▼               ▼         │
│  ┌──────────┐    ┌───────────┐    ┌──────────┐    ┌──────────┐  │
│  │Connector │    │Collectors │    │ Printers │    │  stdout  │  │
│  │(libvirt/ │    │(cpu,mem,  │    │(ncurses, │    │  file    │  │
│  │ proxmox) │    │ disk,...) │    │text,json)│    │  tcp/udp │  │
│  └──────────┘    └───────────┘    └──────────┘    └──────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Troubleshooting

### Common Issues

**"Failed to connect to libvirt"**
```bash
# Check libvirtd is running
sudo systemctl status libvirtd

# Check connection URI
virsh -c qemu:///system list
```

**"Permission denied" for I/O metrics**
```bash
# I/O collector requires root
sudo proxtop --io
```

**"PSI metrics unavailable"**
```bash
# Check kernel version (requires 4.20+)
uname -r

# Check PSI is enabled
cat /proc/pressure/cpu
```

**"No VMs detected"**
```bash
# Verify VMs are running
virsh list --all          # For libvirt
qm list                   # For Proxmox

# Check process visibility
ls /proc/*/cmdline | xargs grep -l qemu
```

**ncurses display corrupted**
```bash
# Reset terminal
reset

# Use text output instead
proxtop --printer=text
```

### Debug Mode

Enable verbose logging:
```bash
proxtop --verbose --printer=text 2>&1 | tee proxtop.log
```

### Performance Tuning

For high VM counts, consider:
- Increase refresh interval: `--frequency=5`
- Disable unused collectors
- Use text/JSON output instead of ncurses
- Run as systemd service for stability

---

## License

See LICENSE file for licensing information.

