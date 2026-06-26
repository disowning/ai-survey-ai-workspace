param(
  [string]$Container = "survey-ai-workspace-postgres",
  [string]$Database = "survey_ai_workspace",
  [string]$User = "survey_ai"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$migrationDir = Join-Path $root "migrations"

if (-not (Test-Path $migrationDir)) {
  throw "Migration directory not found: $migrationDir"
}

$files = Get-ChildItem -LiteralPath $migrationDir -Filter "*.up.sql" | Sort-Object Name
if (-not $files) {
  throw "No migration files found."
}

foreach ($file in $files) {
  Write-Host "Applying $($file.Name)"
  Get-Content -Raw -LiteralPath $file.FullName | docker exec -i $Container psql -v ON_ERROR_STOP=1 -U $User -d $Database
}

Write-Host "Migrations complete."
