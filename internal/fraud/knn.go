package fraud

import (
	"sync"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

const (
	maxWindowBuckets = 2401
)

type searchConfig struct {
	bucketTargetCandidates int
	bucketMaxSearchRadius  int
}

type searchStats struct {
	processedCandidates int
	windowCandidates    int
	bucketsVisited      int
}

type knnScratch struct {
	bucketCandidates [maxWindowBuckets]bucketCandidate
}

var knnScratchPool = sync.Pool{
	New: func() any {
		return &knnScratch{}
	},
}

var defaultSearchConfig = searchConfig{
	bucketTargetCandidates: 128,
	bucketMaxSearchRadius:  2,
}

func SetDefaultSearchConfig(bucketTargetCandidates int, bucketMaxSearchRadius int) {
	if bucketTargetCandidates > 0 {
		defaultSearchConfig.bucketTargetCandidates = bucketTargetCandidates
	}
	if bucketMaxSearchRadius >= 0 && bucketMaxSearchRadius <= 3 {
		defaultSearchConfig.bucketMaxSearchRadius = bucketMaxSearchRadius
	}
}

func FindTop5(query [16]float32, store *dataset.VectorStore) (fraudCount int) {
	fraudCount, _ = findTop5WithStats(query, store, defaultSearchConfig)
	return fraudCount
}

func findTop5WithStats(query [16]float32, store *dataset.VectorStore, cfg searchConfig) (fraudCount int, stats searchStats) {
	if store == nil || store.Count == 0 {
		return 0, searchStats{}
	}

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
	q14 := int32(dataset.QuantizeComponent(query[14]))
	q15 := int32(dataset.QuantizeComponent(query[15]))

	vectors := store.QuantizedVectors
	labels := store.Labels

	var bestDistances [topK]uint64
	var bestLabels [topK]byte

	for index := range bestDistances {
		bestDistances[index] = ^uint64(0)
	}

	amountBucket, hourBucket, dayBucket, tx24hBucket := dataset.BucketCoordinatesFromQuery(
		query[0],
		query[3],
		query[4],
		query[8],
	)

	amountStart, amountEnd, hourStart, hourEnd, dayStart, dayEnd, txStart, txEnd, candidateCount :=
		selectBucketWindow(store.BucketIndex, store.BucketPrefixSums, amountBucket, hourBucket, dayBucket, tx24hBucket, cfg)
	stats.windowCandidates = candidateCount

	if len(store.BucketMeta) > 0 {
		for amountIndex := amountStart; amountIndex <= amountEnd; amountIndex++ {
			for hourIndex := hourStart; hourIndex <= hourEnd; hourIndex++ {
				for dayIndex := dayStart; dayIndex <= dayEnd; dayIndex++ {
					for txIndex := txStart; txIndex <= txEnd; txIndex++ {
						bucketID := dataset.BucketIDFromCoordinates(amountIndex, hourIndex, dayIndex, txIndex)
						meta := store.BucketMeta[bucketID]
						if meta.Count == 0 {
							continue
						}

						offset := int(meta.Offset) * dataset.VectorSize
						count := int(meta.Count)

						stats.processedCandidates += scanQuantizedContiguousSIMD(
							vectors[offset:offset+count*dataset.VectorSize],
							labels[meta.Offset:meta.Offset+meta.Count],
							q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15,
							&bestDistances,
							&bestLabels,
						)
						stats.bucketsVisited++
					}
				}
			}
		}
	} else if len(store.BucketIndex) > 0 {
		for amountIndex := amountStart; amountIndex <= amountEnd; amountIndex++ {
			for hourIndex := hourStart; hourIndex <= hourEnd; hourIndex++ {
				for dayIndex := dayStart; dayIndex <= dayEnd; dayIndex++ {
					for txIndex := txStart; txIndex <= txEnd; txIndex++ {
						bucketID := dataset.BucketIDFromCoordinates(amountIndex, hourIndex, dayIndex, txIndex)
						stats.processedCandidates += scanQuantizedCandidates(
							store.BucketIndex[bucketID],
							vectors,
							labels,
							q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15,
							&bestDistances,
							&bestLabels,
						)
						stats.bucketsVisited++
					}
				}
			}
		}
	} else {
		stats.processedCandidates = scanQuantizedContiguousSIMD(
			vectors,
			labels,
			q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15,
			&bestDistances,
			&bestLabels,
		)
	}

	for _, label := range bestLabels {
		if label == dataset.LabelFraud {
			fraudCount++
		}
	}

	return fraudCount, stats
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

type bucketCandidate struct {
	id       int
	distance float32
}

func scanQuantizedCandidates(
	candidateIDs []uint32,
	vectors []uint16,
	labels []byte,
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
	q14 int32,
	q15 int32,
	bestDistances *[topK]uint64,
	bestLabels *[topK]byte,
) int {
	threshold := bestDistances[topK-1]
	for _, vectorIndex := range candidateIDs {
		baseOffset := int(vectorIndex) * dataset.VectorSize
		v := vectors[baseOffset : baseOffset+dataset.VectorSize]

		d0 := q0 - int32(v[0])
		d1 := q1 - int32(v[1])
		d2 := q2 - int32(v[2])
		d3 := q3 - int32(v[3])
		d4 := q4 - int32(v[4])
		d5 := q5 - int32(v[5])
		d6 := q6 - int32(v[6])
		d7 := q7 - int32(v[7])
		d8 := q8 - int32(v[8])
		d9 := q9 - int32(v[9])
		d10 := q10 - int32(v[10])
		d11 := q11 - int32(v[11])
		d12 := q12 - int32(v[12])
		d13 := q13 - int32(v[13])
		d14 := q14 - int32(v[14])
		d15 := q15 - int32(v[15])

		dist := uint64(int64(d0)*int64(d0)) +
			uint64(int64(d1)*int64(d1)) +
			uint64(int64(d2)*int64(d2)) +
			uint64(int64(d3)*int64(d3)) +
			uint64(int64(d4)*int64(d4)) +
			uint64(int64(d5)*int64(d5)) +
			uint64(int64(d6)*int64(d6)) +
			uint64(int64(d7)*int64(d7)) +
			uint64(int64(d8)*int64(d8)) +
			uint64(int64(d9)*int64(d9)) +
			uint64(int64(d10)*int64(d10)) +
			uint64(int64(d11)*int64(d11)) +
			uint64(int64(d12)*int64(d12)) +
			uint64(int64(d13)*int64(d13)) +
			uint64(int64(d14)*int64(d14)) +
			uint64(int64(d15)*int64(d15))

		if dist < threshold {
			insertTopKUint64(dist, labels[vectorIndex], bestDistances, bestLabels)
			threshold = bestDistances[topK-1]
		}
	}

	return len(candidateIDs)
}

func selectBucketWindow(
	bucketIndex [][]uint32,
	bucketPrefixSums []uint32,
	amountBucket int,
	hourBucket int,
	dayBucket int,
	tx24hBucket int,
	cfg searchConfig,
) (amountStart int, amountEnd int, hourStart int, hourEnd int, dayStart int, dayEnd int, txStart int, txEnd int, candidateCount int) {
	for radius := 0; radius <= cfg.bucketMaxSearchRadius; radius++ {
		amountStart = bucketRangeStart(amountBucket, dataset.AmountBucketCount, radius)
		amountEnd = bucketRangeEnd(amountBucket, dataset.AmountBucketCount, radius)
		hourStart = bucketRangeStart(hourBucket, dataset.HourBucketCount, radius)
		hourEnd = bucketRangeEnd(hourBucket, dataset.HourBucketCount, radius)
		dayStart = bucketRangeStart(dayBucket, dataset.DayBucketCount, radius)
		dayEnd = bucketRangeEnd(dayBucket, dataset.DayBucketCount, radius)
		txStart = bucketRangeStart(tx24hBucket, dataset.Tx24hBucketCount, radius)
		txEnd = bucketRangeEnd(tx24hBucket, dataset.Tx24hBucketCount, radius)

		if len(bucketPrefixSums) > 0 {
			candidateCount = dataset.CountBucketWindowCandidates(
				bucketPrefixSums,
				amountStart,
				amountEnd,
				hourStart,
				hourEnd,
				dayStart,
				dayEnd,
				txStart,
				txEnd,
			)
		} else if len(bucketIndex) > 0 {
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
		} else {
			candidateCount = 0
		}

		if candidateCount >= cfg.bucketTargetCandidates {
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
