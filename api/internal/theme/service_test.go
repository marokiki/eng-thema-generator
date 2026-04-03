package theme

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestPickFallbackPreservesRequestedCategoryAndEnergy(t *testing.T) {
	service := NewService()
	service.apiKey = ""

	prompt := service.Pick(context.Background(), "food-and-home", "gentle", "random", time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC))

	if prompt.Category != "food-and-home" {
		t.Fatalf("expected category food-and-home, got %q", prompt.Category)
	}
	if prompt.Energy != "gentle" {
		t.Fatalf("expected energy gentle, got %q", prompt.Energy)
	}
}

func TestGenerateUsesRequestContext(t *testing.T) {
	service := NewService()
	service.apiKey = "test-key"
	service.baseURL = "https://example.invalid"
	service.model = "test-model"
	service.client.Timeout = 5 * time.Second
	service.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.generate(ctx, time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC), "relationships", "playful", "random")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
