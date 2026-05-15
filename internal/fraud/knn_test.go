package fraud

import (
	"math/rand"
	"testing"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func TestFindTop5MatchesExactSearch(t *testing.T) {
	t.Parallel()

	store := &dataset.VectorStore{
		QuantizedVectors: []uint16{
			dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10), dataset.QuantizeComponent(0.10),
			dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20), dataset.QuantizeComponent(0.20),
			dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30), dataset.QuantizeComponent(0.30),
			dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40), dataset.QuantizeComponent(0.40),
			dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50), dataset.QuantizeComponent(0.50),
			dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60), dataset.QuantizeComponent(0.60),
		},
		Labels: []byte{dataset.LabelLegit, dataset.LabelFraud, dataset.LabelLegit, dataset.LabelFraud, dataset.LabelLegit, dataset.LabelFraud},
		Count:  6,
	}
	dataset.BuildBucketIndex(store)

	query := [16]float32{0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15}

	want := FindTop5(query, store)
	got, _ := findTop5WithStats(query, store, defaultSearchConfig)

	if got != want {
		t.Fatalf("FindTop5 = %d, want %d", got, want)
	}
}

func TestSelectBucketWindow(t *testing.T) {
	t.Parallel()

	store := &dataset.VectorStore{
		QuantizedVectors: make([]uint16, 100*dataset.VectorSize),
		Labels:           make([]byte, 100),
		Count:            100,
	}
	// Distribute vectors into a few buckets
	for i := 0; i < 100; i++ {
		v := generateRandomVector()
		for j := 0; j < dataset.VectorSize; j++ {
			store.QuantizedVectors[i*dataset.VectorSize+j] = dataset.QuantizeComponent(v[j])
		}
	}
	dataset.BuildBucketIndex(store)

	cfg := searchConfig{
		bucketTargetCandidates: 5,
		bucketMaxSearchRadius:  3,
	}

	q := generateRandomVector()
	amountB, hourB, dayB, txB := dataset.BucketCoordinatesFromQuery(q[0], q[3], q[4], q[8])

	_, _, _, _, _, _, _, _, candidateCount := selectBucketWindow(store.BucketIndex, store.BucketPrefixSums, amountB, hourB, dayB, txB, cfg)

	if candidateCount < 0 {
		t.Fatalf("invalid candidate count %d", candidateCount)
	}
}

func generateRandomStore(count int) *dataset.VectorStore {
	vectors := make([]uint16, count*dataset.VectorSize)
	labels := make([]byte, count)

	for i := 0; i < count; i++ {
		v := generateRandomVector()
		for j := 0; j < dataset.VectorSize; j++ {
			vectors[i*dataset.VectorSize+j] = dataset.QuantizeComponent(v[j])
		}
		if rand.Float32() < 0.2 {
			labels[i] = dataset.LabelFraud
		} else {
			labels[i] = dataset.LabelLegit
		}
	}

	return &dataset.VectorStore{
		QuantizedVectors: vectors,
		Labels:           labels,
		Count:            count,
	}
}

func generateRandomVector() [16]float32 {
	var v [16]float32
	for i := 0; i < 16; i++ {
		v[i] = rand.Float32()
	}
	return v
}
