# Package the whole site folder (source + data files) into a tar.gz for uploading to aaPanel.
# If Go is installed, this also cross-compiles the linux binary and includes it.
$ErrorActionPreference = "Stop"

$site = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$leaf = Split-Path $site -Leaf
$out  = Join-Path (Split-Path $site) "$leaf-deploy.tar.gz"

$go = Get-Command go -ErrorAction SilentlyContinue
if ($go) {
    Write-Host "Go detected, cross-compiling linux binary..."
    Push-Location (Join-Path $site "go-server")
    try {
        $env:CGO_ENABLED = "0"
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        go mod tidy
        go build -trimpath -o (Join-Path $site "ajie-server") .
        Write-Host "Built ajie-server and included it in the package"
    }
    finally {
        Pop-Location
    }
}
else {
    Write-Host "Go not detected; deploy.sh on the server will build it"
}

Write-Host "Packaging -> $out"
tar -czf $out -C (Split-Path $site) $leaf
Write-Host "Done: $out"
