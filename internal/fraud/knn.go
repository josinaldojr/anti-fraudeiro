package fraud

import (
	"math"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func FindTop5(query [14]float32, store *dataset.VectorStore) (fraudCount int) {
	if store == nil || store.Count == 0 {
		return 0
	}

	q0 := query[0]
	q1 := query[1]
	q2 := query[2]
	q3 := query[3]
	q4 := query[4]
	q5 := query[5]
	q6 := query[6]
	q7 := query[7]
	q8 := query[8]
	q9 := query[9]
	q10 := query[10]
	q11 := query[11]
	q12 := query[12]
	q13 := query[13]

	vectors := store.Vectors
	labels := store.Labels

	var bestDistances [topK]float32
	var bestLabels [topK]byte

	for index := range bestDistances {
		bestDistances[index] = math.MaxFloat32
	}

	for vectorIndex, baseOffset := 0, 0; vectorIndex < store.Count; vectorIndex, baseOffset = vectorIndex+1, baseOffset+dataset.VectorSize {
		delta0 := q0 - vectors[baseOffset]
		delta1 := q1 - vectors[baseOffset+1]
		delta2 := q2 - vectors[baseOffset+2]
		delta3 := q3 - vectors[baseOffset+3]
		delta4 := q4 - vectors[baseOffset+4]
		delta5 := q5 - vectors[baseOffset+5]
		delta6 := q6 - vectors[baseOffset+6]
		delta7 := q7 - vectors[baseOffset+7]
		delta8 := q8 - vectors[baseOffset+8]
		delta9 := q9 - vectors[baseOffset+9]
		delta10 := q10 - vectors[baseOffset+10]
		delta11 := q11 - vectors[baseOffset+11]
		delta12 := q12 - vectors[baseOffset+12]
		delta13 := q13 - vectors[baseOffset+13]

		distance := delta0*delta0 +
			delta1*delta1 +
			delta2*delta2 +
			delta3*delta3 +
			delta4*delta4 +
			delta5*delta5 +
			delta6*delta6 +
			delta7*delta7 +
			delta8*delta8 +
			delta9*delta9 +
			delta10*delta10 +
			delta11*delta11 +
			delta12*delta12 +
			delta13*delta13
		if distance < bestDistances[topK-1] {
			insertTopK(distance, labels[vectorIndex], &bestDistances, &bestLabels)
		}
	}

	for _, label := range bestLabels {
		if label == dataset.LabelFraud {
			fraudCount++
		}
	}

	return fraudCount
}

func insertTopK(distance float32, label byte, bestDistances *[topK]float32, bestLabels *[topK]byte) {
	insertIndex := topK - 1
	for insertIndex > 0 && distance < bestDistances[insertIndex-1] {
		bestDistances[insertIndex] = bestDistances[insertIndex-1]
		bestLabels[insertIndex] = bestLabels[insertIndex-1]
		insertIndex--
	}

	bestDistances[insertIndex] = distance
	bestLabels[insertIndex] = label
}
