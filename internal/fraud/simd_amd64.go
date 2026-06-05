//go:build amd64 && !noasm
// +build amd64,!noasm

package fraud

import (
	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

// We temporarily disable AVX2 SIMD scanning because it lacks register-based early short-circuiting
// and forces evaluation of all vectors in a batch, introducing severe latency regressions.
var supportsAVX2 = false // hasAVX2()
var supportsExactScanAVX2 = hasAVX2()

func scanQuantizedContiguousSIMD(
	vectors []uint16,
	labels []byte,
	q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15 int32,
	bestDistances *[topK]uint64,
	bestLabels *[topK]byte,
) int {
	if !supportsAVX2 {
		return scanQuantizedContiguous(
			vectors, labels,
			q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15,
			bestDistances, bestLabels,
		)
	}

	count := len(labels)
	if count == 0 {
		return 0
	}

	qArr := [16]int32{q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15}
	threshold := bestDistances[topK-1]

	const batchSize = 128
	var dists [batchSize]uint64

	for i := 0; i < count; i += batchSize {
		end := i + batchSize
		if end > count {
			end = count
		}

		n := end - i
		batchVectors := vectors[i*dataset.VectorSize : end*dataset.VectorSize]

		DistancesAVX2(&qArr, batchVectors, dists[:n])

		for j := 0; j < n; j++ {
			d := dists[j]
			if d < threshold {
				insertTopKUint64(d, labels[i+j], bestDistances, bestLabels)
				threshold = bestDistances[topK-1]
			}
		}
	}

	return count
}

func exactScanSIMD(query [16]float32, store *dataset.VectorStore) int {
	if !supportsExactScanAVX2 || store == nil || store.Count == 0 || len(store.QuantizedVectors) == 0 {
		return exactScanScalar(query, store)
	}

	var bestDistances [topK]uint64
	var bestLabels [topK]byte
	for i := range bestDistances {
		bestDistances[i] = ^uint64(0)
	}

	qArr := [16]int32{
		int32(dataset.QuantizeComponent(query[0])),
		int32(dataset.QuantizeComponent(query[1])),
		int32(dataset.QuantizeComponent(query[2])),
		int32(dataset.QuantizeComponent(query[3])),
		int32(dataset.QuantizeComponent(query[4])),
		int32(dataset.QuantizeComponent(query[5])),
		int32(dataset.QuantizeComponent(query[6])),
		int32(dataset.QuantizeComponent(query[7])),
		int32(dataset.QuantizeComponent(query[8])),
		int32(dataset.QuantizeComponent(query[9])),
		int32(dataset.QuantizeComponent(query[10])),
		int32(dataset.QuantizeComponent(query[11])),
		int32(dataset.QuantizeComponent(query[12])),
		int32(dataset.QuantizeComponent(query[13])),
		int32(dataset.QuantizeComponent(query[14])),
		int32(dataset.QuantizeComponent(query[15])),
	}

	const batchSize = 512
	var distances [batchSize]uint64
	vectors := store.QuantizedVectors
	labels := store.Labels
	threshold := bestDistances[topK-1]

	for start := 0; start < store.Count; start += batchSize {
		end := start + batchSize
		if end > store.Count {
			end = store.Count
		}

		count := end - start
		DistancesAVX2(&qArr, vectors[start*dataset.VectorSize:end*dataset.VectorSize], distances[:count])
		for i := 0; i < count; i++ {
			distance := distances[i]
			if distance < threshold {
				insertTopKUint64(distance, labels[start+i], &bestDistances, &bestLabels)
				threshold = bestDistances[topK-1]
			}
		}
	}

	fraudCount := 0
	for _, label := range bestLabels {
		if label == dataset.LabelFraud {
			fraudCount++
		}
	}
	return fraudCount
}
