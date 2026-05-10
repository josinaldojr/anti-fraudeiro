package fraud

import (
	"strconv"
	"testing"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func TestFindTop5(t *testing.T) {
	t.Parallel()

	query := vectorWithFirstDimension(0)
	store := &dataset.VectorStore{
		Vectors: flattenVectors(
			vectorWithFirstDimension(0.00),
			vectorWithFirstDimension(0.005),
			vectorWithFirstDimension(0.010),
			vectorWithFirstDimension(0.015),
			vectorWithFirstDimension(0.020),
			vectorWithFirstDimension(0.90),
		),
		Labels: []byte{
			dataset.LabelLegit,
			dataset.LabelFraud,
			dataset.LabelFraud,
			dataset.LabelLegit,
			dataset.LabelFraud,
			dataset.LabelLegit,
		},
		Count: 6,
	}
	dataset.BuildBucketIndex(store)

	if fraudCount := FindTop5(query, store); fraudCount != 3 {
		t.Fatalf("FindTop5 fraud count = %d, want 3", fraudCount)
	}

	if fraudCount := FindTop5WithStrategy(query, store, BucketStrategyOrdered); fraudCount != 3 {
		t.Fatalf("FindTop5WithStrategy ordered fraud count = %d, want 3", fraudCount)
	}
	dataset.BuildIVFIndex(store, 2)
	if fraudCount := FindTop5WithStrategy(query, store, BucketStrategyIVF); fraudCount != 3 {
		t.Fatalf("FindTop5WithStrategy ivf fraud count = %d, want 3", fraudCount)
	}

	if fraudCount := FindTop5Exact(query, store); fraudCount != 3 {
		t.Fatalf("FindTop5Exact fraud count = %d, want 3", fraudCount)
	}
}

func BenchmarkFindTop5(b *testing.B) {
	b.ReportAllocs()

	query := sampleRequestVector()
	store := &dataset.VectorStore{
		QuantizedVectors: benchmarkQuantizedVectors(1024),
		Labels:           benchmarkLabels(1024),
		Count:            1024,
	}
	dataset.BuildBucketIndex(store)
	dataset.BuildIVFIndex(store, 32)

	for b.Loop() {
		_ = FindTop5(query, store)
	}
}

func BenchmarkFindTop5BucketConfigs(b *testing.B) {
	b.ReportAllocs()

	query := sampleRequestVector()
	store := &dataset.VectorStore{
		QuantizedVectors: benchmarkQuantizedVectors(1024),
		Labels:           benchmarkLabels(1024),
		Count:            1024,
	}
	dataset.BuildBucketIndex(store)
	dataset.BuildIVFIndex(store, 32)

	configurations := []searchConfig{
		{bucketTargetCandidates: 64, bucketMaxSearchRadius: 2},
		{bucketTargetCandidates: 128, bucketMaxSearchRadius: 2},
		{bucketTargetCandidates: 256, bucketMaxSearchRadius: 2},
		{bucketTargetCandidates: 128, bucketMaxSearchRadius: 3},
		{bucketTargetCandidates: 256, bucketMaxSearchRadius: 3},
	}

	for _, cfg := range configurations {
		cfg := cfg
		b.Run(
			"target_"+strconv.Itoa(cfg.bucketTargetCandidates)+"_radius_"+strconv.Itoa(cfg.bucketMaxSearchRadius),
			func(b *testing.B) {
				for b.Loop() {
					_ = findTop5WithConfig(query, store, BucketStrategyWindow, cfg)
				}
			},
		)
	}
}

func BenchmarkFindTop5StrategyMatrix(b *testing.B) {
	b.ReportAllocs()

	query := sampleRequestVector()
	store := &dataset.VectorStore{
		QuantizedVectors: benchmarkQuantizedVectors(1024),
		Labels:           benchmarkLabels(1024),
		Count:            1024,
	}
	dataset.BuildBucketIndex(store)

	strategies := []BucketStrategy{
		BucketStrategyWindow,
		BucketStrategyOrdered,
		BucketStrategyShortlist,
		BucketStrategyIVF,
	}
	targets := []int{64, 96, 128, 192, 256}

	for _, strategy := range strategies {
		for _, target := range targets {
			strategy := strategy
			target := target

			b.Run(string(strategy)+"_target_"+strconv.Itoa(target), func(b *testing.B) {
				cfg := searchConfig{
					bucketTargetCandidates: target,
					bucketMaxSearchRadius:  3,
					ivfNProbe:              4,
				}
				totalProcessed := 0
				maxProcessed := 0

				for b.Loop() {
					_, stats := findTop5WithStats(query, store, strategy, cfg)
					totalProcessed += stats.processedCandidates
					if stats.processedCandidates > maxProcessed {
						maxProcessed = stats.processedCandidates
					}
				}

				b.ReportMetric(float64(totalProcessed)/float64(b.N), "avg_candidates/op")
				b.ReportMetric(float64(maxProcessed), "max_candidates")
			})
		}
	}
}

func BenchmarkSelectBucketWindow(b *testing.B) {
	b.ReportAllocs()

	store := &dataset.VectorStore{
		QuantizedVectors: benchmarkQuantizedVectors(1024),
		Labels:           benchmarkLabels(1024),
		Count:            1024,
	}
	dataset.BuildBucketIndex(store)
	dataset.BuildIVFIndex(store, 32)

	cfg := searchConfig{bucketTargetCandidates: 256, bucketMaxSearchRadius: 3}

	for b.Loop() {
		_, _, _, _, _, _, _, _, _ = selectBucketWindow(
			store.BucketIndex,
			store.BucketPrefixSums,
			16,
			12,
			3,
			8,
			cfg,
		)
	}
}

func TestSelectBucketWindowUsesTargetCandidates(t *testing.T) {
	t.Parallel()

	buckets := make([][]uint32, dataset.BucketIndexCount)
	centerBucket := dataset.BucketIDFromCoordinates(0, 0, 0, 0)
	neighborBucket := dataset.BucketIDFromCoordinates(1, 0, 0, 0)
	buckets[centerBucket] = []uint32{1, 2, 3, 4, 5}
	buckets[neighborBucket] = make([]uint32, 60)

	_, amountEnd, _, _, _, _, _, _, candidateCount := selectBucketWindow(
		buckets,
		nil,
		0,
		0,
		0,
		0,
		searchConfig{bucketTargetCandidates: 64, bucketMaxSearchRadius: 2},
	)

	if amountEnd < 1 {
		t.Fatalf("expected search radius expansion to include neighbor bucket")
	}

	if candidateCount < 64 {
		t.Fatalf("candidateCount = %d, want at least 64", candidateCount)
	}
}

func TestShortlistCapsProcessedCandidates(t *testing.T) {
	t.Parallel()

	query := sampleRequestVector()
	store := &dataset.VectorStore{
		QuantizedVectors: benchmarkQuantizedVectors(4096),
		Labels:           benchmarkLabels(4096),
		Count:            4096,
	}
	dataset.BuildBucketIndex(store)
	dataset.BuildIVFIndex(store, 64)

	_, stats := findTop5WithStats(query, store, BucketStrategyShortlist, searchConfig{
		bucketTargetCandidates: 96,
		bucketMaxSearchRadius:  3,
	})
	if stats.processedCandidates > 96 {
		t.Fatalf("processedCandidates = %d, want <= 96", stats.processedCandidates)
	}

	_, stats = findTop5WithStats(query, store, BucketStrategyIVF, searchConfig{
		bucketTargetCandidates: 96,
		bucketMaxSearchRadius:  3,
		ivfNProbe:              4,
	})
	if stats.processedCandidates > 96 {
		t.Fatalf("ivf processedCandidates = %d, want <= 96", stats.processedCandidates)
	}
}

func vectorWithFirstDimension(value float32) [14]float32 {
	var vector [14]float32
	vector[0] = value
	return vector
}

func flattenVectors(vectors ...[14]float32) []float32 {
	flattened := make([]float32, 0, len(vectors)*14)

	for _, vector := range vectors {
		flattened = append(flattened, vector[:]...)
	}

	return flattened
}

func sampleRequestVector() [14]float32 {
	vector := [14]float32{
		0.50,
		0.25,
		0.10,
		0.50,
		0.20,
		-1,
		-1,
		0.03,
		0.15,
		0,
		1,
		0,
		0.20,
		0.04,
	}

	return vector
}
