@echo off
setlocal
cd /d "%~dp0"

echo Starting update server...
echo Config: %CD%\config\server.json
echo URL   : http://0.0.0.0:45000
echo.

"%CD%\bin\update-server.exe" -config "%CD%\config\server.json"

endlocal
