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

	bucketStrategy := fraud.NormalizeBucketStrategy(cfg.BucketStrategy)
	if cfg.EnableSecondaryBucketIndex {
		dataset.BuildSecondaryBucketIndex(store)
	}
	if bucketStrategy == fraud.BucketStrategyIVF {
		dataset.BuildIVFIndex(store, cfg.IVFListCount)
	}

	vectorizer := fraud.NewVectorizer(normalization, mccRisk)
	fraud.SetDefaultSearchConfig(cfg.BucketTargetCandidates, cfg.BucketMaxSearchRadius)
	fraud.SetDefaultIVFNProbe(cfg.IVFNProbe)
	scorer := fraud.NewScorer(vectorizer, store, bucketStrategy)
	handler := api.NewHandler(scorer, cfg.MaxConcurrentFraudRequests)

	server := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           api.NewRouter(handler),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	logDatasetStartup(cfg, store, bucketStrategy)

	return server.ListenAndServe()
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

func logDatasetStartup(cfg config.Config, store *dataset.VectorStore, bucketStrategy fraud.BucketStrategy) {
	format := "float32"
	if len(store.QuantizedVectors) > 0 {
		format = "quantized"
	}

	if filepath.Base(cfg.ReferencesPath) == dataset.ExampleReferenceFile {
		log.Printf("WARNING: using example-references.json fallback, official dataset was not found")
	}

	log.Printf("loaded references from %s", cfg.ReferencesPath)
	log.Printf(
		"reference vectors count=%d reference_format=%s bucket_index_enabled=%t secondary_bucket_index_enabled=%t ivf_index_enabled=%t ivf_list_count=%d gomaxprocs=%d gc_percent=%d memory_limit_mib=%d max_concurrent_fraud_requests=%d bucket_strategy=%s",
		store.Count,
		format,
		len(store.BucketIndex) > 0,
		len(store.SecondaryBucketIndex) > 0,
		len(store.IVFLists) > 0,
		len(store.IVFLists),
		runtime.GOMAXPROCS(0),
		cfg.GCPercent,
		cfg.MemoryLimitMiB,
		cfg.MaxConcurrentFraudRequests,
		bucketStrategy,
	)
	log.Printf(
		"bucket_target_candidates=%d bucket_max_search_radius=%d ivf_nprobe=%d",
		cfg.BucketTargetCandidates,
		cfg.BucketMaxSearchRadius,
		cfg.IVFNProbe,
	)
}
