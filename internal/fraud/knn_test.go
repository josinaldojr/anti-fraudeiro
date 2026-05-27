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
	store.BucketIndex = nil

	query := [16]float32{0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15, 0.15}

	want := FindTop5(query, store)
	got, _, _ := findTop5WithStats(query, store, defaultSearchConfig)

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

func TestScanQuantizedContiguousSIMD(t *testing.T) {
	// Generate random vectors and labels
	count := 500
	vectors := make([]uint16, count*dataset.VectorSize)
	labels := make([]byte, count)
	for i := 0; i < count*dataset.VectorSize; i++ {
		vectors[i] = uint16(rand.Intn(65536))
	}
	for i := 0; i < count; i++ {
		if rand.Float32() < 0.2 {
			labels[i] = dataset.LabelFraud
		} else {
			labels[i] = dataset.LabelLegit
		}
	}

	// Generate a random query
	q := generateRandomVector()
	q0 := int32(dataset.QuantizeComponent(q[0]))
	q1 := int32(dataset.QuantizeComponent(q[1]))
	q2 := int32(dataset.QuantizeComponent(q[2]))
	q3 := int32(dataset.QuantizeComponent(q[3]))
	q4 := int32(dataset.QuantizeComponent(q[4]))
	q5 := int32(dataset.QuantizeComponent(q[5]))
	q6 := int32(dataset.QuantizeComponent(q[6]))
	q7 := int32(dataset.QuantizeComponent(q[7]))
	q8 := int32(dataset.QuantizeComponent(q[8]))
	q9 := int32(dataset.QuantizeComponent(q[9]))
	q10 := int32(dataset.QuantizeComponent(q[10]))
	q11 := int32(dataset.QuantizeComponent(q[11]))
	q12 := int32(dataset.QuantizeComponent(q[12]))
	q13 := int32(dataset.QuantizeComponent(q[13]))
	q14 := int32(dataset.QuantizeComponent(q[14]))
	q15 := int32(dataset.QuantizeComponent(q[15]))

	// Run generic scan
	var distsGeneric [topK]uint64
	var labelsGeneric [topK]byte
	for i := range distsGeneric {
		distsGeneric[i] = ^uint64(0)
	}
	countGen := scanQuantizedContiguous(
		vectors, labels,
		q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15,
		&distsGeneric, &labelsGeneric,
	)

	// Run SIMD scan
	var distsSIMD [topK]uint64
	var labelsSIMD [topK]byte
	for i := range distsSIMD {
		distsSIMD[i] = ^uint64(0)
	}
	countSIMD := scanQuantizedContiguousSIMD(
		vectors, labels,
		q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15,
		&distsSIMD, &labelsSIMD,
	)

	if countGen != countSIMD {
		t.Fatalf("count mismatch: generic=%d, simd=%d", countGen, countSIMD)
	}

	for i := 0; i < topK; i++ {
		if distsGeneric[i] != distsSIMD[i] {
			t.Errorf("distance mismatch at index %d: generic=%d, simd=%d", i, distsGeneric[i], distsSIMD[i])
		}
		if labelsGeneric[i] != labelsSIMD[i] {
			t.Errorf("label mismatch at index %d: generic=%d, simd=%d", i, labelsGeneric[i], labelsSIMD[i])
		}
	}
}


