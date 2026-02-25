# Changelog

All notable changes to proxtop are documented in this file.

## [1.1.7] - 2026-02-25

- Added separate LVM view ('l' key) for LVM logical volumes
- Added separate Multipath view ('x' key) for dm-multipath devices
- LVM/multipath detection uses /sys/block/dm-*/dm/uuid (LVM-, mpath- prefixes)
- Device mapper devices show friendly names from /dev/mapper/
- Screen refreshes immediately on any keypress

## [1.1.6] - 2026-02-25

- Fixed physical disk view %UTIL calculation (was showing 0)
- Physical disk view now shows proper per-second rates for all metrics

## [1.1.4] - 2026-02-24

- Fixed PSI collector not returning all metrics in non-verbose mode
- Text printer now displays host-level metrics (PSI, CPU host stats, etc.)
- Added Mb/s and pkts/s rate metrics to physical network view

## [1.1.3] - 2026-02-24

- Fixed per interface network display showing no data for verbose fields

## [1.1.2] - 2026-02-24

- Added sort direction toggle ('r' keybind) - ascending/descending
- Added sort direction indicators (^/v) in column headers
- Fixed numeric sorting in physical device views
- Converted physical views to use standard field system
- Physical view fields now work with field selector
- Added immediate refresh on settings toggle (no delay)
- Verbose fields now hidden by default in field selector

## [1.1.1] - 2026-02-23

- Fixed ncurses column overflow with dynamic width calculation
- Fixed memory unit inconsistency (KB vs bytes)
- Added human-readable units toggle (-H flag, 'u' keybind)
- Added physical disk metrics: QDEPTH, QLEN, SVCTM, AWAIT, MB/s
- Added VM disk latency metrics: LAT/rd, LAT/wr, LAT/fl
- Enhanced esxtop-equivalent metrics coverage
- Improved field selector filtering per view mode
- Updated documentation with new metrics and usage

## [1.1.0] - 2026-02-22

- Added GitHub Actions workflow for automated .deb builds
- Added comprehensive documentation (DOCUMENTATION.md)
- Added detailed metrics reference (METRICS_REFERENCE.md)
- Streamlined build dependencies

## [1.0.0] - 2026-02-21

- Initial release as proxtop (renamed from kvmtop)
- Added Proxmox VE support with QMP connector
- Auto-detection of hypervisor type (libvirt vs Proxmox)

