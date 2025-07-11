package utils

import (
	"fmt"
	"strings"
)

func SanitizeFileName(name string) string {
	replacer := strings.NewReplacer(
		"\\", "＼", "/", "／", ":", "：", "*", "＊",
		"?", "？", "\"", "＂", "<", "＜", ">", "＞", "|", "｜",
	)
	return replacer.Replace(name)
}

// FormatFileSize 将字节数格式化为可读的字符串
func FormatFileSize(totalsize int64) string {
	if totalsize <= 0 {
		return "未知大小"
	}
	const (
		_          = iota
		KB float64 = 1 << (10 * iota)
		MB
		GB
		TB
		PB
	)

	size := float64(totalsize)

	switch {
	case size >= PB:
		return fmt.Sprintf("%.2fPB", size/PB)
	case size >= TB:
		return fmt.Sprintf("%.2fTB", size/TB)
	case size >= GB:
		return fmt.Sprintf("%.2fGB", size/GB)
	case size >= MB:
		return fmt.Sprintf("%.2fMB", size/MB)
	case size >= KB:
		return fmt.Sprintf("%.2fKB", size/KB)
	default:
		return fmt.Sprintf("%dB", totalsize)
	}
}
