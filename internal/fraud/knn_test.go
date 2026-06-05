package fraud

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
)

func TestFindTop5ReturnsBoundedAmbiguousResult(t *testing.T) {
	query := [16]float32{
		0.1, 0.2, 0.3, 0.4,
		0.5, 0.1, 0.2, 0.3,
		0.4, 0.5, 0.1, 0.2,
		0.3, 0.4, 0.5, 0.1,
	}
	store := vectorStoreWithLabels(query, []byte{
		dataset.LabelFraud,
		dataset.LabelFraud,
		dataset.LabelLegit,
		dataset.LabelLegit,
		dataset.LabelLegit,
	})

	boundedFraudCount, bestDistances, stats := findTop5WithStats(query, store, defaultSearchConfig)
	if boundedFraudCount != 2 {
		t.Fatalf("bounded fraud count = %d, want 2", boundedFraudCount)
	}
	if !needsExactFallback(boundedFraudCount, bestDistances) {
		t.Fatalf("expected ambiguous bounded result to require legacy fallback")
	}
	if stats.processedCandidates < 5 {
		t.Fatalf("processed candidates = %d, want at least 5", stats.processedCandidates)
	}

	if actual := FindTop5(query, store); actual != boundedFraudCount {
		t.Fatalf("FindTop5 = %d, want bounded fraud count %d", actual, boundedFraudCount)
	}
}

func TestFindTop5RealDatasetUsesBoundedDefaultPath(t *testing.T) {
	referencesPath := filepath.Join("..", "..", "resources", dataset.BinaryReferenceFile)
	if _, err := os.Stat(referencesPath); err != nil {
		t.Skipf("real dataset not available: %v", err)
	}

	store, err := dataset.LoadVectorStore(referencesPath)
	if err != nil {
		t.Fatalf("LoadVectorStore returned error: %v", err)
	}
	defer store.Close()

	vectorizer := newTestVectorizer()
	query, err := vectorizer.Vectorize(sampleRequest())
	if err != nil {
		t.Fatalf("Vectorize returned error: %v", err)
	}

	boundedFraudCount, bestDistances, stats := findTop5WithStats(query, store, defaultSearchConfig)
	if stats.processedCandidates <= 0 {
		t.Fatalf("expected bounded search to process candidates")
	}
	if stats.processedCandidates >= store.Count {
		t.Fatalf("bounded search processed %d candidates, want less than full dataset %d", stats.processedCandidates, store.Count)
	}

	if actual := FindTop5(query, store); actual != boundedFraudCount {
		t.Fatalf("FindTop5 = %d, want bounded fraud count %d", actual, boundedFraudCount)
	}
	t.Logf("bounded candidates=%d buckets=%d legacyFallback=%v", stats.processedCandidates, stats.bucketsVisited, needsExactFallback(boundedFraudCount, bestDistances))
}

type TestEntry struct {
	Request          FraudScoreRequest `json:"request"`
	ExpectedApproved bool              `json:"expected_approved"`
}

type TestData struct {
	Entries []TestEntry `json:"entries"`
}

func TestQuantizationDiscrepancy(t *testing.T) {
	t.Log("Setting Rinha default search config (256 candidates, radius 3)...")
	SetDefaultSearchConfig(256, 3)

	t.Log("Loading quantized references from BIN...")
	storeQuantized, err := dataset.LoadVectorStore("../../resources/references.bin")
	if err != nil {
		t.Skipf("bin file not available: %v", err)
	}
	defer storeQuantized.Close()

	t.Log("Loading test data...")
	file, err := os.Open("../../.rinha/test/test-data.json")
	if err != nil {
		t.Skipf("test-data.json not available: %v", err)
	}
	defer file.Close()

	var testData TestData
	if err := json.NewDecoder(file).Decode(&testData); err != nil {
		t.Fatalf("failed to decode test data: %v", err)
	}

	vectorizer := newTestVectorizer()

	thresholdsToTest := []uint64{
		200000000,
		150000000,
		100000000,
	}

	sampleSize := 2000

	for _, distThreshold := range thresholdsToTest {
		mismatches := 0
		exactScansCount := 0

		for i := 0; i < sampleSize && i < len(testData.Entries); i++ {
			entry := testData.Entries[i]
			query, _ := vectorizer.Vectorize(entry.Request)

			// Approx search
			var bestDistances [topK]uint64
			fraudCountApprox, bestDistances, _ := findTop5WithStats(query, storeQuantized, defaultSearchConfig)

			// Decision with custom fallback condition
			finalFraudCount := fraudCountApprox
			if (fraudCountApprox > 0 && fraudCountApprox < 5) || bestDistances[topK-1] > distThreshold {
				exactScansCount++
				finalFraudCount = exactScan(query, storeQuantized)
			}

			approvedFallback := finalFraudCount < 3

			// True exact search
			fraudCountExact := exactScan(query, storeQuantized)
			approvedExact := fraudCountExact < 3

			if approvedFallback != approvedExact {
				mismatches++
			}
		}

		t.Logf("Threshold %10d | Mismatches: %4d (%.4f%%) | Exact Scans: %5d/%d (%.2f%%)", 
			distThreshold, mismatches, float64(mismatches)/float64(sampleSize)*100, 
			exactScansCount, sampleSize, float64(exactScansCount)/float64(sampleSize)*100)
	}
}

func vectorStoreWithLabels(query [16]float32, labels []byte) *dataset.VectorStore {
	quantized := make([]uint16, 0, len(labels)*dataset.VectorSize)
	for range labels {
		for _, value := range query {
			quantized = append(quantized, dataset.QuantizeComponent(value))
		}
	}

	store := &dataset.VectorStore{
		QuantizedVectors: quantized,
		Labels:           append([]byte(nil), labels...),
		Count:            len(labels),
	}
	dataset.BuildBucketIndex(store)
	dataset.ReorderStoreByBucket(store)
	return store
}
