package utils

import "fmt"

// HumanReadableBytesBinary 使用二进制单位（GiB, MiB）
func HumanReadableBytesBinary(bytes uint64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	value := float64(bytes)
	base := 1024.0
	i := 0
	for ; i < len(units)-1 && value >= base; i++ {
		value /= base
	}

	return fmt.Sprintf("%.2f %s", value, units[i])
}

// HumanReadableBytes 将字节数格式化为可读字符串（使用十进制 GB）
func HumanReadableBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	base := 1000.0 // 使用 1000 还是 1024？见下方说明
	i := 0
	for ; i < len(units)-1 && value >= base; i++ {
		value /= base
	}

	return fmt.Sprintf("%.2f %s", value, units[i])
}
