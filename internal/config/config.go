package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	HTTPPort          string
	ReferencesPath    string
	MCCRiskPath       string
	NormalizationPath string
}

type Normalization struct {
	MaxAmount            float64 `json:"max_amount"`
	MaxInstallments      float64 `json:"max_installments"`
	AmountVsAvgRatio     float64 `json:"amount_vs_avg_ratio"`
	MaxMinutes           float64 `json:"max_minutes"`
	MaxKM                float64 `json:"max_km"`
	MaxTxCount24h        float64 `json:"max_tx_count_24h"`
	MaxMerchantAvgAmount float64 `json:"max_merchant_avg_amount"`
}

func Load() Config {
	resourceDir := envOrDefault("RESOURCE_DIR", "resources")
	defaultReferencesPath := filepath.Join(resourceDir, "references.json.gz")
	if _, err := os.Stat(defaultReferencesPath); err != nil {
		defaultReferencesPath = filepath.Join(resourceDir, "example-references.json")
	}

	return Config{
		HTTPPort:          envOrDefault("PORT", "9999"),
		ReferencesPath:    envOrDefault("REFERENCES_PATH", defaultReferencesPath),
		MCCRiskPath:       envOrDefault("MCC_RISK_PATH", filepath.Join(resourceDir, "mcc_risk.json")),
		NormalizationPath: envOrDefault("NORMALIZATION_PATH", filepath.Join(resourceDir, "normalization.json")),
	}
}

func (c Config) ListenAddr() string {
	if strings.HasPrefix(c.HTTPPort, ":") {
		return c.HTTPPort
	}

	return ":" + c.HTTPPort
}

func LoadNormalization(path string) (Normalization, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Normalization{}, fmt.Errorf("read normalization file: %w", err)
	}

	var normalization Normalization
	if err := json.Unmarshal(data, &normalization); err != nil {
		return Normalization{}, fmt.Errorf("decode normalization file: %w", err)
	}

	return normalization, nil
}

func LoadMCCRisk(path string) (map[string]float32, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mcc risk file: %w", err)
	}

	riskByMCC := make(map[string]float32)
	if err := json.Unmarshal(data, &riskByMCC); err != nil {
		return nil, fmt.Errorf("decode mcc risk file: %w", err)
	}

	return riskByMCC, nil
}

func envOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}
