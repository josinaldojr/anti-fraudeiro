package fraud

import "strings"

type BucketStrategy string

const (
	BucketStrategyWindow    BucketStrategy = "window"
	BucketStrategyShortlist BucketStrategy = "shortlist"
	BucketStrategyOrdered   BucketStrategy = "ordered"
	BucketStrategyIVF       BucketStrategy = "ivf"
)

func NormalizeBucketStrategy(value string) BucketStrategy {
	switch BucketStrategy(strings.ToLower(strings.TrimSpace(value))) {
	case BucketStrategyShortlist:
		return BucketStrategyShortlist
	case BucketStrategyOrdered:
		return BucketStrategyOrdered
	case BucketStrategyIVF:
		return BucketStrategyIVF
	case "window-legacy":
		return BucketStrategyWindow
	default:
		return BucketStrategyWindow
	}
}
