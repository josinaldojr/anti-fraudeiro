package fraud

import (
	"testing"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func TestFindTop5(t *testing.T) {
	t.Parallel()

	query := vectorWithFirstDimension(0)
	store := &dataset.VectorStore{
		Vectors: flattenVectors(
			vectorWithFirstDimension(0.00),
			vectorWithFirstDimension(0.10),
			vectorWithFirstDimension(0.20),
			vectorWithFirstDimension(0.30),
			vectorWithFirstDimension(0.40),
			vectorWithFirstDimension(10.00),
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

	if fraudCount := FindTop5(query, store); fraudCount != 3 {
		t.Fatalf("FindTop5 fraud count = %d, want 3", fraudCount)
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
