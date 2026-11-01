$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$backupDirectory = Join-Path $projectRoot 'backups'
New-Item -ItemType Directory -Path $backupDirectory -Force | Out-Null
$filename = 'rahcar-' + (Get-Date -Format 'yyyyMMdd-HHmmss') + '.dump'

Push-Location $projectRoot
try {
  docker compose exec -T db pg_dump -U rahcar -d rahcar -Fc -f "/backups/$filename"
  if ($LASTEXITCODE -ne 0) { throw 'PostgreSQL backup failed.' }
  Write-Output (Join-Path $backupDirectory $filename)
}
finally {
  Pop-Location
}
