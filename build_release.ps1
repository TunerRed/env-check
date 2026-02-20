param()
Write-Output "Building windows binaries (PowerShell cross-compile)..."

$out = "dist"
if (-not (Test-Path $out)) { New-Item -ItemType Directory -Path $out | Out-Null }

$env:CGO_ENABLED=0
$env:GOOS = "windows"

Write-Output "Building windows/amd64..."
$env:GOARCH = "amd64"
go build -ldflags "-s -w" -o "$out\env-check-windows-amd64.exe" .
icacls "$out\env-check-windows-amd64.exe" /grant Everyone:RX | Out-Null

Write-Output "Building windows/arm64..."
$env:GOARCH = "arm64"
go build -ldflags "-s -w" -o "$out\env-check-windows-arm64.exe" .
icacls "$out\env-check-windows-arm64.exe" /grant Everyone:RX | Out-Null

# Write-Output "Creating zip packages..."
#foreach ($file in Get-ChildItem -Path $out -File) {
#	$zip = "$($file.FullName).zip"
#	if (Test-Path $zip) { Remove-Item $zip }
#	Compress-Archive -Path $file.FullName -DestinationPath $zip
#}

Write-Output "PowerShell build complete. Check the 'dist' directory."
Write-Output "Build and packaging complete. Files in $out"
