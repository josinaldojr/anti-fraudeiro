package api

import (
	"encoding/json"
	"net/http"
)

type FraudScoreResponse struct {
	Approved   bool    `json:"approved"`
	FraudScore float64 `json:"fraud_score"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

var fraudScoreResponses = [6][]byte{
	[]byte("{\"approved\":true,\"fraud_score\":0.0}\n"),
	[]byte("{\"approved\":true,\"fraud_score\":0.2}\n"),
	[]byte("{\"approved\":true,\"fraud_score\":0.4}\n"),
	[]byte("{\"approved\":false,\"fraud_score\":0.6}\n"),
	[]byte("{\"approved\":false,\"fraud_score\":0.8}\n"),
	[]byte("{\"approved\":false,\"fraud_score\":1.0}\n"),
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(payload)
}

func writeFraudScoreJSON(w http.ResponseWriter, statusCode int, fraudCount int) {
	if fraudCount < 0 {
		fraudCount = 0
	}
	if fraudCount >= len(fraudScoreResponses) {
		fraudCount = len(fraudScoreResponses) - 1
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(fraudScoreResponses[fraudCount])
}

func writeText(w http.ResponseWriter, statusCode int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(body))
}
