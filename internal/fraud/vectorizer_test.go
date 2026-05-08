package fraud

import (
	"testing"

	"github.com/josinaldojr/anti-fraudeiro/internal/config"
)

func TestClamp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    float64
		expected float32
	}{
		{name: "below_zero", input: -1, expected: 0},
		{name: "within_range", input: 0.42, expected: 0.42},
		{name: "above_one", input: 3, expected: 1},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if actual := clamp(testCase.input); actual != testCase.expected {
				t.Fatalf("clamp(%v) = %v, want %v", testCase.input, actual, testCase.expected)
			}
		})
	}
}

func TestWeekdayIndex(t *testing.T) {
	t.Parallel()

	monday := mustParseRFC3339(t, "2026-03-09T10:00:00Z")
	sunday := mustParseRFC3339(t, "2026-03-15T10:00:00Z")

	if actual := weekdayIndex(monday); actual != 0 {
		t.Fatalf("weekdayIndex(monday) = %d, want 0", actual)
	}

	if actual := weekdayIndex(sunday); actual != 6 {
		t.Fatalf("weekdayIndex(sunday) = %d, want 6", actual)
	}
}

func TestVectorizeUsesSentinelWhenLastTransactionIsNil(t *testing.T) {
	t.Parallel()

	vectorizer := newTestVectorizer()

	vector, err := vectorizer.Vectorize(sampleRequest())
	if err != nil {
		t.Fatalf("Vectorize returned error: %v", err)
	}

	if vector[5] != -1 {
		t.Fatalf("vector[5] = %v, want -1", vector[5])
	}

	if vector[6] != -1 {
		t.Fatalf("vector[6] = %v, want -1", vector[6])
	}
}

func TestVectorizeUnknownMerchant(t *testing.T) {
	t.Parallel()

	vectorizer := newTestVectorizer()
	request := sampleRequest()
	request.Customer.KnownMerchants = []string{"MERC-001"}

	vector, err := vectorizer.Vectorize(request)
	if err != nil {
		t.Fatalf("Vectorize returned error: %v", err)
	}

	if vector[11] != 1 {
		t.Fatalf("vector[11] = %v, want 1", vector[11])
	}

	request.Customer.KnownMerchants = []string{"MERC-009", "MERC-123"}
	vector, err = vectorizer.Vectorize(request)
	if err != nil {
		t.Fatalf("Vectorize returned error: %v", err)
	}

	if vector[11] != 0 {
		t.Fatalf("vector[11] = %v, want 0", vector[11])
	}
}

func TestDecisionThreshold(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		fraudCount    int
		expectedScore float64
		expectedPass  bool
	}{
		{name: "score_0_4", fraudCount: 2, expectedScore: 0.4, expectedPass: true},
		{name: "score_0_6", fraudCount: 3, expectedScore: 0.6, expectedPass: false},
		{name: "score_1_0", fraudCount: 5, expectedScore: 1.0, expectedPass: false},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			decision := decisionFromFraudCount(testCase.fraudCount)

			if decision.FraudScore != testCase.expectedScore {
				t.Fatalf("FraudScore = %v, want %v", decision.FraudScore, testCase.expectedScore)
			}

			if decision.Approved != testCase.expectedPass {
				t.Fatalf("Approved = %v, want %v", decision.Approved, testCase.expectedPass)
			}
		})
	}
}

func newTestVectorizer() *Vectorizer {
	return NewVectorizer(config.Normalization{
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
}

func sampleRequest() FraudScoreRequest {
	return FraudScoreRequest{
		ID: "tx-1329056812",
		Transaction: Transaction{
			Amount:       41.12,
			Installments: 2,
			RequestedAt:  "2026-03-09T18:45:53Z",
		},
		Customer: Customer{
			AvgAmount:      82.24,
			TxCount24h:     3,
			KnownMerchants: []string{"MERC-003", "MERC-016"},
		},
		Merchant: Merchant{
			ID:        "MERC-009",
			MCC:       "5411",
			AvgAmount: 60.25,
		},
		Terminal: Terminal{
			IsOnline:    false,
			CardPresent: true,
			KMFromHome:  29.23,
		},
		LastTransaction: nil,
	}
}
