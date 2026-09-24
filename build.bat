@echo off
setlocal

cd /d "%~dp0"

echo.
echo == TNLX Software Updater Build ==
echo.

where go >nul 2>nul
if errorlevel 1 (
    echo ERROR: Go is not installed or not available in PATH.
    exit /b 1
)

echo [1/4] Go version
go version
if errorlevel 1 exit /b 1

echo.
echo [2/4] Running tests
go test ./...
if errorlevel 1 (
    echo ERROR: Tests failed.
    exit /b 1
)

echo.
echo [3/4] Preparing bin folder
if not exist "bin" mkdir "bin"
if errorlevel 1 exit /b 1
if exist "bin\update-agent.exe" del /q "bin\update-agent.exe" || (echo ERROR: Cannot delete bin\update-agent.exe. Close the running program and build again.& exit /b 1)
if exist "bin\update-agent-console.exe" del /q "bin\update-agent-console.exe" || (echo ERROR: Cannot delete bin\update-agent-console.exe. Close the running program and build again.& exit /b 1)
if exist "bin\client-update.exe" del /q "bin\client-update.exe" || (echo ERROR: Cannot delete bin\client-update.exe. Close the running program and build again.& exit /b 1)
if exist "bin\client-update-console.exe" del /q "bin\client-update-console.exe" || (echo ERROR: Cannot delete bin\client-update-console.exe. Close the running program and build again.& exit /b 1)
if exist "bin\client-agent.exe" del /q "bin\client-agent.exe" || (echo ERROR: Cannot delete bin\client-agent.exe. Close the running program and build again.& exit /b 1)

echo.
echo [4/4] Building executables
go build -o "bin\update-server.exe" ".\cmd\update-server"
if errorlevel 1 (
    echo ERROR: Failed to build update-server.exe.
    exit /b 1
)

go build -ldflags="-H windowsgui" -o "bin\client-update.exe" ".\cmd\update-agent"
if errorlevel 1 (
    echo ERROR: Failed to build client-update.exe.
    exit /b 1
)

copy /y "bin\client-update.exe" "bin\client-agent.exe" >nul
if errorlevel 1 (
    echo ERROR: Failed to create client-agent.exe.
    exit /b 1
)

go build -o "bin\client-update-console.exe" ".\cmd\update-agent"
if errorlevel 1 (
    echo ERROR: Failed to build client-update-console.exe.
    exit /b 1
)

if exist "bin\encrypt-agent-config.exe" del /q "bin\encrypt-agent-config.exe" || (echo ERROR: Cannot delete bin\encrypt-agent-config.exe. Close the running program and build again.& exit /b 1)
go build -o "bin\encrypt-agent-config.exe" ".\cmd\encrypt-agent-config"
if errorlevel 1 (
    echo ERROR: Failed to build encrypt-agent-config.exe.
    exit /b 1
)

if exist "Start_update_agent.exe" del /q "Start_update_agent.exe" || (echo ERROR: Cannot delete Start_update_agent.exe. Close the running program and build again.& exit /b 1)
go build -ldflags="-H windowsgui" -o "Start_update_agent.exe" ".\cmd\start-update-agent"
if errorlevel 1 (
    echo ERROR: Failed to build Start_update_agent.exe.
    exit /b 1
)

"%CD%\bin\encrypt-agent-config.exe" -in "%CD%\config\agent.json" -out "%CD%\config\agent.enc"
if errorlevel 1 (
    echo ERROR: Failed to encrypt config\agent.json.
    exit /b 1
)

if not exist "bin\linux-amd64" mkdir "bin\linux-amd64"
if errorlevel 1 exit /b 1
if exist "bin\linux-amd64\update-server" del /q "bin\linux-amd64\update-server" || (echo ERROR: Cannot delete bin\linux-amd64\update-server.& exit /b 1)

set "CGO_ENABLED=0"
set "GOOS=linux"
set "GOARCH=amd64"
go build -o "bin\linux-amd64\update-server" ".\cmd\update-server"
if errorlevel 1 (
    echo ERROR: Failed to build linux-amd64 update-server.
    exit /b 1
)

echo.
echo Build completed successfully.
echo.
echo Output:
echo   bin\update-server.exe
echo   bin\client-update.exe
echo   bin\client-agent.exe
echo   bin\client-update-console.exe
echo   bin\encrypt-agent-config.exe
echo   bin\linux-amd64\update-server
echo   Start_update_agent.exe
echo   config\agent.enc
echo.

endlocal
