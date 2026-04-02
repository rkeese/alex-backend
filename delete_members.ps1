$ErrorActionPreference = "Stop"
$tokenPath = Join-Path $PSScriptRoot "token.txt"

if (-not (Test-Path $tokenPath)) {
    Write-Error "Token file not found at $tokenPath. Create a token.txt file with a valid JWT token."
    exit 1
}

$token = (Get-Content $tokenPath | Select-Object -First 1).Trim()
# Set your Club ID here
$clubId = "YOUR_CLUB_ID"
$baseUrl = "http://localhost:8080/api/v1"

Write-Host "----------------------------------------"
Write-Host "Cleanup Script for Alex Backend"
Write-Host "Club ID: $clubId"
Write-Host "Base URL: $baseUrl"
Write-Host "----------------------------------------"

try {
    Write-Host "Fetching existing members..."
    # Important: Pass Club ID as query parameter so middleware and handler know the context
    $members = Invoke-RestMethod -Uri "$baseUrl/members?club_id=$clubId" -Headers @{Authorization="Bearer $token"}
    
    # PowerShell oddity: valid JSON null result might be $null, single obj is PSCustomObject, list is Array
    if ($null -eq $members) {
        $members = @()
    } elseif ($members -isnot [System.Array]) {
        $members = @($members)
    }

    $count = $members.Count
    Write-Host "Found $count members to delete."

    if ($count -gt 0) {
        foreach ($m in $members) {
            $id = $m.id
            $name = "$($m.first_name) $($m.last_name)"
            # Use explicit subexpression or format operator to avoid interpolation issues
            $deleteUrl = "{0}/members/{1}?club_id={2}" -f $baseUrl, $m.id, $clubId
            Write-Host "Deleting: $name ($($m.id))..." -NoNewline
            Write-Host " [DEBUG URL: $deleteUrl]" -ForegroundColor Gray
            try {
                Invoke-RestMethod -Uri $deleteUrl -Method Delete -Headers @{Authorization="Bearer $token"}
                Write-Host " OK" -ForegroundColor Green
            } catch {
                Write-Host " FAILED" -ForegroundColor Red
                Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
            }
        }
    } else {
        Write-Host "No members found to delete." -ForegroundColor Yellow
    }
    Write-Host "----------------------------------------"
    Write-Host "Cleanup complete."
} catch {
    Write-Host "Fatal Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        # Try to read error body
        try {
            $stream = $_.Exception.Response.GetResponseStream()
            if ($stream) {
                $reader = New-Object System.IO.StreamReader($stream)
                $body = $reader.ReadToEnd()
                Write-Host "Server Response: $body" -ForegroundColor Red
            }
        } catch {}
    }
}
