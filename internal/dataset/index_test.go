package dataset

import (
	"testing"
)

func TestCountBucketWindowCandidatesMatchesBucketLengths(t *testing.T) {
	t.Parallel()

	// Create vectors with known positions for the 6D cell index
	// dim 0: amount
	// dim 5: minutes_since_last
	// dim 7: km_from_home
	// dim 8: tx_count_24h
	// dim 2: amount_vs_avg
	// dim 12: mcc_risk
	store := &VectorStore{
		QuantizedVectors: makeVector(
			QuantizeComponent(0.10), 0, 0, 0, 0, QuantizeComponent(0.20), 0, QuantizeComponent(0.30), 0, 0, 0, 0, QuantizeComponent(0.40), 0, 0, 0,
			QuantizeComponent(0.11), 0, QuantizeComponent(0.21), 0, 0, QuantizeComponent(0.31), 0, QuantizeComponent(0.41), 0, 0, 0, 0, QuantizeComponent(0.51), 0, 0, 0,
			QuantizeComponent(0.75), 0, QuantizeComponent(0.80), 0, 0, QuantizeComponent(0.20), 0, QuantizeComponent(0.60), 0, 0, 0, 0, QuantizeComponent(0.10), 0, 0, 0,
		),
		Labels: []byte{LabelLegit, LabelFraud, LabelLegit},
		Count:  3,
	}
	BuildBucketIndex(store)

	aStart, aEnd := 0, 3
	mStart, mEnd := 0, 3
	kStart, kEnd := 0, 3
	tStart, tEnd := 0, 3
	vStart, vEnd := 0, 3
	rStart, rEnd := 0, 3

	expected := 0
	for ai := aStart; ai <= aEnd; ai++ {
		for mi := mStart; mi <= mEnd; mi++ {
			for ki := kStart; ki <= kEnd; ki++ {
				for ti := tStart; ti <= tEnd; ti++ {
					for vi := vStart; vi <= vEnd; vi++ {
						for ri := rStart; ri <= rEnd; ri++ {
							bucketID := BucketIDFromCoordinates(ai, mi, ki, ti, vi, ri)
							expected += len(store.BucketIndex[bucketID])
						}
					}
				}
			}
		}
	}

	actual := CountBucketWindowCandidates(
		store.BucketPrefixSums,
		aStart, aEnd, mStart, mEnd, kStart, kEnd, tStart, tEnd, vStart, vEnd, rStart, rEnd,
	)

	if actual != expected {
		t.Fatalf("CountBucketWindowCandidates = %d, want %d", actual, expected)
	}
}

func makeVector(values ...uint16) []uint16 {
	v := make([]uint16, len(values))
	copy(v, values)
	return v
}
