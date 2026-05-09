package fraud

import (
	"testing"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func BenchmarkScorerScore(b *testing.B) {
	b.ReportAllocs()

	scorer := NewScorer(newTestVectorizer(), &dataset.VectorStore{
		QuantizedVectors: benchmarkQuantizedVectors(1024),
		Labels:           benchmarkLabels(1024),
		Count:            1024,
	})
	dataset.BuildBucketIndex(scorer.store)
	request := sampleRequest()

	for b.Loop() {
		if _, err := scorer.Score(request); err != nil {
			b.Fatalf("Score returned error: %v", err)
		}
	}
}

func BenchmarkScorerScoreLargeKnownMerchantSet(b *testing.B) {
	b.ReportAllocs()

	scorer := NewScorer(newTestVectorizer(), &dataset.VectorStore{
		QuantizedVectors: benchmarkQuantizedVectors(1024),
		Labels:           benchmarkLabels(1024),
		Count:            1024,
	})
	dataset.BuildBucketIndex(scorer.store)
	request := sampleRequest()
	request.Customer.KnownMerchants = benchmarkKnownMerchants(128)
	request.Customer.KnownMerchants[96] = request.Merchant.ID
	request.Customer.Finalize()

	for b.Loop() {
		if _, err := scorer.Score(request); err != nil {
			b.Fatalf("Score returned error: %v", err)
		}
	}
}

func benchmarkVectors(count int) []float32 {
	values := make([]float32, 0, count*14)

	for i := 0; i < count; i++ {
		base := float32(i%100) / 100
		vector := [14]float32{
			base,
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
		values = append(values, vector[:]...)
	}

	return values
}

func benchmarkQuantizedVectors(count int) []uint16 {
	values := benchmarkVectors(count)
	quantized := make([]uint16, len(values))

	for index, value := range values {
		quantized[index] = dataset.QuantizeComponent(value)
	}

	return quantized
}

func benchmarkLabels(count int) []byte {
	labels := make([]byte, count)

	for i := 0; i < count; i++ {
		if i%4 == 0 {
			labels[i] = dataset.LabelFraud
		}
	}

	return labels
}

func benchmarkKnownMerchants(count int) []string {
	merchants := make([]string, count)
	for i := 0; i < count; i++ {
		merchants[i] = "MERC-BENCH-" + string(rune('A'+(i%26))) + string(rune('0'+(i%10)))
	}
	return merchants
}
