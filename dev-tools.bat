@echo off
REM PAW Blockchain Development Tools Helper
REM Windows Batch Script Wrapper

setlocal enabledelayedexpansion

echo ========================================
echo   PAW Blockchain Development Tools
echo ========================================
echo.

if "%1"=="" goto :help
if "%1"=="help" goto :help
if "%1"=="-h" goto :help
if "%1"=="--help" goto :help

if "%1"=="setup" goto :setup
if "%1"=="clean" goto :clean
if "%1"=="format" goto :format
if "%1"=="test" goto :test
if "%1"=="build" goto :build
if "%1"=="lint" goto :lint
if "%1"=="dev" goto :dev

echo Unknown command: %1
goto :help

:help
echo Usage: dev-tools.bat [command]
echo.
echo Commands:
echo   setup   - Run development environment setup
echo   build   - Build pawd and pawcli binaries
echo   test    - Run all tests
echo   lint    - Run linter
echo   format  - Format all code
echo   clean   - Clean build artifacts
echo   dev     - Start Docker development environment
echo   help    - Show this help message
echo.
echo Examples:
echo   dev-tools.bat setup
echo   dev-tools.bat build
echo   dev-tools.bat test
echo.
goto :end

:setup
echo Running development setup...
powershell -ExecutionPolicy Bypass -File scripts\dev-setup.ps1
goto :end

:clean
echo Running cleanup...
powershell -ExecutionPolicy Bypass -File scripts\clean.ps1
goto :end

:format
echo Formatting code...
powershell -ExecutionPolicy Bypass -File scripts\format-all.ps1
goto :end

:test
echo Running tests...
go test -v -race ./...
goto :end

:build
echo Building binaries...
mkdir build 2>nul
go build -o build\pawd.exe .\cmd\pawd
go build -o build\pawcli.exe .\cmd\pawcli
echo Build complete: build\pawd.exe, build\pawcli.exe
goto :end

:lint
echo Running linter...
golangci-lint run --timeout=10m
goto :end

:dev
echo Starting Docker development environment...
docker-compose -f docker-compose.dev.yml up --build
goto :end

:end
endlocal
