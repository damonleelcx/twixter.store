#!/bin/bash

# Docker 镜像名称和标签
IMAGE_NAME="damonleelcx/twixter.store-backend"
TAG="${1:-latest}"

echo "Building Docker image: ${IMAGE_NAME}:${TAG}"

# 构建 Docker 镜像
docker build -t ${IMAGE_NAME}:${TAG} .

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Pushing image to Docker Hub..."
    
    # 推送镜像到 Docker Hub
    docker push ${IMAGE_NAME}:${TAG}
    
    if [ $? -eq 0 ]; then
        echo "Push successful!"
        echo "Image available at: ${IMAGE_NAME}:${TAG}"
    else
        echo "Push failed. Make sure you're logged in to Docker Hub:"
        echo "  docker login"
    fi
else
    echo "Build failed!"
    exit 1
fi
