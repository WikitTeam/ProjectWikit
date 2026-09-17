# Installs a ProjectWikit release on Windows.
#
#   irm https://github.com/WikitTeam/ProjectWikit/releases/latest/download/install.ps1 | iex
#   irm <mirror>/latest/download/install.ps1 | iex
#
# Piped into iex, the script takes its options from environment variables:
# PWIKIT_VERSION, PWIKIT_DIR, PWIKIT_MIRROR and PWIKIT_NO_PATH. Saved to a file,
# it also takes them as parameters:
#
#   .\install.ps1 -Version v1.0.0 -Dir D:\pwikit -Mirror <url> -NoPath
param(
    [string]$Version = $env:PWIKIT_VERSION,
    [string]$Dir = $env:PWIKIT_DIR,
    [string]$Mirror = $env:PWIKIT_MIRROR,
    [switch]$NoPath = [bool]$env:PWIKIT_NO_PATH
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

function Fail([string]$Message) {
    Write-Host "install.ps1: $Message" -ForegroundColor Red
    throw $Message
}

# A mirror that hands out this script writes its own address here, so the
# script downloads from that mirror without trying GitHub first.
$ServedBy = ''

$Releases = if ($env:PWIKIT_RELEASES_URL) { $env:PWIKIT_RELEASES_URL } elseif ($ServedBy) { $ServedBy } else { 'https://github.com/WikitTeam/ProjectWikit/releases' }
$Releases = $Releases.TrimEnd('/')
if (-not $Mirror) { $Mirror = $ServedBy }
if ($Mirror) { $Mirror = $Mirror.TrimEnd('/') }
$FirstTimeout = if ($ServedBy -and $Releases -eq $ServedBy.TrimEnd('/')) { 600 } else { 60 }
if (-not $Dir) { $Dir = (Get-Location).Path }

if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64' -and $env:PROCESSOR_ARCHITEW6432 -ne 'AMD64') {
    if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
        Write-Host 'No ARM64 build exists; installing the x64 build, which Windows 11 runs through emulation.'
    } else {
        Fail "no release is built for $($env:PROCESSOR_ARCHITECTURE)"
    }
}

if (Test-Path (Join-Path $Dir 'pwikit.exe')) {
    Fail "$Dir already holds pwikit; update it with: $Dir\pwikit.exe update"
}
if ((Test-Path $Dir) -and (Get-ChildItem -Force $Dir | Select-Object -First 1)) {
    Fail "$Dir is not empty; run this from an empty directory, or pass -Dir with a new or empty one"
}

function Get-ReleaseFile([string]$Path, [string]$OutFile) {
    try {
        Invoke-WebRequest -UseBasicParsing -TimeoutSec $FirstTimeout -Uri "$Releases/$Path" -OutFile $OutFile
        return
    } catch {
        if (-not $Mirror -or $Mirror -eq $Releases) { throw }
    }
    Write-Host "GitHub could not be reached, trying $Mirror"
    Invoke-WebRequest -UseBasicParsing -TimeoutSec 600 -Uri "$Mirror/$Path" -OutFile $OutFile
}

$work = Join-Path ([IO.Path]::GetTempPath()) ("pwikit-install-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $work | Out-Null
try {
    $platform = 'windows-amd64'
    if (-not $Version) {
        try { Get-ReleaseFile 'latest/download/latest.json' (Join-Path $work 'latest.json') }
        catch { Fail 'could not fetch latest.json; check the network, or pass -Mirror' }
        $latest = Get-Content -Raw (Join-Path $work 'latest.json') | ConvertFrom-Json
        $Version = $latest.version
        $entry = $latest.packages.$platform
        if (-not $entry) { Fail "latest.json has no package for $platform" }
        $file = $entry.file
        $want = $entry.sha256
    } else {
        if (-not $Version.StartsWith('v')) { $Version = "v$Version" }
        $file = "pwikit-$Version-$platform.zip"
        try { Get-ReleaseFile "download/$Version/SHA256SUMS" (Join-Path $work 'SHA256SUMS') }
        catch { Fail "could not fetch the checksums of $Version; does that release exist?" }
        $want = $null
        foreach ($line in Get-Content (Join-Path $work 'SHA256SUMS')) {
            $parts = $line -split '\s+'
            if ($parts.Count -ge 2 -and $parts[1] -eq $file) { $want = $parts[0] }
        }
        if (-not $want) { Fail "release $Version has no package for $platform" }
    }

    Write-Host "Downloading pwikit $Version for $platform"
    $archive = Join-Path $work $file
    try { Get-ReleaseFile "download/$Version/$file" $archive }
    catch { Fail "could not download $file" }
    $got = (Get-FileHash -Algorithm SHA256 $archive).Hash.ToLowerInvariant()
    if ($got -ne $want.ToLowerInvariant()) {
        Fail "$file has sha256 $got, want $want; the download is damaged or was altered"
    }

    $unpacked = Join-Path $work 'unpacked'
    Expand-Archive -Path $archive -DestinationPath $unpacked
    $top = Join-Path $unpacked "pwikit-$Version-$platform"
    if (-not (Test-Path (Join-Path $top 'pwikit.exe'))) { Fail "$file does not hold pwikit.exe" }

    New-Item -ItemType Directory -Force -Path $Dir | Out-Null
    Copy-Item (Join-Path $top 'pwikit.exe') $Dir
    foreach ($doc in 'LICENSE', 'NOTICE') {
        if (Test-Path (Join-Path $top $doc)) { Copy-Item (Join-Path $top $doc) $Dir }
    }
    Write-Host "Installed pwikit $Version into $Dir"

    if ($Mirror) {
        $saved = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        & (Join-Path $Dir 'pwikit.exe') update mirror -data-dir $Dir $Mirror *> $null
        $ErrorActionPreference = $saved
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Could not write the mirror into $Dir\pwikit.toml; set it later with: $Dir\pwikit.exe update mirror $Mirror"
        }
    }

    if (-not $NoPath) {
        & (Join-Path $Dir 'pwikit.exe') path install
        if ($LASTEXITCODE -ne 0) {
            Write-Host "pwikit is installed, but could not be made runnable by name; run $Dir\pwikit.exe path install later"
        }
    }

    Write-Host ''
    Write-Host 'Next, create the site from that directory:'
    Write-Host "  cd $Dir"
    Write-Host '  .\pwikit.exe createsite -slug main -title "My Wiki" -headline "A wiki" -domain wiki.example.org -media-domain files.example.org'
    Write-Host 'Then follow the quick start: https://github.com/WikitTeam/ProjectWikit/blob/main/docs/en/quickstart.md'
} finally {
    Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue
}
