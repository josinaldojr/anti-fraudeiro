$ErrorActionPreference = "Stop"

git.exe submodule update --init --recursive
New-Item -ItemType Directory -Force -Path resources | Out-Null
Copy-Item .rinha/resources/references.json.gz resources/references.json.gz -Force
Copy-Item .rinha/resources/mcc_risk.json resources/mcc_risk.json -Force
Copy-Item .rinha/resources/normalization.json resources/normalization.json -Force
go run ./cmd/preprocess -input resources/references.json.gz -output resources/references.bin
