[CmdletBinding()]
param(
    [ValidateSet("windows", "linux")]
    [string]$Os,
    [ValidateSet("amd64", "arm64")]
    [string]$Arch,
    [switch]$AllTargets
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$skillRoot = Split-Path -Parent $scriptDir
$binRoot = Join-Path $skillRoot "bin"

$goCommand = Get-Command go.exe -ErrorAction SilentlyContinue
if (-not $goCommand) {
    throw "No se encontro go.exe en PATH. Instala Go para recompilar db-cli."
}

$goSource = Get-ChildItem -Path $scriptDir -Filter *.go -File -ErrorAction SilentlyContinue
if (-not $goSource) {
    throw "No hay archivos Go en '$scriptDir'. Agrega el core de db-cli antes de compilar."
}

$programFilesGo = Join-Path ${env:ProgramFiles} "Go"
if (Test-Path $programFilesGo) {
    $env:GOROOT = $programFilesGo
}

if (-not $env:GOPATH -or -not (Test-Path $env:GOPATH)) {
    $env:GOPATH = Join-Path $env:USERPROFILE "go"
}

function Get-HostArch {
    if ($env:PROCESSOR_ARCHITECTURE -match "ARM64") {
        return "arm64"
    }
    return "amd64"
}

if ($AllTargets) {
    $targets = @(
        @{ Goos = "windows"; Goarch = "amd64" },
        @{ Goos = "windows"; Goarch = "arm64" },
        @{ Goos = "linux"; Goarch = "amd64" },
        @{ Goos = "linux"; Goarch = "arm64" }
    )
} else {
    if (-not $Os) {
        $Os = "windows"
    }
    if (-not $Arch) {
        $Arch = Get-HostArch
    }
    $targets = @(@{ Goos = $Os; Goarch = $Arch })
}

New-Item -ItemType Directory -Force -Path $binRoot | Out-Null

Write-Host "Compilando db-cli..."
Write-Host "  go: $($goCommand.Source)"
Write-Host "  GOROOT: $env:GOROOT"
Write-Host "  GOPATH: $env:GOPATH"

Push-Location $scriptDir
try {
    & $goCommand.Source mod tidy

    foreach ($target in $targets) {
        $goos = $target.Goos
        $goarch = $target.Goarch
        $targetDir = Join-Path $binRoot "$goos-$goarch"
        $binaryName = if ($goos -eq "windows") { "db-cli.exe" } else { "db-cli" }
        $outputPath = Join-Path $targetDir $binaryName

        New-Item -ItemType Directory -Force -Path $targetDir | Out-Null

        Write-Host "  target: $goos/$goarch"
        $env:CGO_ENABLED = "0"
        $env:GOOS = $goos
        $env:GOARCH = $goarch
        & $goCommand.Source build -buildvcs=false -o $outputPath .

        if (-not (Test-Path $outputPath)) {
            throw "La compilacion termino sin generar '$outputPath'."
        }

        if ($goos -eq "windows" -and $goarch -eq (Get-HostArch)) {
            Copy-Item -Force $outputPath (Join-Path $binRoot "db-cli.exe")
        }
    }
} finally {
    Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    Pop-Location
}

Get-ChildItem -Recurse -File $binRoot | Select-Object FullName, Length, LastWriteTime
