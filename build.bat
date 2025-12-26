@echo off
setlocal enabledelayedexpansion

:: Talaria - Comprehensive Backup System
:: Build script for Windows (equivalent to Makefile)

:: Colors (Windows 10+)
set "GREEN=[92m"
set "YELLOW=[93m"
set "RED=[91m"
set "CYAN=[96m"
set "NC=[0m"

:: Variables
set "BINARY_DIR=bin"
set "CONFIG_FILE=configs\talaria.yaml"

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

:: Binary names
set "SERVER_BINARY=talaria-server-windows-amd64.exe"
set "CLIENT_BINARY=talaria-client-windows-amd64.exe"

:: Parse command
if "%1"=="" goto :build
if "%1"=="help" goto :help
if "%1"=="-h" goto :help
if "%1"=="--help" goto :help
if "%1"=="build" goto :build
if "%1"=="build-server" goto :build-server
if "%1"=="build-client" goto :build-client
if "%1"=="build-linux" goto :build-linux
if "%1"=="build-linux-arm64" goto :build-linux-arm64
if "%1"=="build-windows" goto :build-windows
if "%1"=="build-windows-arm64" goto :build-windows-arm64
if "%1"=="build-darwin" goto :build-darwin
if "%1"=="build-darwin-arm64" goto :build-darwin-arm64
if "%1"=="build-all" goto :build-all
if "%1"=="test" goto :test
if "%1"=="test-unit" goto :test-unit
if "%1"=="test-integration" goto :test-integration
if "%1"=="test-cover" goto :test-cover
if "%1"=="lint" goto :lint
if "%1"=="fmt" goto :fmt
if "%1"=="vet" goto :vet
if "%1"=="check" goto :check
if "%1"=="clean" goto :clean
if "%1"=="run-server" goto :run-server
if "%1"=="run-client" goto :run-client
if "%1"=="proto" goto :proto
if "%1"=="deps" goto :deps
if "%1"=="deps-update" goto :deps-update
if "%1"=="generate" goto :generate
if "%1"=="web-install" goto :web-install
if "%1"=="web-build" goto :web-build
if "%1"=="web-dev" goto :web-dev
if "%1"=="docker-build" goto :docker-build
if "%1"=="docker-push" goto :docker-push

echo %RED%Unknown command: %1%NC%
echo Run 'build.bat help' for usage
exit /b 1

:: ==================== BUILD TARGETS ====================

:build
echo %GREEN%Building all binaries for Windows...%NC%
call :build-server
if errorlevel 1 exit /b 1
call :build-client
if errorlevel 1 exit /b 1
echo %GREEN%All binaries built successfully%NC%
goto :eof

:build-server
echo %CYAN%Building Talaria server...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
go build %LDFLAGS% -o "%BINARY_DIR%\%SERVER_BINARY%" .\cmd\talaria-server
if errorlevel 1 (
    echo %RED%Failed to build server%NC%
    exit /b 1
)
echo   %GREEN%Created: %BINARY_DIR%\%SERVER_BINARY%%NC%
goto :eof

:build-client
echo %CYAN%Building Talaria client...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
go build %LDFLAGS% -o "%BINARY_DIR%\%CLIENT_BINARY%" .\cmd\talaria-client
if errorlevel 1 (
    echo %RED%Failed to build client%NC%
    exit /b 1
)
echo   %GREEN%Created: %BINARY_DIR%\%CLIENT_BINARY%%NC%
goto :eof

:: ==================== CROSS-COMPILATION ====================

:build-linux
echo %GREEN%Building all binaries for Linux (amd64) with Zig...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=1
set CC=zig cc -target x86_64-linux-musl
set CXX=zig c++ -target x86_64-linux-musl
go build %LDFLAGS% -ldflags "-linkmode=external -extldflags=-static" -o "%BINARY_DIR%\talaria-server-linux-amd64" .\cmd\talaria-server
go build %LDFLAGS% -ldflags "-linkmode=external -extldflags=-static" -o "%BINARY_DIR%\talaria-client-linux-amd64" .\cmd\talaria-client
set GOOS=
set GOARCH=
set CGO_ENABLED=
set CC=
set CXX=
echo %GREEN%Linux (amd64) binaries built successfully%NC%
goto :eof

:build-linux-arm64
echo %GREEN%Building all binaries for Linux (arm64) with Zig...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=linux
set GOARCH=arm64
set CGO_ENABLED=1
set CC=zig cc -target aarch64-linux-musl
set CXX=zig c++ -target aarch64-linux-musl
go build %LDFLAGS% -ldflags "-linkmode=external -extldflags=-static" -o "%BINARY_DIR%\talaria-server-linux-arm64" .\cmd\talaria-server
go build %LDFLAGS% -ldflags "-linkmode=external -extldflags=-static" -o "%BINARY_DIR%\talaria-client-linux-arm64" .\cmd\talaria-client
set GOOS=
set GOARCH=
set CGO_ENABLED=
set CC=
set CXX=
echo %GREEN%Linux (arm64) binaries built successfully%NC%
goto :eof

:build-windows
echo %GREEN%Building all binaries for Windows (amd64)...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=windows
set GOARCH=amd64
go build %LDFLAGS% -o "%BINARY_DIR%\talaria-server-windows-amd64.exe" .\cmd\talaria-server
go build %LDFLAGS% -o "%BINARY_DIR%\talaria-client-windows-amd64.exe" .\cmd\talaria-client
set GOOS=
set GOARCH=
echo %GREEN%Windows (amd64) binaries built successfully%NC%
goto :eof

:build-windows-arm64
echo %GREEN%Building all binaries for Windows (arm64)...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=windows
set GOARCH=arm64
go build %LDFLAGS% -o "%BINARY_DIR%\talaria-server-windows-arm64.exe" .\cmd\talaria-server
go build %LDFLAGS% -o "%BINARY_DIR%\talaria-client-windows-arm64.exe" .\cmd\talaria-client
set GOOS=
set GOARCH=
echo %GREEN%Windows (arm64) binaries built successfully%NC%
goto :eof

:build-darwin
echo %GREEN%Building all binaries for macOS (amd64)...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=darwin
set GOARCH=amd64
set CGO_ENABLED=0
go build %LDFLAGS% -o "%BINARY_DIR%\talaria-server-darwin-amd64" .\cmd\talaria-server
go build %LDFLAGS% -tags notray -o "%BINARY_DIR%\talaria-client-darwin-amd64" .\cmd\talaria-client
set GOOS=
set GOARCH=
set CGO_ENABLED=
echo %GREEN%macOS (amd64) binaries built successfully%NC%
goto :eof

:build-darwin-arm64
echo %GREEN%Building all binaries for macOS (arm64/Apple Silicon)...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=darwin
set GOARCH=arm64
set CGO_ENABLED=0
go build %LDFLAGS% -o "%BINARY_DIR%\talaria-server-darwin-arm64" .\cmd\talaria-server
go build %LDFLAGS% -tags notray -o "%BINARY_DIR%\talaria-client-darwin-arm64" .\cmd\talaria-client
set GOOS=
set GOARCH=
set CGO_ENABLED=
echo %GREEN%macOS (arm64) binaries built successfully%NC%
goto :eof

:build-all
echo %GREEN%Building for all platforms...%NC%
call :build-linux
call :build-linux-arm64
call :build-windows
call :build-windows-arm64
call :build-darwin
call :build-darwin-arm64
echo %GREEN%All platform binaries built successfully%NC%
goto :eof

:: ==================== TEST TARGETS ====================

:test
call :test-unit
echo %GREEN%All tests completed%NC%
goto :eof

:test-unit
echo %CYAN%Running unit tests...%NC%
go test -v -short -coverprofile=coverage.out ./...
if errorlevel 1 (
    echo %RED%Unit tests failed%NC%
    exit /b 1
)
go tool cover -html=coverage.out -o coverage.html
echo %GREEN%Unit tests passed. Coverage report: coverage.html%NC%
goto :eof

:test-integration
echo %CYAN%Running integration tests...%NC%
go test -v -tags=integration ./...
if errorlevel 1 (
    echo %RED%Integration tests failed%NC%
    exit /b 1
)
echo %GREEN%Integration tests passed%NC%
goto :eof

:test-cover
echo %CYAN%Running tests with coverage...%NC%
go test -v -coverprofile=coverage.out -covermode=atomic ./...
if errorlevel 1 (
    echo %RED%Tests failed%NC%
    exit /b 1
)
go tool cover -func=coverage.out
echo %GREEN%Coverage report generated%NC%
goto :eof

:: ==================== CODE QUALITY ====================

:lint
echo %CYAN%Running linter...%NC%
golangci-lint run ./...
if errorlevel 1 (
    echo %RED%Linting failed%NC%
    exit /b 1
)
echo %GREEN%Linting passed%NC%
goto :eof

:fmt
echo %CYAN%Formatting code...%NC%
gofmt -s -w .
go mod tidy
echo %GREEN%Code formatted%NC%
goto :eof

:vet
echo %CYAN%Running go vet...%NC%
go vet ./...
if errorlevel 1 (
    echo %RED%go vet found issues%NC%
    exit /b 1
)
echo %GREEN%go vet passed%NC%
goto :eof

:check
echo %CYAN%Running all checks...%NC%
call :fmt
call :vet
call :lint
call :test
echo %GREEN%All checks passed%NC%
goto :eof

:: ==================== CLEAN ====================

:clean
echo %CYAN%Cleaning build artifacts...%NC%
if exist "%BINARY_DIR%" rmdir /s /q "%BINARY_DIR%"
if exist "coverage.out" del "coverage.out"
if exist "coverage.html" del "coverage.html"
echo %GREEN%Cleaned%NC%
goto :eof

:: ==================== RUN TARGETS ====================

:run-server
call :build-server
if errorlevel 1 exit /b 1
echo %GREEN%Starting Talaria server...%NC%
"%BINARY_DIR%\%SERVER_BINARY%" -config "%CONFIG_FILE%"
goto :eof

:run-client
call :build-client
if errorlevel 1 exit /b 1
echo %GREEN%Starting Talaria client...%NC%
"%BINARY_DIR%\%CLIENT_BINARY%"
goto :eof

:: ==================== PROTO ====================

:proto
echo %CYAN%Generating protobuf code...%NC%
buf generate
if errorlevel 1 (
    echo %RED%Proto generation failed%NC%
    exit /b 1
)
echo %GREEN%Proto files generated%NC%
goto :eof

:: ==================== DEPENDENCIES ====================

:deps
echo %CYAN%Downloading dependencies...%NC%
go mod download
go mod verify
echo %GREEN%Dependencies downloaded%NC%
goto :eof

:deps-update
echo %CYAN%Updating dependencies...%NC%
go get -u ./...
go mod tidy
echo %GREEN%Dependencies updated%NC%
goto :eof

:: ==================== GENERATE ====================

:generate
echo %CYAN%Generating code...%NC%
go generate ./...
echo %GREEN%Code generated%NC%
goto :eof

:: ==================== WEB DASHBOARD ====================

:web-install
echo %CYAN%Installing web dashboard dependencies...%NC%
cd web && npm install
echo %GREEN%Web dependencies installed%NC%
goto :eof

:web-build
echo %CYAN%Building web dashboard...%NC%
cd web && npm run build
echo %GREEN%Web dashboard built%NC%
goto :eof

:web-dev
echo %CYAN%Starting web dashboard dev server...%NC%
cd web && npm run dev
goto :eof

:: ==================== DOCKER ====================

:docker-build
echo %CYAN%Building Docker images...%NC%
docker build -t talaria-server:%VERSION% -f deploy\docker\Dockerfile.server .
docker build -t talaria-client:%VERSION% -f deploy\docker\Dockerfile.client .
echo %GREEN%Docker images built%NC%
goto :eof

:docker-push
echo %CYAN%Pushing Docker images...%NC%
docker push talaria-server:%VERSION%
docker push talaria-client:%VERSION%
echo %GREEN%Docker images pushed%NC%
goto :eof

:: ==================== HELP ====================

:help
echo.
echo %GREEN%Talaria - Comprehensive Backup System%NC%
echo %GREEN%=====================================%NC%
echo.
echo Usage: build.bat [command]
echo.
echo %YELLOW%Build targets (Windows):%NC%
echo   build              Build all binaries for Windows
echo   build-server       Build server binary
echo   build-client       Build client binary
echo.
echo %YELLOW%Cross-compilation targets:%NC%
echo   build-linux        Build all binaries for Linux (amd64)
echo   build-linux-arm64  Build all binaries for Linux (arm64)
echo   build-windows      Build all binaries for Windows (amd64)
echo   build-windows-arm64 Build all binaries for Windows (arm64)
echo   build-darwin       Build all binaries for macOS (amd64)
echo   build-darwin-arm64 Build all binaries for macOS (arm64/Apple Silicon)
echo   build-all          Build for all supported platforms
echo.
echo %YELLOW%Test targets:%NC%
echo   test               Run all tests
echo   test-unit          Run unit tests with coverage
echo   test-integration   Run integration tests
echo   test-cover         Run tests with coverage report
echo.
echo %YELLOW%Code quality:%NC%
echo   lint               Run golangci-lint
echo   fmt                Format code and tidy modules
echo   vet                Run go vet
echo   check              Run fmt, vet, lint, and test
echo.
echo %YELLOW%Run targets:%NC%
echo   run-server         Build and run server
echo   run-client         Build and run client
echo.
echo %YELLOW%Protobuf:%NC%
echo   proto              Generate protobuf code
echo.
echo %YELLOW%Web Dashboard:%NC%
echo   web-install        Install web dependencies
echo   web-build          Build web dashboard
echo   web-dev            Start web dev server
echo.
echo %YELLOW%Docker:%NC%
echo   docker-build       Build Docker images
echo   docker-push        Push Docker images
echo.
echo %YELLOW%Other:%NC%
echo   deps               Download dependencies
echo   deps-update        Update dependencies
echo   generate           Run go generate
echo   clean              Remove build artifacts
echo   help               Show this help message
echo.
echo %CYAN%Binary naming: talaria-{component}-{os}-{arch}[.exe]%NC%
echo Example: talaria-server-linux-amd64, talaria-client-windows-amd64.exe
echo.
goto :eof
