package currency

import (
	"strconv"
	"strings"
)

// FormatIDR formats an integer IDR amount using Indonesian dot thousands separators
// (e.g., 1250000 -> "Rp 1.250.000").
func FormatIDR(amount int64) string {
	prefix := "Rp "
	if amount < 0 {
		prefix = "-Rp "
		amount = -amount
	}

	s := strconv.FormatInt(amount, 10)
	var b strings.Builder
	b.WriteString(prefix)
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	return b.String()
}
