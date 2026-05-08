package main

import (
	"log"
	"net/http"
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
	scorer := fraud.NewScorer(vectorizer, store)
	handler := api.NewHandler(scorer)

	server := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           api.NewRouter(handler),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("anti-fraudeiro listening on %s with %d reference vectors", cfg.ListenAddr(), store.Count)

	return server.ListenAndServe()
}
