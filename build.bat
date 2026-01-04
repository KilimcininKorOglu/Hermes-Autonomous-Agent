@echo off
setlocal enabledelayedexpansion

:: Hermes - AI-Powered Application Development
:: Build script for Windows

:: Variables
set "BINARY_DIR=bin"
set "BINARY_NAME=hermes"

:: Get version from git
for /f "tokens=*" %%i in ('git describe --tags --always --dirty 2^>nul') do set "VERSION=%%i"
if "%VERSION%"=="" set "VERSION=dev"

:: Get build time
for /f "tokens=*" %%i in ('powershell -command "Get-Date -Format 'yyyy-MM-dd_HH:mm:ss'"') do set "BUILD_TIME=%%i"

:: Get commit hash
for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set "COMMIT=%%i"
if "%COMMIT%"=="" set "COMMIT=unknown"

:: LDFLAGS
set "LDFLAGS=-ldflags "-X main.Version=%VERSION% -X main.BuildTime=%BUILD_TIME% -X main.Commit=%COMMIT%""

:: Parse command
if "%1"=="" goto :build-all
if "%1"=="help" goto :help
if "%1"=="-h" goto :help
if "%1"=="--help" goto :help
if "%1"=="build" goto :build
if "%1"=="build-linux" goto :build-linux
if "%1"=="build-linux-arm64" goto :build-linux-arm64
if "%1"=="build-windows" goto :build-windows
if "%1"=="build-windows-arm64" goto :build-windows-arm64
if "%1"=="build-darwin" goto :build-darwin
if "%1"=="build-darwin-arm64" goto :build-darwin-arm64
if "%1"=="build-all" goto :build-all
if "%1"=="test" goto :test
if "%1"=="test-short" goto :test-short
if "%1"=="lint" goto :lint
if "%1"=="fmt" goto :fmt
if "%1"=="vet" goto :vet
if "%1"=="check" goto :check
if "%1"=="clean" goto :clean
if "%1"=="run" goto :run
if "%1"=="run-tui" goto :run-tui
if "%1"=="deps" goto :deps
if "%1"=="deps-update" goto :deps-update
if "%1"=="install" goto :install

echo Unknown command: %1
echo Run 'build.bat help' for usage
exit /b 1

:: ==================== BUILD TARGETS ====================

:build
echo Building Hermes for Windows (amd64)...
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
echo Generating Windows resources...
if exist "%GOPATH%\bin\go-winres.exe" (
    "%GOPATH%\bin\go-winres.exe" make --in winres\winres.json --out cmd\hermes\rsrc_windows
)
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe" .\cmd\hermes
if errorlevel 1 (
    echo Build failed
    exit /b 1
)
echo Created: %BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe
goto :eof

:: ==================== CROSS-COMPILATION ====================

:build-linux
echo Building Hermes for Linux (amd64)...
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-linux-amd64" .\cmd\hermes
set GOOS=
set GOARCH=
set CGO_ENABLED=
echo Created: %BINARY_DIR%\%BINARY_NAME%-linux-amd64
goto :eof

:build-linux-arm64
echo Building Hermes for Linux (arm64)...
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=linux
set GOARCH=arm64
set CGO_ENABLED=0
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-linux-arm64" .\cmd\hermes
set GOOS=
set GOARCH=
set CGO_ENABLED=
echo Created: %BINARY_DIR%\%BINARY_NAME%-linux-arm64
goto :eof

:build-windows
echo Building Hermes for Windows (amd64)...
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
echo Generating Windows resources...
if exist "%GOPATH%\bin\go-winres.exe" (
    "%GOPATH%\bin\go-winres.exe" make --in winres\winres.json --out cmd\hermes\rsrc_windows
)
set GOOS=windows
set GOARCH=amd64
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe" .\cmd\hermes
set GOOS=
set GOARCH=
echo Created: %BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe
goto :eof

:build-windows-arm64
echo Building Hermes for Windows (arm64)...
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
echo Generating Windows resources...
if exist "%GOPATH%\bin\go-winres.exe" (
    "%GOPATH%\bin\go-winres.exe" make --in winres\winres.json --out cmd\hermes\rsrc_windows
)
set GOOS=windows
set GOARCH=arm64
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-windows-arm64.exe" .\cmd\hermes
set GOOS=
set GOARCH=
echo Created: %BINARY_DIR%\%BINARY_NAME%-windows-arm64.exe
goto :eof

:build-darwin
echo Building Hermes for macOS (amd64)...
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=darwin
set GOARCH=amd64
set CGO_ENABLED=0
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-darwin-amd64" .\cmd\hermes
set GOOS=
set GOARCH=
set CGO_ENABLED=
echo Created: %BINARY_DIR%\%BINARY_NAME%-darwin-amd64
goto :eof

:build-darwin-arm64
echo Building Hermes for macOS (arm64/Apple Silicon)...
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=darwin
set GOARCH=arm64
set CGO_ENABLED=0
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-darwin-arm64" .\cmd\hermes
set GOOS=
set GOARCH=
set CGO_ENABLED=
echo Created: %BINARY_DIR%\%BINARY_NAME%-darwin-arm64
goto :eof

:build-all
echo Building for all platforms...
call :build-linux
call :build-linux-arm64
call :build-windows
call :build-windows-arm64
call :build-darwin
call :build-darwin-arm64
echo All platform binaries built successfully
goto :eof

:: ==================== TEST TARGETS ====================

:test
echo Running tests...
go test -v -race -coverprofile=coverage.out ./...
if errorlevel 1 (
    echo Tests failed
    exit /b 1
)
go tool cover -html=coverage.out -o coverage.html
echo Tests passed. Coverage report: coverage.html
goto :eof

:test-short
echo Running short tests...
go test -v -short ./...
if errorlevel 1 (
    echo Tests failed
    exit /b 1
)
echo Short tests passed
goto :eof

:: ==================== CODE QUALITY ====================

:lint
echo Running linter...
golangci-lint run ./...
if errorlevel 1 (
    echo Linting failed
    exit /b 1
)
echo Linting passed
goto :eof

:fmt
echo Formatting code...
gofmt -s -w .
go mod tidy
echo Code formatted
goto :eof

:vet
echo Running go vet...
go vet ./...
if errorlevel 1 (
    echo go vet found issues
    exit /b 1
)
echo go vet passed
goto :eof

:check
echo Running all checks...
call :fmt
call :vet
call :lint
call :test
echo All checks passed
goto :eof

:: ==================== CLEAN ====================

:clean
echo Cleaning build artifacts...
if exist "%BINARY_DIR%" rmdir /s /q "%BINARY_DIR%"
if exist "coverage.out" del "coverage.out"
if exist "coverage.html" del "coverage.html"
if exist "hermes.exe" del "hermes.exe"
echo Cleaned
goto :eof

:: ==================== RUN TARGETS ====================

:run
call :build
if errorlevel 1 exit /b 1
echo Starting Hermes...
"%BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe" status
goto :eof

:run-tui
call :build
if errorlevel 1 exit /b 1
echo Starting Hermes TUI...
"%BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe" tui
goto :eof

:: ==================== DEPENDENCIES ====================

:deps
echo Downloading dependencies...
go mod download
go mod verify
echo Dependencies downloaded
goto :eof

:deps-update
echo Updating dependencies...
go get -u ./...
go mod tidy
echo Dependencies updated
goto :eof

:: ==================== INSTALL ====================

:install
call :build
if errorlevel 1 exit /b 1
echo Installing Hermes to GOPATH\bin...
copy "%BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe" "%GOPATH%\bin\hermes.exe"
echo Installed
goto :eof

:: ==================== HELP ====================

:help
echo.
echo Hermes - AI-Powered Application Development
echo ========================================
echo.
echo Usage: build.bat [command]
echo.
echo Build targets:
echo   (no command)       Build for all platforms (default)
echo   build              Build for Windows only
echo   build-linux        Build for Linux (amd64)
echo   build-linux-arm64  Build for Linux (arm64)
echo   build-windows      Build for Windows (amd64)
echo   build-windows-arm64 Build for Windows (arm64)
echo   build-darwin       Build for macOS (amd64)
echo   build-darwin-arm64 Build for macOS (arm64/Apple Silicon)
echo   build-all          Build for all platforms
echo.
echo Test targets:
echo   test               Run all tests with coverage
echo   test-short         Run short tests only
echo.
echo Code quality:
echo   lint               Run golangci-lint
echo   fmt                Format code and tidy modules
echo   vet                Run go vet
echo   check              Run fmt, vet, lint, and test
echo.
echo Run targets:
echo   run                Build and show status
echo   run-tui            Build and run TUI
echo.
echo Other:
echo   deps               Download dependencies
echo   deps-update        Update dependencies
echo   install            Install to GOPATH\bin
echo   clean              Remove build artifacts
echo   help               Show this help message
echo.
goto :eof

