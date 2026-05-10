package main

import (
	"encoding/json"
	"flag"
	"log"
	"path/filepath"
	"time"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
	"github.com/josinaldojr/anti-fraudeiro/internal/fraud"
)

type evaluationSummary struct {
	ReferencesCount int               `json:"references_count"`
	Queries         int               `json:"queries"`
	ExactDuration   string            `json:"exact_duration"`
	IVF             ivfIndexSummary   `json:"ivf"`
	Strategies      []strategySummary `json:"strategies"`
}

type ivfIndexSummary struct {
	ListCount         int     `json:"list_count"`
	NonEmptyLists     int     `json:"non_empty_lists"`
	AvgBucketsPerList float64 `json:"avg_buckets_per_list"`
	MaxBucketsPerList int     `json:"max_buckets_per_list"`
	AvgVectorsPerList float64 `json:"avg_vectors_per_list"`
	MaxVectorsPerList int     `json:"max_vectors_per_list"`
	AvgBucketSpread   float64 `json:"avg_bucket_spread"`
	MaxBucketSpread   float64 `json:"max_bucket_spread"`
}

type strategySummary struct {
	Name                string  `json:"name"`
	Duration            string  `json:"duration"`
	FraudCountMatches   int     `json:"fraud_count_matches"`
	FraudCountMatchRate float64 `json:"fraud_count_match_rate"`
	DecisionMatches     int     `json:"decision_matches"`
	DecisionMatchRate   float64 `json:"decision_match_rate"`
	AvgProcessed        float64 `json:"avg_processed_candidates"`
	MaxProcessed        int     `json:"max_processed_candidates"`
	AvgWindowCandidates float64 `json:"avg_window_candidates"`
	MaxWindowCandidates int     `json:"max_window_candidates"`
	AvgProbedLists      float64 `json:"avg_probed_lists"`
	MaxProbedLists      int     `json:"max_probed_lists"`
	AvgBucketsVisited   float64 `json:"avg_buckets_visited"`
	MaxBucketsVisited   int     `json:"max_buckets_visited"`
	AvgAvailable        float64 `json:"avg_available_candidates"`
	MaxAvailable        int     `json:"max_available_candidates"`
	AvgShortlist        float64 `json:"avg_shortlist_candidates"`
	MaxShortlist        int     `json:"max_shortlist_candidates"`
	ShortlistTruncated  int     `json:"shortlist_truncated_queries"`
	SecondaryFallbacks  int     `json:"secondary_fallback_queries"`
}

func main() {
	inputPath := flag.String("input", filepath.Join("resources", dataset.BinaryReferenceFile), "path to the references dataset")
	queryCount := flag.Int("queries", 2000, "number of dataset vectors to sample as evaluation queries")
	ivfListCount := flag.Int("ivf-lists", 128, "number of IVF coarse lists")
	ivfNProbe := flag.Int("ivf-nprobe", 4, "number of IVF lists to probe")
	targetCandidates := flag.Int("target", 256, "target candidate count for bucketed strategies")
	maxRadius := flag.Int("radius", 3, "max search radius for window-like strategies")
	flag.Parse()

	store, err := dataset.LoadVectorStore(*inputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	dataset.BuildIVFIndex(store, *ivfListCount)
	fraud.SetDefaultSearchConfig(*targetCandidates, *maxRadius)
	fraud.SetDefaultIVFNProbe(*ivfNProbe)

	totalQueries := min(max(*queryCount, 1), store.Count)
	queryIndices := sampleQueryIndices(store.Count, totalQueries)

	exactCounts := make([]int, totalQueries)
	exactStart := time.Now()
	for index, queryIndex := range queryIndices {
		query := queryVector(store, queryIndex)
		exactCounts[index] = fraud.FindTop5Exact(query, store)
	}
	exactDuration := time.Since(exactStart)

	strategies := []fraud.BucketStrategy{
		fraud.BucketStrategyWindow,
		fraud.BucketStrategyOrdered,
		fraud.BucketStrategyIVF,
	}

	summaries := make([]strategySummary, 0, len(strategies))
	for _, strategy := range strategies {
		matchCount := 0
		decisionMatchCount := 0
		totalProcessed := 0
		maxProcessed := 0
		totalWindowCandidates := 0
		maxWindowCandidates := 0
		totalProbedLists := 0
		maxProbedLists := 0
		totalBucketsVisited := 0
		maxBucketsVisited := 0
		totalAvailable := 0
		maxAvailable := 0
		totalShortlist := 0
		maxShortlist := 0
		truncatedCount := 0
		secondaryFallbacks := 0
		start := time.Now()
		for index, queryIndex := range queryIndices {
			query := queryVector(store, queryIndex)
			fraudCount, stats := fraud.FindTop5WithStatsForEval(query, store, strategy)
			if fraudCount == exactCounts[index] {
				matchCount++
			}
			if approvedFromFraudCount(fraudCount) == approvedFromFraudCount(exactCounts[index]) {
				decisionMatchCount++
			}
			totalProcessed += stats.ProcessedCandidates
			if stats.ProcessedCandidates > maxProcessed {
				maxProcessed = stats.ProcessedCandidates
			}
			totalWindowCandidates += stats.WindowCandidates
			if stats.WindowCandidates > maxWindowCandidates {
				maxWindowCandidates = stats.WindowCandidates
			}
			totalProbedLists += stats.ProbedLists
			if stats.ProbedLists > maxProbedLists {
				maxProbedLists = stats.ProbedLists
			}
			totalBucketsVisited += stats.BucketsVisited
			if stats.BucketsVisited > maxBucketsVisited {
				maxBucketsVisited = stats.BucketsVisited
			}
			totalAvailable += stats.AvailableCandidates
			if stats.AvailableCandidates > maxAvailable {
				maxAvailable = stats.AvailableCandidates
			}
			totalShortlist += stats.ShortlistCandidates
			if stats.ShortlistCandidates > maxShortlist {
				maxShortlist = stats.ShortlistCandidates
			}
			if stats.ShortlistTruncated {
				truncatedCount++
			}
			if stats.SecondaryFallback {
				secondaryFallbacks++
			}
		}

		duration := time.Since(start)
		summaries = append(summaries, strategySummary{
			Name:                string(strategy),
			Duration:            duration.String(),
			FraudCountMatches:   matchCount,
			FraudCountMatchRate: float64(matchCount) / float64(totalQueries),
			DecisionMatches:     decisionMatchCount,
			DecisionMatchRate:   float64(decisionMatchCount) / float64(totalQueries),
			AvgProcessed:        float64(totalProcessed) / float64(totalQueries),
			MaxProcessed:        maxProcessed,
			AvgWindowCandidates: float64(totalWindowCandidates) / float64(totalQueries),
			MaxWindowCandidates: maxWindowCandidates,
			AvgProbedLists:      float64(totalProbedLists) / float64(totalQueries),
			MaxProbedLists:      maxProbedLists,
			AvgBucketsVisited:   float64(totalBucketsVisited) / float64(totalQueries),
			MaxBucketsVisited:   maxBucketsVisited,
			AvgAvailable:        float64(totalAvailable) / float64(totalQueries),
			MaxAvailable:        maxAvailable,
			AvgShortlist:        float64(totalShortlist) / float64(totalQueries),
			MaxShortlist:        maxShortlist,
			ShortlistTruncated:  truncatedCount,
			SecondaryFallbacks:  secondaryFallbacks,
		})
	}

	summary := evaluationSummary{
		ReferencesCount: store.Count,
		Queries:         totalQueries,
		ExactDuration:   exactDuration.String(),
		IVF:             summarizeIVFIndex(store),
		Strategies:      summaries,
	}

	encoder := json.NewEncoder(log.Writer())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		log.Fatal(err)
	}
}

func summarizeIVFIndex(store *dataset.VectorStore) ivfIndexSummary {
	if len(store.IVFLists) == 0 {
		return ivfIndexSummary{}
	}

	totalBuckets := 0
	totalVectors := 0
	maxBuckets := 0
	maxVectors := 0
	nonEmptyLists := 0
	totalSpread := 0.0
	maxSpread := 0.0

	for listIndex, list := range store.IVFLists {
		if len(list) == 0 {
			continue
		}
		nonEmptyLists++
		totalBuckets += len(list)
		if len(list) > maxBuckets {
			maxBuckets = len(list)
		}

		listVectors := 0
		for _, bucketID := range list {
			listVectors += len(store.BucketIndex[int(bucketID)])
		}
		totalVectors += listVectors
		if listVectors > maxVectors {
			maxVectors = listVectors
		}

		listSpread := computeIVFListSpread(store, listIndex, list)
		totalSpread += listSpread
		if listSpread > maxSpread {
			maxSpread = listSpread
		}
	}

	if nonEmptyLists == 0 {
		return ivfIndexSummary{ListCount: len(store.IVFLists)}
	}

	return ivfIndexSummary{
		ListCount:         len(store.IVFLists),
		NonEmptyLists:     nonEmptyLists,
		AvgBucketsPerList: float64(totalBuckets) / float64(nonEmptyLists),
		MaxBucketsPerList: maxBuckets,
		AvgVectorsPerList: float64(totalVectors) / float64(nonEmptyLists),
		MaxVectorsPerList: maxVectors,
		AvgBucketSpread:   totalSpread / float64(nonEmptyLists),
		MaxBucketSpread:   maxSpread,
	}
}

func computeIVFListSpread(store *dataset.VectorStore, listIndex int, list []uint32) float64 {
	if len(store.IVFCentroids) == 0 || len(store.IVFBucketSummaries) == 0 {
		return 0
	}

	baseOffset := listIndex * dataset.IVFCoarseDimensions
	totalDistance := 0.0
	for _, bucketID := range list {
		summaryOffset := int(bucketID) * dataset.IVFCoarseDimensions
		totalDistance += float64(ivfDistance(
			store.IVFBucketSummaries[summaryOffset:summaryOffset+dataset.IVFCoarseDimensions],
			store.IVFCentroids[baseOffset:baseOffset+dataset.IVFCoarseDimensions],
		))
	}

	return totalDistance / float64(len(list))
}

func ivfDistance(left []float32, right []float32) float32 {
	d0 := left[0] - right[0]
	d1 := left[1] - right[1]
	d2 := left[2] - right[2]
	d3 := left[3] - right[3]
	d4 := left[4] - right[4]
	d5 := left[5] - right[5]
	return d0*d0 + d1*d1 + d2*d2 + d3*d3 + d4*d4 + d5*d5
}

func sampleQueryIndices(count int, samples int) []int {
	if samples >= count {
		indices := make([]int, count)
		for index := range indices {
			indices[index] = index
		}
		return indices
	}

	indices := make([]int, 0, samples)
	stride := count / samples
	if stride < 1 {
		stride = 1
	}

	for index := 0; len(indices) < samples && index < count; index += stride {
		indices = append(indices, index)
	}

	for len(indices) < samples {
		indices = append(indices, count-1)
	}

	return indices
}

func queryVector(store *dataset.VectorStore, index int) [dataset.VectorSize]float32 {
	var query [dataset.VectorSize]float32
	offset := index * dataset.VectorSize

	if len(store.QuantizedVectors) > 0 {
		for vectorIndex := range query {
			query[vectorIndex] = dataset.DequantizeComponent(store.QuantizedVectors[offset+vectorIndex])
		}
		return query
	}

	copy(query[:], store.Vectors[offset:offset+dataset.VectorSize])
	return query
}

func approvedFromFraudCount(fraudCount int) bool {
	return fraudCount < 3
}

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func max(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
