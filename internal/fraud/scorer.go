package fraud

import "github.com/josinaldojr/anti-fraudeiro/internal/dataset"

type Scorer struct {
	vectorizer *Vectorizer
	store      *dataset.VectorStore
	strategy   BucketStrategy
}

func NewScorer(vectorizer *Vectorizer, store *dataset.VectorStore, strategies ...BucketStrategy) *Scorer {
	strategy := BucketStrategyWindow
	if len(strategies) > 0 {
		strategy = strategies[0]
	}

	return &Scorer{
		vectorizer: vectorizer,
		store:      store,
		strategy:   strategy,
	}
}

func (s *Scorer) Score(request FraudScoreRequest) (Decision, error) {
	fraudCount, err := s.ScoreFraudCount(request)
	if err != nil {
		return Decision{}, err
	}

	return decisionFromFraudCount(fraudCount), nil
}

func (s *Scorer) ScoreFraudCount(request FraudScoreRequest) (int, error) {
	vector, err := s.vectorizer.Vectorize(request)
	if err != nil {
		return 0, err
	}

	return FindTop5WithStrategy(vector, s.store, s.strategy), nil
}

func decisionFromFraudCount(fraudCount int) Decision {
	fraudScore := float64(fraudCount) / float64(topK)

	return Decision{
		Approved:   fraudScore < 0.6,
		FraudScore: fraudScore,
	}
}
