package provider

import "strings"

var airportCities = map[string]string{
	"CGK": "Jakarta",
	"DPS": "Denpasar",
	"SUB": "Surabaya",
	"SOC": "Solo",
	"UPG": "Makassar",
	"JOG": "Yogyakarta",
	"YIA": "Yogyakarta",
	"KNO": "Medan",
	"BPN": "Balikpapan",
	"SIN": "Singapore",
	"KUL": "Kuala Lumpur",
}

// ResolveCity returns explicitCity when non-empty, or maps well-known IATA airport codes
// to city names, falling back to the uppercased airport code.
func ResolveCity(airportCode, explicitCity string) string {
	if trimmed := strings.TrimSpace(explicitCity); trimmed != "" {
		return trimmed
	}
	code := strings.ToUpper(strings.TrimSpace(airportCode))
	if city, ok := airportCities[code]; ok {
		return city
	}
	return code
}
