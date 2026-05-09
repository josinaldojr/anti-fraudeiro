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
	errTrailingJSON    = errors.New("trailing json payload")
)

func rejectTrailingJSON(decoder interface{ Decode(v any) error }) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errTrailingJSON
		}
		return err
	}

	return nil
}

func decodeFraudScoreRequest(w http.ResponseWriter, r *http.Request) (fraud.FraudScoreRequest, error) {
	limitedBody := http.MaxBytesReader(w, r.Body, maxFraudScoreRequestBodyBytes)
	defer limitedBody.Close()

	decoder := json.NewDecoder(limitedBody)
	decoder.DisallowUnknownFields()

	var request fraud.FraudScoreRequest
	if err := decoder.Decode(&request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return fraud.FraudScoreRequest{}, errRequestTooLarge
		}
		return fraud.FraudScoreRequest{}, err
	}

	if err := rejectTrailingJSON(decoder); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return fraud.FraudScoreRequest{}, errRequestTooLarge
		}
		return fraud.FraudScoreRequest{}, err
	}

	if _, err := io.Copy(io.Discard, limitedBody); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return fraud.FraudScoreRequest{}, errRequestTooLarge
		}
		return fraud.FraudScoreRequest{}, err
	}

	request.Customer.Finalize()
	return request, nil
}
