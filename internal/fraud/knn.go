package fraud

import (
	"math"
	"sort"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

const (
	bucketTargetCandidates                = 256
	bucketMaxSearchRadius                 = 2
	candidateSeenLimit                    = 4096
	secondaryFallbackMaxPrimaryCandidates = 96
	secondaryFallbackMaxBucketSize        = 256
	maxWindowBuckets                      = 625
)

func FindTop5(query [14]float32, store *dataset.VectorStore) (fraudCount int) {
	return FindTop5WithStrategy(query, store, BucketStrategyWindow)
}

func FindTop5WithStrategy(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy) (fraudCount int) {
	if store == nil || store.Count == 0 {
		return 0
	}

	if len(store.QuantizedVectors) > 0 {
		return findTop5Quantized(query, store, strategy)
	}

	return findTop5Float32(query, store, strategy)
}

func FindTop5Exact(query [14]float32, store *dataset.VectorStore) (fraudCount int) {
	if store == nil || store.Count == 0 {
		return 0
	}

	if len(store.QuantizedVectors) > 0 {
		return findTop5Quantized(query, store, "")
	}

	return findTop5Float32(query, store, "")
}

func findTop5Float32(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy) (fraudCount int) {
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

	if len(store.BucketIndex) > 0 && strategy != "" {
		return findTop5Float32Bucketed(query, store, strategy, bestDistances, bestLabels)
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

func findTop5Float32Bucketed(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy, bestDistances [topK]float32, bestLabels [topK]byte) (fraudCount int) {
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
	amountBucket, hourBucket, dayBucket, tx24hBucket := dataset.BucketCoordinatesFromQuery(q0, q3, q4, q8)
	amountStart, amountEnd, hourStart, hourEnd, dayStart, dayEnd, txStart, txEnd, candidateCount :=
		selectBucketWindow(store.BucketIndex, amountBucket, hourBucket, dayBucket, tx24hBucket)
	if candidateCount < topK {
		return findTop5Float32(query, &dataset.VectorStore{Vectors: vectors, Labels: labels, Count: store.Count}, "")
	}

	trackSeen := len(store.SecondaryBucketIndex) > 0
	var seen [candidateSeenLimit]uint32
	seenCount := 0
	if strategy == BucketStrategyOrdered {
		var bucketCandidates [maxWindowBuckets]bucketCandidate
		bucketCount, _ := collectOrderedBucketCandidates(
			bucketCandidates[:],
			store.BucketIndex,
			q0,
			q3,
			q4,
			q8,
			amountStart,
			amountEnd,
			hourStart,
			hourEnd,
			dayStart,
			dayEnd,
			txStart,
			txEnd,
		)
		processedCandidates := 0
		for bucketIndex := 0; bucketIndex < bucketCount; bucketIndex++ {
			bucketID := bucketCandidates[bucketIndex].id
			bucketVectors := store.BucketIndex[bucketID]
			processedCandidates += len(bucketVectors)
			for _, vectorIndex := range bucketVectors {
				if trackSeen && seenCount < candidateSeenLimit {
					seen[seenCount] = vectorIndex
					seenCount++
				}
				baseOffset := int(vectorIndex) * dataset.VectorSize
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
			if processedCandidates >= bucketTargetCandidates && processedCandidates >= topK {
				break
			}
		}
	} else {
		for amountIndex := amountStart; amountIndex <= amountEnd; amountIndex++ {
			for hourIndex := hourStart; hourIndex <= hourEnd; hourIndex++ {
				for dayIndex := dayStart; dayIndex <= dayEnd; dayIndex++ {
					for txIndex := txStart; txIndex <= txEnd; txIndex++ {
						bucketID := dataset.BucketIDFromCoordinates(amountIndex, hourIndex, dayIndex, txIndex)
						for _, vectorIndex := range store.BucketIndex[bucketID] {
							if trackSeen && seenCount < candidateSeenLimit {
								seen[seenCount] = vectorIndex
								seenCount++
							}
							baseOffset := int(vectorIndex) * dataset.VectorSize
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
					}
				}
			}
		}
	}

	if trackSeen && candidateCount <= secondaryFallbackMaxPrimaryCandidates {
		amountBucket2, hourBucket2, dayBucket2, riskBucket2 := dataset.SecondaryBucketCoordinatesFromQuery(q0, q3, q4, q12)
		secondaryBucketID := dataset.SecondaryBucketIDFromCoordinates(amountBucket2, hourBucket2, dayBucket2, riskBucket2)
		secondaryCandidates := store.SecondaryBucketIndex[secondaryBucketID]
		if len(secondaryCandidates) <= secondaryFallbackMaxBucketSize {
			for _, vectorIndex := range secondaryCandidates {
				if containsCandidateID(seen[:seenCount], vectorIndex) {
					continue
				}
				if seenCount < candidateSeenLimit {
					seen[seenCount] = vectorIndex
					seenCount++
				}

				baseOffset := int(vectorIndex) * dataset.VectorSize
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
		}
	}

	for _, label := range bestLabels {
		if label == dataset.LabelFraud {
			fraudCount++
		}
	}

	return fraudCount
}

func findTop5Quantized(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy) (fraudCount int) {
	q0 := int32(dataset.QuantizeComponent(query[0]))
	q1 := int32(dataset.QuantizeComponent(query[1]))
	q2 := int32(dataset.QuantizeComponent(query[2]))
	q3 := int32(dataset.QuantizeComponent(query[3]))
	q4 := int32(dataset.QuantizeComponent(query[4]))
	q5 := int32(dataset.QuantizeComponent(query[5]))
	q6 := int32(dataset.QuantizeComponent(query[6]))
	q7 := int32(dataset.QuantizeComponent(query[7]))
	q8 := int32(dataset.QuantizeComponent(query[8]))
	q9 := int32(dataset.QuantizeComponent(query[9]))
	q10 := int32(dataset.QuantizeComponent(query[10]))
	q11 := int32(dataset.QuantizeComponent(query[11]))
	q12 := int32(dataset.QuantizeComponent(query[12]))
	q13 := int32(dataset.QuantizeComponent(query[13]))

	vectors := store.QuantizedVectors
	labels := store.Labels

	var bestDistances [topK]uint64
	var bestLabels [topK]byte

	for index := range bestDistances {
		bestDistances[index] = ^uint64(0)
	}

	if len(store.BucketIndex) > 0 && strategy != "" {
		return findTop5QuantizedBucketed(store, strategy, q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, bestDistances, bestLabels)
	}

	for vectorIndex, baseOffset := 0, 0; vectorIndex < store.Count; vectorIndex, baseOffset = vectorIndex+1, baseOffset+dataset.VectorSize {
		delta0 := q0 - int32(vectors[baseOffset])
		delta1 := q1 - int32(vectors[baseOffset+1])
		delta2 := q2 - int32(vectors[baseOffset+2])
		delta3 := q3 - int32(vectors[baseOffset+3])
		delta4 := q4 - int32(vectors[baseOffset+4])
		delta5 := q5 - int32(vectors[baseOffset+5])
		delta6 := q6 - int32(vectors[baseOffset+6])
		delta7 := q7 - int32(vectors[baseOffset+7])
		delta8 := q8 - int32(vectors[baseOffset+8])
		delta9 := q9 - int32(vectors[baseOffset+9])
		delta10 := q10 - int32(vectors[baseOffset+10])
		delta11 := q11 - int32(vectors[baseOffset+11])
		delta12 := q12 - int32(vectors[baseOffset+12])
		delta13 := q13 - int32(vectors[baseOffset+13])

		distance := squareUint64(delta0) +
			squareUint64(delta1) +
			squareUint64(delta2) +
			squareUint64(delta3) +
			squareUint64(delta4) +
			squareUint64(delta5) +
			squareUint64(delta6) +
			squareUint64(delta7) +
			squareUint64(delta8) +
			squareUint64(delta9) +
			squareUint64(delta10) +
			squareUint64(delta11) +
			squareUint64(delta12) +
			squareUint64(delta13)
		if distance < bestDistances[topK-1] {
			insertTopKUint64(distance, labels[vectorIndex], &bestDistances, &bestLabels)
		}
	}

	for _, label := range bestLabels {
		if label == dataset.LabelFraud {
			fraudCount++
		}
	}

	return fraudCount
}

func findTop5QuantizedBucketed(
	store *dataset.VectorStore,
	strategy BucketStrategy,
	q0 int32,
	q1 int32,
	q2 int32,
	q3 int32,
	q4 int32,
	q5 int32,
	q6 int32,
	q7 int32,
	q8 int32,
	q9 int32,
	q10 int32,
	q11 int32,
	q12 int32,
	q13 int32,
	bestDistances [topK]uint64,
	bestLabels [topK]byte,
) (fraudCount int) {
	vectors := store.QuantizedVectors
	labels := store.Labels
	amountBucket, hourBucket, dayBucket, tx24hBucket := dataset.BucketCoordinatesFromQuery(
		dataset.DequantizeComponent(uint16(q0)),
		dataset.DequantizeComponent(uint16(q3)),
		dataset.DequantizeComponent(uint16(q4)),
		dataset.DequantizeComponent(uint16(q8)),
	)

	amountStart, amountEnd, hourStart, hourEnd, dayStart, dayEnd, txStart, txEnd, candidateCount :=
		selectBucketWindow(store.BucketIndex, amountBucket, hourBucket, dayBucket, tx24hBucket)
	if candidateCount < topK {
		return findTop5Quantized(
			[14]float32{
				dataset.DequantizeComponent(uint16(q0)),
				dataset.DequantizeComponent(uint16(q1)),
				dataset.DequantizeComponent(uint16(q2)),
				dataset.DequantizeComponent(uint16(q3)),
				dataset.DequantizeComponent(uint16(q4)),
				dataset.DequantizeComponent(uint16(q5)),
				dataset.DequantizeComponent(uint16(q6)),
				dataset.DequantizeComponent(uint16(q7)),
				dataset.DequantizeComponent(uint16(q8)),
				dataset.DequantizeComponent(uint16(q9)),
				dataset.DequantizeComponent(uint16(q10)),
				dataset.DequantizeComponent(uint16(q11)),
				dataset.DequantizeComponent(uint16(q12)),
				dataset.DequantizeComponent(uint16(q13)),
			},
			&dataset.VectorStore{QuantizedVectors: vectors, Labels: labels, Count: store.Count},
			"",
		)
	}

	trackSeen := len(store.SecondaryBucketIndex) > 0
	var seen [candidateSeenLimit]uint32
	seenCount := 0

	if strategy == BucketStrategyOrdered {
		queryAmount := dataset.DequantizeComponent(uint16(q0))
		queryHour := dataset.DequantizeComponent(uint16(q3))
		queryDay := dataset.DequantizeComponent(uint16(q4))
		queryTx24h := dataset.DequantizeComponent(uint16(q8))
		var bucketCandidates [maxWindowBuckets]bucketCandidate
		bucketCount, _ := collectOrderedBucketCandidates(
			bucketCandidates[:],
			store.BucketIndex,
			queryAmount,
			queryHour,
			queryDay,
			queryTx24h,
			amountStart,
			amountEnd,
			hourStart,
			hourEnd,
			dayStart,
			dayEnd,
			txStart,
			txEnd,
		)
		processedCandidates := 0
		for bucketIndex := 0; bucketIndex < bucketCount; bucketIndex++ {
			bucketID := bucketCandidates[bucketIndex].id
			bucketVectors := store.BucketIndex[bucketID]
			processedCandidates += len(bucketVectors)
			for _, vectorIndex := range bucketVectors {
				if trackSeen && seenCount < candidateSeenLimit {
					seen[seenCount] = vectorIndex
					seenCount++
				}
				baseOffset := int(vectorIndex) * dataset.VectorSize
				delta0 := q0 - int32(vectors[baseOffset])
				delta1 := q1 - int32(vectors[baseOffset+1])
				delta2 := q2 - int32(vectors[baseOffset+2])
				delta3 := q3 - int32(vectors[baseOffset+3])
				delta4 := q4 - int32(vectors[baseOffset+4])
				delta5 := q5 - int32(vectors[baseOffset+5])
				delta6 := q6 - int32(vectors[baseOffset+6])
				delta7 := q7 - int32(vectors[baseOffset+7])
				delta8 := q8 - int32(vectors[baseOffset+8])
				delta9 := q9 - int32(vectors[baseOffset+9])
				delta10 := q10 - int32(vectors[baseOffset+10])
				delta11 := q11 - int32(vectors[baseOffset+11])
				delta12 := q12 - int32(vectors[baseOffset+12])
				delta13 := q13 - int32(vectors[baseOffset+13])

				distance := squareUint64(delta0) +
					squareUint64(delta1) +
					squareUint64(delta2) +
					squareUint64(delta3) +
					squareUint64(delta4) +
					squareUint64(delta5) +
					squareUint64(delta6) +
					squareUint64(delta7) +
					squareUint64(delta8) +
					squareUint64(delta9) +
					squareUint64(delta10) +
					squareUint64(delta11) +
					squareUint64(delta12) +
					squareUint64(delta13)
				if distance < bestDistances[topK-1] {
					insertTopKUint64(distance, labels[vectorIndex], &bestDistances, &bestLabels)
				}
			}
			if processedCandidates >= bucketTargetCandidates && processedCandidates >= topK {
				break
			}
		}
	} else {
		for amountIndex := amountStart; amountIndex <= amountEnd; amountIndex++ {
			for hourIndex := hourStart; hourIndex <= hourEnd; hourIndex++ {
				for dayIndex := dayStart; dayIndex <= dayEnd; dayIndex++ {
					for txIndex := txStart; txIndex <= txEnd; txIndex++ {
						bucketID := dataset.BucketIDFromCoordinates(amountIndex, hourIndex, dayIndex, txIndex)
						for _, vectorIndex := range store.BucketIndex[bucketID] {
							if trackSeen && seenCount < candidateSeenLimit {
								seen[seenCount] = vectorIndex
								seenCount++
							}
							baseOffset := int(vectorIndex) * dataset.VectorSize
							delta0 := q0 - int32(vectors[baseOffset])
							delta1 := q1 - int32(vectors[baseOffset+1])
							delta2 := q2 - int32(vectors[baseOffset+2])
							delta3 := q3 - int32(vectors[baseOffset+3])
							delta4 := q4 - int32(vectors[baseOffset+4])
							delta5 := q5 - int32(vectors[baseOffset+5])
							delta6 := q6 - int32(vectors[baseOffset+6])
							delta7 := q7 - int32(vectors[baseOffset+7])
							delta8 := q8 - int32(vectors[baseOffset+8])
							delta9 := q9 - int32(vectors[baseOffset+9])
							delta10 := q10 - int32(vectors[baseOffset+10])
							delta11 := q11 - int32(vectors[baseOffset+11])
							delta12 := q12 - int32(vectors[baseOffset+12])
							delta13 := q13 - int32(vectors[baseOffset+13])

							distance := squareUint64(delta0) +
								squareUint64(delta1) +
								squareUint64(delta2) +
								squareUint64(delta3) +
								squareUint64(delta4) +
								squareUint64(delta5) +
								squareUint64(delta6) +
								squareUint64(delta7) +
								squareUint64(delta8) +
								squareUint64(delta9) +
								squareUint64(delta10) +
								squareUint64(delta11) +
								squareUint64(delta12) +
								squareUint64(delta13)
							if distance < bestDistances[topK-1] {
								insertTopKUint64(distance, labels[vectorIndex], &bestDistances, &bestLabels)
							}
						}
					}
				}
			}
		}
	}

	if trackSeen && candidateCount <= secondaryFallbackMaxPrimaryCandidates {
		amountBucket2, hourBucket2, dayBucket2, riskBucket2 := dataset.SecondaryBucketCoordinatesFromQuery(
			dataset.DequantizeComponent(uint16(q0)),
			dataset.DequantizeComponent(uint16(q3)),
			dataset.DequantizeComponent(uint16(q4)),
			dataset.DequantizeComponent(uint16(q12)),
		)
		secondaryBucketID := dataset.SecondaryBucketIDFromCoordinates(amountBucket2, hourBucket2, dayBucket2, riskBucket2)
		secondaryCandidates := store.SecondaryBucketIndex[secondaryBucketID]
		if len(secondaryCandidates) <= secondaryFallbackMaxBucketSize {
			for _, vectorIndex := range secondaryCandidates {
				if containsCandidateID(seen[:seenCount], vectorIndex) {
					continue
				}
				if seenCount < candidateSeenLimit {
					seen[seenCount] = vectorIndex
					seenCount++
				}

				baseOffset := int(vectorIndex) * dataset.VectorSize
				delta0 := q0 - int32(vectors[baseOffset])
				delta1 := q1 - int32(vectors[baseOffset+1])
				delta2 := q2 - int32(vectors[baseOffset+2])
				delta3 := q3 - int32(vectors[baseOffset+3])
				delta4 := q4 - int32(vectors[baseOffset+4])
				delta5 := q5 - int32(vectors[baseOffset+5])
				delta6 := q6 - int32(vectors[baseOffset+6])
				delta7 := q7 - int32(vectors[baseOffset+7])
				delta8 := q8 - int32(vectors[baseOffset+8])
				delta9 := q9 - int32(vectors[baseOffset+9])
				delta10 := q10 - int32(vectors[baseOffset+10])
				delta11 := q11 - int32(vectors[baseOffset+11])
				delta12 := q12 - int32(vectors[baseOffset+12])
				delta13 := q13 - int32(vectors[baseOffset+13])

				distance := squareUint64(delta0) +
					squareUint64(delta1) +
					squareUint64(delta2) +
					squareUint64(delta3) +
					squareUint64(delta4) +
					squareUint64(delta5) +
					squareUint64(delta6) +
					squareUint64(delta7) +
					squareUint64(delta8) +
					squareUint64(delta9) +
					squareUint64(delta10) +
					squareUint64(delta11) +
					squareUint64(delta12) +
					squareUint64(delta13)
				if distance < bestDistances[topK-1] {
					insertTopKUint64(distance, labels[vectorIndex], &bestDistances, &bestLabels)
				}
			}
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

func insertTopKUint64(distance uint64, label byte, bestDistances *[topK]uint64, bestLabels *[topK]byte) {
	insertIndex := topK - 1
	for insertIndex > 0 && distance < bestDistances[insertIndex-1] {
		bestDistances[insertIndex] = bestDistances[insertIndex-1]
		bestLabels[insertIndex] = bestLabels[insertIndex-1]
		insertIndex--
	}

	bestDistances[insertIndex] = distance
	bestLabels[insertIndex] = label
}

func squareUint64(value int32) uint64 {
	unsigned := int64(value)
	if unsigned < 0 {
		unsigned = -unsigned
	}

	return uint64(unsigned * unsigned)
}

func containsCandidateID(candidateIDs []uint32, value uint32) bool {
	for _, candidateID := range candidateIDs {
		if candidateID == value {
			return true
		}
	}

	return false
}

type bucketCandidate struct {
	id         int
	lowerBound float32
}

func collectOrderedBucketCandidates(
	candidates []bucketCandidate,
	bucketIndex [][]uint32,
	queryAmount float32,
	queryHour float32,
	queryDay float32,
	queryTx24h float32,
	amountStart int,
	amountEnd int,
	hourStart int,
	hourEnd int,
	dayStart int,
	dayEnd int,
	txStart int,
	txEnd int,
) (int, int) {
	count := 0
	candidateCount := 0

	for amountIndex := amountStart; amountIndex <= amountEnd; amountIndex++ {
		for hourIndex := hourStart; hourIndex <= hourEnd; hourIndex++ {
			for dayIndex := dayStart; dayIndex <= dayEnd; dayIndex++ {
				for txIndex := txStart; txIndex <= txEnd; txIndex++ {
					bucketID := dataset.BucketIDFromCoordinates(amountIndex, hourIndex, dayIndex, txIndex)
					bucketSize := len(bucketIndex[bucketID])
					if bucketSize == 0 {
						continue
					}

					candidates[count] = bucketCandidate{
						id: bucketID,
						lowerBound: coarseLowerBound(
							queryAmount,
							queryHour,
							queryDay,
							queryTx24h,
							amountIndex,
							hourIndex,
							dayIndex,
							txIndex,
						),
					}
					count++
					candidateCount += bucketSize
				}
			}
		}
	}

	sort.Slice(candidates[:count], func(left int, right int) bool {
		if candidates[left].lowerBound == candidates[right].lowerBound {
			return candidates[left].id < candidates[right].id
		}

		return candidates[left].lowerBound < candidates[right].lowerBound
	})

	return count, candidateCount
}

func coarseLowerBound(
	queryAmount float32,
	queryHour float32,
	queryDay float32,
	queryTx24h float32,
	amountBucket int,
	hourBucket int,
	dayBucket int,
	txBucket int,
) float32 {
	amountDistance := intervalDistance(queryAmount, amountBucket, dataset.AmountBucketCount)
	hourDistance := intervalDistance(queryHour, hourBucket, dataset.HourBucketCount)
	dayDistance := intervalDistance(queryDay, dayBucket, dataset.DayBucketCount)
	txDistance := intervalDistance(queryTx24h, txBucket, dataset.Tx24hBucketCount)

	return amountDistance*amountDistance +
		hourDistance*hourDistance +
		dayDistance*dayDistance +
		txDistance*txDistance
}

func intervalDistance(query float32, bucket int, bucketCount int) float32 {
	minimum := float32(bucket) / float32(bucketCount)
	maximum := float32(bucket+1) / float32(bucketCount)

	if query < minimum {
		return minimum - query
	}
	if query > maximum {
		return query - maximum
	}

	return 0
}

func selectBucketWindow(
	bucketIndex [][]uint32,
	amountBucket int,
	hourBucket int,
	dayBucket int,
	tx24hBucket int,
) (amountStart int, amountEnd int, hourStart int, hourEnd int, dayStart int, dayEnd int, txStart int, txEnd int, candidateCount int) {
	for radius := 0; radius <= bucketMaxSearchRadius; radius++ {
		amountStart = bucketRangeStart(amountBucket, dataset.AmountBucketCount, radius)
		amountEnd = bucketRangeEnd(amountBucket, dataset.AmountBucketCount, radius)
		hourStart = bucketRangeStart(hourBucket, dataset.HourBucketCount, radius)
		hourEnd = bucketRangeEnd(hourBucket, dataset.HourBucketCount, radius)
		dayStart = bucketRangeStart(dayBucket, dataset.DayBucketCount, radius)
		dayEnd = bucketRangeEnd(dayBucket, dataset.DayBucketCount, radius)
		txStart = bucketRangeStart(tx24hBucket, dataset.Tx24hBucketCount, radius)
		txEnd = bucketRangeEnd(tx24hBucket, dataset.Tx24hBucketCount, radius)

		candidateCount = 0
		for amountIndex := amountStart; amountIndex <= amountEnd; amountIndex++ {
			for hourIndex := hourStart; hourIndex <= hourEnd; hourIndex++ {
				for dayIndex := dayStart; dayIndex <= dayEnd; dayIndex++ {
					for txIndex := txStart; txIndex <= txEnd; txIndex++ {
						candidateCount += len(bucketIndex[dataset.BucketIDFromCoordinates(amountIndex, hourIndex, dayIndex, txIndex)])
					}
				}
			}
		}

		if candidateCount >= bucketTargetCandidates || candidateCount >= topK {
			return amountStart, amountEnd, hourStart, hourEnd, dayStart, dayEnd, txStart, txEnd, candidateCount
		}
	}

	return amountStart, amountEnd, hourStart, hourEnd, dayStart, dayEnd, txStart, txEnd, candidateCount
}

func bucketRangeStart(center int, bucketCount int, radius int) int {
	start := center - radius
	if start < 0 {
		return 0
	}

	return start
}

func bucketRangeEnd(center int, bucketCount int, radius int) int {
	end := center + radius
	if end >= bucketCount {
		return bucketCount - 1
	}

	return end
}
