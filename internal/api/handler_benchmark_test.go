package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkDecodeFraudScoreRequest(b *testing.B) {
	body := validFraudScoreRequestJSON()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		request := httptest.NewRequest(http.MethodPost, "/fraud-score", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		b.StartTimer()

		req, err := decodeFraudScoreRequest(recorder, request)
		if err != nil {
			b.Fatalf("decodeFraudScoreRequest returned error: %v", err)
		}
		releaseFraudScoreRequest(req)
	}
}

func BenchmarkFraudScoreEndToEnd(b *testing.B) {
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
