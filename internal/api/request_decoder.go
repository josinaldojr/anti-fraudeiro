package api

import (
	"errors"
	"io"
	"net/http"

	json "github.com/goccy/go-json"
	"github.com/josinaldojr/anti-fraudeiro/internal/fraud"
)

var (
	errRequestTooLarge = errors.New("request payload too large")
)

func decodeFraudScoreRequest(w http.ResponseWriter, r *http.Request) (fraud.FraudScoreRequest, error) {
	limitedBody := http.MaxBytesReader(w, r.Body, maxFraudScoreRequestBodyBytes)
	defer limitedBody.Close()

	body, err := io.ReadAll(limitedBody)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return fraud.FraudScoreRequest{}, errRequestTooLarge
		}
		return fraud.FraudScoreRequest{}, err
	}

	var request fraud.FraudScoreRequest
	if err := json.Unmarshal(body, &request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return fraud.FraudScoreRequest{}, errRequestTooLarge
		}
		return fraud.FraudScoreRequest{}, err
	}

	request.Customer.Finalize()
	return request, nil
}
