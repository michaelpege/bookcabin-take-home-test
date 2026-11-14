package batik

// wireSearchRequest represents Batik Air's native Stage-A query contract.
type wireSearchRequest struct {
	Origin        string
	Destination   string
	DepartureDate string
	CabinClass    string
	Passengers    int
}

type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Results []rawFlight `json:"results"`
}

type rawFlight struct {
	FlightNumber      string          `json:"flightNumber"`
	AirlineName       string          `json:"airlineName"`
	AirlineIATA       string          `json:"airlineIATA"`
	Origin            string          `json:"origin"`
	Destination       string          `json:"destination"`
	DepartureDateTime string          `json:"departureDateTime"`
	ArrivalDateTime   string          `json:"arrivalDateTime"`
	TravelTime        string          `json:"travelTime"`
	NumberOfStops     int             `json:"numberOfStops"`
	Connections       []rawConnection `json:"connections"`
	Fare              rawFare         `json:"fare"`
	SeatsAvailable    int             `json:"seatsAvailable"`
	AircraftModel     string          `json:"aircraftModel"`
	BaggageInfo       string          `json:"baggageInfo"`
	OnboardServices   []string        `json:"onboardServices"`
}

type rawConnection struct {
	StopAirport  string `json:"stopAirport"`
	StopDuration string `json:"stopDuration"`
}

type rawFare struct {
	BasePrice    int64  `json:"basePrice"`
	Taxes        int64  `json:"taxes"`
	TotalPrice   int64  `json:"totalPrice"`
	CurrencyCode string `json:"currencyCode"`
	Class        string `json:"class"`
}
