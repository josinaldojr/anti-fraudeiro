APP_NAME=anti-fraudeiro
IMAGE=ghcr.io/josinaldojr/anti-fraudeiro:latest

.PHONY: prepare-resources preprocess eval-coarse test run docker-build docker-push docker-up docker-down curl-ready curl-fraud-score rinha-smoke rinha-test

prepare-resources:
	sh ./scripts/prepare-resources.sh

preprocess:
	go run ./cmd/preprocess

eval-coarse:
	go run ./cmd/eval-coarse

test:
	go test ./...

run:
	go run ./cmd/api

docker-build: prepare-resources
	docker build -t $(IMAGE) .

docker-push:
	docker push $(IMAGE)

docker-up: prepare-resources
	docker compose up --build -d

docker-down:
	docker compose down

curl-ready:
	curl -i http://localhost:9999/ready

curl-fraud-score:
	curl -i -X POST http://localhost:9999/fraud-score \
		-H "Content-Type: application/json" \
		-d '{"id":"tx-3576980410","transaction":{"amount":384.88,"installments":3,"requested_at":"2026-03-11T20:23:35Z"},"customer":{"avg_amount":769.76,"tx_count_24h":3,"known_merchants":["MERC-009","MERC-001","MERC-001"]},"merchant":{"id":"MERC-001","mcc":"5912","avg_amount":298.95},"terminal":{"is_online":false,"card_present":true,"km_from_home":13.7090520965},"last_transaction":{"timestamp":"2026-03-11T14:58:35Z","km_from_current":18.8626479774}}'

rinha-smoke:
	sh ./scripts/run-rinha-smoke.sh

rinha-test:
	sh ./scripts/run-rinha-test.sh
