package fraud

import (
	"testing"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func BenchmarkScorerScore(b *testing.B) {
	b.ReportAllocs()

	scorer := NewScorer(newTestVectorizer(), &dataset.VectorStore{
		Vectors: benchmarkVectors(1024),
		Labels:  benchmarkLabels(1024),
		Count:   1024,
	})
	request := sampleRequest()

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

func benchmarkLabels(count int) []byte {
	labels := make([]byte, count)

	for i := 0; i < count; i++ {
		if i%4 == 0 {
			labels[i] = dataset.LabelFraud
		}
	}

	return labels
}
