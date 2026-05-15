FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY resources ./resources

RUN test -f resources/references.json.gz
RUN test -f resources/mcc_risk.json
RUN test -f resources/normalization.json
RUN go run ./cmd/preprocess -input resources/references.json.gz -output resources/references.bin
RUN test -f resources/references.bin
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/anti-fraudeiro ./cmd/api

FROM alpine:3.22

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /out/anti-fraudeiro /app/anti-fraudeiro
COPY --from=builder /app/resources /app/resources

USER app

EXPOSE 8080

CMD ["/app/anti-fraudeiro"]
