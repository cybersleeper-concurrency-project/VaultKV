
# chaos_test.ps1

Write-Host "=== CHAOS TEST STARTED ===" -ForegroundColor Cyan -BackgroundColor Black

# 1. Start Cluster (Assumes running from project root)
if (Test-Path ".\start_cluster.ps1") {
    Write-Host "Starting Cluster..."
    & .\start_cluster.ps1
} else {
    Write-Error "Please run this script from the project root directory (e.g., .\test\chaos_test.ps1)."
    exit 1
}

# 2. Wait for stabilization
Write-Host "Waiting 5s for cluster to stabilize..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# 3. Start Load Test (crash_test) concurrently
# Using Start-Job keeps the console clean for the script output, but receiving output is tricky live.
# Using Start-Process -NoNewWindow mixes output, which is what we want to see the errors.
Write-Host "Starting Crash Test (20 iterations)..." -ForegroundColor Yellow
$loadTest = Start-Process go -ArgumentList "run .\test\crash_test\main.go" -NoNewWindow -PassThru

# 4. Let it run for ~10 seconds (approx 2 iterations)
Start-Sleep -Seconds 10

# 5. KILL NODE B (Port 8082)
Write-Host "`n!!! KILLING NODE B (Port 8082) !!!" -ForegroundColor Red -BackgroundColor Black
try {
    # Find the process listening on port 8082
    $conn = Get-NetTCPConnection -LocalPort 8082 -State Listen -ErrorAction Stop
    $pidToKill = $conn.OwningProcess
    Stop-Process -Id $pidToKill -Force
    Write-Host "Node B (PID: $pidToKill) Killed."
} catch {
    Write-Warning "Could not find/kill Node B on port 8082: $_"
}

# 6. Let it run in DEGRADED State for ~10 seconds
Write-Host "Running in DEGRADED State for 10s..."
Start-Sleep -Seconds 10

# 7. Restart Node B
Write-Host "`n!!! RESTARTING NODE B !!!" -ForegroundColor Green -BackgroundColor Black
Start-Process -FilePath ".\node.exe" -ArgumentList "-port", "8082" -NoNewWindow
Write-Host "Node B Restarted."

# 8. Wait for Load Test to finish
Write-Host "Waiting for load test to finish..."
$loadTest | Wait-Process

Write-Host "`n=== CHAOS TEST FINISHED ===" -ForegroundColor Cyan
