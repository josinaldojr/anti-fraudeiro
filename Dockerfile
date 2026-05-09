FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY resources ./resources

RUN test -f resources/references.bin
RUN test -f resources/mcc_risk.json
RUN test -f resources/normalization.json
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/anti-fraudeiro ./cmd/api

FROM alpine:3.22

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /out/anti-fraudeiro /app/anti-fraudeiro
COPY --from=builder /app/resources /app/resources

USER app

EXPOSE 8080

CMD ["/app/anti-fraudeiro"]
