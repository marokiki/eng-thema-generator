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

func TestReviewEnglishFallsBackWithoutAPIKey(t *testing.T) {
	service := NewService()
	service.apiKey = ""

	advice := service.ReviewEnglish(context.Background(), "i want improve my english because very important")

	if advice.Summary == "" {
		t.Fatal("expected summary to be set")
	}
	if len(advice.Strengths) != 2 {
		t.Fatalf("expected 2 strengths, got %d", len(advice.Strengths))
	}
	if len(advice.Suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(advice.Suggestions))
	}
	if len(advice.Alternatives) != 2 {
		t.Fatalf("expected 2 alternatives, got %d", len(advice.Alternatives))
	}
	if advice.Polished == "" {
		t.Fatal("expected polished text to be set")
	}
}

func TestReviewEnglishEmptyInputGetsStarterAdvice(t *testing.T) {
	service := NewService()

	advice := service.ReviewEnglish(context.Background(), "   ")

	if advice.Polished == "" {
		t.Fatal("expected polished example for empty input")
	}
	if advice.Focus == "" {
		t.Fatal("expected focus message for empty input")
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
