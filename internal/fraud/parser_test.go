package fraud

import (
	"testing"
)

func TestParseFraudScoreRequestStringFieldsSurviveSourceOverwrite(t *testing.T) {
	data := []byte(`{
  "id": "tx-3576980410",
  "transaction": {
    "amount": 384.88,
    "installments": 3,
    "requested_at": "2026-03-11T20:23:35Z"
  },
  "customer": {
    "avg_amount": 769.76,
    "tx_count_24h": 3,
    "known_merchants": ["MERC-009", "MERC-001", "MERC-002"]
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
}`)

	request := &FraudScoreRequest{
		Customer: Customer{
			KnownMerchants: make([]string, 0, 16),
		},
		LastTransaction: &LastTransaction{},
	}

	if err := ParseFraudScoreRequest(data, request); err != nil {
		t.Fatalf("ParseFraudScoreRequest returned error: %v", err)
	}
	request.Customer.Finalize()

	for i := range data {
		data[i] = 'X'
	}

	if request.ID != "tx-3576980410" {
		t.Fatalf("ID = %q, want %q", request.ID, "tx-3576980410")
	}
	if request.Merchant.ID != "MERC-001" {
		t.Fatalf("Merchant.ID = %q, want %q", request.Merchant.ID, "MERC-001")
	}
	if request.Merchant.MCC != "5912" {
		t.Fatalf("Merchant.MCC = %q, want %q", request.Merchant.MCC, "5912")
	}
	if !request.Customer.HasKnownMerchant("MERC-001") {
		t.Fatalf("expected known merchant lookup to use original values")
	}

	vectorizer := newTestVectorizer()
	vector, err := vectorizer.Vectorize(*request)
	if err != nil {
		t.Fatalf("Vectorize returned error: %v", err)
	}
	if vector[11] != 0 {
		t.Fatalf("vector[11] = %v, want 0 for known merchant", vector[11])
	}
	if vector[12] != 0.20 {
		t.Fatalf("vector[12] = %v, want 0.20 for MCC risk", vector[12])
	}
}

func BenchmarkParseFraudScoreRequest(b *testing.B) {
	data := []byte(`{
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
}`)

	request := &FraudScoreRequest{
		Customer: Customer{
			KnownMerchants: make([]string, 0, 16),
		},
		LastTransaction: &LastTransaction{},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := ParseFraudScoreRequest(data, request); err != nil {
			b.Fatal(err)
		}
	}
}
