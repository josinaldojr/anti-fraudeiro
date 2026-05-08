package fraud

import (
	"encoding/json"
	"fmt"
	"time"
)

const topK = 5

type FraudScoreRequest struct {
	ID              string           `json:"id"`
	Transaction     Transaction      `json:"transaction"`
	Customer        Customer         `json:"customer"`
	Merchant        Merchant         `json:"merchant"`
	Terminal        Terminal         `json:"terminal"`
	LastTransaction *LastTransaction `json:"last_transaction"`
}

type Transaction struct {
	Amount       float64   `json:"amount"`
	Installments int       `json:"installments"`
	RequestedAt  Timestamp `json:"requested_at"`
}

type Customer struct {
	AvgAmount      float64  `json:"avg_amount"`
	TxCount24h     int      `json:"tx_count_24h"`
	KnownMerchants []string `json:"known_merchants"`
}

type Merchant struct {
	ID        string  `json:"id"`
	MCC       string  `json:"mcc"`
	AvgAmount float64 `json:"avg_amount"`
}

type Terminal struct {
	IsOnline    bool    `json:"is_online"`
	CardPresent bool    `json:"card_present"`
	KMFromHome  float64 `json:"km_from_home"`
}

type LastTransaction struct {
	Timestamp     Timestamp `json:"timestamp"`
	KMFromCurrent float64   `json:"km_from_current"`
}

type Decision struct {
	Approved   bool
	FraudScore float64
}

type Timestamp struct {
	unixNano int64
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("timestamp must be an RFC3339 string: %w", err)
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return fmt.Errorf("invalid RFC3339 timestamp %q: %w", value, err)
	}

	t.unixNano = parsed.UTC().UnixNano()
	return nil
}

func (t Timestamp) Time() time.Time {
	return time.Unix(0, t.unixNano).UTC()
}

func (t Timestamp) UnixNano() int64 {
	return t.unixNano
}
