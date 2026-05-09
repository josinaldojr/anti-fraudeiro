$ErrorActionPreference = "Stop"

$baseUrl = if ($env:BASE_URL) { $env:BASE_URL } else { "http://localhost:9999" }
$k6DockerNetwork = if ($env:K6_DOCKER_NETWORK) { $env:K6_DOCKER_NETWORK } else { "" }
$env:K6_NO_USAGE_REPORT = "true"

if (Get-Command k6 -ErrorAction SilentlyContinue) {
    k6 run --env "BASE_URL=$baseUrl" scripts/k6/smoke.js
    exit 0
}

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
    "-v", "${PWD}:/workdir",
    "-w", "/workdir",
    "grafana/k6:latest",
    "run", "--env", "BASE_URL=$dockerBaseUrl", "scripts/k6/smoke.js"
)

docker @dockerArgs
