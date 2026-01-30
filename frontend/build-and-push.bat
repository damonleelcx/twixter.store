@echo off
REM Docker 镜像名称和标签
set IMAGE_NAME=damonleelcx/twixter.store-frontend
set TAG=%1
if "%TAG%"=="" set TAG=latest

echo Building Docker image: %IMAGE_NAME%:%TAG%

REM 构建 Docker 镜像
docker build -t %IMAGE_NAME%:%TAG% .

if %errorlevel% equ 0 (
    echo Build successful!
    echo Pushing image to Docker Hub...
    
    REM 推送镜像到 Docker Hub
    docker push %IMAGE_NAME%:%TAG%
    
    if %errorlevel% equ 0 (
        echo Push successful!
        echo Image available at: %IMAGE_NAME%:%TAG%
    ) else (
        echo Push failed. Make sure you're logged in to Docker Hub:
        echo   docker login
    )
) else (
    echo Build failed!
    exit /b 1
)
