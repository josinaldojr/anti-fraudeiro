package fraud

import (
	"testing"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func TestExactScanSIMDMatchesScalar(t *testing.T) {
	store := &dataset.VectorStore{
		QuantizedVectors: benchmarkQuantizedVectors(4096),
		Labels:           benchmarkLabels(4096),
		Count:            4096,
	}

	vectorizer := newTestVectorizer()
	query, err := vectorizer.Vectorize(sampleRequest())
	if err != nil {
		t.Fatalf("Vectorize returned error: %v", err)
	}

	scalar := exactScanScalar(query, store)
	simd := exactScanSIMD(query, store)
	if simd != scalar {
		t.Fatalf("exactScanSIMD=%d, want scalar=%d", simd, scalar)
	}
}
