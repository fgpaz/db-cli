[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Args
)

$ErrorActionPreference = "Stop"

$scriptPath = if (-not [string]::IsNullOrWhiteSpace($PSCommandPath)) {
    $PSCommandPath
} else {
    $MyInvocation.MyCommand.Path
}

if ([string]::IsNullOrWhiteSpace($scriptPath)) {
    throw "No se pudo resolver la ruta de db-cli.ps1."
}

$scriptDir = Split-Path -Parent $scriptPath
$skillRoot = Split-Path -Parent $scriptDir
$arch = if ($env:PROCESSOR_ARCHITECTURE -match "ARM64") { "arm64" } else { "amd64" }
$binaryPath = Join-Path $skillRoot "bin\windows-$arch\db-cli.exe"
$fallbackPath = Join-Path $skillRoot "bin\db-cli.exe"

function Find-NearestEnvFile {
    param(
        [string]$StartPath
    )

    if ([string]::IsNullOrWhiteSpace($StartPath)) {
        $StartPath = $PWD.Path
    }

    $cursor = (Resolve-Path -LiteralPath $StartPath).Path
    while ($true) {
        $candidate = Join-Path $cursor "infra\.env"
        if (Test-Path -LiteralPath $candidate) {
            return $candidate
        }

        $parent = Split-Path -Parent $cursor
        if ([string]::IsNullOrWhiteSpace($parent) -or $parent -eq $cursor) {
            return $null
        }
        $cursor = $parent
    }
}

function Import-EnvFile {
    param(
        [string]$Path
    )

    foreach ($line in Get-Content -LiteralPath $Path) {
        $trimmed = $line.Trim()
        if ([string]::IsNullOrWhiteSpace($trimmed) -or $trimmed.StartsWith("#")) {
            continue
        }
        if ($trimmed.StartsWith("export ")) {
            $trimmed = $trimmed.Substring(7).Trim()
        }
        $index = $trimmed.IndexOf("=")
        if ($index -lt 1) {
            continue
        }

        $key = $trimmed.Substring(0, $index).Trim()
        if ($key -notmatch '^[A-Za-z_][A-Za-z0-9_]*$') {
            continue
        }
        $existing = Get-Item -Path "Env:$key" -ErrorAction SilentlyContinue
        if ($null -eq $existing) {
            $value = $trimmed.Substring($index + 1).Trim()
            if ($value.Length -ge 2) {
                if (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'"))) {
                    $value = $value.Substring(1, $value.Length - 2)
                }
            }
            Set-Item -Path "Env:$key" -Value $value
        }
    }
}

$envFile = Find-NearestEnvFile -StartPath (Get-Location).Path
if ($envFile) {
    Import-EnvFile -Path $envFile
}

if (-not (Test-Path $binaryPath) -and (Test-Path $fallbackPath)) {
    $binaryPath = $fallbackPath
}

if (-not (Test-Path $binaryPath)) {
    Write-Error "No se encontro db-cli.exe en '$binaryPath'. Compilalo con: & '$scriptDir\build.ps1' -AllTargets"
}

if ($null -eq $Args) {
    $Args = @()
} else {
    $Args = @($Args)
}

& $binaryPath @Args
$exitCode = $LASTEXITCODE
if ($null -ne $exitCode) {
    exit $exitCode
}
