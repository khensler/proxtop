# proxtop Metrics Reference

This document provides a comprehensive reference of all metrics collected by proxtop, organized by collector. Each metric includes its data source, description, collection cycle, and availability mode.

## Legend

| Symbol | Meaning |
|--------|---------|
| 🔄 lookup | Metric is collected during the lookup phase (VM discovery) |
| 📊 collect | Metric is collected during the collect phase (periodic sampling) |
| 📝 verbose | Only available with `--verbose` flag |
| 🔒 internal | Internal metric, not exported but used for calculations |
| ⚠️ root | Requires root/sudo privileges |

---

## Table of Contents

1. [Base VM Metrics](#base-vm-metrics)
2. [CPU Collector](#cpu-collector---cpu)
3. [Memory Collector](#memory-collector---mem)
4. [Network Collector](#network-collector---net)
5. [Disk Collector](#disk-collector---disk)
6. [I/O Collector](#io-collector---io)
7. [Host Collector](#host-collector---host)
8. [PSI Collector](#psi-collector---pressure)
9. [Metric Calculations](#metric-calculations)

---

## Base VM Metrics

These metrics are always collected for each virtual machine, regardless of which collectors are enabled.

| Metric | Source | Description | Unit |
|--------|--------|-------------|------|
| `UUID` | libvirt/QMP | Unique identifier for the virtual machine | string |
| `name` | libvirt/QMP | Human-readable name of the virtual machine | string |
| `PID` | /proc | Process ID of the QEMU process on the host | integer |

---

## CPU Collector (`--cpu`)

Monitors CPU utilization, frequency, and steal time for both the host and virtual machines.

### Host Metrics

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `cpu_cores` | /proc/cpuinfo | Total number of physical CPU cores | count | 🔄 lookup |
| `cpu_curfreq` | /sys/devices/system/cpu/cpu*/cpufreq | Current mean frequency across all CPU cores | MHz | 🔄 lookup |
| `cpu_user` | /proc/stat | Time spent executing user-space processes | % | 📊 collect |
| `cpu_system` | /proc/stat | Time spent executing kernel-space code | % | 📊 collect |
| `cpu_idle` | /proc/stat | Time spent idle (no tasks running) | % | 📊 collect |
| `cpu_steal` | /proc/stat | Time stolen by hypervisor for other VMs | % | 📊 collect |

#### Verbose Mode Host Metrics 📝

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `cpu_minfreq` | /sys/devices/system/cpu/cpu*/cpufreq | Minimum supported CPU frequency | MHz | 🔄 lookup |
| `cpu_maxfreq` | /sys/devices/system/cpu/cpu*/cpufreq | Maximum supported CPU frequency | MHz | 🔄 lookup |
| `cpu_nice` | /proc/stat | Time spent on niced (low priority) user processes | % | 📊 collect |
| `cpu_iowait` | /proc/stat | Time spent waiting for I/O completion | % | 📊 collect |
| `cpu_irq` | /proc/stat | Time spent handling hardware interrupts | % | 📊 collect |
| `cpu_softirq` | /proc/stat | Time spent handling software interrupts | % | 📊 collect |
| `cpu_guest` | /proc/stat | Time spent running guest VMs | % | 📊 collect |
| `cpu_guestnice` | /proc/stat | Time spent running niced guest VMs | % | 📊 collect |

### VM Metrics

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `cpu_cores` | libvirt/QMP | Number of virtual CPU cores assigned to VM | count | 🔄 lookup |
| `cpu_total` | calculated | Total CPU utilization across all vCPUs | % | 📊 collect |
| `cpu_steal` | calculated | CPU time stolen due to host contention | % | 📊 collect |

#### Verbose Mode VM Metrics 📝

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `cpu_other_total` | calculated | CPU overhead from I/O and emulation threads | % | 📊 collect |
| `cpu_other_steal` | calculated | Steal time for overhead threads | % | 📊 collect |

#### Internal Metrics 🔒

| Metric | Source | Description | Cycle |
|--------|--------|-------------|-------|
| `cpu_threadIDs` | libvirt + /proc | List of thread IDs for vCPU threads | 🔄 lookup |
| `cpu_otherThreadIDs` | libvirt + /proc | List of thread IDs for non-vCPU threads | 🔄 lookup |
| `cpu_times_${pid}` | /proc/${pid}/schedstat | CPU time counter for each vCPU thread | 📊 collect |
| `cpu_runqueues_${pid}` | /proc/${pid}/schedstat | Run queue wait time for each vCPU thread | 📊 collect |
| `cpu_other_times_${pid}` | /proc/${pid}/schedstat | CPU time for overhead threads | 📊 collect |
| `cpu_other_runqueues_${pid}` | /proc/${pid}/schedstat | Run queue wait for overhead threads | 📊 collect |

---

## Memory Collector (`--mem`)

Monitors memory allocation, usage, and page faults.

### Host Metrics

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `ram_Total` | /proc/meminfo | Total physical RAM installed | KB | 📊 collect |
| `ram_Free` | /proc/meminfo | Completely unused RAM | KB | 📊 collect |
| `ram_Available` | /proc/meminfo | RAM available for new allocations | KB | 📊 collect |

#### Verbose Mode Host Metrics 📝

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `ram_Buffers` | /proc/meminfo | Memory used for kernel buffers | KB | 📊 collect |
| `ram_Cached` | /proc/meminfo | Memory used for page cache | KB | 📊 collect |
| `ram_SwapCached` | /proc/meminfo | Swap memory also in RAM | KB | 📊 collect |
| `ram_Active` | /proc/meminfo | Recently used memory | KB | 📊 collect |
| `ram_Inactive` | /proc/meminfo | Less recently used memory | KB | 📊 collect |
| `ram_ActiveAnon` | /proc/meminfo | Active anonymous memory | KB | 📊 collect |
| `ram_InactiveAnon` | /proc/meminfo | Inactive anonymous memory | KB | 📊 collect |
| `ram_ActiveFile` | /proc/meminfo | Active file-backed memory | KB | 📊 collect |
| `ram_InactiveFile` | /proc/meminfo | Inactive file-backed memory | KB | 📊 collect |
| `ram_Unevictable` | /proc/meminfo | Memory that cannot be reclaimed | KB | 📊 collect |
| `ram_Mlocked` | /proc/meminfo | Memory locked with mlock() | KB | 📊 collect |
| `ram_SwapTotal` | /proc/meminfo | Total swap space | KB | 📊 collect |
| `ram_SwapFree` | /proc/meminfo | Unused swap space | KB | 📊 collect |
| `ram_Dirty` | /proc/meminfo | Memory waiting to be written to disk | KB | 📊 collect |
| `ram_Writeback` | /proc/meminfo | Memory actively being written to disk | KB | 📊 collect |
| `ram_AnonPages` | /proc/meminfo | Anonymous mapped memory | KB | 📊 collect |
| `ram_Mapped` | /proc/meminfo | Files mapped into memory | KB | 📊 collect |
| `ram_Shmem` | /proc/meminfo | Shared memory (tmpfs, etc.) | KB | 📊 collect |
| `ram_Slab` | /proc/meminfo | Kernel slab allocator memory | KB | 📊 collect |
| `ram_SReclaimable` | /proc/meminfo | Reclaimable slab memory | KB | 📊 collect |
| `ram_SUnreclaim` | /proc/meminfo | Non-reclaimable slab memory | KB | 📊 collect |
| `ram_KernelStack` | /proc/meminfo | Kernel stack memory | KB | 📊 collect |
| `ram_PageTables` | /proc/meminfo | Memory for page tables | KB | 📊 collect |
| `ram_NFSUnstable` | /proc/meminfo | NFS pages not yet committed | KB | 📊 collect |
| `ram_Bounce` | /proc/meminfo | Bounce buffer memory | KB | 📊 collect |
| `ram_WritebackTmp` | /proc/meminfo | Temporary writeback memory | KB | 📊 collect |
| `ram_CommitLimit` | /proc/meminfo | Total memory available for allocation | KB | 📊 collect |
| `ram_CommittedAS` | /proc/meminfo | Total memory committed | KB | 📊 collect |
| `ram_VmallocTotal` | /proc/meminfo | Total vmalloc address space | KB | 📊 collect |
| `ram_VmallocUsed` | /proc/meminfo | Used vmalloc space | KB | 📊 collect |
| `ram_VmallocChunk` | /proc/meminfo | Largest contiguous vmalloc block | KB | 📊 collect |
| `ram_HardwareCorrupted` | /proc/meminfo | Memory with hardware errors | KB | 📊 collect |
| `ram_AnonHugePages` | /proc/meminfo | Anonymous huge pages | KB | 📊 collect |
| `ram_ShmemHugePages` | /proc/meminfo | Shared memory huge pages | KB | 📊 collect |
| `ram_ShmemPmdMapped` | /proc/meminfo | Shared memory PMD mapped | KB | 📊 collect |
| `ram_HugePagesTotal` | /proc/meminfo | Total huge pages configured | count | 📊 collect |
| `ram_HugePagesFree` | /proc/meminfo | Free huge pages | count | 📊 collect |
| `ram_HugePagesRsvd` | /proc/meminfo | Reserved huge pages | count | 📊 collect |
| `ram_HugePagesSurp` | /proc/meminfo | Surplus huge pages | count | 📊 collect |
| `ram_Hugepagesize` | /proc/meminfo | Size of each huge page | KB | 📊 collect |
| `ram_Hugetlb` | /proc/meminfo | Total huge page memory | KB | 📊 collect |
| `ram_DirectMap4k` | /proc/meminfo | Memory mapped with 4K pages | KB | 📊 collect |
| `ram_DirectMap2M` | /proc/meminfo | Memory mapped with 2M pages | KB | 📊 collect |
| `ram_DirectMap1G` | /proc/meminfo | Memory mapped with 1G pages | KB | 📊 collect |

### VM Metrics

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `ram_total` | libvirt/QMP | Maximum memory the VM can use | KB | 🔄 lookup |
| `ram_used` | libvirt/QMP | Currently allocated memory | KB | 🔄 lookup |

#### Verbose Mode VM Metrics 📝

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `ram_vsize` | /proc/${pid}/stat | Virtual memory size of QEMU process | bytes | 📊 collect |
| `ram_rss` | /proc/${pid}/stat | Resident set size (physical memory used) | bytes | 📊 collect |
| `ram_minflt` | /proc/${pid}/stat | Minor page faults (no disk I/O) | count | 📊 collect |
| `ram_cminflt` | /proc/${pid}/stat | Minor faults including children | count | 📊 collect |
| `ram_majflt` | /proc/${pid}/stat | Major page faults (required disk I/O) | count | 📊 collect |
| `ram_cmajflt` | /proc/${pid}/stat | Major faults including children | count | 📊 collect |

---

## Network Collector (`--net`)

Monitors network traffic across physical and virtual interfaces.

### Host Metrics

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `net_host_receivedBytes` | /proc/net/dev | Total bytes received (sum over relevant interfaces) | bytes | 📊 collect |
| `net_host_transmittedBytes` | /proc/net/dev | Total bytes transmitted (sum over relevant interfaces) | bytes | 📊 collect |
| `net_host_speed` | /sys/class/net/${dev}/speed | Network device maximum link speed | Mbps | 🔄 lookup |

#### Verbose Mode Host Metrics 📝

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `net_host_receivedPackets` | /proc/net/dev | Total packets received | count | 📊 collect |
| `net_host_receivedErrs` | /proc/net/dev | Receive errors | count | 📊 collect |
| `net_host_receivedDrop` | /proc/net/dev | Dropped incoming packets | count | 📊 collect |
| `net_host_receivedFifo` | /proc/net/dev | FIFO buffer errors (receive) | count | 📊 collect |
| `net_host_receivedFrame` | /proc/net/dev | Framing errors on receive | count | 📊 collect |
| `net_host_receivedCompressed` | /proc/net/dev | Compressed packets received | count | 📊 collect |
| `net_host_receivedMulticast` | /proc/net/dev | Multicast frames received | count | 📊 collect |
| `net_host_transmittedPackets` | /proc/net/dev | Total packets transmitted | count | 📊 collect |
| `net_host_transmittedErrs` | /proc/net/dev | Transmit errors | count | 📊 collect |
| `net_host_transmittedDrop` | /proc/net/dev | Dropped outgoing packets | count | 📊 collect |
| `net_host_transmittedFifo` | /proc/net/dev | FIFO buffer errors (transmit) | count | 📊 collect |
| `net_host_transmittedColls` | /proc/net/dev | Packet collisions during transmit | count | 📊 collect |
| `net_host_transmittedCarrier` | /proc/net/dev | Carrier losses during transmit | count | 📊 collect |
| `net_host_transmittedCompressed` | /proc/net/dev | Compressed packets transmitted | count | 📊 collect |

#### Internal Metrics 🔐

| Metric | Source | Description | Cycle |
|--------|--------|-------------|-------|
| `net_host_ifs` | libvirt | List of relevant physical network interfaces | 🔄 lookup |

### VM Metrics

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `net_receivedBytes` | /proc/${pid}/net/dev | Bytes received by this VM | bytes | 📊 collect |
| `net_transmittedBytes` | /proc/${pid}/net/dev | Bytes transmitted by this VM | bytes | 📊 collect |

#### Verbose Mode VM Metrics 📝

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `net_receivedPackets` | /proc/${pid}/net/dev | Packets received | count | 📊 collect |
| `net_receivedErrs` | /proc/${pid}/net/dev | Receive errors | count | 📊 collect |
| `net_receivedDrop` | /proc/${pid}/net/dev | Dropped incoming packets | count | 📊 collect |
| `net_receivedFifo` | /proc/${pid}/net/dev | FIFO buffer errors (receive) | count | 📊 collect |
| `net_receivedFrame` | /proc/${pid}/net/dev | Framing errors | count | 📊 collect |
| `net_receivedCompressed` | /proc/${pid}/net/dev | Compressed packets received | count | 📊 collect |
| `net_receivedMulticast` | /proc/${pid}/net/dev | Multicast frames received | count | 📊 collect |
| `net_transmittedPackets` | /proc/${pid}/net/dev | Packets transmitted | count | 📊 collect |
| `net_transmittedErrs` | /proc/${pid}/net/dev | Transmit errors | count | 📊 collect |
| `net_transmittedDrop` | /proc/${pid}/net/dev | Dropped outgoing packets | count | 📊 collect |
| `net_transmittedFifo` | /proc/${pid}/net/dev | FIFO buffer errors (transmit) | count | 📊 collect |
| `net_transmittedColls` | /proc/${pid}/net/dev | Packet collisions | count | 📊 collect |
| `net_transmittedCarrier` | /proc/${pid}/net/dev | Carrier losses | count | 📊 collect |
| `net_transmittedCompressed` | /proc/${pid}/net/dev | Compressed packets transmitted | count | 📊 collect |

#### Internal Metrics 🔐

| Metric | Source | Description | Cycle |
|--------|--------|-------------|-------|
| `net_interfaces` | libvirt | List of virtual network interfaces for this VM | 🔄 lookup |

---

## Disk Collector (`--disk`)

Monitors block storage devices and I/O operations. Use `--storedev` to manually specify host storage devices. This collector provides esxtop-style storage metrics.

### Host Metrics

| Metric | Display Name | Source | Description | Unit | Cycle |
|--------|--------------|--------|-------------|------|-------|
| `disk_device_reads` | READS/s | /proc/diskstats | Read operations per second | ops/s | 📊 collect |
| `disk_device_writes` | WRITES/s | /proc/diskstats | Write operations per second | ops/s | 📊 collect |
| `disk_device_mbread` | MBRD/s | /proc/diskstats | MB read per second (sectors × 512) | MB/s | 📊 collect |
| `disk_device_mbwrite` | MBWR/s | /proc/diskstats | MB written per second (sectors × 512) | MB/s | 📊 collect |
| `disk_device_ioutil` | %UTIL | calculated | Device busy percentage (time doing I/Os) | % | 📊 collect |
| `disk_device_currentops` | QDEPTH | /proc/diskstats | Current I/O queue depth (ops in progress) | count | 📊 collect |
| `disk_device_queuelen` | QLEN | calculated | Average queue length (weighted time / elapsed) | count | 📊 collect |
| `disk_device_servicetime` | SVCTM | calculated | Average service time per I/O | ms | 📊 collect |
| `disk_device_await` | AWAIT | calculated | Average wait time per I/O (queue + service) | ms | 📊 collect |

#### Verbose Mode Host Metrics 📝

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `disk_device_readsmerged` | /proc/diskstats | Adjacent reads merged | count | 📊 collect |
| `disk_device_sectorsread` | /proc/diskstats | Sectors read | count | 📊 collect |
| `disk_device_timereading` | /proc/diskstats | Time spent reading | ms | 📊 collect |
| `disk_device_writesmerged` | /proc/diskstats | Adjacent writes merged | count | 📊 collect |
| `disk_device_sectorswritten` | /proc/diskstats | Sectors written | count | 📊 collect |
| `disk_device_timewriting` | /proc/diskstats | Time spent writing | ms | 📊 collect |
| `disk_device_timeforops` | /proc/diskstats | Total time spent on I/Os | ms | 📊 collect |
| `disk_device_weightedtimeforops` | /proc/diskstats | Weighted time doing I/Os | ms | 📊 collect |
| `disk_device_count` | /proc/diskstats | Number of relevant disks | count | 🔄 lookup |

#### Internal Metrics 🔐

| Metric | Source | Description | Cycle |
|--------|--------|-------------|-------|
| `disk_sources` | libvirt | List of relevant disk devices | 🔄 lookup |

### VM Metrics

| Metric | Display Name | Source | Description | Unit | Cycle |
|--------|--------------|--------|-------------|------|-------|
| `disk_size_capacity` | SIZE | libvirt/QMP | Maximum virtual disk capacity | bytes | 🔄 lookup |
| `disk_size_allocation` | ALLOC | libvirt/QMP | Currently allocated disk space | bytes | 🔄 lookup |
| `disk_ioutil` | %UTIL | calculated | Estimated I/O utilization for VM | % | 📊 collect |
| `disk_reads` | READS/s | libvirt/QMP | Read operations per second | ops/s | 📊 collect |
| `disk_writes` | WRITES/s | libvirt/QMP | Write operations per second | ops/s | 📊 collect |
| `disk_mbread` | MBRD/s | libvirt/QMP | MB read per second | MB/s | 📊 collect |
| `disk_mbwrite` | MBWR/s | libvirt/QMP | MB written per second | MB/s | 📊 collect |
| `disk_lat_rd` | LAT/rd | libvirt/QMP | Average read latency | ms | 📊 collect |
| `disk_lat_wr` | LAT/wr | libvirt/QMP | Average write latency | ms | 📊 collect |
| `disk_lat_fl` | LAT/fl | libvirt/QMP | Average flush latency | ms | 📊 collect |
| `disk_lat_avg` | LAT/avg | calculated | Average latency across all ops | ms | 📊 collect |

#### Verbose Mode VM Metrics 📝

| Metric | Display Name | Source | Description | Unit | Cycle |
|--------|--------------|--------|-------------|------|-------|
| `disk_size_physical` | PHYSICAL | libvirt | Physical space for virtual disks | bytes | 🔄 lookup |
| `disk_stats_flushreq` | FLUSH/s | libvirt | Cache flush requests per second | ops/s | 📊 collect |
| `disk_stats_rdtotaltimes` | RDTM | libvirt | Total time spent on reads | ms | 📊 collect |
| `disk_stats_wrtotaltimes` | WRTM | libvirt | Total time spent on writes | ms | 📊 collect |
| `disk_stats_flushtotaltimes` | FLTM | libvirt | Total time spent on flushes | ms | 📊 collect |
| `disk_delayblkio` | BLKIO | /proc/${pid}/stat | Aggregated block I/O delays | ticks | 📊 collect |

#### Internal Metrics 🔐

| Metric | Source | Description | Cycle |
|--------|--------|-------------|-------|
| `disk_sources` | libvirt | List of virtual disk sources | 🔄 lookup |

---

## I/O Collector (`--io`) ⚠️

Extends disk metrics with process-level I/O statistics. **Requires root access** to `/proc/${pid}/io`.

### Host Metrics

No host-level metrics are collected by this collector.

### VM Metrics

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `io_read_bytes` | /proc/${pid}/io | Bytes read directly from disk | bytes | 📊 collect |
| `io_write_bytes` | /proc/${pid}/io | Bytes originally dirtied in page cache | bytes | 📊 collect |

#### Verbose Mode VM Metrics 📝

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `io_rchar` | /proc/${pid}/io | Bytes read via any read-like syscall | bytes | 📊 collect |
| `io_wchar` | /proc/${pid}/io | Bytes written via any write-like syscall | bytes | 📊 collect |
| `io_syscr` | /proc/${pid}/io | Read-like system calls performed | count | 📊 collect |
| `io_syscw` | /proc/${pid}/io | Write-like system calls performed | count | 📊 collect |
| `io_cancelled_write_bytes` | /proc/${pid}/io | Bytes "un-dirtied" (e.g., ftruncate) | bytes | 📊 collect |

---

## Host Collector (`--host`)

Provides host identification information useful for multi-host deployments.

### Host Metrics

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `host_name` | /proc/sys/kernel/hostname | Hostname of the hypervisor | string | 🔄 lookup |

#### Verbose Mode Host Metrics 📝 ⚠️

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `host_uuid` | /sys/devices/virtual/dmi/id/product_uuid | DMI UUID of the host (requires root) | string | 🔄 lookup |

### VM Metrics

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `host_name` | /proc/sys/kernel/hostname | Hostname of the hypervisor running this VM | string | 🔄 lookup |

---

## PSI Collector (`--pressure`)

Monitors Pressure Stall Information (PSI) to detect resource shortages before they cause visible performance degradation. **Requires kernel 4.20+**.

For more information, see: https://facebookmicrosites.github.io/psi/docs/overview

### Host Metrics

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `psi_some_cpu_avg60` | /proc/pressure/cpu | Time some tasks delayed for CPU (60s window) | % | 📊 collect |
| `psi_some_io_avg60` | /proc/pressure/io | Time some tasks delayed for I/O (60s window) | % | 📊 collect |
| `psi_full_io_avg60` | /proc/pressure/io | Time all tasks delayed for I/O (60s window) | % | 📊 collect |
| `psi_some_mem_avg60` | /proc/pressure/mem | Time some tasks delayed for memory (60s window) | % | 📊 collect |
| `psi_full_mem_avg60` | /proc/pressure/mem | Time all tasks delayed for memory (60s window) | % | 📊 collect |

#### Verbose Mode Host Metrics 📝

| Metric | Source | Description | Unit | Cycle |
|--------|--------|-------------|------|-------|
| `psi_some_cpu_avg10` | /proc/pressure/cpu | CPU pressure, some tasks (10s window) | % | 📊 collect |
| `psi_some_cpu_avg300` | /proc/pressure/cpu | CPU pressure, some tasks (300s window) | % | 📊 collect |
| `psi_some_cpu_total` | /proc/pressure/cpu | Total CPU delay for some tasks | μs | 📊 collect |
| `psi_some_io_avg10` | /proc/pressure/io | I/O pressure, some tasks (10s window) | % | 📊 collect |
| `psi_some_io_avg300` | /proc/pressure/io | I/O pressure, some tasks (300s window) | % | 📊 collect |
| `psi_some_io_total` | /proc/pressure/io | Total I/O delay for some tasks | μs | 📊 collect |
| `psi_full_io_avg10` | /proc/pressure/io | I/O pressure, all tasks (10s window) | % | 📊 collect |
| `psi_full_io_avg300` | /proc/pressure/io | I/O pressure, all tasks (300s window) | % | 📊 collect |
| `psi_full_io_total` | /proc/pressure/io | Total I/O delay for all tasks | μs | 📊 collect |
| `psi_some_mem_avg10` | /proc/pressure/mem | Memory pressure, some tasks (10s window) | % | 📊 collect |
| `psi_some_mem_avg300` | /proc/pressure/mem | Memory pressure, some tasks (300s window) | % | 📊 collect |
| `psi_some_mem_total` | /proc/pressure/mem | Total memory delay for some tasks | μs | 📊 collect |
| `psi_full_mem_avg10` | /proc/pressure/mem | Memory pressure, all tasks (10s window) | % | 📊 collect |
| `psi_full_mem_avg300` | /proc/pressure/mem | Memory pressure, all tasks (300s window) | % | 📊 collect |
| `psi_full_mem_total` | /proc/pressure/mem | Total memory delay for all tasks | μs | 📊 collect |

### VM Metrics

No VM-level metrics are collected by this collector. PSI data is host-level only.

---

## Metric Calculations

Several metrics are derived from raw data using formulas:

### CPU Utilization

VM CPU metrics are calculated from scheduler statistics:

```
cpu_total = Δcpu_times / Δtime × 100
cpu_steal = Δcpu_runqueues / Δtime × 100
```

Where:
- `Δcpu_times`: Change in CPU time counters from `/proc/${pid}/schedstat`
- `Δcpu_runqueues`: Change in run queue wait time
- `Δtime`: Time elapsed between measurements

### Disk I/O Utilization

**Host ioutil:**
```
disk_device_ioutil = (Δtimeforops / Δtime) × 100
```

**VM ioutil (estimated):**
```
disk_ioutil = host_ioutil × (vm_io_requests / total_io_requests)
```

### Disk Queue Metrics

**Queue Size:**
```
disk_device_queuesize = Δweightedtimeforops / Δtime
```

**Queue Time:**
```
disk_device_queuetime = Δweightedtimeforops / (Δreads + Δreadsmerged + Δwrites + Δwritesmerged + currentops)
```

**Service Time:**
```
disk_device_servicetime = Δweightedtimeforops / (Δreads + Δreadsmerged + Δwrites + Δwritesmerged)
```

---

## Notes

1. **Collection Cycles**: Metrics marked with "lookup" are collected during VM discovery (less frequently), while "collect" metrics are gathered every sampling interval.

2. **Verbose Fields**: In the ncurses UI, verbose-only fields are hidden by default but can be enabled via the field selector (press `f`). In CLI mode, use `--verbose` to enable all additional metrics.

3. **Human-Readable Mode**: Enable with `-H` command-line flag or press `u` in the ncurses UI to format byte values as KB/MB/GB/TB. When active, the status bar displays `[H]`. Changes take effect immediately.

4. **Root Requirements**: Some metrics (I/O collector, host UUID) require root privileges to access protected proc filesystem entries.

5. **Proxmox vs Libvirt**: On Proxmox VE, QMP (QEMU Machine Protocol) is used instead of libvirt API for faster metric collection.

6. **Rate Metrics**: Most counter-based metrics (bytes, packets, etc.) should be interpreted as rates when comparing across time intervals.

7. **esxtop Equivalent Metrics**: proxtop provides many esxtop-equivalent metrics for VMware-to-KVM/Proxmox migrations:
   - `%UTIL` = Device busy percentage (like esxtop GAVG/DAVG context)
   - `QDEPTH` = Current queue depth (like esxtop ACTV)
   - `QLEN` = Average queue length (like esxtop QUED)
   - `SVCTM` = Service time (like esxtop DAVG)
   - `AWAIT` = Wait time (like esxtop KAVG)
   - `LAT/rd`, `LAT/wr`, `LAT/fl` = Per-operation type latencies

8. **Sorting**: Use `<`/`>` to change sort column, `r` to toggle ascending/descending. The sorted column shows `^` (ascending) or `v` (descending). Numeric columns are sorted numerically.

9. **Physical Device Views**: Press `p` for physical network interfaces or `s` for physical disk devices. These views use the same field selector as VM views - press `f` to show/hide columns.

10. **Physical Network Interface Metrics** (`p` key in ncurses): Per-interface statistics from `/proc/net/dev` with calculated rates:
    - `net_MbRX/s` = Megabits/sec received
    - `net_MbTX/s` = Megabits/sec transmitted
    - `net_PKTRX/s` = Packets/sec received
    - `net_PKTTX/s` = Packets/sec transmitted
    - Raw counters (RX-Bytes, TX-Bytes, etc.) are also available via field selector

11. **Physical Disk Device Metrics** (`s` key in ncurses): Per-device statistics from `/proc/diskstats` with calculated rates:
    - `dsk_READS/s` = Read operations per second
    - `dsk_WRITES/s` = Write operations per second
    - `dsk_MBRD/s` = MB/sec read throughput
    - `dsk_MBWR/s` = MB/sec write throughput
    - `dsk_%UTIL` = Device busy percentage (0-100%)
    - `dsk_QDEPTH` = Current queue depth (I/Os in flight)
    - `dsk_SVCTM` = Average service time per I/O (ms)
    - `dsk_AWAIT` = Average wait time per I/O including queue time (ms)

12. **Host Storage Device Views**: Three separate views for different storage types, each with the same metrics as above:

    - **Physical Disks** (`s` key): SCSI/SATA (`sd*`), NVMe (`nvme*`), virtio (`vd*`), partitions (`sda1`, etc.)
    - **LVM Volumes** (`l` key): LVM logical volumes (detected via `LVM-` UUID prefix in `/sys/block/dm-*/dm/uuid`)
    - **Multipath Devices** (`x` key): DM-Multipath devices (detected via `mpath-` UUID prefix in `/sys/block/dm-*/dm/uuid`)

    Device mapper devices (`dm-*`) are displayed with their friendly names from `/dev/mapper/` instead of the raw `dm-X` name.

    Excluded from all views: `loop*` (loopback), `ram*` (RAM disks)

