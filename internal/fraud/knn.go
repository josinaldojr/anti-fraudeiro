package fraud

import (
	"math"
	"sort"
	"sync"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

const (
	candidateSeenLimit                    = 4096
	secondaryFallbackMaxPrimaryCandidates = 96
	secondaryFallbackMaxBucketSize        = 256
	maxWindowBuckets                      = 2401
	maxIVFLists                           = 256
	maxIVFBucketCandidates                = 4096
	maxShortlistCandidates                = 256
)

type searchConfig struct {
	bucketTargetCandidates int
	bucketMaxSearchRadius  int
	ivfNProbe              int
}

type searchStats struct {
	processedCandidates int
	windowCandidates    int
	probedLists         int
	bucketsVisited      int
	availableCandidates int
	shortlistCandidates int
	shortlistTruncated  bool
	secondaryFallback   bool
}

type SearchStatsForEval struct {
	ProcessedCandidates int
	WindowCandidates    int
	ProbedLists         int
	BucketsVisited      int
	AvailableCandidates int
	ShortlistCandidates int
	ShortlistTruncated  bool
	SecondaryFallback   bool
}

type knnScratch struct {
	bucketCandidates [maxWindowBuckets]bucketCandidate
	ivfLists         [maxIVFLists]bucketCandidate
	ivfBuckets       [maxIVFBucketCandidates]bucketCandidate
	shortlist        [maxShortlistCandidates]uint32
	seen             [candidateSeenLimit]uint32
}

var knnScratchPool = sync.Pool{
	New: func() any {
		return &knnScratch{}
	},
}

var defaultSearchConfig = searchConfig{
	bucketTargetCandidates: 256,
	bucketMaxSearchRadius:  3,
	ivfNProbe:              4,
}

func SetDefaultSearchConfig(bucketTargetCandidates int, bucketMaxSearchRadius int) {
	if bucketTargetCandidates > 0 {
		defaultSearchConfig.bucketTargetCandidates = bucketTargetCandidates
	}
	if bucketMaxSearchRadius >= 0 && bucketMaxSearchRadius <= 3 {
		defaultSearchConfig.bucketMaxSearchRadius = bucketMaxSearchRadius
	}
	defaultSearchConfig = normalizeSearchConfig(defaultSearchConfig)
}

func SetDefaultIVFNProbe(ivfNProbe int) {
	if ivfNProbe > 0 {
		defaultSearchConfig.ivfNProbe = ivfNProbe
	}
	defaultSearchConfig = normalizeSearchConfig(defaultSearchConfig)
}

func FindTop5(query [14]float32, store *dataset.VectorStore) (fraudCount int) {
	return FindTop5WithStrategy(query, store, BucketStrategyWindow)
}

func FindTop5WithStrategy(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy) (fraudCount int) {
	return findTop5WithConfig(query, store, strategy, defaultSearchConfig)
}

func FindTop5WithStatsForEval(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy) (fraudCount int, stats SearchStatsForEval) {
	fraudCount, internalStats := findTop5WithStats(query, store, strategy, defaultSearchConfig)
	return fraudCount, SearchStatsForEval{
		ProcessedCandidates: internalStats.processedCandidates,
		WindowCandidates:    internalStats.windowCandidates,
		ProbedLists:         internalStats.probedLists,
		BucketsVisited:      internalStats.bucketsVisited,
		AvailableCandidates: internalStats.availableCandidates,
		ShortlistCandidates: internalStats.shortlistCandidates,
		ShortlistTruncated:  internalStats.shortlistTruncated,
		SecondaryFallback:   internalStats.secondaryFallback,
	}
}

func findTop5WithConfig(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy, cfg searchConfig) (fraudCount int) {
	fraudCount, _ = findTop5WithStats(query, store, strategy, cfg)
	return fraudCount
}

func findTop5WithStats(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy, cfg searchConfig) (fraudCount int, stats searchStats) {
	if store == nil || store.Count == 0 {
		return 0, searchStats{}
	}

	cfg = normalizeSearchConfig(cfg)

	if len(store.QuantizedVectors) > 0 {
		return findTop5Quantized(query, store, strategy, cfg)
	}

	return findTop5Float32(query, store, strategy, cfg)
}

func FindTop5Exact(query [14]float32, store *dataset.VectorStore) (fraudCount int) {
	if store == nil || store.Count == 0 {
		return 0
	}

	if len(store.QuantizedVectors) > 0 {
		fraudCount, _ = findTop5Quantized(query, store, "", normalizeSearchConfig(defaultSearchConfig))
		return fraudCount
	}

	fraudCount, _ = findTop5Float32(query, store, "", normalizeSearchConfig(defaultSearchConfig))
	return fraudCount
}

func normalizeSearchConfig(cfg searchConfig) searchConfig {
	if cfg.bucketTargetCandidates < topK {
		cfg.bucketTargetCandidates = topK
	}
	if cfg.bucketTargetCandidates > maxShortlistCandidates {
		cfg.bucketTargetCandidates = maxShortlistCandidates
	}
	if cfg.bucketMaxSearchRadius < 0 {
		cfg.bucketMaxSearchRadius = 0
	}
	if cfg.bucketMaxSearchRadius > 3 {
		cfg.bucketMaxSearchRadius = 3
	}
	if cfg.ivfNProbe < 1 {
		cfg.ivfNProbe = 1
	}
	if cfg.ivfNProbe > maxIVFLists {
		cfg.ivfNProbe = maxIVFLists
	}

	return cfg
}

func findTop5Float32(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy, cfg searchConfig) (fraudCount int, stats searchStats) {
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
		return findTop5Float32Bucketed(query, store, strategy, cfg, bestDistances, bestLabels)
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
	stats.processedCandidates = store.Count

	for _, label := range bestLabels {
		if label == dataset.LabelFraud {
			fraudCount++
		}
	}

	return fraudCount, stats
}

func findTop5Float32Bucketed(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy, cfg searchConfig, bestDistances [topK]float32, bestLabels [topK]byte) (fraudCount int, stats searchStats) {
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
	if strategy == BucketStrategyIVF && len(store.IVFLists) == 0 {
		strategy = BucketStrategyWindow
	}
	amountBucket, hourBucket, dayBucket, tx24hBucket := dataset.BucketCoordinatesFromQuery(q0, q3, q4, q8)
	amountStart, amountEnd, hourStart, hourEnd, dayStart, dayEnd, txStart, txEnd, candidateCount :=
		selectBucketWindow(store.BucketIndex, store.BucketPrefixSums, amountBucket, hourBucket, dayBucket, tx24hBucket, cfg)
	stats.windowCandidates = candidateCount

	trackSeen := len(store.SecondaryBucketIndex) > 0
	scratch := knnScratchPool.Get().(*knnScratch)
	defer knnScratchPool.Put(scratch)
	seenCount := 0
	switch strategy {
	case BucketStrategyIVF:
		ivfStats := collectIVFShortlist(
			scratch.shortlist[:cfg.bucketTargetCandidates],
			scratch.ivfLists[:],
			scratch.ivfBuckets[:],
			store,
			dataset.BuildIVFQueryCoords(q0, q3, q4, q8, q2, q12),
			cfg.ivfNProbe,
		)
		stats.probedLists = ivfStats.probedLists
		stats.bucketsVisited = ivfStats.bucketCount
		stats.availableCandidates = ivfStats.availableCandidates
		stats.shortlistCandidates = ivfStats.shortlistCount
		stats.shortlistTruncated = ivfStats.truncated
		stats.processedCandidates = scanFloatCandidates(
			scratch.shortlist[:ivfStats.shortlistCount],
			vectors,
			labels,
			q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13,
			&bestDistances,
			&bestLabels,
			trackSeen,
			&scratch.seen,
			&seenCount,
		)
	case BucketStrategyOrdered:
		bucketCount, _ := collectOrderedBucketCandidates(
			scratch.bucketCandidates[:],
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
		stats.bucketsVisited = bucketCount
		for bucketIndex := 0; bucketIndex < bucketCount; bucketIndex++ {
			bucketID := scratch.bucketCandidates[bucketIndex].id
			bucketVectors := store.BucketIndex[bucketID]
			processedCandidates += scanFloatCandidates(
				bucketVectors,
				vectors,
				labels,
				q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13,
				&bestDistances,
				&bestLabels,
				trackSeen,
				&scratch.seen,
				&seenCount,
			)
			if processedCandidates >= cfg.bucketTargetCandidates {
				break
			}
		}
		stats.processedCandidates = processedCandidates
	case BucketStrategyShortlist:
		shortlistCount := collectShortlist(
			scratch.shortlist[:cfg.bucketTargetCandidates],
			scratch.bucketCandidates[:],
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
		stats.shortlistCandidates = shortlistCount
		stats.shortlistTruncated = shortlistCount == cfg.bucketTargetCandidates && candidateCount > shortlistCount
		stats.processedCandidates = scanFloatCandidates(
			scratch.shortlist[:shortlistCount],
			vectors,
			labels,
			q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13,
			&bestDistances,
			&bestLabels,
			trackSeen,
			&scratch.seen,
			&seenCount,
		)
	default:
		for amountIndex := amountStart; amountIndex <= amountEnd; amountIndex++ {
			for hourIndex := hourStart; hourIndex <= hourEnd; hourIndex++ {
				for dayIndex := dayStart; dayIndex <= dayEnd; dayIndex++ {
					for txIndex := txStart; txIndex <= txEnd; txIndex++ {
						bucketID := dataset.BucketIDFromCoordinates(amountIndex, hourIndex, dayIndex, txIndex)
						stats.processedCandidates += scanFloatCandidates(
							store.BucketIndex[bucketID],
							vectors,
							labels,
							q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13,
							&bestDistances,
							&bestLabels,
							trackSeen,
							&scratch.seen,
							&seenCount,
						)
						stats.bucketsVisited++
					}
				}
			}
		}
	}

	if trackSeen && stats.processedCandidates < topK && candidateCount <= secondaryFallbackMaxPrimaryCandidates {
		amountBucket2, hourBucket2, dayBucket2, riskBucket2 := dataset.SecondaryBucketCoordinatesFromQuery(q0, q3, q4, q12)
		secondaryBucketID := dataset.SecondaryBucketIDFromCoordinates(amountBucket2, hourBucket2, dayBucket2, riskBucket2)
		secondaryCandidates := store.SecondaryBucketIndex[secondaryBucketID]
		if len(secondaryCandidates) <= secondaryFallbackMaxBucketSize {
			stats.secondaryFallback = true
			stats.processedCandidates += scanFloatSecondaryCandidates(
				secondaryCandidates,
				cfg.bucketTargetCandidates-stats.processedCandidates,
				vectors,
				labels,
				q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13,
				&bestDistances,
				&bestLabels,
				scratch.seen[:seenCount],
				&scratch.seen,
				&seenCount,
			)
		}
	}

	for _, label := range bestLabels {
		if label == dataset.LabelFraud {
			fraudCount++
		}
	}

	return fraudCount, stats
}

func findTop5Quantized(query [14]float32, store *dataset.VectorStore, strategy BucketStrategy, cfg searchConfig) (fraudCount int, stats searchStats) {
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
		return findTop5QuantizedBucketed(store, strategy, cfg, q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13, bestDistances, bestLabels)
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
	stats.processedCandidates = store.Count

	for _, label := range bestLabels {
		if label == dataset.LabelFraud {
			fraudCount++
		}
	}

	return fraudCount, stats
}

func findTop5QuantizedBucketed(
	store *dataset.VectorStore,
	strategy BucketStrategy,
	cfg searchConfig,
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
) (fraudCount int, stats searchStats) {
	vectors := store.QuantizedVectors
	labels := store.Labels
	if strategy == BucketStrategyIVF && len(store.IVFLists) == 0 {
		strategy = BucketStrategyWindow
	}
	amountBucket, hourBucket, dayBucket, tx24hBucket := dataset.BucketCoordinatesFromQuery(
		dataset.DequantizeComponent(uint16(q0)),
		dataset.DequantizeComponent(uint16(q3)),
		dataset.DequantizeComponent(uint16(q4)),
		dataset.DequantizeComponent(uint16(q8)),
	)

	amountStart, amountEnd, hourStart, hourEnd, dayStart, dayEnd, txStart, txEnd, candidateCount :=
		selectBucketWindow(store.BucketIndex, store.BucketPrefixSums, amountBucket, hourBucket, dayBucket, tx24hBucket, cfg)
	stats.windowCandidates = candidateCount

	trackSeen := len(store.SecondaryBucketIndex) > 0
	scratch := knnScratchPool.Get().(*knnScratch)
	defer knnScratchPool.Put(scratch)
	seenCount := 0

	switch strategy {
	case BucketStrategyIVF:
		queryCoords := dataset.BuildIVFQueryCoords(
			dataset.DequantizeComponent(uint16(q0)),
			dataset.DequantizeComponent(uint16(q3)),
			dataset.DequantizeComponent(uint16(q4)),
			dataset.DequantizeComponent(uint16(q8)),
			dataset.DequantizeComponent(uint16(q2)),
			dataset.DequantizeComponent(uint16(q12)),
		)
		ivfStats := collectIVFShortlist(
			scratch.shortlist[:cfg.bucketTargetCandidates],
			scratch.ivfLists[:],
			scratch.ivfBuckets[:],
			store,
			queryCoords,
			cfg.ivfNProbe,
		)
		stats.probedLists = ivfStats.probedLists
		stats.bucketsVisited = ivfStats.bucketCount
		stats.availableCandidates = ivfStats.availableCandidates
		stats.shortlistCandidates = ivfStats.shortlistCount
		stats.shortlistTruncated = ivfStats.truncated
		stats.processedCandidates = scanQuantizedCandidates(
			scratch.shortlist[:ivfStats.shortlistCount],
			vectors,
			labels,
			q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13,
			&bestDistances,
			&bestLabels,
			trackSeen,
			&scratch.seen,
			&seenCount,
		)
	case BucketStrategyOrdered:
		queryAmount := dataset.DequantizeComponent(uint16(q0))
		queryHour := dataset.DequantizeComponent(uint16(q3))
		queryDay := dataset.DequantizeComponent(uint16(q4))
		queryTx24h := dataset.DequantizeComponent(uint16(q8))
		bucketCount, _ := collectOrderedBucketCandidates(
			scratch.bucketCandidates[:],
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
		stats.bucketsVisited = bucketCount
		for bucketIndex := 0; bucketIndex < bucketCount; bucketIndex++ {
			bucketID := scratch.bucketCandidates[bucketIndex].id
			bucketVectors := store.BucketIndex[bucketID]
			processedCandidates += scanQuantizedCandidates(
				bucketVectors,
				vectors,
				labels,
				q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13,
				&bestDistances,
				&bestLabels,
				trackSeen,
				&scratch.seen,
				&seenCount,
			)
			if processedCandidates >= cfg.bucketTargetCandidates {
				break
			}
		}
		stats.processedCandidates = processedCandidates
	case BucketStrategyShortlist:
		shortlistCount := collectShortlist(
			scratch.shortlist[:cfg.bucketTargetCandidates],
			scratch.bucketCandidates[:],
			store.BucketIndex,
			dataset.DequantizeComponent(uint16(q0)),
			dataset.DequantizeComponent(uint16(q3)),
			dataset.DequantizeComponent(uint16(q4)),
			dataset.DequantizeComponent(uint16(q8)),
			amountStart,
			amountEnd,
			hourStart,
			hourEnd,
			dayStart,
			dayEnd,
			txStart,
			txEnd,
		)
		stats.shortlistCandidates = shortlistCount
		stats.shortlistTruncated = shortlistCount == cfg.bucketTargetCandidates && candidateCount > shortlistCount
		stats.processedCandidates = scanQuantizedCandidates(
			scratch.shortlist[:shortlistCount],
			vectors,
			labels,
			q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13,
			&bestDistances,
			&bestLabels,
			trackSeen,
			&scratch.seen,
			&seenCount,
		)
	default:
		for amountIndex := amountStart; amountIndex <= amountEnd; amountIndex++ {
			for hourIndex := hourStart; hourIndex <= hourEnd; hourIndex++ {
				for dayIndex := dayStart; dayIndex <= dayEnd; dayIndex++ {
					for txIndex := txStart; txIndex <= txEnd; txIndex++ {
						bucketID := dataset.BucketIDFromCoordinates(amountIndex, hourIndex, dayIndex, txIndex)
						stats.processedCandidates += scanQuantizedCandidates(
							store.BucketIndex[bucketID],
							vectors,
							labels,
							q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13,
							&bestDistances,
							&bestLabels,
							trackSeen,
							&scratch.seen,
							&seenCount,
						)
						stats.bucketsVisited++
					}
				}
			}
		}
	}

	if trackSeen && stats.processedCandidates < topK && candidateCount <= secondaryFallbackMaxPrimaryCandidates {
		amountBucket2, hourBucket2, dayBucket2, riskBucket2 := dataset.SecondaryBucketCoordinatesFromQuery(
			dataset.DequantizeComponent(uint16(q0)),
			dataset.DequantizeComponent(uint16(q3)),
			dataset.DequantizeComponent(uint16(q4)),
			dataset.DequantizeComponent(uint16(q12)),
		)
		secondaryBucketID := dataset.SecondaryBucketIDFromCoordinates(amountBucket2, hourBucket2, dayBucket2, riskBucket2)
		secondaryCandidates := store.SecondaryBucketIndex[secondaryBucketID]
		if len(secondaryCandidates) <= secondaryFallbackMaxBucketSize {
			stats.secondaryFallback = true
			stats.processedCandidates += scanQuantizedSecondaryCandidates(
				secondaryCandidates,
				cfg.bucketTargetCandidates-stats.processedCandidates,
				vectors,
				labels,
				q0, q1, q2, q3, q4, q5, q6, q7, q8, q9, q10, q11, q12, q13,
				&bestDistances,
				&bestLabels,
				scratch.seen[:seenCount],
				&scratch.seen,
				&seenCount,
			)
		}
	}

	for _, label := range bestLabels {
		if label == dataset.LabelFraud {
			fraudCount++
		}
	}

	return fraudCount, stats
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
	id       int
	distance float32
}

type ivfShortlistStats struct {
	probedLists         int
	bucketCount         int
	availableCandidates int
	shortlistCount      int
	truncated           bool
}

type bucketCandidateSlice []bucketCandidate

func (c bucketCandidateSlice) Len() int {
	return len(c)
}

func (c bucketCandidateSlice) Less(left int, right int) bool {
	if c[left].distance == c[right].distance {
		return c[left].id < c[right].id
	}

	return c[left].distance < c[right].distance
}

func (c bucketCandidateSlice) Swap(left int, right int) {
	c[left], c[right] = c[right], c[left]
}

func collectShortlist(
	shortlist []uint32,
	bucketCandidates []bucketCandidate,
	buckets [][]uint32,
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
) int {
	bucketCount, _ := collectCenteredBucketCandidates(
		bucketCandidates,
		buckets,
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

	shortlistCount := 0
	for candidateIndex := 0; candidateIndex < bucketCount && shortlistCount < len(shortlist); candidateIndex++ {
		bucketVectors := buckets[bucketCandidates[candidateIndex].id]
		remaining := len(shortlist) - shortlistCount
		if len(bucketVectors) > remaining {
			copy(shortlist[shortlistCount:], bucketVectors[:remaining])
			shortlistCount += remaining
			break
		}

		copy(shortlist[shortlistCount:], bucketVectors)
		shortlistCount += len(bucketVectors)
	}

	return shortlistCount
}

func collectIVFShortlist(
	shortlist []uint32,
	listCandidates []bucketCandidate,
	bucketCandidates []bucketCandidate,
	store *dataset.VectorStore,
	queryCoords [dataset.IVFCoarseDimensions]float32,
	nprobe int,
) ivfShortlistStats {
	listCount := collectNearestIVFLists(
		listCandidates,
		store.IVFCentroids,
		queryCoords,
		nprobe,
	)
	if listCount == 0 {
		return ivfShortlistStats{}
	}

	bucketCount, availableCandidates := collectIVFBucketCandidates(
		bucketCandidates,
		store.IVFLists,
		store.BucketIndex,
		store.IVFBucketSummaries,
		queryCoords,
		listCandidates[:listCount],
	)
	if bucketCount == 0 {
		return ivfShortlistStats{probedLists: listCount}
	}

	shortlistCount := 0
	for candidateIndex := 0; candidateIndex < bucketCount && shortlistCount < len(shortlist); candidateIndex++ {
		bucketID := bucketCandidates[candidateIndex].id
		bucketVectors := store.BucketIndex[bucketID]
		remaining := len(shortlist) - shortlistCount
		if len(bucketVectors) > remaining {
			copy(shortlist[shortlistCount:], bucketVectors[:remaining])
			shortlistCount += remaining
			break
		}

		copy(shortlist[shortlistCount:], bucketVectors)
		shortlistCount += len(bucketVectors)
	}

	return ivfShortlistStats{
		probedLists:         listCount,
		bucketCount:         bucketCount,
		availableCandidates: availableCandidates,
		shortlistCount:      shortlistCount,
		truncated:           availableCandidates > shortlistCount,
	}
}

func collectNearestIVFLists(
	candidates []bucketCandidate,
	centroids []float32,
	queryCoords [dataset.IVFCoarseDimensions]float32,
	nprobe int,
) int {
	if len(centroids) == 0 || len(candidates) == 0 {
		return 0
	}

	if nprobe > len(candidates) {
		nprobe = len(candidates)
	}
	if totalLists := len(centroids) / dataset.IVFCoarseDimensions; nprobe > totalLists {
		nprobe = totalLists
	}
	if nprobe <= 0 {
		return 0
	}

	count := 0
	for listIndex, baseOffset := 0, 0; baseOffset < len(centroids); listIndex, baseOffset = listIndex+1, baseOffset+dataset.IVFCoarseDimensions {
		distance := datasetIVFDistance(queryCoords, centroids, baseOffset)

		if count < nprobe {
			candidates[count] = bucketCandidate{id: listIndex, distance: distance}
			count++
			if count == nprobe {
				sort.Sort(sort.Reverse(bucketCandidateSlice(candidates[:count])))
			}
			continue
		}

		if distance >= candidates[0].distance {
			continue
		}

		candidates[0] = bucketCandidate{id: listIndex, distance: distance}
		sort.Sort(sort.Reverse(bucketCandidateSlice(candidates[:count])))
	}

	sort.Sort(bucketCandidateSlice(candidates[:count]))
	return count
}

func collectIVFBucketCandidates(
	candidates []bucketCandidate,
	lists [][]uint32,
	bucketIndex [][]uint32,
	bucketSummaries []float32,
	queryCoords [dataset.IVFCoarseDimensions]float32,
	selectedLists []bucketCandidate,
) (int, int) {
	if len(candidates) == 0 {
		return 0, 0
	}

	count := 0
	availableCandidates := 0
	for _, listCandidate := range selectedLists {
		for _, bucketID := range lists[listCandidate.id] {
			availableCandidates += len(bucketIndex[int(bucketID)])
			baseOffset := int(bucketID) * dataset.IVFCoarseDimensions
			distance := datasetIVFDistance(queryCoords, bucketSummaries, baseOffset)

			if count < len(candidates) {
				candidates[count] = bucketCandidate{id: int(bucketID), distance: distance}
				count++
				continue
			}

			worstIndex := 0
			worstDistance := candidates[0].distance
			for index := 1; index < count; index++ {
				if candidates[index].distance > worstDistance {
					worstIndex = index
					worstDistance = candidates[index].distance
				}
			}
			if distance < worstDistance {
				candidates[worstIndex] = bucketCandidate{id: int(bucketID), distance: distance}
			}
		}
	}

	sort.Sort(bucketCandidateSlice(candidates[:count]))
	return count, availableCandidates
}

func datasetIVFDistance(queryCoords [dataset.IVFCoarseDimensions]float32, values []float32, baseOffset int) float32 {
	d0 := queryCoords[0] - values[baseOffset]
	d1 := queryCoords[1] - values[baseOffset+1]
	d2 := queryCoords[2] - values[baseOffset+2]
	d3 := queryCoords[3] - values[baseOffset+3]
	d4 := queryCoords[4] - values[baseOffset+4]
	d5 := queryCoords[5] - values[baseOffset+5]
	return d0*d0 + d1*d1 + d2*d2 + d3*d3 + d4*d4 + d5*d5
}

func scanFloatCandidates(
	candidateIDs []uint32,
	vectors []float32,
	labels []byte,
	q0 float32,
	q1 float32,
	q2 float32,
	q3 float32,
	q4 float32,
	q5 float32,
	q6 float32,
	q7 float32,
	q8 float32,
	q9 float32,
	q10 float32,
	q11 float32,
	q12 float32,
	q13 float32,
	bestDistances *[topK]float32,
	bestLabels *[topK]byte,
	trackSeen bool,
	seen *[candidateSeenLimit]uint32,
	seenCount *int,
) int {
	for _, vectorIndex := range candidateIDs {
		if trackSeen && *seenCount < candidateSeenLimit {
			seen[*seenCount] = vectorIndex
			*seenCount = *seenCount + 1
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
			insertTopK(distance, labels[vectorIndex], bestDistances, bestLabels)
		}
	}

	return len(candidateIDs)
}

func scanFloatSecondaryCandidates(
	candidateIDs []uint32,
	limit int,
	vectors []float32,
	labels []byte,
	q0 float32,
	q1 float32,
	q2 float32,
	q3 float32,
	q4 float32,
	q5 float32,
	q6 float32,
	q7 float32,
	q8 float32,
	q9 float32,
	q10 float32,
	q11 float32,
	q12 float32,
	q13 float32,
	bestDistances *[topK]float32,
	bestLabels *[topK]byte,
	existing []uint32,
	seen *[candidateSeenLimit]uint32,
	seenCount *int,
) int {
	if limit <= 0 {
		return 0
	}

	appended := 0
	for _, vectorIndex := range candidateIDs {
		if appended >= limit || containsCandidateID(existing, vectorIndex) {
			continue
		}
		if *seenCount < candidateSeenLimit {
			seen[*seenCount] = vectorIndex
			*seenCount = *seenCount + 1
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
			insertTopK(distance, labels[vectorIndex], bestDistances, bestLabels)
		}
		appended++
	}

	return appended
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
	bestDistances *[topK]uint64,
	bestLabels *[topK]byte,
	trackSeen bool,
	seen *[candidateSeenLimit]uint32,
	seenCount *int,
) int {
	for _, vectorIndex := range candidateIDs {
		if trackSeen && *seenCount < candidateSeenLimit {
			seen[*seenCount] = vectorIndex
			*seenCount = *seenCount + 1
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
			insertTopKUint64(distance, labels[vectorIndex], bestDistances, bestLabels)
		}
	}

	return len(candidateIDs)
}

func scanQuantizedSecondaryCandidates(
	candidateIDs []uint32,
	limit int,
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
	bestDistances *[topK]uint64,
	bestLabels *[topK]byte,
	existing []uint32,
	seen *[candidateSeenLimit]uint32,
	seenCount *int,
) int {
	if limit <= 0 {
		return 0
	}

	appended := 0
	for _, vectorIndex := range candidateIDs {
		if appended >= limit || containsCandidateID(existing, vectorIndex) {
			continue
		}
		if *seenCount < candidateSeenLimit {
			seen[*seenCount] = vectorIndex
			*seenCount = *seenCount + 1
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
			insertTopKUint64(distance, labels[vectorIndex], bestDistances, bestLabels)
		}
		appended++
	}

	return appended
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
						distance: coarseLowerBound(
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

	sort.Sort(bucketCandidateSlice(candidates[:count]))

	return count, candidateCount
}

func collectCenteredBucketCandidates(
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
						distance: bucketCenterDistance(
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

	sort.Sort(bucketCandidateSlice(candidates[:count]))

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

func bucketCenterDistance(
	queryAmount float32,
	queryHour float32,
	queryDay float32,
	queryTx24h float32,
	amountBucket int,
	hourBucket int,
	dayBucket int,
	txBucket int,
) float32 {
	amountCenter := bucketCenter(amountBucket, dataset.AmountBucketCount)
	hourCenter := bucketCenter(hourBucket, dataset.HourBucketCount)
	dayCenter := bucketCenter(dayBucket, dataset.DayBucketCount)
	txCenter := bucketCenter(txBucket, dataset.Tx24hBucketCount)

	amountDistance := queryAmount - amountCenter
	hourDistance := queryHour - hourCenter
	dayDistance := queryDay - dayCenter
	txDistance := queryTx24h - txCenter

	return amountDistance*amountDistance +
		hourDistance*hourDistance +
		dayDistance*dayDistance +
		txDistance*txDistance
}

func bucketCenter(bucket int, bucketCount int) float32 {
	return (float32(bucket) + 0.5) / float32(bucketCount)
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
		} else {
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
