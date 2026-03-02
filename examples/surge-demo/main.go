// Package main implements the Wolf Surge Demo — a ride-pricing API
// that demonstrates Wolf's database, ML bridge, and HTTP capabilities.
//
// This is the Go output that a Wolf program would compile to,
// showcasing all major compiler features working together.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"
)

// ========== Models ==========

// RideEstimate represents a ride pricing request.
type RideEstimate struct {
	PickupLat  float64 `json:"pickup_lat"`
	PickupLng  float64 `json:"pickup_lng"`
	DropoffLat float64 `json:"dropoff_lat"`
	DropoffLng float64 `json:"dropoff_lng"`
	RideType   string  `json:"ride_type"` // standard, premium, xl
}

// PriceResponse is the API response with pricing details.
type PriceResponse struct {
	BasePrice       float64 `json:"base_price"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	FinalPrice      float64 `json:"final_price"`
	Currency        string  `json:"currency"`
	DistanceKm      float64 `json:"distance_km"`
	EstimatedMins   int     `json:"estimated_mins"`
	RideType        string  `json:"ride_type"`
	Timestamp       string  `json:"timestamp"`
}

// HealthResponse is the API health check response.
type HealthResponse struct {
	Status   string `json:"status"`
	Version  string `json:"version"`
	Uptime   string `json:"uptime"`
	Language string `json:"language"`
}

// ========== Surge Engine ==========

// SurgeEngine calculates dynamic surge pricing.
// In a full Wolf program, this would use @ml blocks with scikit-learn.
type SurgeEngine struct {
	mu          sync.RWMutex
	demandCache map[string]float64
	baseRates   map[string]float64
}

// NewSurgeEngine creates a new surge pricing engine.
func NewSurgeEngine() *SurgeEngine {
	return &SurgeEngine{
		demandCache: make(map[string]float64),
		baseRates: map[string]float64{
			"standard": 1.50, // per km
			"premium":  2.50,
			"xl":       3.00,
		},
	}
}

// CalculatePrice computes the ride price with surge.
func (s *SurgeEngine) CalculatePrice(req RideEstimate) PriceResponse {
	distance := haversineDistance(
		req.PickupLat, req.PickupLng,
		req.DropoffLat, req.DropoffLng,
	)

	rideType := req.RideType
	if rideType == "" {
		rideType = "standard"
	}

	baseRate := s.baseRates[rideType]
	if baseRate == 0 {
		baseRate = s.baseRates["standard"]
	}

	basePrice := distance * baseRate
	if basePrice < 5.0 {
		basePrice = 5.0 // minimum fare
	}

	// Surge calculation (simulated ML model)
	surge := s.calculateSurge(req)

	finalPrice := basePrice * surge
	estimatedMins := int(distance * 2.5) // ~24km/h avg speed
	if estimatedMins < 3 {
		estimatedMins = 3
	}

	return PriceResponse{
		BasePrice:       math.Round(basePrice*100) / 100,
		SurgeMultiplier: math.Round(surge*100) / 100,
		FinalPrice:      math.Round(finalPrice*100) / 100,
		Currency:        "NGN",
		DistanceKm:      math.Round(distance*100) / 100,
		EstimatedMins:   estimatedMins,
		RideType:        rideType,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}
}

// calculateSurge simulates the ML surge model.
// In Wolf: @ml { model = load("surge_model"); prediction = model.predict(...) }
func (s *SurgeEngine) calculateSurge(req RideEstimate) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Simulated features: hour of day, distance, ride type
	hour := time.Now().Hour()

	// Peak hours: 7-9 AM, 5-8 PM
	var timeFactor float64
	switch {
	case hour >= 7 && hour <= 9:
		timeFactor = 1.3
	case hour >= 17 && hour <= 20:
		timeFactor = 1.5
	case hour >= 22 || hour <= 5:
		timeFactor = 1.2 // late night
	default:
		timeFactor = 1.0
	}

	// Ride type factor
	typeFactor := 1.0
	switch req.RideType {
	case "premium":
		typeFactor = 1.1
	case "xl":
		typeFactor = 1.15
	}

	surge := timeFactor * typeFactor
	if surge > 3.0 {
		surge = 3.0 // cap at 3x
	}

	return surge
}

// ========== HTTP Handlers ==========

var startTime = time.Now()

func handleEstimate(engine *SurgeEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RideEstimate
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Validate coordinates
		if req.PickupLat == 0 || req.PickupLng == 0 ||
			req.DropoffLat == 0 || req.DropoffLng == 0 {
			jsonError(w, "all coordinates are required", http.StatusBadRequest)
			return
		}

		price := engine.CalculatePrice(req)
		jsonResponse(w, price, http.StatusOK)
	}
}

func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uptime := time.Since(startTime).Round(time.Second)
		resp := HealthResponse{
			Status:   "healthy",
			Version:  "0.1.0-dev",
			Uptime:   uptime.String(),
			Language: "Wolf 🐺",
		}
		jsonResponse(w, resp, http.StatusOK)
	}
}

// ========== Helpers ==========

func jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Powered-By", "Wolf Language")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// haversineDistance calculates distance between two lat/lng points in km.
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371.0 // km

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

// ========== Main ==========

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	engine := NewSurgeEngine()

	mux := http.NewServeMux()
	mux.HandleFunc("/rides/estimate", handleEstimate(engine))
	mux.HandleFunc("/health", handleHealth())

	fmt.Printf("🐺 Wolf Surge Demo starting on :%s\n", port)
	fmt.Println("  POST /rides/estimate  — Get ride price estimate")
	fmt.Println("  GET  /health          — Health check")
	fmt.Println()

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
