#!/bin/sh
set -eu

git submodule update --init --recursive
mkdir -p resources
cp .rinha/resources/references.json.gz resources/references.json.gz
cp .rinha/resources/mcc_risk.json resources/mcc_risk.json
cp .rinha/resources/normalization.json resources/normalization.json
go run ./cmd/preprocess -input resources/references.json.gz -output resources/references.bin
