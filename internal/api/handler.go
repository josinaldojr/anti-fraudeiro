package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/josinaldojr/anti-fraudeiro/internal/fraud"
)

const maxFraudScoreRequestBodyBytes int64 = 16 * 1024

type Handler struct {
	scorer *fraud.Scorer
}

func NewHandler(scorer *fraud.Scorer) *Handler {
	return &Handler{scorer: scorer}
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	writeText(w, http.StatusOK, "ok")
}

func (h *Handler) FraudScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
		return
	}

	request, err := decodeFraudScoreRequest(w, r)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{Error: "request payload too large"})
			return
		}

		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request payload"})
		return
	}

	decision, err := h.scorer.Score(request)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, FraudScoreResponse{
		Approved:   decision.Approved,
		FraudScore: decision.FraudScore,
	})
}

func decodeFraudScoreRequest(w http.ResponseWriter, r *http.Request) (fraud.FraudScoreRequest, error) {
	defer r.Body.Close()

	limitedBody := http.MaxBytesReader(w, r.Body, maxFraudScoreRequestBodyBytes)
	defer limitedBody.Close()

	var request fraud.FraudScoreRequest
	decoder := json.NewDecoder(limitedBody)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		return fraud.FraudScoreRequest{}, err
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fraud.FraudScoreRequest{}, errors.New("trailing json payload")
		}
		return fraud.FraudScoreRequest{}, err
	}

	return request, nil
}
