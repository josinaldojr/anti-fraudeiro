package dataset

import "testing"

func TestBuildSecondaryBucketIndex(t *testing.T) {
	t.Parallel()

	store := &VectorStore{
		QuantizedVectors: []uint16{
			QuantizeComponent(0.10), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.20), QuantizeComponent(0.30), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.40), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.50), QuantizeComponent(0),
			QuantizeComponent(0.15), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.25), QuantizeComponent(0.35), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.45), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0), QuantizeComponent(0.55), QuantizeComponent(0),
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
