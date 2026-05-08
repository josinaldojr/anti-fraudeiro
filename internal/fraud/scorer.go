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
	vector, err := s.vectorizer.Vectorize(request)
	if err != nil {
		return Decision{}, err
	}

	fraudCount := FindTop5(vector, s.store)

	return decisionFromFraudCount(fraudCount), nil
}

func decisionFromFraudCount(fraudCount int) Decision {
	fraudScore := float64(fraudCount) / float64(topK)

	return Decision{
		Approved:   fraudScore < 0.6,
		FraudScore: fraudScore,
	}
}
