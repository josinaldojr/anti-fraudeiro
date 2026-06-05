package config

import "testing"

func TestLoadUsesPerformanceTunedDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("RESOURCE_DIR", "")
	t.Setenv("REFERENCES_PATH", "")
	t.Setenv("MCC_RISK_PATH", "")
	t.Setenv("NORMALIZATION_PATH", "")
	t.Setenv("GOMAXPROCS", "")
	t.Setenv("GC_PERCENT", "")
	t.Setenv("MEMORY_LIMIT_MIB", "")
	t.Setenv("MAX_CONCURRENT_FRAUD_REQUESTS", "")
	t.Setenv("BUCKET_TARGET_CANDIDATES", "")
	t.Setenv("BUCKET_MAX_SEARCH_RADIUS", "")

	cfg := Load()

	if cfg.GCPercent != 500 {
		t.Fatalf("GCPercent = %d, want 500", cfg.GCPercent)
	}
	if cfg.MemoryLimitMiB != 0 {
		t.Fatalf("MemoryLimitMiB = %d, want 0", cfg.MemoryLimitMiB)
	}
	if cfg.BucketTargetCandidates != 448 {
		t.Fatalf("BucketTargetCandidates = %d, want 448", cfg.BucketTargetCandidates)
	}
	if cfg.BucketMaxSearchRadius != 3 {
		t.Fatalf("BucketMaxSearchRadius = %d, want 3", cfg.BucketMaxSearchRadius)
	}
}

func TestLoadHonorsRuntimeOverrides(t *testing.T) {
	t.Setenv("GC_PERCENT", "350")
	t.Setenv("MEMORY_LIMIT_MIB", "192")
	t.Setenv("MAX_CONCURRENT_FRAUD_REQUESTS", "32")
	t.Setenv("BUCKET_TARGET_CANDIDATES", "64")
	t.Setenv("BUCKET_MAX_SEARCH_RADIUS", "1")

	cfg := Load()

	if cfg.GCPercent != 350 {
		t.Fatalf("GCPercent = %d, want 350", cfg.GCPercent)
	}
	if cfg.MemoryLimitMiB != 192 {
		t.Fatalf("MemoryLimitMiB = %d, want 192", cfg.MemoryLimitMiB)
	}
	if cfg.MaxConcurrentFraudRequests != 32 {
		t.Fatalf("MaxConcurrentFraudRequests = %d, want 32", cfg.MaxConcurrentFraudRequests)
	}
	if cfg.BucketTargetCandidates != 64 {
		t.Fatalf("BucketTargetCandidates = %d, want 64", cfg.BucketTargetCandidates)
	}
	if cfg.BucketMaxSearchRadius != 1 {
		t.Fatalf("BucketMaxSearchRadius = %d, want 1", cfg.BucketMaxSearchRadius)
	}
}
