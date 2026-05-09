package main

import (
	"log"
	"net/http"
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

	vectorizer := fraud.NewVectorizer(normalization, mccRisk)
	bucketStrategy := fraud.NormalizeBucketStrategy(cfg.BucketStrategy)
	scorer := fraud.NewScorer(vectorizer, store, bucketStrategy)
	handler := api.NewHandler(scorer, cfg.MaxConcurrentFraudRequests)

	server := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           api.NewRouter(handler),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf(
		"anti-fraudeiro listening on %s with %d reference vectors (gomaxprocs=%d gc_percent=%d memory_limit_mib=%d max_concurrent_fraud_requests=%d bucket_strategy=%s)",
		cfg.ListenAddr(),
		store.Count,
		runtime.GOMAXPROCS(0),
		cfg.GCPercent,
		cfg.MemoryLimitMiB,
		cfg.MaxConcurrentFraudRequests,
		bucketStrategy,
	)

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
