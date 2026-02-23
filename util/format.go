package util

import (
	"fmt"
	"strconv"
)

// FormatBytes converts a byte value to human-readable format (KB, MB, GB, TB)
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1fT", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d", bytes)
	}
}

// FormatBytesFromString converts a string byte value to human-readable format
func FormatBytesFromString(bytesStr string) string {
	bytes, err := strconv.ParseUint(bytesStr, 10, 64)
	if err != nil {
		return bytesStr // Return original if not a valid number
	}
	return FormatBytes(bytes)
}

