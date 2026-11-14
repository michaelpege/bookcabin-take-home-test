package currency

import "testing"

func TestFormatIDR(t *testing.T) {
	tests := []struct {
		amount int64
		want   string
	}{
		{amount: 0, want: "Rp 0"},
		{amount: 500, want: "Rp 500"},
		{amount: 485000, want: "Rp 485.000"},
		{amount: 1250000, want: "Rp 1.250.000"},
		{amount: 12500000, want: "Rp 12.500.000"},
		{amount: -50000, want: "-Rp 50.000"},
	}

	for _, tc := range tests {
		if got := FormatIDR(tc.amount); got != tc.want {
			t.Errorf("FormatIDR(%d) = %q, want %q", tc.amount, got, tc.want)
		}
	}
}
