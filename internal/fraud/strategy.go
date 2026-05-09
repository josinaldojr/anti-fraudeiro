package fraud

import "strings"

type BucketStrategy string

const (
	BucketStrategyWindow    BucketStrategy = "window"
	BucketStrategyShortlist BucketStrategy = "shortlist"
	BucketStrategyOrdered   BucketStrategy = "ordered"
)

func NormalizeBucketStrategy(value string) BucketStrategy {
	switch BucketStrategy(strings.ToLower(strings.TrimSpace(value))) {
	case BucketStrategyShortlist:
		return BucketStrategyShortlist
	case BucketStrategyOrdered:
		return BucketStrategyOrdered
	case "window-legacy":
		return BucketStrategyWindow
	default:
		return BucketStrategyWindow
	}
}
