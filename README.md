# anti-fraudeiro

Bootstrap inicial em Go para a Rinha de Backend 2026, com foco em uma base correta, simples e pronta para evoluir. Nesta etapa o projeto já expõe a API pedida, vetoriza a transação em 14 dimensões, faz KNN brute force sobre `resources/example-references.json` e responde `approved` e `fraud_score`.

## Objetivo

Construir uma solução enxuta para detecção de fraude com busca vetorial, priorizando:

- corretude da regra da competição;
- estrutura de código clara para otimizações futuras;
- execução local simples;
- conteinerização com `nginx` + 2 instâncias da API.

## Estrutura

```text
anti-fraudeiro/
├── cmd/api/main.go
├── internal/api/
├── internal/config/
├── internal/dataset/
├── internal/fraud/
├── resources/
├── docker/nginx.conf
├── Dockerfile
├── docker-compose.yml
├── info.json
├── Makefile
└── README.md
```

## Como rodar localmente

Pré-requisito: Go 1.25+.

```bash
go run ./cmd/api
```

Por padrão a API sobe na porta `9999` e carrega:

- `resources/example-references.json`
- `resources/mcc_risk.json`
- `resources/normalization.json`

Variáveis úteis:

- `PORT`
- `RESOURCE_DIR`
- `REFERENCES_PATH`
- `MCC_RISK_PATH`
- `NORMALIZATION_PATH`

## Como rodar testes

```bash
go test ./...
```

Ou via Makefile:

```bash
make test
```

## Como subir com Docker Compose

```bash
docker compose up --build
```

Isso sobe:

- `nginx` escutando na porta `9999`;
- `api1` na porta interna `8080`;
- `api2` na porta interna `8080`.

Para derrubar:

```bash
docker compose down
```

## Como testar /ready

```bash
curl -i http://localhost:9999/ready
```

Resposta esperada:

```text
HTTP/1.1 200 OK
ok
```

## Como testar /fraud-score

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

Resposta esperada:

```json
{
  "approved": true,
  "fraud_score": 0.0
}
```

## Makefile

Comandos disponíveis:

- `make test`
- `make run`
- `make docker-up`
- `make docker-down`
- `make curl-ready`
- `make curl-fraud-score`

## Próximos passos

- trocar `example-references.json` pelo `references.json.gz` real;
- gerar um formato binário compacto para carregar os vetores;
- reduzir alocações e parsing no hot path;
- avaliar quantização;
- testar buckets por faixa e outras estratégias antes de ANN;
- comparar brute force com alternativas como ANN quando a base real entrar.

## Notas

- A distância usada no KNN é euclidiana ao quadrado, sem `sqrt`.
- O hot path mantém apenas os 5 melhores candidatos, sem ordenar o dataset inteiro.
- Há espaço claro para otimizações futuras sem mudar a API pública.

