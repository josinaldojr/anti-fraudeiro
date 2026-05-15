package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/josinaldojr/anti-fraudeiro/internal/config"
	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
	"github.com/josinaldojr/anti-fraudeiro/internal/fraud"
)

func TestFraudScoreRejectsTrailingJSON(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestScorer(), 0)
	body := validFraudScoreRequestJSON() + `{"extra":true}`
	request := httptest.NewRequest(http.MethodPost, "/fraud-score", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.FraudScore(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestFraudScoreRejectsPayloadTooLarge(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestScorer(), 0)
	body := validFraudScoreRequestJSON() + strings.Repeat(" ", int(maxFraudScoreRequestBodyBytes))
	request := httptest.NewRequest(http.MethodPost, "/fraud-score", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.FraudScore(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestFraudScoreAcceptsValidPayload(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestScorer(), 0)
	request := httptest.NewRequest(http.MethodPost, "/fraud-score", strings.NewReader(validFraudScoreRequestJSON()))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.FraudScore(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func newTestScorer() *fraud.Scorer {
	vectorizer := fraud.NewVectorizer(config.Normalization{
		MaxAmount:            10000,
		MaxInstallments:      12,
		AmountVsAvgRatio:     10,
		MaxMinutes:           1440,
		MaxKM:                1000,
		MaxTxCount24h:        20,
		MaxMerchantAvgAmount: 10000,
	}, map[string]float32{
		"5411": 0.15,
		"5912": 0.20,
	})

	store := &dataset.VectorStore{
		Vectors: repeatVector(5, [16]float32{0.03, 0.2, 0.05, 0.8, 0.0, -1, -1, 0.02, 0.15, 0, 1, 0, 0.2, 0.03}),
		Labels:  []byte{dataset.LabelLegit, dataset.LabelLegit, dataset.LabelLegit, dataset.LabelFraud, dataset.LabelLegit},
		Count:   5,
	}

	return fraud.NewScorer(vectorizer, store)
}

func repeatVector(count int, vector [16]float32) []float32 {
	values := make([]float32, 0, count*dataset.VectorSize)
	for i := 0; i < count; i++ {
		values = append(values, vector[:]...)
	}
	return values
}

func validFraudScoreRequestJSON() string {
	return `{
  "id": "tx-3576980410",
  "transaction": {
    "amount": 384.88,
    "installments": 3,
    "requested_at": "2026-03-11T20:23:35Z"
  },
  "customer": {
    "avg_amount": 769.76,
    "tx_count_24h": 3,
    "known_merchants": ["MERC-009", "MERC-001", "MERC-001"]
  },
  "merchant": {
    "id": "MERC-001",
    "mcc": "5912",
    "avg_amount": 298.95
  },
  "terminal": {
    "is_online": false,
    "card_present": true,
    "km_from_home": 13.7090520965
  },
  "last_transaction": {
    "timestamp": "2026-03-11T14:58:35Z",
    "km_from_current": 18.8626479774
  }
}`
}
