$ErrorActionPreference = 'Stop'

$Repository = if ($env:OPENPORT_REPOSITORY) { $env:OPENPORT_REPOSITORY } else { 'SinghAman21/openport' }
$Version = if ($env:OPENPORT_VERSION) { $env:OPENPORT_VERSION } else { 'latest' }
$InstallDirectory = if ($env:OPENPORT_INSTALL_DIR) { $env:OPENPORT_INSTALL_DIR } else { Join-Path $HOME '.local\bin' }

function Fail([string]$Message) {
    throw "OpenPort install failed: $Message"
}

$Architecture = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
switch ($Architecture.ToUpperInvariant()) {
    'AMD64' { $Asset = 'openport-windows-amd64.exe' }
    'ARM64' { $Asset = 'openport-windows-arm64.exe' }
    default { Fail "unsupported Windows architecture: $Architecture (supported: AMD64, ARM64)" }
}

if ($Version -eq 'latest') {
    $Url = "https://github.com/$Repository/releases/latest/download/$Asset"
} else {
    $Tag = if ($Version.StartsWith('v')) { $Version } else { "v$Version" }
    $Url = "https://github.com/$Repository/releases/download/$Tag/$Asset"
}

$TemporaryDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("openport-" + [System.Guid]::NewGuid().ToString('N'))
$TemporaryBinary = Join-Path $TemporaryDirectory $Asset

try {
    New-Item -ItemType Directory -Path $TemporaryDirectory -Force | Out-Null
    Write-Host "Downloading OpenPort ($Asset) ..."
    Invoke-WebRequest -Uri $Url -OutFile $TemporaryBinary -UseBasicParsing

    if (-not (Test-Path $TemporaryBinary) -or (Get-Item $TemporaryBinary).Length -eq 0) {
        Fail 'downloaded file is empty'
    }

    New-Item -ItemType Directory -Path $InstallDirectory -Force | Out-Null
    $Target = Join-Path $InstallDirectory 'openport.exe'
    Move-Item -Force $TemporaryBinary $Target

    $UserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $PathEntries = @($UserPath -split ';' | Where-Object { $_ })
    $NormalizedInstallDirectory = [System.IO.Path]::GetFullPath($InstallDirectory).TrimEnd('\')
    $AlreadyInPath = $PathEntries | Where-Object {
        try { [System.IO.Path]::GetFullPath($_).TrimEnd('\') -ieq $NormalizedInstallDirectory } catch { $false }
    }

    if (-not $AlreadyInPath) {
        $NewUserPath = (($PathEntries + $InstallDirectory) -join ';')
        [Environment]::SetEnvironmentVariable('Path', $NewUserPath, 'User')
        $env:Path = "$InstallDirectory;$env:Path"
        $PathUpdated = $true
    } else {
        $PathUpdated = $false
    }

    Write-Host ""
    Write-Host "OpenPort installed to $Target" -ForegroundColor Green
    if ($PathUpdated) {
        Write-Host "Your user PATH was updated. Open a new PowerShell window to use 'openport' everywhere."
    }

    if (Get-Command openport.exe -ErrorAction SilentlyContinue) {
        Write-Host 'Verified:' -NoNewline
        & $Target version
    } else {
        Write-Host "Run this in a new PowerShell window: openport"
    }
}
catch {
    Fail $_.Exception.Message
}
finally {
    if (Test-Path $TemporaryDirectory) {
        Remove-Item -Recurse -Force $TemporaryDirectory -ErrorAction SilentlyContinue
    }
}
