package garuda

// wireSearchRequest represents Garuda's native Stage-A query contract.
type wireSearchRequest struct {
	Origin        string
	Destination   string
	DepartureDate string
	CabinClass    string
	Passengers    int
}

type apiResponse struct {
	Status  string      `json:"status"`
	Flights []rawFlight `json:"flights"`
}

type rawFlight struct {
	FlightID        string       `json:"flight_id"`
	Airline         string       `json:"airline"`
	AirlineCode     string       `json:"airline_code"`
	Departure       rawEndpoint  `json:"departure"`
	Arrival         rawEndpoint  `json:"arrival"`
	DurationMinutes int          `json:"duration_minutes"`
	Stops           int          `json:"stops"`
	Aircraft        string       `json:"aircraft"`
	Price           rawPrice     `json:"price"`
	Segments        []rawSegment `json:"segments"`
	AvailableSeats  int          `json:"available_seats"`
	FareClass       string       `json:"fare_class"`
	Baggage         rawBaggage   `json:"baggage"`
	Amenities       []string     `json:"amenities"`
}

type rawEndpoint struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Time     string `json:"time"`
	Terminal string `json:"terminal"`
}

type rawSegment struct {
	FlightNumber    string      `json:"flight_number"`
	Departure       rawEndpoint `json:"departure"`
	Arrival         rawEndpoint `json:"arrival"`
	DurationMinutes int         `json:"duration_minutes"`
	LayoverMinutes  int         `json:"layover_minutes"`
}

type rawPrice struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type rawBaggage struct {
	CarryOn int `json:"carry_on"`
	Checked int `json:"checked"`
}
