@echo off
setlocal
cd /d "%~dp0"

echo Starting update agent...
echo Config: %CD%\config\agent.json
echo URL   : http://127.0.0.1:45100
echo.

"%CD%\bin\client-update-console.exe" -config "%CD%\config\agent.json"

endlocal
