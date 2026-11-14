package lion

// wireSearchRequest represents Lion Air's native Stage-A query contract.
type wireSearchRequest struct {
	Origin        string
	Destination   string
	DepartureDate string
	CabinClass    string
	Passengers    int
}

type apiResponse struct {
	Success bool        `json:"success"`
	Data    apiDataBody `json:"data"`
}

type apiDataBody struct {
	AvailableFlights []rawFlight `json:"available_flights"`
}

type rawFlight struct {
	ID         string       `json:"id"`
	Carrier    rawCarrier   `json:"carrier"`
	Route      rawRoute     `json:"route"`
	Schedule   rawSchedule  `json:"schedule"`
	FlightTime int          `json:"flight_time"`
	IsDirect   bool         `json:"is_direct"`
	StopCount  int          `json:"stop_count"`
	Layovers   []rawLayover `json:"layovers"`
	Pricing    rawPricing   `json:"pricing"`
	SeatsLeft  int          `json:"seats_left"`
	PlaneType  string       `json:"plane_type"`
	Services   rawServices  `json:"services"`
}

type rawCarrier struct {
	Name string `json:"name"`
	IATA string `json:"iata"`
}

type rawRoute struct {
	From rawAirport `json:"from"`
	To   rawAirport `json:"to"`
}

type rawAirport struct {
	Code string `json:"code"`
	Name string `json:"name"`
	City string `json:"city"`
}

type rawSchedule struct {
	Departure         string `json:"departure"`
	DepartureTimezone string `json:"departure_timezone"`
	Arrival           string `json:"arrival"`
	ArrivalTimezone   string `json:"arrival_timezone"`
}

type rawLayover struct {
	Airport         string `json:"airport"`
	DurationMinutes int    `json:"duration_minutes"`
}

type rawPricing struct {
	Total    int64  `json:"total"`
	Currency string `json:"currency"`
	FareType string `json:"fare_type"`
}

type rawServices struct {
	WifiAvailable    bool       `json:"wifi_available"`
	MealsIncluded    bool       `json:"meals_included"`
	BaggageAllowance rawBaggage `json:"baggage_allowance"`
}

type rawBaggage struct {
	Cabin string `json:"cabin"`
	Hold  string `json:"hold"`
}
