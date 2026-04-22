package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func humanizeBytesIEC(b float64) string {
	if math.IsNaN(b) || math.IsInf(b, 0) {
		return fmt.Sprintf("%f", b)
	}
	if b < 0 {
		return "-" + humanizeBytesIEC(-b)
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	i := 0
	for i < len(units)-1 && b >= 1024 {
		b /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", b, units[i])
	}
	return fmt.Sprintf("%.2f %s", b, units[i])
}

func humanizeDurationSeconds(sec float64) string {
	if math.IsNaN(sec) || math.IsInf(sec, 0) {
		return fmt.Sprintf("%f", sec)
	}
	d := time.Duration(sec * float64(time.Second))
	// Keep it compact for tables.
	if d < time.Second {
		return d.String()
	}
	// Truncate to seconds.
	d = (d / time.Second) * time.Second
	return d.String()
}

func humanizeNumber(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Sprintf("%f", v)
	}
	// If it looks like an integer, add separators.
	if math.Abs(v-math.Round(v)) < 1e-9 && math.Abs(v) < 9e15 {
		return commaInt64(int64(math.Round(v)))
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func commaInt64(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return sign + s
	}
	var b strings.Builder
	b.WriteString(sign)
	pre := len(s) % 3
	if pre == 0 {
		pre = 3
	}
	b.WriteString(s[:pre])
	for i := pre; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func formatValueHuman(metric string, value float64) string {
	switch {
	case strings.HasSuffix(metric, "_bytes"):
		return humanizeBytesIEC(value)
	case strings.HasSuffix(metric, "_seconds") || strings.HasSuffix(metric, "_seconds_total"):
		return humanizeDurationSeconds(value)
	default:
		return humanizeNumber(value)
	}
}

