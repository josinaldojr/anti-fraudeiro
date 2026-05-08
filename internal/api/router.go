package api

import "net/http"

func NewRouter(handler *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", handler.Ready)
	mux.HandleFunc("/fraud-score", handler.FraudScore)

	return mux
}
