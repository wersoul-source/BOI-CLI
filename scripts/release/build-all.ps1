$ErrorActionPreference = "Stop"

$platforms = @(
    "windows/amd64",
    "windows/arm64",
    "linux/amd64",
    "linux/arm64",
    "android/arm64",
    "darwin/amd64",
    "darwin/arm64"
)

$outdir = "bin"
New-Item -ItemType Directory -Force -Path $outdir | Out-Null

foreach ($platform in $platforms) {
    $goos, $goarch = $platform -split '/'
    $output = "$outdir\boi-${goos}-${goarch}"
    if ($goos -eq "windows") {
        $output += ".exe"
    }
    Write-Host "Building $goos/$goarch..."
    $env:GOOS = $goos
    $env:GOARCH = $goarch
    go build -o $output .\cmd\boi
}

Write-Host ""
Write-Host "Build complete. Binaries:"
Get-ChildItem "$outdir\boi-*" | Format-Table Name, Length
