#!/usr/bin/env bash
# K3s 部署脚本：构建前后端镜像、推送、导入 k3s、重启 Deployment
# 在项目根目录执行：./deploy-k3s.sh  或  bash deploy-k3s.sh

set -e

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

echo "==> 构建前端镜像..."
cd frontend
[ -f k8s/.env ] && set -a && . ./k8s/.env && set +a
docker build -t damonleelcx/twixter.store-frontend:latest \
  --build-arg NEXT_PUBLIC_API_URL="https://api.twixter.store" \
  --build-arg NEXT_PUBLIC_APP_URL="https://www.twixter.store" \
  --build-arg NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY="${NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY:-}" \
  --build-arg NEXT_PUBLIC_PAYPAL_CLIENT_ID="${NEXT_PUBLIC_PAYPAL_CLIENT_ID:-}" \
  --build-arg BACKEND_URL="http://twixter-backend:8080" \
  .
cd ..

echo "==> 构建后端镜像..."
cd backend
docker build -t damonleelcx/twixter.store-backend:latest .
cd ..

echo "==> 推送镜像到 Docker Hub..."
docker push damonleelcx/twixter.store-backend:latest
docker push damonleelcx/twixter.store-frontend:latest

echo "==> 导入镜像到 k3s containerd..."
docker save damonleelcx/twixter.store-backend:latest | sudo k3s ctr images import -
docker save damonleelcx/twixter.store-frontend:latest | sudo k3s ctr images import -

echo "==> 重启 Deployment..."
kubectl rollout restart deployment/twixter-backend
kubectl rollout restart deployment/twixter-frontend

echo "==> 部署完成."
