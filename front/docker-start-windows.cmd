@echo off
setlocal

rem Always run from the directory that contains this script.
rem This avoids PowerShell path syntax and supports directory names with spaces.
pushd "%~dp0"

echo [1/3] Checking Docker Desktop...
docker version >nul 2>&1
if errorlevel 1 goto docker_error

echo [2/3] Building the frontend image...
docker build -f Dockerfile.dev -t video-front-dev .
if errorlevel 1 goto build_error

echo [3/3] Starting the frontend...
echo.
echo Open this URL in your browser:
echo http://localhost:5173
echo.
echo Press Ctrl+C to stop.
echo.

docker run --rm -it -p 5173:5173 ^
  --mount "type=bind,source=%CD%,target=/app" ^
  --mount "type=volume,target=/app/node_modules" ^
  video-front-dev

set "RESULT=%ERRORLEVEL%"
popd
exit /b %RESULT%

:docker_error
echo.
echo ERROR: Docker Desktop is not running.
echo Start Docker Desktop, wait until it is ready, and run this file again.
echo.
popd
pause
exit /b 1

:build_error
echo.
echo ERROR: Failed to build the frontend image.
echo Review the Docker error shown above.
echo.
popd
pause
exit /b 1
