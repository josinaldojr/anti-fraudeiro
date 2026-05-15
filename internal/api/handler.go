package api

import (
	"errors"
	"net/http"

	"github.com/josinaldojr/anti-fraudeiro/internal/fraud"
)

const maxFraudScoreRequestBodyBytes int64 = 16 * 1024

type Handler struct {
	scorer *fraud.Scorer
	limiter *fraudScoreLimiter
}

func NewHandler(scorer *fraud.Scorer, maxConcurrentFraudRequests int) *Handler {
	return &Handler{
		scorer: scorer,
		limiter: newFraudScoreLimiter(maxConcurrentFraudRequests),
	}
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
		if errors.Is(err, errRequestTooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{Error: "request payload too large"})
			return
		}

		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request payload"})
		return
	}
	defer releaseFraudScoreRequest(request)

	if err := h.limiter.Acquire(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "request canceled"})
		return
	}
	defer h.limiter.Release()

	fraudCount, err := h.scorer.ScoreFraudCount(*request)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeFraudScoreJSON(w, http.StatusOK, fraudCount)
}
