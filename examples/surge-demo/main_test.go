package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// ========== Surge Engine Tests ==========

func TestNewSurgeEngine(t *testing.T) {
	e := NewSurgeEngine()
	if e == nil {
		t.Fatal("Expected non-nil engine")
	}
	if len(e.baseRates) != 3 {
		t.Errorf("Expected 3 base rates, got %d", len(e.baseRates))
	}
}

func TestCalculatePrice_Standard(t *testing.T) {
	e := NewSurgeEngine()
	req := RideEstimate{
		PickupLat: 6.5244, PickupLng: 3.3792, // Lagos
		DropoffLat: 6.4541, DropoffLng: 3.3947, // Victoria Island
		RideType: "standard",
	}
	price := e.CalculatePrice(req)

	if price.DistanceKm <= 0 {
		t.Error("Distance should be positive")
	}
	if price.BasePrice < 5.0 {
		t.Error("Base price should be at least minimum fare")
	}
	if price.SurgeMultiplier < 1.0 {
		t.Error("Surge should be >= 1.0")
	}
	if price.FinalPrice < price.BasePrice {
		t.Error("Final price should be >= base price")
	}
	if price.Currency != "NGN" {
		t.Errorf("Expected NGN currency, got %s", price.Currency)
	}
	if price.RideType != "standard" {
		t.Errorf("Expected standard, got %s", price.RideType)
	}
	if price.EstimatedMins < 3 {
		t.Error("Estimated minutes should be >= 3")
	}
	if price.Timestamp == "" {
		t.Error("Expected timestamp")
	}
}

func TestCalculatePrice_Premium(t *testing.T) {
	e := NewSurgeEngine()
	standard := e.CalculatePrice(RideEstimate{
		PickupLat: 6.5, PickupLng: 3.3,
		DropoffLat: 6.6, DropoffLng: 3.4,
		RideType: "standard",
	})
	premium := e.CalculatePrice(RideEstimate{
		PickupLat: 6.5, PickupLng: 3.3,
		DropoffLat: 6.6, DropoffLng: 3.4,
		RideType: "premium",
	})

	if premium.BasePrice <= standard.BasePrice {
		t.Error("Premium should cost more than standard")
	}
}

func TestCalculatePrice_XL(t *testing.T) {
	e := NewSurgeEngine()
	req := RideEstimate{
		PickupLat: 6.5, PickupLng: 3.3,
		DropoffLat: 6.6, DropoffLng: 3.4,
		RideType: "xl",
	}
	price := e.CalculatePrice(req)
	if price.RideType != "xl" {
		t.Errorf("Expected xl, got %s", price.RideType)
	}
}

func TestCalculatePrice_DefaultRideType(t *testing.T) {
	e := NewSurgeEngine()
	req := RideEstimate{
		PickupLat: 6.5, PickupLng: 3.3,
		DropoffLat: 6.6, DropoffLng: 3.4,
	}
	price := e.CalculatePrice(req)
	if price.RideType != "standard" {
		t.Errorf("Expected default standard, got %s", price.RideType)
	}
}

func TestCalculatePrice_MinimumFare(t *testing.T) {
	e := NewSurgeEngine()
	// Very short distance should hit minimum fare
	req := RideEstimate{
		PickupLat: 6.5000, PickupLng: 3.3000,
		DropoffLat: 6.5001, DropoffLng: 3.3001,
		RideType: "standard",
	}
	price := e.CalculatePrice(req)
	if price.BasePrice < 5.0 {
		t.Errorf("Expected minimum fare 5.0, got %f", price.BasePrice)
	}
}

func TestSurgeMultiplierCap(t *testing.T) {
	e := NewSurgeEngine()
	req := RideEstimate{
		PickupLat: 6.5, PickupLng: 3.3,
		DropoffLat: 6.6, DropoffLng: 3.4,
		RideType: "standard",
	}
	price := e.CalculatePrice(req)
	if price.SurgeMultiplier > 3.0 {
		t.Errorf("Surge should be capped at 3.0, got %f", price.SurgeMultiplier)
	}
}

// ========== Haversine Tests ==========

func TestHaversineDistance_Zero(t *testing.T) {
	d := haversineDistance(6.5, 3.3, 6.5, 3.3)
	if d != 0.0 {
		t.Errorf("Same point should have 0 distance, got %f", d)
	}
}

func TestHaversineDistance_Positive(t *testing.T) {
	// Lagos to Victoria Island (~8km)
	d := haversineDistance(6.5244, 3.3792, 6.4541, 3.3947)
	if d < 5.0 || d > 15.0 {
		t.Errorf("Expected ~8km, got %f", d)
	}
}

func TestHaversineDistance_Symmetric(t *testing.T) {
	d1 := haversineDistance(6.5, 3.3, 6.6, 3.4)
	d2 := haversineDistance(6.6, 3.4, 6.5, 3.3)
	diff := math.Abs(d1 - d2)
	if diff > 0.001 {
		t.Errorf("Distance should be symmetric: %f vs %f", d1, d2)
	}
}

// ========== HTTP Handler Tests ==========

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handleHealth()(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp HealthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "healthy" {
		t.Errorf("Expected 'healthy', got '%s'", resp.Status)
	}
	if resp.Language != "Wolf 🐺" {
		t.Errorf("Expected 'Wolf 🐺', got '%s'", resp.Language)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected JSON content type")
	}
	if w.Header().Get("X-Powered-By") != "Wolf Language" {
		t.Error("Expected X-Powered-By header")
	}
}

func TestEstimateEndpoint_Success(t *testing.T) {
	engine := NewSurgeEngine()
	body := `{
		"pickup_lat": 6.5244,
		"pickup_lng": 3.3792,
		"dropoff_lat": 6.4541,
		"dropoff_lng": 3.3947,
		"ride_type": "standard"
	}`

	req := httptest.NewRequest("POST", "/rides/estimate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleEstimate(engine)(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp PriceResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.FinalPrice <= 0 {
		t.Error("Expected positive final price")
	}
	if resp.DistanceKm <= 0 {
		t.Error("Expected positive distance")
	}
}

func TestEstimateEndpoint_InvalidMethod(t *testing.T) {
	engine := NewSurgeEngine()
	req := httptest.NewRequest("GET", "/rides/estimate", nil)
	w := httptest.NewRecorder()

	handleEstimate(engine)(w, req)

	if w.Code != 405 {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

func TestEstimateEndpoint_InvalidBody(t *testing.T) {
	engine := NewSurgeEngine()
	req := httptest.NewRequest("POST", "/rides/estimate",
		strings.NewReader("not json"))
	w := httptest.NewRecorder()

	handleEstimate(engine)(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestEstimateEndpoint_MissingCoords(t *testing.T) {
	engine := NewSurgeEngine()
	body := `{"ride_type": "standard"}`
	req := httptest.NewRequest("POST", "/rides/estimate",
		strings.NewReader(body))
	w := httptest.NewRecorder()

	handleEstimate(engine)(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestEstimateEndpoint_PremiumPricing(t *testing.T) {
	engine := NewSurgeEngine()
	body := `{
		"pickup_lat": 6.5244, "pickup_lng": 3.3792,
		"dropoff_lat": 6.4541, "dropoff_lng": 3.3947,
		"ride_type": "premium"
	}`
	req := httptest.NewRequest("POST", "/rides/estimate",
		strings.NewReader(body))
	w := httptest.NewRecorder()

	handleEstimate(engine)(w, req)

	var resp PriceResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.RideType != "premium" {
		t.Errorf("Expected premium, got %s", resp.RideType)
	}
}

// ========== Concurrent Request Test ==========

func TestConcurrentRequests(t *testing.T) {
	engine := NewSurgeEngine()
	handler := handleEstimate(engine)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := `{
				"pickup_lat": 6.5244, "pickup_lng": 3.3792,
				"dropoff_lat": 6.4541, "dropoff_lng": 3.3947,
				"ride_type": "standard"
			}`
			req := httptest.NewRequest("POST", "/rides/estimate",
				strings.NewReader(body))
			w := httptest.NewRecorder()
			handler(w, req)
			if w.Code != 200 {
				errors <- fmt.Errorf("got status %d", w.Code)
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}
}


