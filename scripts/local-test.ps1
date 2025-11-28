# Local Go test runner for PAW
# Usage: pwsh -File scripts/local-test.ps1

$ErrorActionPreference = 'Stop'
Write-Host "Running local Go test suite with go1.23.1 (clean caches)" -ForegroundColor Cyan
$env:GOTOOLCHAIN = "go1.23.1"
& go clean -cache -testcache
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& go test ./...
exit $LASTEXITCODE
