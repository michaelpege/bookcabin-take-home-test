package timeutil

import (
	"testing"
	"time"
)

func TestLoadLocationAndAliases(t *testing.T) {
	tests := []struct {
		input      string
		wantOffset int
		wantErr    bool
	}{
		{input: "WIB", wantOffset: 7 * 3600},
		{input: "WITA", wantOffset: 8 * 3600},
		{input: "WIT", wantOffset: 9 * 3600},
		{input: "Asia/Jakarta", wantOffset: 7 * 3600},
		{input: "Asia/Makassar", wantOffset: 8 * 3600},
		{input: "Asia/Jayapura", wantOffset: 9 * 3600},
		{input: "", wantErr: true},
		{input: "Invalid/Zone", wantErr: true},
	}

	ref := time.Date(2025, 12, 15, 12, 0, 0, 0, time.UTC)
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			loc, err := LoadLocation(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			_, offset := ref.In(loc).Zone()
			if offset != tc.wantOffset {
				t.Fatalf("expected offset %d, got %d", tc.wantOffset, offset)
			}
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		tz      string
		wantUTC string
		wantErr bool
	}{
		{
			name:    "garuda rfc3339",
			raw:     "2025-12-15T06:00:00+07:00",
			wantUTC: "2025-12-14T23:00:00Z",
		},
		{
			name:    "batik numeric offset without colon",
			raw:     "2025-12-15T10:00:00+0800",
			wantUTC: "2025-12-15T02:00:00Z",
		},
		{
			name:    "lion air naive jakarta",
			raw:     "2025-12-15T05:30:00",
			tz:      "Asia/Jakarta",
			wantUTC: "2025-12-14T22:30:00Z",
		},
		{
			name:    "lion air naive makassar",
			raw:     "2025-12-15T08:15:00",
			tz:      "Asia/Makassar",
			wantUTC: "2025-12-15T00:15:00Z",
		},
		{
			name:    "naive without tz fails",
			raw:     "2025-12-15T08:15:00",
			tz:      "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTimestamp(tc.raw, tc.tz)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.UTC().Format(time.RFC3339) != tc.wantUTC {
				t.Fatalf("expected %s, got %s", tc.wantUTC, got.UTC().Format(time.RFC3339))
			}
		})
	}
}

func TestDurationParsingAndFormatting(t *testing.T) {
	tests := []struct {
		raw     string
		wantMin int
		wantFmt string
		wantErr bool
	}{
		{raw: "1h 45m", wantMin: 105, wantFmt: "1h 45m"},
		{raw: "3h 5m", wantMin: 185, wantFmt: "3h 5m"},
		{raw: "55m", wantMin: 55, wantFmt: "55m"},
		{raw: "2h", wantMin: 120, wantFmt: "2h"},
		{raw: "invalid", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := ParseDurationMinutes(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantMin {
				t.Fatalf("expected %d minutes, got %d", tc.wantMin, got)
			}
			if formatted := FormatDuration(got); formatted != tc.wantFmt {
				t.Fatalf("expected %q, got %q", tc.wantFmt, formatted)
			}
		})
	}

	if mins := HoursToMinutes(4.33); mins != 260 {
		t.Fatalf("expected 4.33 hours -> 260 minutes, got %d", mins)
	}
	if fmtStr := FormatDuration(260); fmtStr != "4h 20m" {
		t.Fatalf("expected 260 -> 4h 20m, got %q", fmtStr)
	}
}
