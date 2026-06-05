package fraud

import (
	"sync"
	"unsafe"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

const (
	maxWindowBuckets               = 4913 // 17^3 max (radius 2 across 3 dims, or ~radius 1 across 6D = 729)
	exactFallbackDistanceThreshold = 300000000
	maxExpandRadius                = 5 // covers all cells (11 of 12 amount buckets, all 8 of others)
)

type searchConfig struct {
	bucketTargetCandidates    int
	bucketMaxSearchRadius     int
	bucketEarlyExitCandidates int
}

type searchStats struct {
	processedCandidates int
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
	bucketTargetCandidates:    512,
	bucketMaxSearchRadius:     3,
	bucketEarlyExitCandidates: 256,
}

func SetDefaultSearchConfig(bucketTargetCandidates int, bucketMaxSearchRadius int) {
	if bucketTargetCandidates > 0 {
		defaultSearchConfig.bucketTargetCandidates = bucketTargetCandidates
		defaultSearchConfig.bucketEarlyExitCandidates = bucketTargetCandidates / 2
	}
	if bucketMaxSearchRadius >= 0 && bucketMaxSearchRadius <= 5 {
		defaultSearchConfig.bucketMaxSearchRadius = bucketMaxSearchRadius
	}
}

func FindTop5(query [16]float32, store *dataset.VectorStore) (fraudCount int) {
	fraudCount, _, stats := findTop5WithStats(query, store, defaultSearchConfig)
	_ = stats
	return fraudCount
}

func needsExactFallback(fraudCount int, bestDistances [topK]uint64) bool {
	return fraudCount == 2 || bestDistances[topK-1] > exactFallbackDistanceThreshold
}

func exactScan(query [16]float32, store *dataset.VectorStore) int {
	return exactScanSIMD(query, store)
}

func expandedScan(query [16]float32, store *dataset.VectorStore, bestDistances [topK]uint64) int {
	if store == nil || store.Count == 0 || len(store.QuantizedVectors) == 0 || len(store.BucketMeta) == 0 {
		return exactScanSIMD(query, store)
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

	var bestLabels [topK]byte
	vectors := store.QuantizedVectors
	labels := store.Labels
	meta := store.BucketMeta

	amountBucket, minutesBucket, kmHomeBucket, txCountBucket, amountVsAvgBucket, mccRiskBucket :=
		dataset.BucketCoordinatesFromQuery(
			query[0], query[5], query[7], query[8], query[2], query[12],
		)

	for radius := defaultSearchConfig.bucketMaxSearchRadius + 1; radius <= maxExpandRadius; radius++ {
		aStart := bucketRangeStart(amountBucket, dataset.AmountBucketCount, radius)
		aEnd := bucketRangeEnd(amountBucket, dataset.AmountBucketCount, radius)
		mStart := bucketRangeStart(minutesBucket, dataset.MinutesSinceLastCount, radius)
		mEnd := bucketRangeEnd(minutesBucket, dataset.MinutesSinceLastCount, radius)
		kStart := bucketRangeStart(kmHomeBucket, dataset.KMFromHomeCount, radius)
		kEnd := bucketRangeEnd(kmHomeBucket, dataset.KMFromHomeCount, radius)
		tStart := bucketRangeStart(txCountBucket, dataset.Tx24hBucketCount, radius)
		tEnd := bucketRangeEnd(txCountBucket, dataset.Tx24hBucketCount, radius)
		vStart := bucketRangeStart(amountVsAvgBucket, dataset.AmountVsAvgBucketCount, radius)
		vEnd := bucketRangeEnd(amountVsAvgBucket, dataset.AmountVsAvgBucketCount, radius)
		rStart := bucketRangeStart(mccRiskBucket, dataset.MCCRiskBucketCount, radius)
		rEnd := bucketRangeEnd(mccRiskBucket, dataset.MCCRiskBucketCount, radius)

		for ai := aStart; ai <= aEnd; ai++ {
			for mi := mStart; mi <= mEnd; mi++ {
				for ki := kStart; ki <= kEnd; ki++ {
					for ti := tStart; ti <= tEnd; ti++ {
						for vi := vStart; vi <= vEnd; vi++ {
							for ri := rStart; ri <= rEnd; ri++ {
								ad := ai - amountBucket
								if ad < 0 {
									ad = -ad
								}
								md := mi - minutesBucket
								if md < 0 {
									md = -md
								}
								kd := ki - kmHomeBucket
								if kd < 0 {
									kd = -kd
								}
								td := ti - txCountBucket
								if td < 0 {
									td = -td
								}
								vd := vi - amountVsAvgBucket
								if vd < 0 {
									vd = -vd
								}
								rd := ri - mccRiskBucket
								if rd < 0 {
									rd = -rd
								}
								maxDist := ad
								if md > maxDist {
									maxDist = md
								}
								if kd > maxDist {
									maxDist = kd
								}
								if td > maxDist {
									maxDist = td
								}
								if vd > maxDist {
									maxDist = vd
								}
								if rd > maxDist {
									maxDist = rd
								}
								if maxDist != radius {
									continue
								}

								cellID := dataset.BucketIDFromCoordinates(ai, mi, ki, ti, vi, ri)
								m := meta[cellID]
								if m.Count == 0 {
									continue
								}

								offset := int(m.Offset) * dataset.VectorSize
								count := int(m.Count)

								scanQuantizedContiguousSIMD(
									vectors[offset:offset+count*dataset.VectorSize],
									labels[m.Offset:m.Offset+uint32(m.Count)],
									q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15,
									&bestDistances,
									&bestLabels,
								)
							}
						}
					}
				}
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

func exactScanScalar(query [16]float32, store *dataset.VectorStore) int {
	var exactBestDistances [topK]uint64
	var exactBestLabels [topK]byte
	for i := range exactBestDistances {
		exactBestDistances[i] = ^uint64(0)
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

	scanQuantizedContiguous(
		store.QuantizedVectors,
		store.Labels,
		q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15,
		&exactBestDistances,
		&exactBestLabels,
	)

	exactFraudCount := 0
	for _, label := range exactBestLabels {
		if label == dataset.LabelFraud {
			exactFraudCount++
		}
	}
	return exactFraudCount
}

func findTop5WithStats(query [16]float32, store *dataset.VectorStore, cfg searchConfig) (fraudCount int, bestDistances [topK]uint64, stats searchStats) {
	if store == nil || store.Count == 0 {
		var emptyDists [topK]uint64
		for i := range emptyDists {
			emptyDists[i] = ^uint64(0)
		}
		return 0, emptyDists, searchStats{}
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

	var bestLabels [topK]byte

	for index := range bestDistances {
		bestDistances[index] = ^uint64(0)
	}

	amountBucket, minutesBucket, kmHomeBucket, txCountBucket, amountVsAvgBucket, mccRiskBucket :=
		dataset.BucketCoordinatesFromQuery(
			query[0], query[5], query[7], query[8], query[2], query[12],
		)

	if len(store.BucketMeta) == dataset.BucketIndexCount {
		meta := store.BucketMeta
		for radius := 0; radius <= cfg.bucketMaxSearchRadius; radius++ {
			aStart := bucketRangeStart(amountBucket, dataset.AmountBucketCount, radius)
			aEnd := bucketRangeEnd(amountBucket, dataset.AmountBucketCount, radius)
			mStart := bucketRangeStart(minutesBucket, dataset.MinutesSinceLastCount, radius)
			mEnd := bucketRangeEnd(minutesBucket, dataset.MinutesSinceLastCount, radius)
			kStart := bucketRangeStart(kmHomeBucket, dataset.KMFromHomeCount, radius)
			kEnd := bucketRangeEnd(kmHomeBucket, dataset.KMFromHomeCount, radius)
			tStart := bucketRangeStart(txCountBucket, dataset.Tx24hBucketCount, radius)
			tEnd := bucketRangeEnd(txCountBucket, dataset.Tx24hBucketCount, radius)
			vStart := bucketRangeStart(amountVsAvgBucket, dataset.AmountVsAvgBucketCount, radius)
			vEnd := bucketRangeEnd(amountVsAvgBucket, dataset.AmountVsAvgBucketCount, radius)
			rStart := bucketRangeStart(mccRiskBucket, dataset.MCCRiskBucketCount, radius)
			rEnd := bucketRangeEnd(mccRiskBucket, dataset.MCCRiskBucketCount, radius)

			for ai := aStart; ai <= aEnd; ai++ {
				for mi := mStart; mi <= mEnd; mi++ {
					for ki := kStart; ki <= kEnd; ki++ {
						for ti := tStart; ti <= tEnd; ti++ {
							for vi := vStart; vi <= vEnd; vi++ {
								for ri := rStart; ri <= rEnd; ri++ {
									cellID := dataset.BucketIDFromCoordinates(ai, mi, ki, ti, vi, ri)
									m := meta[cellID]
									if m.Count == 0 {
										continue
									}

									offset := int(m.Offset) * dataset.VectorSize
									count := int(m.Count)

									stats.processedCandidates += scanQuantizedContiguousSIMD(
										vectors[offset:offset+count*dataset.VectorSize],
										labels[m.Offset:m.Offset+uint32(m.Count)],
										q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15,
										&bestDistances,
										&bestLabels,
									)
									stats.bucketsVisited++
								}
							}
						}
					}
				}
			}

			earlyExit := cfg.bucketEarlyExitCandidates
			if earlyExit <= 0 {
				earlyExit = 128
			}
			if stats.processedCandidates >= cfg.bucketTargetCandidates || (radius >= 2 && stats.processedCandidates >= earlyExit) {
				break
			}
		}
	} else if len(store.BucketIndex) > 0 {
		bucketIndex := store.BucketIndex
		for radius := 0; radius <= cfg.bucketMaxSearchRadius; radius++ {
			aStart := bucketRangeStart(amountBucket, dataset.AmountBucketCount, radius)
			aEnd := bucketRangeEnd(amountBucket, dataset.AmountBucketCount, radius)
			mStart := bucketRangeStart(minutesBucket, dataset.MinutesSinceLastCount, radius)
			mEnd := bucketRangeEnd(minutesBucket, dataset.MinutesSinceLastCount, radius)
			kStart := bucketRangeStart(kmHomeBucket, dataset.KMFromHomeCount, radius)
			kEnd := bucketRangeEnd(kmHomeBucket, dataset.KMFromHomeCount, radius)
			tStart := bucketRangeStart(txCountBucket, dataset.Tx24hBucketCount, radius)
			tEnd := bucketRangeEnd(txCountBucket, dataset.Tx24hBucketCount, radius)
			vStart := bucketRangeStart(amountVsAvgBucket, dataset.AmountVsAvgBucketCount, radius)
			vEnd := bucketRangeEnd(amountVsAvgBucket, dataset.AmountVsAvgBucketCount, radius)
			rStart := bucketRangeStart(mccRiskBucket, dataset.MCCRiskBucketCount, radius)
			rEnd := bucketRangeEnd(mccRiskBucket, dataset.MCCRiskBucketCount, radius)

			for ai := aStart; ai <= aEnd; ai++ {
				for mi := mStart; mi <= mEnd; mi++ {
					for ki := kStart; ki <= kEnd; ki++ {
						for ti := tStart; ti <= tEnd; ti++ {
							for vi := vStart; vi <= vEnd; vi++ {
								for ri := rStart; ri <= rEnd; ri++ {
									cellID := dataset.BucketIDFromCoordinates(ai, mi, ki, ti, vi, ri)
									stats.processedCandidates += scanQuantizedCandidates(
										bucketIndex[cellID],
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
				}
			}

			earlyExit := cfg.bucketEarlyExitCandidates
			if earlyExit <= 0 {
				earlyExit = 128
			}
			if stats.processedCandidates >= cfg.bucketTargetCandidates || (radius >= 2 && stats.processedCandidates >= earlyExit) {
				break
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

	return fraudCount, bestDistances, stats
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
	q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15 int32,
	bestDistances *[topK]uint64,
	bestLabels *[topK]byte,
) int {
	threshold := bestDistances[topK-1]
	for _, vectorIndex := range candidateIDs {
		baseOffset := int(vectorIndex) * dataset.VectorSize
		ptr := (*[16]uint16)(unsafe.Pointer(&vectors[baseOffset]))

		d0 := q0 - int32(ptr[0])
		d1 := q1 - int32(ptr[1])
		d2 := q2 - int32(ptr[2])
		d3 := q3 - int32(ptr[3])

		dist := uint64(int64(d0)*int64(d0)) +
			uint64(int64(d1)*int64(d1)) +
			uint64(int64(d2)*int64(d2)) +
			uint64(int64(d3)*int64(d3))

		if dist >= threshold {
			continue
		}

		d4 := q4 - int32(ptr[4])
		d5 := q5 - int32(ptr[5])
		d6 := q6 - int32(ptr[6])
		d7 := q7 - int32(ptr[7])

		dist += uint64(int64(d4)*int64(d4)) +
			uint64(int64(d5)*int64(d5)) +
			uint64(int64(d6)*int64(d6)) +
			uint64(int64(d7)*int64(d7))

		if dist >= threshold {
			continue
		}

		d8 := q8 - int32(ptr[8])
		d9 := q9 - int32(ptr[9])
		d10 := q10 - int32(ptr[10])
		d11 := q11 - int32(ptr[11])

		dist += uint64(int64(d8)*int64(d8)) +
			uint64(int64(d9)*int64(d9)) +
			uint64(int64(d10)*int64(d10)) +
			uint64(int64(d11)*int64(d11))

		if dist >= threshold {
			continue
		}

		d12 := q12 - int32(ptr[12])
		d13 := q13 - int32(ptr[13])
		d14 := q14 - int32(ptr[14])
		d15 := q15 - int32(ptr[15])

		dist += uint64(int64(d12)*int64(d12)) +
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
	amountBucket, minutesBucket, kmHomeBucket, tx24hBucket, amountVsAvgBucket, mccRiskBucket int,
	cfg searchConfig,
) (aStart, aEnd, mStart, mEnd, kStart, kEnd, tStart, tEnd, vStart, vEnd, rStart, rEnd int, candidateCount int) {
	for radius := 0; radius <= cfg.bucketMaxSearchRadius; radius++ {
		aStart = bucketRangeStart(amountBucket, dataset.AmountBucketCount, radius)
		aEnd = bucketRangeEnd(amountBucket, dataset.AmountBucketCount, radius)
		mStart = bucketRangeStart(minutesBucket, dataset.MinutesSinceLastCount, radius)
		mEnd = bucketRangeEnd(minutesBucket, dataset.MinutesSinceLastCount, radius)
		kStart = bucketRangeStart(kmHomeBucket, dataset.KMFromHomeCount, radius)
		kEnd = bucketRangeEnd(kmHomeBucket, dataset.KMFromHomeCount, radius)
		tStart = bucketRangeStart(tx24hBucket, dataset.Tx24hBucketCount, radius)
		tEnd = bucketRangeEnd(tx24hBucket, dataset.Tx24hBucketCount, radius)
		vStart = bucketRangeStart(amountVsAvgBucket, dataset.AmountVsAvgBucketCount, radius)
		vEnd = bucketRangeEnd(amountVsAvgBucket, dataset.AmountVsAvgBucketCount, radius)
		rStart = bucketRangeStart(mccRiskBucket, dataset.MCCRiskBucketCount, radius)
		rEnd = bucketRangeEnd(mccRiskBucket, dataset.MCCRiskBucketCount, radius)

		if len(bucketPrefixSums) > 0 {
			candidateCount = dataset.CountBucketWindowCandidates(
				bucketPrefixSums,
				aStart, aEnd, mStart, mEnd, kStart, kEnd, tStart, tEnd, vStart, vEnd, rStart, rEnd,
			)
		} else if len(bucketIndex) > 0 {
			candidateCount = 0
			for ai := aStart; ai <= aEnd; ai++ {
				for mi := mStart; mi <= mEnd; mi++ {
					for ki := kStart; ki <= kEnd; ki++ {
						for ti := tStart; ti <= tEnd; ti++ {
							for vi := vStart; vi <= vEnd; vi++ {
								for ri := rStart; ri <= rEnd; ri++ {
									candidateCount += len(bucketIndex[dataset.BucketIDFromCoordinates(ai, mi, ki, ti, vi, ri)])
								}
							}
						}
					}
				}
			}
		}

		earlyExit := cfg.bucketEarlyExitCandidates
		if earlyExit <= 0 {
			earlyExit = 128
		}
		if candidateCount >= cfg.bucketTargetCandidates || (radius >= 2 && candidateCount >= earlyExit) {
			return
		}
	}

	return
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

func scanQuantizedContiguous(
	vectors []uint16,
	labels []byte,
	q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, q14, q15 int32,
	bestDistances *[topK]uint64,
	bestLabels *[topK]byte,
) int {
	count := len(labels)
	if count == 0 {
		return 0
	}

	threshold := bestDistances[topK-1]

	for i := 0; i < count; i++ {
		baseOffset := i * dataset.VectorSize
		ptr := (*[16]uint16)(unsafe.Pointer(&vectors[baseOffset]))

		d0 := q0 - int32(ptr[0])
		d1 := q1 - int32(ptr[1])
		d2 := q2 - int32(ptr[2])
		d3 := q3 - int32(ptr[3])

		dist := uint64(int64(d0)*int64(d0)) +
			uint64(int64(d1)*int64(d1)) +
			uint64(int64(d2)*int64(d2)) +
			uint64(int64(d3)*int64(d3))

		if dist >= threshold {
			continue
		}

		d4 := q4 - int32(ptr[4])
		d5 := q5 - int32(ptr[5])
		d6 := q6 - int32(ptr[6])
		d7 := q7 - int32(ptr[7])

		dist += uint64(int64(d4)*int64(d4)) +
			uint64(int64(d5)*int64(d5)) +
			uint64(int64(d6)*int64(d6)) +
			uint64(int64(d7)*int64(d7))

		if dist >= threshold {
			continue
		}

		d8 := q8 - int32(ptr[8])
		d9 := q9 - int32(ptr[9])
		d10 := q10 - int32(ptr[10])
		d11 := q11 - int32(ptr[11])

		dist += uint64(int64(d8)*int64(d8)) +
			uint64(int64(d9)*int64(d9)) +
			uint64(int64(d10)*int64(d10)) +
			uint64(int64(d11)*int64(d11))

		if dist >= threshold {
			continue
		}

		d12 := q12 - int32(ptr[12])
		d13 := q13 - int32(ptr[13])
		d14 := q14 - int32(ptr[14])
		d15 := q15 - int32(ptr[15])

		dist += uint64(int64(d12)*int64(d12)) +
			uint64(int64(d13)*int64(d13)) +
			uint64(int64(d14)*int64(d14)) +
			uint64(int64(d15)*int64(d15))

		if dist < threshold {
			insertTopKUint64(dist, labels[i], bestDistances, bestLabels)
			threshold = bestDistances[topK-1]
		}
	}

	return count
}
