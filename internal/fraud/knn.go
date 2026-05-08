package fraud

import (
	"math"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func FindTop5(query [14]float32, store *dataset.VectorStore) (fraudCount int) {
	if store == nil || store.Count == 0 {
		return 0
	}

	var bestDistances [topK]float32
	var bestLabels [topK]byte

	for index := range bestDistances {
		bestDistances[index] = math.MaxFloat32
	}

	for vectorIndex := 0; vectorIndex < store.Count; vectorIndex++ {
		baseOffset := vectorIndex * dataset.VectorSize

		var distance float32
		for dimension := 0; dimension < dataset.VectorSize; dimension++ {
			delta := query[dimension] - store.Vectors[baseOffset+dimension]
			distance += delta * delta
		}

		worstIndex := indexOfWorstDistance(bestDistances)
		if distance < bestDistances[worstIndex] {
			bestDistances[worstIndex] = distance
			bestLabels[worstIndex] = store.Labels[vectorIndex]
		}
	}

	for _, label := range bestLabels {
		if label == dataset.LabelFraud {
			fraudCount++
		}
	}

	return fraudCount
}

func indexOfWorstDistance(distances [topK]float32) int {
	worstIndex := 0

	for index := 1; index < topK; index++ {
		if distances[index] > distances[worstIndex] {
			worstIndex = index
		}
	}

	return worstIndex
}
