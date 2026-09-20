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
if exist "bin\update-agent.exe" del /q "bin\update-agent.exe"
if exist "bin\update-agent-console.exe" del /q "bin\update-agent-console.exe"
if exist "bin\client-update.exe" del /q "bin\client-update.exe"
if exist "bin\client-update-console.exe" del /q "bin\client-update-console.exe"
if exist "bin\client-agent.exe" del /q "bin\client-agent.exe"

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

echo.
echo Build completed successfully.
echo.
echo Output:
echo   bin\update-server.exe
echo   bin\client-update.exe
echo   bin\client-agent.exe
echo   bin\client-update-console.exe
echo.

endlocal
