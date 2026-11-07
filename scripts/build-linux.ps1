param(
    [string]$Version = "v0.1.0"
)

Write-Host "==> Starting Linux build for version $Version..." -ForegroundColor Cyan
go run ./scripts/package.go -v $Version
