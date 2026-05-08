package fraud

import (
	"time"

	"github.com/josinaldojr/anti-fraudeiro/internal/config"
)

type Vectorizer struct {
	normalization           config.Normalization
	mccRisk                 map[string]float32
	maxAmountInv            float64
	maxInstallmentsInv      float64
	amountVsAvgRatioInv     float64
	maxMinutesInv           float64
	maxKMInv                float64
	maxTxCount24hInv        float64
	maxMerchantAvgAmountInv float64
}

func NewVectorizer(normalization config.Normalization, mccRisk map[string]float32) *Vectorizer {
	return &Vectorizer{
		normalization:           normalization,
		mccRisk:                 mccRisk,
		maxAmountInv:            inverseOrOne(normalization.MaxAmount),
		maxInstallmentsInv:      inverseOrOne(normalization.MaxInstallments),
		amountVsAvgRatioInv:     inverseOrOne(normalization.AmountVsAvgRatio),
		maxMinutesInv:           inverseOrOne(normalization.MaxMinutes),
		maxKMInv:                inverseOrOne(normalization.MaxKM),
		maxTxCount24hInv:        inverseOrOne(normalization.MaxTxCount24h),
		maxMerchantAvgAmountInv: inverseOrOne(normalization.MaxMerchantAvgAmount),
	}
}

func (v *Vectorizer) Vectorize(request FraudScoreRequest) ([14]float32, error) {
	var vector [14]float32

	requestedAt := request.Transaction.RequestedAt.Time()

	vector[0] = clamp(request.Transaction.Amount * v.maxAmountInv)
	vector[1] = clamp(float64(request.Transaction.Installments) * v.maxInstallmentsInv)

	customerAverage := request.Customer.AvgAmount
	if customerAverage <= 0 {
		customerAverage = 1.0
	}
	vector[2] = clamp((request.Transaction.Amount / customerAverage) * v.amountVsAvgRatioInv)

	vector[3] = float32(requestedAt.Hour()) / 23.0
	vector[4] = float32(weekdayIndex(requestedAt)) / 6.0

	if request.LastTransaction == nil {
		vector[5] = -1
		vector[6] = -1
	} else {
		minutesSinceLast := float64(request.Transaction.RequestedAt.UnixNano()-request.LastTransaction.Timestamp.UnixNano()) / float64(time.Minute)
		vector[5] = clamp(minutesSinceLast * v.maxMinutesInv)
		vector[6] = clamp(request.LastTransaction.KMFromCurrent * v.maxKMInv)
	}

	vector[7] = clamp(request.Terminal.KMFromHome * v.maxKMInv)
	vector[8] = clamp(float64(request.Customer.TxCount24h) * v.maxTxCount24hInv)
	vector[9] = boolToFloat32(request.Terminal.IsOnline)
	vector[10] = boolToFloat32(request.Terminal.CardPresent)
	vector[11] = boolToFloat32(isUnknownMerchant(request.Merchant.ID, request.Customer.KnownMerchants))
	vector[12] = v.lookupMCCRisk(request.Merchant.MCC)
	vector[13] = clamp(request.Merchant.AvgAmount * v.maxMerchantAvgAmountInv)

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

func inverseOrOne(value float64) float64 {
	if value <= 0 {
		return 1
	}

	return 1 / value
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
