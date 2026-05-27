package fraud

import "github.com/josinaldojr/anti-fraudeiro/internal/dataset"

type Scorer struct {
	vectorizer *Vectorizer
	store      *dataset.VectorStore
}

func NewScorer(vectorizer *Vectorizer, store *dataset.VectorStore) *Scorer {
	return &Scorer{
		vectorizer: vectorizer,
		store:      store,
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

	return FindTop5(vector, s.store), nil
}

func decisionFromFraudCount(fraudCount int) Decision {
	fraudScore := float64(fraudCount) / float64(topK)

	return Decision{
		Approved:   fraudCount < 3,
		FraudScore: fraudScore,
	}
}
