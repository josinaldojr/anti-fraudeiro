package fraud

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	json "github.com/goccy/go-json"
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
	AvgAmount        float64  `json:"avg_amount"`
	TxCount24h       int      `json:"tx_count_24h"`
	KnownMerchants   []string `json:"known_merchants"`
	knownMerchantSet map[string]struct{}
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
	value, err := parseJSONString(data)
	if err != nil {
		return fmt.Errorf("timestamp must be an RFC3339 string: %w", err)
	}

	parsed, err := ParseTimestamp(value)
	if err != nil {
		return err
	}

	*t = parsed
	return nil
}

func (t Timestamp) Time() time.Time {
	return time.Unix(0, t.unixNano).UTC()
}

func (t Timestamp) UnixNano() int64 {
	return t.unixNano
}

func ParseTimestamp(value string) (Timestamp, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return Timestamp{}, fmt.Errorf("invalid RFC3339 timestamp %q: %w", value, err)
	}

	return Timestamp{unixNano: parsed.UTC().UnixNano()}, nil
}

func (c *Customer) UnmarshalJSON(data []byte) error {
	type customerAlias struct {
		AvgAmount      float64  `json:"avg_amount"`
		TxCount24h     int      `json:"tx_count_24h"`
		KnownMerchants []string `json:"known_merchants"`
	}

	var decoded customerAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	c.AvgAmount = decoded.AvgAmount
	c.TxCount24h = decoded.TxCount24h
	c.KnownMerchants = decoded.KnownMerchants
	c.knownMerchantSet = buildKnownMerchantSet(decoded.KnownMerchants)

	return nil
}

func (c *Customer) Finalize() {
	c.knownMerchantSet = buildKnownMerchantSet(c.KnownMerchants)
}

func (c Customer) HasKnownMerchant(merchantID string) bool {
	if len(c.knownMerchantSet) > 0 {
		_, found := c.knownMerchantSet[merchantID]
		return found
	}

	for _, knownMerchant := range c.KnownMerchants {
		if knownMerchant == merchantID {
			return true
		}
	}

	return false
}

func buildKnownMerchantSet(knownMerchants []string) map[string]struct{} {
	if len(knownMerchants) < 8 {
		return nil
	}

	knownMerchantSet := make(map[string]struct{}, len(knownMerchants))
	for _, knownMerchant := range knownMerchants {
		knownMerchantSet[knownMerchant] = struct{}{}
	}

	return knownMerchantSet
}

func parseJSONString(data []byte) (string, error) {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return "", fmt.Errorf("invalid JSON string")
	}

	raw := data[1 : len(data)-1]
	if bytes.IndexByte(raw, '\\') == -1 {
		return string(raw), nil
	}

	value, err := strconv.Unquote(string(data))
	if err != nil {
		return "", err
	}

	return value, nil
}
