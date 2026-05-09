package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkDecodeFraudScoreRequest(b *testing.B) {
	b.ReportAllocs()

	body := validFraudScoreRequestJSON()

	for b.Loop() {
		request := httptest.NewRequest(http.MethodPost, "/fraud-score", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		if _, err := decodeFraudScoreRequest(recorder, request); err != nil {
			b.Fatalf("decodeFraudScoreRequest returned error: %v", err)
		}
	}
}

func BenchmarkHandlerFraudScore(b *testing.B) {
	b.ReportAllocs()

	handler := NewHandler(newTestScorer(), 0)
	body := validFraudScoreRequestJSON()

	for b.Loop() {
		request := httptest.NewRequest(http.MethodPost, "/fraud-score", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		handler.FraudScore(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
	}
}
