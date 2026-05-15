package dataset

import (
	"math"
	"testing"
)

func TestBuildSecondaryBucketIndex(t *testing.T) {
	t.Parallel()

	store := &VectorStore{
		QuantizedVectors: []uint16{
			QuantizeComponent(0.10), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.20), QuantizeComponent(0.30), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.40), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.50), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0),
			QuantizeComponent(0.15), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.25), QuantizeComponent(0.35), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.45), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.55), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0),
		},
		Labels: []byte{LabelLegit, LabelFraud},
		Count:  2,
	}

	BuildSecondaryBucketIndex(store)

	if len(store.SecondaryBucketIndex) != SecondaryBucketIndexCount {
		t.Fatalf("secondary bucket index count = %d, want %d", len(store.SecondaryBucketIndex), SecondaryBucketIndexCount)
	}

	bucketID := secondaryBucketIDFromQuantized(store.QuantizedVectors, 0)
	if len(store.SecondaryBucketIndex[bucketID]) == 0 {
		t.Fatalf("expected populated secondary bucket")
	}
}

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

func TestBuildIVFIndex(t *testing.T) {
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
	BuildIVFIndex(store, 2)

	if len(store.IVFCentroids) != 2*IVFCoarseDimensions {
		t.Fatalf("ivf centroids len = %d, want %d", len(store.IVFCentroids), 2*IVFCoarseDimensions)
	}
	if len(store.IVFLists) != 2 {
		t.Fatalf("ivf lists len = %d, want 2", len(store.IVFLists))
	}

	totalBuckets := 0
	for _, list := range store.IVFLists {
		totalBuckets += len(list)
	}
	if totalBuckets == 0 {
		t.Fatalf("expected non-empty ivf lists")
	}
}

func TestBuildIVFIndexUsesBucketContentSummary(t *testing.T) {
	t.Parallel()

	store := &VectorStore{
		Vectors: []float32{
			0.10, 0, 0.20, 0.25, 0.30, 0, 0, 0, 0.40, 0, 0, 0, 0.80, 0, 0, 0,
			0.11, 0, 0.60, 0.26, 0.31, 0, 0, 0, 0.41, 0, 0, 0, 0.20, 0, 0, 0,
		},
		Labels: []byte{LabelLegit, LabelFraud},
		Count:  2,
	}
	BuildBucketIndex(store)
	BuildIVFIndex(store, 1)

	amountBucket, hourBucket, dayBucket, txBucket := BucketCoordinatesFromQuery(
		store.Vectors[0],
		store.Vectors[3],
		store.Vectors[4],
		store.Vectors[8],
	)
	bucketID := BucketIDFromCoordinates(amountBucket, hourBucket, dayBucket, txBucket)

	if len(store.IVFBucketSummaries) != BucketIndexCount*IVFCoarseDimensions {
		t.Fatalf("ivf bucket summaries len = %d, want %d", len(store.IVFBucketSummaries), BucketIndexCount*IVFCoarseDimensions)
	}

	baseOffset := bucketID * IVFCoarseDimensions
	expected := []float32{0.105, 0.255, 0.305, 0.405, 0.40, 0.50}
	for index, want := range expected {
		got := store.IVFBucketSummaries[baseOffset+index]
		if math.Abs(float64(got-want)) > 1e-6 {
			t.Fatalf("bucket summary dim %d = %f, want %f", index, got, want)
		}
	}
}


