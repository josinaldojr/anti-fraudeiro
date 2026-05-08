package api

import (
	"bytes"
	"errors"
	"io"

	json "github.com/goccy/go-json"
	"github.com/josinaldojr/anti-fraudeiro/internal/fraud"
)

var (
	errRequestTooLarge = errors.New("request payload too large")
	errTrailingJSON    = errors.New("trailing json payload")
)

func decodeFraudScoreRequest(r io.Reader) (requestPayload []byte, err error) {
	limitedBody := &io.LimitedReader{R: r, N: maxFraudScoreRequestBodyBytes + 1}

	requestPayload, err = io.ReadAll(limitedBody)
	if err != nil {
		return nil, err
	}

	if int64(len(requestPayload)) > maxFraudScoreRequestBodyBytes {
		return nil, errRequestTooLarge
	}

	return requestPayload, nil
}

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

func unmarshalFraudScoreRequest(payload []byte) (fraud.FraudScoreRequest, error) {
	var request fraud.FraudScoreRequest

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		return fraud.FraudScoreRequest{}, err
	}

	if err := rejectTrailingJSON(decoder); err != nil {
		return fraud.FraudScoreRequest{}, err
	}

	return request, nil
}
