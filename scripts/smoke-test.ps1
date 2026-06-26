param(
  [string]$ApiBaseUrl = "http://localhost:8080"
)

$ErrorActionPreference = "Stop"
$base = $ApiBaseUrl.TrimEnd("/")

function Invoke-JsonPost {
  param(
    [string]$Path,
    [object]$Body
  )

  Invoke-RestMethod -Method Post -Uri "$base$Path" -ContentType "application/json" -Body ($Body | ConvertTo-Json -Depth 8)
}

Write-Host "Checking health..."
$health = Invoke-RestMethod "$base/health"
if ($health.status -ne "ok") {
  throw "Health check failed."
}

$status = Invoke-RestMethod "$base/api/system/status"
if (-not $status.database_ok) {
  throw "Database status check failed."
}
Write-Host "AI configured: $($status.ai_configured); embeddings configured: $($status.embedding_configured)"

Write-Host "Binding profile..."
$profile = Invoke-JsonPost "/api/profiles/bind" @{
  profile_key = "profile-smoke"
  profile_name = "Profile Smoke"
  remark = "Created by smoke test"
}

Write-Host "Creating site..."
$site = Invoke-JsonPost "/api/sites" @{
  site_key = "smoke-site"
  site_name = "Smoke Survey"
  domain = "smoke.example.com"
}

Write-Host "Creating survey..."
$survey = Invoke-JsonPost "/api/surveys" @{
  profile_id = $profile.id
  site_key = $site.site_key
  survey_title = "Smoke Test Survey"
  survey_url = "https://smoke.example.com/survey"
}

Write-Host "Saving page snapshot..."
$snapshot = Invoke-JsonPost "/api/page-snapshots" @{
  profile_id = $profile.id
  site_key = $site.site_key
  survey_id = $survey.id
  url = "https://smoke.example.com/survey/page-1"
  page_title = "Smoke Page"
  question_text = "How often do you buy soft drinks?"
  options_text = "Daily; Weekly; Monthly; Never"
  page_text = "How often do you buy soft drinks? Daily Weekly Monthly Never"
  language = "en"
}

Write-Host "Saving note..."
$note = Invoke-JsonPost "/api/notes" @{
  profile_id = $profile.id
  site_key = $site.site_key
  survey_id = $survey.id
  page_snapshot_id = $snapshot.id
  note_text = "Smoke note for beverage frequency wording."
}

Write-Host "Searching knowledge..."
$query = [uri]::EscapeDataString("beverage frequency")
$knowledge = Invoke-RestMethod "$base/api/knowledge/search?profile_id=$($profile.id)&site_key=$($site.site_key)&q=$query"

Write-Host "Smoke test complete."
[pscustomobject]@{
  profile_id = $profile.id
  site_key = $site.site_key
  survey_id = $survey.id
  snapshot_id = $snapshot.id
  note_id = $note.id
  knowledge_results = $knowledge.items.Count
  ai_configured = $status.ai_configured
  embedding_configured = $status.embedding_configured
}
