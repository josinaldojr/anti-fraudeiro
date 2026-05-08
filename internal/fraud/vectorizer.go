package fraud

import (
	"fmt"
	"time"

	"github.com/josinaldojr/anti-fraudeiro/internal/config"
)

type Vectorizer struct {
	normalization config.Normalization
	mccRisk       map[string]float32
}

func NewVectorizer(normalization config.Normalization, mccRisk map[string]float32) *Vectorizer {
	return &Vectorizer{
		normalization: normalization,
		mccRisk:       mccRisk,
	}
}

func (v *Vectorizer) Vectorize(request FraudScoreRequest) ([14]float32, error) {
	var vector [14]float32

	requestedAt, err := time.Parse(time.RFC3339, request.Transaction.RequestedAt)
	if err != nil {
		return vector, fmt.Errorf("parse transaction.requested_at: %w", err)
	}
	requestedAt = requestedAt.UTC()

	vector[0] = clampNormalized(request.Transaction.Amount, v.normalization.MaxAmount)
	vector[1] = clampNormalized(float64(request.Transaction.Installments), v.normalization.MaxInstallments)

	customerAverage := request.Customer.AvgAmount
	if customerAverage <= 0 {
		customerAverage = 1.0
	}
	vector[2] = clamp((request.Transaction.Amount / customerAverage) / positiveOrOne(v.normalization.AmountVsAvgRatio))

	vector[3] = float32(requestedAt.Hour()) / 23.0
	vector[4] = float32(weekdayIndex(requestedAt)) / 6.0

	if request.LastTransaction == nil {
		vector[5] = -1
		vector[6] = -1
	} else {
		lastTimestamp, err := time.Parse(time.RFC3339, request.LastTransaction.Timestamp)
		if err != nil {
			return vector, fmt.Errorf("parse last_transaction.timestamp: %w", err)
		}

		minutesSinceLast := requestedAt.Sub(lastTimestamp.UTC()).Minutes()
		vector[5] = clamp(minutesSinceLast / positiveOrOne(v.normalization.MaxMinutes))
		vector[6] = clampNormalized(request.LastTransaction.KMFromCurrent, v.normalization.MaxKM)
	}

	vector[7] = clampNormalized(request.Terminal.KMFromHome, v.normalization.MaxKM)
	vector[8] = clampNormalized(float64(request.Customer.TxCount24h), v.normalization.MaxTxCount24h)
	vector[9] = boolToFloat32(request.Terminal.IsOnline)
	vector[10] = boolToFloat32(request.Terminal.CardPresent)
	vector[11] = boolToFloat32(isUnknownMerchant(request.Merchant.ID, request.Customer.KnownMerchants))
	vector[12] = v.lookupMCCRisk(request.Merchant.MCC)
	vector[13] = clampNormalized(request.Merchant.AvgAmount, v.normalization.MaxMerchantAvgAmount)

	return vector, nil
}

func clamp(value float64) float32 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return float32(value)
	}
}

func clampNormalized(value float64, maximum float64) float32 {
	return clamp(value / positiveOrOne(maximum))
}

func positiveOrOne(value float64) float64 {
	if value <= 0 {
		return 1
	}

	return value
}

func weekdayIndex(t time.Time) int {
	if t.Weekday() == time.Sunday {
		return 6
	}

	return int(t.Weekday()) - 1
}

func boolToFloat32(value bool) float32 {
	if value {
		return 1
	}

	return 0
}

func isUnknownMerchant(merchantID string, knownMerchants []string) bool {
	for _, knownMerchant := range knownMerchants {
		if knownMerchant == merchantID {
			return false
		}
	}

	return true
}

func (v *Vectorizer) lookupMCCRisk(mcc string) float32 {
	if risk, found := v.mccRisk[mcc]; found {
		return risk
	}

	return 0.5
}
