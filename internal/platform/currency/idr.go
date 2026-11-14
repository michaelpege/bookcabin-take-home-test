package currency

import (
	"strconv"
	"strings"
)

// FormatIDR formats an integer IDR amount using Indonesian dot thousands separators
// (e.g., 1250000 -> "Rp 1.250.000").
func FormatIDR(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}

	digits := strconv.FormatInt(amount, 10)
	n := len(digits)
	if n <= 3 {
		if negative {
			return "-Rp " + digits
		}
		return "Rp " + digits
	}

	var b strings.Builder
	if negative {
		b.WriteString("-Rp ")
	} else {
		b.WriteString("Rp ")
	}

	rem := n % 3
	if rem > 0 {
		b.WriteString(digits[:rem])
		if n > rem {
			b.WriteByte('.')
		}
	}
	for i := rem; i < n; i += 3 {
		if i > rem {
			b.WriteByte('.')
		}
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}
