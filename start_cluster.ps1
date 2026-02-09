# start_cluster.ps1

# 1. Kill any existing Go processes to free up ports
Write-Host "Cleaning up old processes..." -ForegroundColor Yellow
Stop-Process -Name "node", "router" -ErrorAction SilentlyContinue

# 2. Compile the binaries
#    Compiling is better than 'go run' for background processes
Write-Host "Compiling Node..." -ForegroundColor Cyan
go build -o node.exe main.go

Write-Host "Compiling Router..." -ForegroundColor Cyan
go build -o router.exe router/router.go

# 3. Start 3 Nodes (Ports 8081, 8082, 8083)
Write-Host "Starting Node A on :8081..." -ForegroundColor Green
Start-Process -FilePath ".\node.exe" -ArgumentList "-port", "8081" -NoNewWindow

Write-Host "Starting Node B on :8082..." -ForegroundColor Green
Start-Process -FilePath ".\node.exe" -ArgumentList "-port", "8082" -NoNewWindow

Write-Host "Starting Node C on :8083..." -ForegroundColor Green
Start-Process -FilePath ".\node.exe" -ArgumentList "-port", "8083" -NoNewWindow

# 4. Start Router (Port 8080)
Write-Host "Starting Router on :8080..." -ForegroundColor Magenta
Start-Process -FilePath ".\router.exe" -NoNewWindow

Write-Host "------------------------------------------------"
Write-Host "CLUSTER IS LIVE!" -ForegroundColor Cyan
Write-Host "Router: http://localhost:8080"
Write-Host "Nodes:  :8081, :8082, :8083"
Write-Host "To stop: Run 'Stop-Process -Name node, router -Force' or close this window."
Write-Host "------------------------------------------------"