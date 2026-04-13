// Package gflights is flightgoat's Google Flights backend. It has two
// implementations wired in priority order:
//
//  1. Native: krisukox/google-flights-api, a pure-Go reverse-engineered client
//     that talks directly to Google's internal protobuf endpoints. No Python,
//     no subprocess, no API key.
//  2. Fallback: the fli Python CLI (pipx install flights). Used when the
//     native backend fails (e.g. Google's abuse-detection demands the
//     GOOGLE_ABUSE_EXEMPTION cookie that only a browser can solve).
//
// The two backends return the same normalized flightgoat types so callers
// don't care which one served the request. Each response carries a Source
// string so users can see which backend answered.
package gflights

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	krisukox "github.com/krisukox/google-flights-api/flights"
	"golang.org/x/text/currency"
	"golang.org/x/text/language"
)

// ErrFliNotInstalled is returned when the fli fallback binary can't be found
// AND the native backend already failed.
var ErrFliNotInstalled = errors.New("fli not installed (run: pipx install flights)")

// Airport matches the nested shape used across the normalized return types.
type Airport struct {
	Code string `json:"code"`
	Name string `json:"name,omitempty"`
}

// Airline matches the nested airline object.
type Airline struct {
	Code string `json:"code"`
	Name string `json:"name,omitempty"`
}

// Leg is one hop of a multi-stop itinerary.
type Leg struct {
	DepartureAirport Airport `json:"departure_airport"`
	ArrivalAirport   Airport `json:"arrival_airport"`
	DepartureTime    string  `json:"departure_time"`
	ArrivalTime      string  `json:"arrival_time"`
	DurationMinutes  int     `json:"duration"`
	Airline          Airline `json:"airline"`
	FlightNumber     string  `json:"flight_number"`
}

// Flight is one itinerary (possibly multi-leg).
type Flight struct {
	DurationMinutes int     `json:"duration"`
	Stops           int     `json:"stops"`
	Legs            []Leg   `json:"legs"`
	Price           float64 `json:"price"`
	Currency        string  `json:"currency"`
}

// SearchResult is the normalized envelope returned by Search.
type SearchResult struct {
	Success    bool     `json:"success"`
	Source     string   `json:"source"` // "native-go" or "fli-fallback"
	DataSource string   `json:"data_source"`
	SearchType string   `json:"search_type"`
	TripType   string   `json:"trip_type"`
	Query      SearchQuery `json:"query"`
	Count      int      `json:"count"`
	Flights    []Flight `json:"flights"`
}

// SearchQuery echoes the user's query back in the response envelope.
type SearchQuery struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	ReturnDate    string `json:"return_date,omitempty"`
	CabinClass    string `json:"cabin_class"`
	MaxStops      string `json:"max_stops"`
}

// DatePrice is one row in the cheapest-dates output.
type DatePrice struct {
	DepartureDate string  `json:"departure_date"`
	ReturnDate    string  `json:"return_date,omitempty"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency,omitempty"`
}

// DatesResult is the normalized envelope returned by Dates.
type DatesResult struct {
	Success    bool        `json:"success"`
	Source     string      `json:"source"`
	DataSource string      `json:"data_source"`
	SearchType string      `json:"search_type"`
	Query      SearchQuery `json:"query"`
	Count      int         `json:"count"`
	Dates      []DatePrice `json:"dates"`
}

// SearchOptions are the knobs users can pass to a flight search.
type SearchOptions struct {
	Origin        string
	Destination   string
	DepartureDate string
	ReturnDate    string
	TimeWindow    string
	Airlines      []string
	CabinClass    string
	MaxStops      string
	SortBy        string
	Passengers    int
	ExcludeBasic  bool
	// ForceFallback forces the fli Python subprocess path for debugging.
	ForceFallback bool
}

// Search runs a flight search. Tries the native Go backend first, falls back
// to the fli Python CLI if the native path fails.
func Search(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	if !opts.ForceFallback {
		if result, err := searchNative(ctx, opts); err == nil {
			return result, nil
		} else if !isRetryableErr(err) {
			// Non-retryable: bubble up without bothering the fallback
			return nil, err
		}
	}
	return searchFli(ctx, opts)
}

// searchNative uses krisukox/google-flights-api for a pure-Go call to
// Google Flights' internal protobuf endpoint.
func searchNative(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	session, err := krisukox.New()
	if err != nil {
		return nil, fmt.Errorf("native session: %w", err)
	}
	depDate, err := time.Parse("2006-01-02", opts.DepartureDate)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: want YYYY-MM-DD", opts.DepartureDate)
	}
	// krisukox insists on a return date even for OneWay; synthesize one if
	// the user didn't provide it.
	retDate := depDate.AddDate(0, 0, 7)
	if opts.ReturnDate != "" {
		rd, err := time.Parse("2006-01-02", opts.ReturnDate)
		if err != nil {
			return nil, fmt.Errorf("invalid return date %q: want YYYY-MM-DD", opts.ReturnDate)
		}
		retDate = rd
	}

	stops := krisukox.AnyStops
	switch strings.ToLower(strings.ReplaceAll(opts.MaxStops, "-", "_")) {
	case "0", "nonstop", "non_stop":
		stops = krisukox.Nonstop
	case "1", "one_stop":
		stops = krisukox.Stop1
	case "2", "two_plus_stops":
		stops = krisukox.Stop2
	}
	cabin := krisukox.Economy
	switch strings.ToLower(opts.CabinClass) {
	case "premium_economy", "premium-economy", "premium":
		cabin = krisukox.PremiumEconomy
	case "business":
		cabin = krisukox.Business
	case "first":
		cabin = krisukox.First
	}
	tripType := krisukox.OneWay
	if opts.ReturnDate != "" {
		tripType = krisukox.RoundTrip
	}

	passengers := 1
	if opts.Passengers > 0 {
		passengers = opts.Passengers
	}

	offers, _, err := session.GetOffers(ctx, krisukox.Args{
		Date:        depDate,
		ReturnDate:  retDate,
		SrcAirports: []string{opts.Origin},
		DstAirports: []string{opts.Destination},
		Options: krisukox.Options{
			Travelers: krisukox.Travelers{Adults: passengers},
			Currency:  currency.USD,
			Stops:     stops,
			Class:     cabin,
			TripType:  tripType,
			Lang:      language.English,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("native GetOffers: %w", err)
	}

	flights := make([]Flight, 0, len(offers))
	for _, o := range offers {
		f := Flight{
			DurationMinutes: int(o.FlightDuration.Minutes()),
			Stops:           len(o.Flight) - 1,
			Price:           o.Price,
			Currency:        "USD",
		}
		for _, leg := range o.Flight {
			f.Legs = append(f.Legs, Leg{
				DepartureAirport: Airport{Code: leg.DepAirportCode, Name: leg.DepAirportName},
				ArrivalAirport:   Airport{Code: leg.ArrAirportCode, Name: leg.ArrAirportName},
				DepartureTime:    leg.DepTime.Format(time.RFC3339),
				ArrivalTime:      leg.ArrTime.Format(time.RFC3339),
				DurationMinutes:  int(leg.Duration.Minutes()),
				Airline:          Airline{Name: leg.AirlineName, Code: leg.FlightNumber[:2]},
				FlightNumber:     leg.FlightNumber,
			})
		}
		flights = append(flights, f)
	}

	return &SearchResult{
		Success:    true,
		Source:     "native-go",
		DataSource: "google_flights",
		SearchType: "flights",
		TripType:   strings.ToUpper(strings.TrimSpace(string(tripTypeName(tripType)))),
		Query: SearchQuery{
			Origin:        opts.Origin,
			Destination:   opts.Destination,
			DepartureDate: opts.DepartureDate,
			ReturnDate:    opts.ReturnDate,
			CabinClass:    strings.ToUpper(opts.CabinClass),
			MaxStops:      strings.ToUpper(opts.MaxStops),
		},
		Count:   len(flights),
		Flights: flights,
	}, nil
}

func tripTypeName(t krisukox.TripType) string {
	if t == krisukox.RoundTrip {
		return "round_trip"
	}
	return "one_way"
}

// searchFli is the fallback path using the fli Python CLI via subprocess.
func searchFli(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	fliPath, err := exec.LookPath("fli")
	if err != nil {
		return nil, ErrFliNotInstalled
	}
	args := []string{"flights", opts.Origin, opts.Destination, opts.DepartureDate, "--format", "json"}
	if opts.ReturnDate != "" {
		args = append(args, "--return", opts.ReturnDate)
	}
	if opts.TimeWindow != "" {
		args = append(args, "--time", opts.TimeWindow)
	}
	if len(opts.Airlines) > 0 {
		args = append(args, "--airlines", strings.Join(opts.Airlines, " "))
	}
	if opts.CabinClass != "" {
		args = append(args, "--class", strings.ToUpper(opts.CabinClass))
	}
	if opts.MaxStops != "" {
		args = append(args, "--stops", strings.ToUpper(opts.MaxStops))
	}
	if opts.SortBy != "" {
		args = append(args, "--sort", strings.ToUpper(opts.SortBy))
	}
	if opts.Passengers > 1 {
		args = append(args, "--passengers", strconv.Itoa(opts.Passengers))
	}
	if opts.ExcludeBasic {
		args = append(args, "--exclude-basic")
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
	}
	out, err := exec.CommandContext(ctx, fliPath, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("fli flights failed: %w", err)
	}
	var fliRaw struct {
		Success    bool   `json:"success"`
		DataSource string `json:"data_source"`
		SearchType string `json:"search_type"`
		TripType   string `json:"trip_type"`
		Count      int    `json:"count"`
		Flights    []Flight `json:"flights"`
	}
	if err := json.Unmarshal(out, &fliRaw); err != nil {
		return nil, fmt.Errorf("parsing fli output: %w", err)
	}
	if !fliRaw.Success {
		return nil, fmt.Errorf("fli reported failure (check origin/destination IATA codes and date)")
	}
	return &SearchResult{
		Success:    true,
		Source:     "fli-fallback",
		DataSource: fliRaw.DataSource,
		SearchType: fliRaw.SearchType,
		TripType:   fliRaw.TripType,
		Query: SearchQuery{
			Origin:        opts.Origin,
			Destination:   opts.Destination,
			DepartureDate: opts.DepartureDate,
			ReturnDate:    opts.ReturnDate,
			CabinClass:    strings.ToUpper(opts.CabinClass),
			MaxStops:      strings.ToUpper(opts.MaxStops),
		},
		Count:   fliRaw.Count,
		Flights: fliRaw.Flights,
	}, nil
}

// DatesOptions drives a cheapest-dates query.
type DatesOptions struct {
	Origin      string
	Destination string
	From        string
	To          string
	Duration    int
	Airlines    []string
	RoundTrip   bool
	MaxStops    string
	CabinClass  string
	Sort        bool
}

// Dates runs a cheapest-dates query. The native backend does not expose a
// calendar endpoint, so we always use fli for this. If fli isn't installed,
// returns ErrFliNotInstalled.
func Dates(ctx context.Context, opts DatesOptions) (*DatesResult, error) {
	fliPath, err := exec.LookPath("fli")
	if err != nil {
		return nil, ErrFliNotInstalled
	}
	args := []string{"dates", opts.Origin, opts.Destination, "--format", "json"}
	if opts.From != "" {
		args = append(args, "--from", opts.From)
	}
	if opts.To != "" {
		args = append(args, "--to", opts.To)
	}
	if opts.Duration > 0 {
		args = append(args, "--duration", strconv.Itoa(opts.Duration))
	}
	if len(opts.Airlines) > 0 {
		args = append(args, "--airlines", strings.Join(opts.Airlines, " "))
	}
	if opts.RoundTrip {
		args = append(args, "--round")
	}
	if opts.MaxStops != "" {
		args = append(args, "--stops", strings.ToUpper(opts.MaxStops))
	}
	if opts.CabinClass != "" {
		args = append(args, "--class", strings.ToUpper(opts.CabinClass))
	}
	if opts.Sort {
		args = append(args, "--sort")
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 90*time.Second)
		defer cancel()
	}
	out, err := exec.CommandContext(ctx, fliPath, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("fli dates failed: %w", err)
	}
	var fliRaw struct {
		Success    bool        `json:"success"`
		DataSource string      `json:"data_source"`
		SearchType string      `json:"search_type"`
		Count      int         `json:"count"`
		Dates      []DatePrice `json:"dates"`
	}
	if err := json.Unmarshal(out, &fliRaw); err != nil {
		return nil, fmt.Errorf("parsing fli dates output: %w", err)
	}
	if !fliRaw.Success {
		return nil, fmt.Errorf("fli reported failure (check origin/destination IATA codes and date range)")
	}
	return &DatesResult{
		Success:    true,
		Source:     "fli-fallback",
		DataSource: fliRaw.DataSource,
		SearchType: fliRaw.SearchType,
		Query: SearchQuery{
			Origin:      opts.Origin,
			Destination: opts.Destination,
		},
		Count: fliRaw.Count,
		Dates: fliRaw.Dates,
	}, nil
}

// isRetryableErr returns true when a native-backend error is the kind that
// typically means "Google bot-blocked us, try the fli fallback." Other errors
// (bad IATA code, parse errors) should fail fast without calling fli.
func isRetryableErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, signal := range []string{"abuse", "captcha", "blocked", "429", "timeout", "401", "403"} {
		if strings.Contains(msg, signal) {
			return true
		}
	}
	// Also retry on raw-response parsing failures which are often a sign
	// Google returned an HTML challenge page instead of protobuf.
	for _, signal := range []string{"unmarshal", "invalid character", "unexpected eof"} {
		if strings.Contains(msg, signal) {
			return true
		}
	}
	return false
}
