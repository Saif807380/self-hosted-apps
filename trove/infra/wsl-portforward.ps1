# WSL2 Port Forwarding for Trove
# Run as Administrator in PowerShell
# Usage: .\wsl-portforward.ps1

$ErrorActionPreference = "Stop"

# Get WSL2 IP
$wslIp = (wsl hostname -I).Trim().Split(" ")[0]
if (-not $wslIp) {
    Write-Error "Could not get WSL2 IP. Is WSL running?"
    exit 1
}
Write-Host "WSL2 IP: $wslIp"

$ports = @(3000, 3443, 8080)

# Clear existing rules
foreach ($port in $ports) {
    netsh interface portproxy delete v4tov4 listenport=$port listenaddress=0.0.0.0 2>$null
}

# Add port forwarding
foreach ($port in $ports) {
    netsh interface portproxy add v4tov4 listenport=$port listenaddress=0.0.0.0 connectport=$port connectaddress=$wslIp
    Write-Host "Forwarding 0.0.0.0:$port -> ${wslIp}:$port"
}

# Add firewall rules (idempotent — removes first if exists)
# Clean up legacy rule name if present
netsh advfirewall firewall delete rule name="Trove HTTPS" 2>$null
$ruleName = "Trove WSL2 Ports"
netsh advfirewall firewall delete rule name="$ruleName" 2>$null
netsh advfirewall firewall add rule name="$ruleName" dir=in action=allow protocol=tcp localport="3000,3443,8080"

Write-Host "`nPort forwarding active. Verify with:"
Write-Host "  netsh interface portproxy show v4tov4"
