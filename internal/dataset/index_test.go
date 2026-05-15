package dataset

import (
	"testing"
)

func TestCountBucketWindowCandidatesMatchesBucketLengths(t *testing.T) {
	t.Parallel()

	store := &VectorStore{
		QuantizedVectors: []uint16{
			QuantizeComponent(0.10), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.20), QuantizeComponent(0.30), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.40), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.50), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0),
			QuantizeComponent(0.11), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.21), QuantizeComponent(0.31), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.41), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.51), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0),
			QuantizeComponent(0.75), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.80), QuantizeComponent(0.20), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.60), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.10), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0),
		},
		Labels: []byte{LabelLegit, LabelFraud, LabelLegit},
		Count:  3,
	}
	BuildBucketIndex(store)

	amountStart, amountEnd := 0, 5
	hourStart, hourEnd := 0, 8
	dayStart, dayEnd := 0, 4
	txStart, txEnd := 0, 10

	expected := 0
	for amountIndex := amountStart; amountIndex <= amountEnd; amountIndex++ {
		for hourIndex := hourStart; hourIndex <= hourEnd; hourIndex++ {
			for dayIndex := dayStart; dayIndex <= dayEnd; dayIndex++ {
				for txIndex := txStart; txIndex <= txEnd; txIndex++ {
					expected += len(store.BucketIndex[BucketIDFromCoordinates(amountIndex, hourIndex, dayIndex, txIndex)])
				}
			}
		}
	}

	actual := CountBucketWindowCandidates(
		store.BucketPrefixSums,
		amountStart,
		amountEnd,
		hourStart,
		hourEnd,
		dayStart,
		dayEnd,
		txStart,
		txEnd,
	)

	if actual != expected {
		t.Fatalf("CountBucketWindowCandidates = %d, want %d", actual, expected)
	}
}
