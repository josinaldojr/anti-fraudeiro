package api

import (
	"errors"
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

	requestPayload, err := decodeFraudScoreRequest(r.Body)
	if err != nil {
		if errors.Is(err, errRequestTooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{Error: "request payload too large"})
			return
		}

		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request payload"})
		return
	}

	request, err := unmarshalFraudScoreRequest(requestPayload)
	if err != nil {
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
