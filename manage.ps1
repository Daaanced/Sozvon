param(
    [string]$Action = "start"
)

$GatewayPath = "D:\Sozvon\Gateway"
$AuthPath    = "D:\Sozvon\Services\Auth_Service"
$ChatPath    = "D:\Sozvon\Services\Chat_Service"
$UserPath    = "D:\Sozvon\Services\User_Service"
$VoicePath   = "D:\Sozvon\Services\Voice_Service"

$ClientPath  = "D:\Sozvon\sozvon-client"

$NginxExe    = "C:\Users\chern\Downloads\nginx-1.31.1\nginx-1.31.1\nginx.exe"

function Start-ServiceWindow($title, $workdir, $command)
{
    Start-Process powershell `
        -ArgumentList "-NoExit", "-Command", "cd '$workdir'; $command" `
        -WindowStyle Normal
}

function Build-GoService($path)
{
    Write-Host "Building $path"
    Push-Location $path
    go build .
    Pop-Location
}

switch ($Action)
{
    "build" {

        Build-GoService $GatewayPath
        Build-GoService $AuthPath
        Build-GoService $ChatPath
        Build-GoService $UserPath
        Build-GoService $VoicePath

        Write-Host "Build completed."
    }

    "start" {

        Start-ServiceWindow "Gateway" $GatewayPath ".\Gateway.exe"

        Start-ServiceWindow "Auth" $AuthPath ".\Auth_Service.exe"

        Start-ServiceWindow "Chat" $ChatPath ".\Chat_Service.exe"

        Start-ServiceWindow "User" $UserPath ".\User_Service.exe"

        Start-ServiceWindow "Voice" $VoicePath ".\Voice_Service.exe"

        Start-ServiceWindow "Client" $ClientPath "serve -s dist"

        Start-Process $NginxExe

        Write-Host "All services started."
    }

    "stop" {

        Get-Process Gateway -ErrorAction SilentlyContinue | Stop-Process -Force

        Get-Process Auth_Service -ErrorAction SilentlyContinue | Stop-Process -Force

        Get-Process Chat_Service -ErrorAction SilentlyContinue | Stop-Process -Force

        Get-Process User_Service -ErrorAction SilentlyContinue | Stop-Process -Force

        Get-Process Voice_Service -ErrorAction SilentlyContinue | Stop-Process -Force

        Get-Process nginx -ErrorAction SilentlyContinue | Stop-Process -Force

        Get-Process node -ErrorAction SilentlyContinue |
            Where-Object {
                $_.Path -like "*serve*"
            } |
            Stop-Process -Force

        Write-Host "All services stopped."
    }

    "restart" {

        & $PSCommandPath stop
        Start-Sleep 2
        & $PSCommandPath start
    }

    default {
        Write-Host "Usage:"
        Write-Host ".\manage.ps1 build"
        Write-Host ".\manage.ps1 start"
        Write-Host ".\manage.ps1 stop"
        Write-Host ".\manage.ps1 restart"
    }
}