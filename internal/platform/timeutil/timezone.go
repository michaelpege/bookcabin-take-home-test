package timeutil

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // Embed IANA timezone data for minimal container runtimes.
)

var tzAliases = map[string]string{
	"WIB":  "Asia/Jakarta",
	"WITA": "Asia/Makassar",
	"WIT":  "Asia/Jayapura",
}

var airportTZ = map[string]string{
	// WIB (+07:00)
	"CGK": "Asia/Jakarta",
	"HLP": "Asia/Jakarta",
	"SUB": "Asia/Jakarta",
	"SOC": "Asia/Jakarta",
	"JOG": "Asia/Jakarta",
	"YIA": "Asia/Jakarta",
	"BDO": "Asia/Jakarta",
	"SRG": "Asia/Jakarta",
	"KNO": "Asia/Jakarta",
	"PLM": "Asia/Jakarta",
	"PDG": "Asia/Jakarta",
	"PKU": "Asia/Jakarta",
	"BTH": "Asia/Jakarta",
	"PNK": "Asia/Jakarta",
	// WITA (+08:00)
	"DPS": "Asia/Makassar",
	"UPG": "Asia/Makassar",
	"BPN": "Asia/Makassar",
	"LOP": "Asia/Makassar",
	"MDC": "Asia/Makassar",
	"BDJ": "Asia/Makassar",
	"KOE": "Asia/Makassar",
	// WIT (+09:00)
	"DJJ": "Asia/Jayapura",
	"SOQ": "Asia/Jayapura",
	"MKQ": "Asia/Jayapura",
	"AMQ": "Asia/Jayapura",
}

// LoadLocation resolves an IANA timezone name or Indonesian abbreviation (WIB, WITA, WIT).
func LoadLocation(name string) (*time.Location, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, fmt.Errorf("timezone name cannot be empty")
	}
	if canonical, ok := tzAliases[strings.ToUpper(trimmed)]; ok {
		trimmed = canonical
	}
	loc, err := time.LoadLocation(trimmed)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", name, err)
	}
	return loc, nil
}

// AirportLocation returns the Indonesian *time.Location for a known IATA airport code.
func AirportLocation(iata string) (*time.Location, error) {
	tz, ok := airportTZ[strings.ToUpper(strings.TrimSpace(iata))]
	if !ok {
		return nil, fmt.Errorf("unknown airport timezone for %q", iata)
	}
	return LoadLocation(tz)
}
