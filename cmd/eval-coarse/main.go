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
	Strategies      []strategySummary `json:"strategies"`
}

type strategySummary struct {
	Name                string  `json:"name"`
	Duration            string  `json:"duration"`
	FraudCountMatches   int     `json:"fraud_count_matches"`
	FraudCountMatchRate float64 `json:"fraud_count_match_rate"`
	DecisionMatches     int     `json:"decision_matches"`
	DecisionMatchRate   float64 `json:"decision_match_rate"`
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
		start := time.Now()
		for index, queryIndex := range queryIndices {
			query := queryVector(store, queryIndex)
			fraudCount := fraud.FindTop5WithStrategy(query, store, strategy)
			if fraudCount == exactCounts[index] {
				matchCount++
			}
			if approvedFromFraudCount(fraudCount) == approvedFromFraudCount(exactCounts[index]) {
				decisionMatchCount++
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
		})
	}

	summary := evaluationSummary{
		ReferencesCount: store.Count,
		Queries:         totalQueries,
		ExactDuration:   exactDuration.String(),
		Strategies:      summaries,
	}

	encoder := json.NewEncoder(log.Writer())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		log.Fatal(err)
	}
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
