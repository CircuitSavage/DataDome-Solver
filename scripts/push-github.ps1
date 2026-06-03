# Push DataDome-Solver to https://github.com/CircuitSavage/DataDome-Solver
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

$GhDir = Join-Path $env:TEMP "gh-cli"
$Gh = Get-ChildItem -Path $GhDir -Recurse -Filter "gh.exe" -ErrorAction SilentlyContinue | Select-Object -First 1
if (-not $Gh) {
    Write-Host "Downloading GitHub CLI..."
    $zip = Join-Path $env:TEMP "gh.zip"
    Invoke-WebRequest -Uri "https://github.com/cli/cli/releases/download/v2.63.2/gh_2.63.2_windows_amd64.zip" -OutFile $zip -UseBasicParsing
    Expand-Archive -Path $zip -DestinationPath $GhDir -Force
    $Gh = Get-ChildItem -Path $GhDir -Recurse -Filter "gh.exe" | Select-Object -First 1
}
$GhExe = $Gh.FullName

Set-Location $Root

$authOk = $false
& $GhExe auth status *>$null
if ($LASTEXITCODE -eq 0) { $authOk = $true }

if (-not $authOk) {
    Write-Host ""
    Write-Host ">>> Open https://github.com/login/device and enter the code shown below <<<"
    Write-Host ""
    & $GhExe auth login -h github.com -p https -w
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

git branch -M main 2>$null
git remote remove origin 2>$null

& $GhExe repo view CircuitSavage/DataDome-Solver *>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "Creating CircuitSavage/DataDome-Solver ..."
    & $GhExe repo create CircuitSavage/DataDome-Solver --public `
        --description "Production-ready Go library & CLI to generate DataDome fingerprints, encrypt jspl payloads, and obtain session cookies — no browser required." `
        --source $Root --remote origin --push
} else {
    git remote add origin https://github.com/CircuitSavage/DataDome-Solver.git
    git push -u origin main
}

if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "Live at https://github.com/CircuitSavage/DataDome-Solver"
