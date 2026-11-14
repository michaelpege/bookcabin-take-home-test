package airasia

// wireSearchRequest represents AirAsia's native Stage-A query contract.
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
	FlightCode    string    `json:"flight_code"`
	Airline       string    `json:"airline"`
	FromAirport   string    `json:"from_airport"`
	ToAirport     string    `json:"to_airport"`
	DepartTime    string    `json:"depart_time"`
	ArriveTime    string    `json:"arrive_time"`
	DurationHours float64   `json:"duration_hours"`
	DirectFlight  bool      `json:"direct_flight"`
	Stops         []rawStop `json:"stops"`
	PriceIDR      int64     `json:"price_idr"`
	Seats         int       `json:"seats"`
	CabinClass    string    `json:"cabin_class"`
	BaggageNote   string    `json:"baggage_note"`
}

type rawStop struct {
	Airport         string `json:"airport"`
	WaitTimeMinutes int    `json:"wait_time_minutes"`
}
