package main

import (
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/josinaldojr/anti-fraudeiro/internal/api"
	"github.com/josinaldojr/anti-fraudeiro/internal/config"
	"github.com/josinaldojr/anti-fraudeiro/internal/dataset"
	"github.com/josinaldojr/anti-fraudeiro/internal/fraud"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.Load()
	applyRuntimeTuning(cfg)

	normalization, err := config.LoadNormalization(cfg.NormalizationPath)
	if err != nil {
		return err
	}

	mccRisk, err := config.LoadMCCRisk(cfg.MCCRiskPath)
	if err != nil {
		return err
	}

	store, err := dataset.LoadVectorStore(cfg.ReferencesPath)
	if err != nil {
		return err
	}
	defer store.Close()

	// Warm-up mmap data to avoid page faults during the test
	warmupDataset(store)

	vectorizer := fraud.NewVectorizer(normalization, mccRisk)
	fraud.SetDefaultSearchConfig(cfg.BucketTargetCandidates, cfg.BucketMaxSearchRadius)
	scorer := fraud.NewScorer(vectorizer, store)
	handler := api.NewHandler(scorer, cfg.MaxConcurrentFraudRequests)

	// Prime the pools
	api.WarmupPools()

	server := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           api.NewRouter(handler),
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logDatasetStartup(cfg, store)

	return server.ListenAndServe()
}

func warmupDataset(store *dataset.VectorStore) {
	// Touch all labels and some vectors to trigger page-in
	sum := 0
	for i := 0; i < store.Count; i++ {
		sum += int(store.Labels[i])
		if i%1000 == 0 && len(store.QuantizedVectors) > 0 {
			sum += int(store.QuantizedVectors[i*dataset.VectorSize])
		}
	}
	_ = sum
}

func applyRuntimeTuning(cfg config.Config) {
	if cfg.GOMAXPROCS > 0 {
		runtime.GOMAXPROCS(cfg.GOMAXPROCS)
	}

	if cfg.GCPercent >= 0 {
		debug.SetGCPercent(cfg.GCPercent)
	}

	if cfg.MemoryLimitMiB > 0 {
		debug.SetMemoryLimit(cfg.MemoryLimitMiB << 20)
	}
}

func logDatasetStartup(cfg config.Config, store *dataset.VectorStore) {
	format := "float32"
	if len(store.QuantizedVectors) > 0 {
		format = "quantized"
	}

	if filepath.Base(cfg.ReferencesPath) == dataset.ExampleReferenceFile {
		log.Printf("WARNING: using example-references.json fallback, official dataset was not found")
	}

	log.Printf("loaded references from %s", cfg.ReferencesPath)
	log.Printf(
		"reference vectors count=%d reference_format=%s bucket_index_enabled=%t gomaxprocs=%d gc_percent=%d memory_limit_mib=%d max_concurrent_fraud_requests=%d",
		store.Count,
		format,
		len(store.BucketIndex) > 0 || len(store.BucketMeta) > 0 || len(store.BucketPrefixSums) > 0,
		runtime.GOMAXPROCS(0),
		cfg.GCPercent,
		cfg.MemoryLimitMiB,
		cfg.MaxConcurrentFraudRequests,
	)
	log.Printf(
		"bucket_target_candidates=%d bucket_max_search_radius=%d",
		cfg.BucketTargetCandidates,
		cfg.BucketMaxSearchRadius,
	)
}
