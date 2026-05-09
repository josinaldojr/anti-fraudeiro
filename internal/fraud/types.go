package fraud

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

const topK = 5
const knownMerchantSetThreshold = 16

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
	parsed, err := parseTimestampJSON(data)
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
	if parsed, ok := parseTimestampRFC3339UTCString(value); ok {
		return parsed, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return Timestamp{}, fmt.Errorf("invalid RFC3339 timestamp %q: %w", value, err)
	}

	return Timestamp{unixNano: parsed.UTC().UnixNano()}, nil
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
	return buildKnownMerchantSetWithThreshold(knownMerchants, knownMerchantSetThreshold)
}

func buildKnownMerchantSetWithThreshold(knownMerchants []string, threshold int) map[string]struct{} {
	if len(knownMerchants) < threshold {
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

func parseTimestampJSON(data []byte) (Timestamp, error) {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return Timestamp{}, fmt.Errorf("timestamp must be an RFC3339 string: invalid JSON string")
	}

	raw := data[1 : len(data)-1]
	if bytes.IndexByte(raw, '\\') == -1 {
		if parsed, ok := parseTimestampRFC3339UTCBytes(raw); ok {
			return parsed, nil
		}
		return ParseTimestamp(string(raw))
	}

	value, err := strconv.Unquote(string(data))
	if err != nil {
		return Timestamp{}, fmt.Errorf("timestamp must be an RFC3339 string: %w", err)
	}

	return ParseTimestamp(value)
}

func parseTimestampRFC3339UTCString(value string) (Timestamp, bool) {
	if len(value) != len("2026-03-11T20:23:35Z") {
		return Timestamp{}, false
	}

	return parseTimestampRFC3339UTCBytes([]byte(value))
}

func parseTimestampRFC3339UTCBytes(value []byte) (Timestamp, bool) {
	if len(value) != len("2026-03-11T20:23:35Z") {
		return Timestamp{}, false
	}
	if value[4] != '-' || value[7] != '-' || value[10] != 'T' || value[13] != ':' || value[16] != ':' || value[19] != 'Z' {
		return Timestamp{}, false
	}

	year, ok := parse4Digits(value[0], value[1], value[2], value[3])
	if !ok {
		return Timestamp{}, false
	}
	month, ok := parse2Digits(value[5], value[6])
	if !ok {
		return Timestamp{}, false
	}
	day, ok := parse2Digits(value[8], value[9])
	if !ok {
		return Timestamp{}, false
	}
	hour, ok := parse2Digits(value[11], value[12])
	if !ok {
		return Timestamp{}, false
	}
	minute, ok := parse2Digits(value[14], value[15])
	if !ok {
		return Timestamp{}, false
	}
	second, ok := parse2Digits(value[17], value[18])
	if !ok {
		return Timestamp{}, false
	}
	if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59 || second > 59 {
		return Timestamp{}, false
	}

	parsed := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
	if parsed.Year() != year || parsed.Month() != time.Month(month) || parsed.Day() != day || parsed.Hour() != hour || parsed.Minute() != minute || parsed.Second() != second {
		return Timestamp{}, false
	}

	return Timestamp{unixNano: parsed.UnixNano()}, true
}

func parse2Digits(left byte, right byte) (int, bool) {
	if left < '0' || left > '9' || right < '0' || right > '9' {
		return 0, false
	}

	return int(left-'0')*10 + int(right-'0'), true
}

func parse4Digits(a byte, b byte, c byte, d byte) (int, bool) {
	ab, ok := parse2Digits(a, b)
	if !ok {
		return 0, false
	}
	cd, ok := parse2Digits(c, d)
	if !ok {
		return 0, false
	}

	return ab*100 + cd, true
}
