<a id="readme-top"></a>

# anti-fraudeiro

Go backend for the [Rinha de Backend 2026](https://github.com/zanfranceschi/rinha-de-backend-2026), focused on fraud detection with vector search under the challenge resource constraints.

## About The Project

`anti-fraudeiro` is a lean HTTP service that receives a transaction payload, converts it to the official 14-dimensional feature vector, runs a KNN search against the challenge reference dataset, and returns:

```json
{
  "approved": true,
  "fraud_score": 0
}
```

The current implementation prioritizes correctness, simplicity, and a clean baseline for future optimization work. The search path supports:

- `net/http`
- `goccy/go-json`
- bucketed approximate KNN
- quantized contiguous bucket scans
- squared Euclidean distance
- fixed top-5 tracking without sorting the entire dataset

### Built With

- [Go](https://go.dev/)
- `net/http`
- [goccy/go-json](https://github.com/goccy/go-json)
- Docker
- Docker Compose
- Nginx

## Repository Layout

```text
cmd/
  api/
    main.go
internal/
  api/         HTTP handlers and router
  config/      runtime configuration and resource loading
  dataset/     reference dataset loader
  fraud/       request models, vectorization, KNN and scoring
resources/     challenge resources and local fallback dataset
docker/        Nginx configuration
```

## Getting Started

### Prerequisites

- Go 1.25+
- Docker and Docker Compose

### Resources

The service expects the official challenge resource files under `resources/`:

- `references.bin`
- `references.json.gz`
- `mcc_risk.json`
- `normalization.json`

Load priority:

1. `resources/references.bin`
2. `resources/references.json.gz`
3. `resources/example-references.json`

For local development, the application falls back to `resources/example-references.json` when the official dataset is not present.

### Configuration

Environment variables:

- `PORT`
- `RESOURCE_DIR`
- `REFERENCES_PATH`
- `MCC_RISK_PATH`
- `NORMALIZATION_PATH`
- `GOMAXPROCS`
- `GC_PERCENT`
- `MEMORY_LIMIT_MIB`
- `MAX_CONCURRENT_FRAUD_REQUESTS`
- `BUCKET_TARGET_CANDIDATES`
- `BUCKET_MAX_SEARCH_RADIUS`

Default HTTP port is `9999`.

## Usage

### Run Locally

```bash
go run ./cmd/api
```

### Run Tests

```bash
go test ./...
```

### Run Rinha k6 Tests

Smoke test:

```bash
sh ./scripts/run-rinha-smoke.sh
```

Full challenge test:

```bash
sh ./scripts/run-rinha-test.sh
```

PowerShell equivalents:

```powershell
./scripts/run-rinha-smoke.ps1
./scripts/run-rinha-test.ps1
```

Optional environment variables:

- `BASE_URL` default: `http://localhost:9999`
- `RESULTS_PATH` default: `artifacts/rinha/results.json`
- `K6_DOCKER_NETWORK` optional: Docker network name used when the `k6` wrapper falls back to the containerized runner

### Generate Compact Binary Dataset

```bash
go run ./cmd/preprocess
```

Or via Makefile:

```bash
make preprocess
```

Default conversion:

- input: `resources/references.json.gz`
- output: `resources/references.bin`

### Run With Docker Compose

```bash
docker compose up --build
```

The stack exposes:

- `GET /ready`
- `POST /fraud-score`

through:

```text
http://localhost:9999
```

### Example Requests

Ready endpoint:

```bash
curl -i http://localhost:9999/ready
```

Fraud score endpoint:

```bash
curl -i -X POST http://localhost:9999/fraud-score \
  -H "Content-Type: application/json" \
  -d '{
    "id": "tx-3576980410",
    "transaction": {
      "amount": 384.88,
      "installments": 3,
      "requested_at": "2026-03-11T20:23:35Z"
    },
    "customer": {
      "avg_amount": 769.76,
      "tx_count_24h": 3,
      "known_merchants": ["MERC-009", "MERC-001", "MERC-001"]
    },
    "merchant": {
      "id": "MERC-001",
      "mcc": "5912",
      "avg_amount": 298.95
    },
    "terminal": {
      "is_online": false,
      "card_present": true,
      "km_from_home": 13.7090520965
    },
    "last_transaction": {
      "timestamp": "2026-03-11T14:58:35Z",
      "km_from_current": 18.8626479774
    }
  }'
```

## Development

Useful commands:

- `make preprocess`
- `make test`
- `make run`
- `make docker-up`
- `make docker-down`
- `make curl-ready`
- `make curl-fraud-score`
- `make rinha-smoke`
- `make rinha-test`

## Architecture Notes

- The public port is `9999`, as required by the challenge.
- The load balancer is Nginx and performs round-robin distribution only.
- The application instances run behind the load balancer on port `8080`.
- The dataset loader supports compact binary, plain JSON, and `gzip`-compressed JSON.
- The current reference store is kept in contiguous slices for lower overhead on the hot path.
- Runtime tuning is configurable through `GOMAXPROCS`, `GC_PERCENT`, `MEMORY_LIMIT_MIB`, `MAX_CONCURRENT_FRAUD_REQUESTS`, `BUCKET_TARGET_CANDIDATES`, and `BUCKET_MAX_SEARCH_RADIUS`.
- The binary dataset is reordered by bucket so each candidate window is scanned as contiguous quantized slices, minimizing pointer chasing and improving cache locality.
- Higher `BUCKET_TARGET_CANDIDATES` and `BUCKET_MAX_SEARCH_RADIUS` recover accuracy at the cost of more candidate work; the checked-in defaults reflect the best baseline currently tracked in `artifacts/rinha/`.

## Challenge References

- [Official repository](https://github.com/zanfranceschi/rinha-de-backend-2026)
- [API documentation](https://github.com/zanfranceschi/rinha-de-backend-2026/blob/main/docs/br/API.md)
- [Detection rules](https://github.com/zanfranceschi/rinha-de-backend-2026/blob/main/docs/br/REGRAS_DE_DETECCAO.md)
- [Dataset documentation](https://github.com/zanfranceschi/rinha-de-backend-2026/blob/main/docs/br/DATASET.md)
- [Architecture constraints](https://github.com/zanfranceschi/rinha-de-backend-2026/blob/main/docs/br/ARQUITETURA.md)
- [Submission rules](https://github.com/zanfranceschi/rinha-de-backend-2026/blob/main/docs/br/SUBMISSAO.md)

<p align="right">(<a href="#readme-top">back to top</a>)</p>
