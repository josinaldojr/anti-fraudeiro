package fraud

import "strings"

type BucketStrategy string

const (
	BucketStrategyWindow  BucketStrategy = "window"
	BucketStrategyOrdered BucketStrategy = "ordered"
)

func NormalizeBucketStrategy(value string) BucketStrategy {
	switch BucketStrategy(strings.ToLower(strings.TrimSpace(value))) {
	case BucketStrategyOrdered:
		return BucketStrategyOrdered
	default:
		return BucketStrategyWindow
	}
}
