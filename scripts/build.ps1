$ErrorActionPreference = 'Stop'
Push-Location (Join-Path $PSScriptRoot '..')
try {
    $version = (Get-Content -Raw version.txt).Trim()
    $packagePath = Join-Path 'dist' "auto-yuumi-$version-windows-amd64"
    New-Item -ItemType Directory -Path (Join-Path $packagePath 'assets') -Force | Out-Null
    go build -trimpath -ldflags '-s -w -H=windowsgui' -o dist/auto-yuumi.exe .
    if ($LASTEXITCODE -ne 0) { throw 'Falha na build' }
    Copy-Item dist/auto-yuumi.exe $packagePath
    Copy-Item config.json,README.md $packagePath
    Copy-Item assets/attached.png,assets/detached.png,assets/w_ready.png (Join-Path $packagePath 'assets')
    $zipPath = "$packagePath.zip"
    Compress-Archive -Path "$packagePath/*" -DestinationPath $zipPath -Force
    $checksum = (Get-FileHash -Algorithm SHA256 -LiteralPath $zipPath).Hash.ToLower()
    "$checksum  $(Split-Path -Leaf $zipPath)" | Set-Content -Encoding utf8 (Join-Path 'dist' 'SHA256SUMS.txt')
} finally { Pop-Location }
