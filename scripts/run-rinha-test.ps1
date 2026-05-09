$ErrorActionPreference = "Stop"

$baseUrl = if ($env:BASE_URL) { $env:BASE_URL } else { "http://localhost:9999" }
$resultsPath = if ($env:RESULTS_PATH) { $env:RESULTS_PATH } else { "artifacts/rinha/results.json" }
$k6DockerNetwork = if ($env:K6_DOCKER_NETWORK) { $env:K6_DOCKER_NETWORK } else { "" }
$resultsDir = Split-Path -Parent $resultsPath

if ($resultsDir) {
    New-Item -ItemType Directory -Force -Path $resultsDir | Out-Null
}

$env:K6_NO_USAGE_REPORT = "true"

if (Get-Command k6 -ErrorAction SilentlyContinue) {
    k6 run --env "BASE_URL=$baseUrl" --env "RESULTS_PATH=$resultsPath" scripts/k6/test.js
} else {
    $dockerBaseUrl = $baseUrl
    if ($dockerBaseUrl -eq "http://localhost:9999") {
        $dockerBaseUrl = "http://host.docker.internal:9999"
    }

    $dockerArgs = @("run", "--rm")
    if ($k6DockerNetwork) {
        $dockerArgs += @("--network", $k6DockerNetwork)
    }
    $dockerArgs += @(
        "-e", "K6_NO_USAGE_REPORT=true",
        "-e", "BASE_URL=$dockerBaseUrl",
        "-e", "RESULTS_PATH=$resultsPath",
        "-v", "${PWD}:/workdir",
        "-w", "/workdir",
        "grafana/k6:latest",
        "run", "--env", "BASE_URL=$dockerBaseUrl", "--env", "RESULTS_PATH=$resultsPath", "scripts/k6/test.js"
    )

    docker @dockerArgs
}

if (Test-Path $resultsPath) {
    Get-Content $resultsPath
}
