type RequestLike = {
  headers?: Record<string, string | string[] | undefined>;
};

type ResponseLike = {
  statusCode: number;
  setHeader(name: string, value: string): void;
  end(body?: string): void;
};

const repository = "SinghAman21/openport";

const posixInstaller = `#!/usr/bin/env sh
set -eu

REPOSITORY="${repository}"
VERSION="\${OPENPORT_VERSION:-latest}"
INSTALL_DIR="\${OPENPORT_INSTALL_DIR:-\${HOME}/.local/bin}"

fail() {
  printf '%s\\n' "OpenPort install failed: $*" >&2
  exit 1
}

[ -n "\${HOME:-}" ] || fail "HOME is not set; set OPENPORT_INSTALL_DIR to a writable directory"
[ -n "$VERSION" ] || fail "OPENPORT_VERSION cannot be empty"
case "$VERSION" in
  latest) ;;
  *[!A-Za-z0-9._-]*) fail "OPENPORT_VERSION may contain only letters, numbers, dots, underscores, and hyphens" ;;
esac

OS="$(uname -s 2>/dev/null || true)"
ARCH="$(uname -m 2>/dev/null || true)"
case "$OS" in
  Linux) PLATFORM="linux" ;;
  Darwin) PLATFORM="darwin" ;;
  *) fail "unsupported operating system: $OS (supported: Linux, macOS)" ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) fail "unsupported $OS architecture: $ARCH (supported: amd64, arm64)" ;;
esac

ASSET="openport-$PLATFORM-$ARCH"
if [ "$VERSION" = "latest" ]; then
  URL="https://github.com/$REPOSITORY/releases/latest/download/$ASSET"
else
  case "$VERSION" in v*) TAG="$VERSION" ;; *) TAG="v$VERSION" ;; esac
  URL="https://github.com/$REPOSITORY/releases/download/$TAG/$ASSET"
fi

command -v mktemp >/dev/null 2>&1 || fail "mktemp is required"
TEMP_DIR="$(mktemp -d 2>/dev/null)" || fail "could not create a temporary directory"
trap 'rm -rf "$TEMP_DIR"' EXIT HUP INT TERM
TEMP_BINARY="$TEMP_DIR/$ASSET"

printf 'Downloading OpenPort (%s) ...\\n' "$ASSET"
if command -v curl >/dev/null 2>&1; then
  curl --fail --location --silent --show-error --proto '=https' --tlsv1.2 "$URL" -o "$TEMP_BINARY" || fail "could not download $URL"
elif command -v wget >/dev/null 2>&1; then
  wget --https-only -qO "$TEMP_BINARY" "$URL" || fail "could not download $URL"
else
  fail "curl or wget is required"
fi

[ -s "$TEMP_BINARY" ] || fail "downloaded file is empty"
mkdir -p "$INSTALL_DIR" || fail "could not create $INSTALL_DIR"
[ -w "$INSTALL_DIR" ] || fail "$INSTALL_DIR is not writable"
TARGET="$INSTALL_DIR/openport"
install -m 755 "$TEMP_BINARY" "$TARGET" || fail "could not install $TARGET"

printf '\\nOpenPort installed to %s\\n' "$TARGET"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) printf 'Run: openport\\n' ;;
  *)
    printf 'Add this user-writable directory to PATH, then open a new shell:\\n'
    printf '  export PATH="%s:$PATH"\\n' "$INSTALL_DIR"
    ;;
esac
`;

const powerShellInstaller = `$ErrorActionPreference = 'Stop'

$Repository = '${repository}'
$Version = if ($env:OPENPORT_VERSION) { $env:OPENPORT_VERSION.Trim() } else { 'latest' }
$InstallDirectory = if ($env:OPENPORT_INSTALL_DIR) { $env:OPENPORT_INSTALL_DIR } else { Join-Path $HOME '.local\\bin' }

function Fail([string]$Message) {
    throw "OpenPort install failed: $Message"
}

if ([string]::IsNullOrWhiteSpace($Version)) {
    Fail 'OPENPORT_VERSION cannot be empty'
}
if ($Version -ne 'latest' -and $Version -notmatch '^[A-Za-z0-9._-]+$') {
    Fail 'OPENPORT_VERSION may contain only letters, numbers, dots, underscores, and hyphens'
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
    Invoke-WebRequest -Uri $Url -OutFile $TemporaryBinary

    if (-not (Test-Path -LiteralPath $TemporaryBinary) -or (Get-Item -LiteralPath $TemporaryBinary).Length -eq 0) {
        Fail 'downloaded file is empty'
    }

    New-Item -ItemType Directory -Path $InstallDirectory -Force | Out-Null
    if (-not (Test-Path -LiteralPath $InstallDirectory -PathType Container)) {
        Fail "could not create $InstallDirectory"
    }

    $Target = Join-Path $InstallDirectory 'openport.exe'
    Copy-Item -LiteralPath $TemporaryBinary -Destination $Target -Force

    $UserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $PathEntries = @($UserPath -split ';' | Where-Object { $_ })
    $NormalizedInstallDirectory = [System.IO.Path]::GetFullPath($InstallDirectory).TrimEnd('\\')
    $AlreadyInPath = $PathEntries | Where-Object {
        try { [System.IO.Path]::GetFullPath($_).TrimEnd('\\') -ieq $NormalizedInstallDirectory } catch { $false }
    }

    if (-not $AlreadyInPath) {
        [Environment]::SetEnvironmentVariable('Path', (($PathEntries + $InstallDirectory) -join ';'), 'User')
        $env:Path = "$InstallDirectory;$env:Path"
    }

    Write-Host ""
    Write-Host "OpenPort installed to $Target" -ForegroundColor Green
    if (-not $AlreadyInPath) {
        Write-Host "Your user PATH was updated. Open a new PowerShell window, then run: openport"
    } else {
        Write-Host "Run: openport"
    }
}
catch {
    Fail $_.Exception.Message
}
finally {
    if (Test-Path -LiteralPath $TemporaryDirectory) {
        Remove-Item -LiteralPath $TemporaryDirectory -Recurse -Force -ErrorAction SilentlyContinue
    }
}
`;

function isWindowsPowerShellRequest(userAgent: string): boolean {
  return /windows|win32|powershell/i.test(userAgent);
}

export default function handler(req: RequestLike, res: ResponseLike): void {
  const userAgentHeader = req.headers?.["user-agent"];
  const userAgent = Array.isArray(userAgentHeader)
    ? userAgentHeader.join(" ")
    : userAgentHeader ?? "";

  res.statusCode = 200;
  res.setHeader("Content-Type", "text/plain; charset=utf-8");
  res.setHeader("Cache-Control", "no-store");
  res.end(isWindowsPowerShellRequest(userAgent) ? powerShellInstaller : posixInstaller);
}
