# 前端 Docker 说明

## 从 .env 加载环境变量

两种方式任选其一：

1. **docker compose**（推荐）：自动从项目根目录的 `.env` 加载  
   ```bash
   cd frontend
   docker compose up --build
   ```

2. **docker run**：通过 `--env-file` 注入  
   ```bash
   cd frontend
   docker build -t frontend .
   docker run -p 3000:3000 --env-file .env frontend
   ```

## 构建时变量（NEXT_PUBLIC_*）

`NEXT_PUBLIC_*` 会在构建时打进前端 bundle，若需在镜像中固定，可在 build 时传入：

```bash
docker build -t frontend \
  --build-arg NEXT_PUBLIC_API_URL=https://api.example.com \
  --build-arg NEXT_PUBLIC_APP_URL=https://app.example.com \
  .
```

或通过 `.env` 在 `docker compose build` 时被用作构建环境，Compose 会从 `env_file` 注入。

## 端口

默认监听 3000，可通过 `-p 3000:3000` 映射。
