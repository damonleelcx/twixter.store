# Docker 构建和部署指南

## 快速开始

### 构建和推送 Docker 镜像

#### 使用脚本（推荐）

**Windows:**
```bash
build-and-push.bat [tag]
```

**Linux/Mac:**
```bash
chmod +x build-and-push.sh
./build-and-push.sh [tag]
```

如果不指定标签，默认使用 `latest`。

#### 手动构建和推送

1. **构建镜像:**
```bash
docker build -t damonleelcx/twixter.store-backend:latest .
```

2. **登录 Docker Hub:**
```bash
docker login
```

3. **推送镜像:**
```bash
docker push damonleelcx/twixter.store-backend:latest
```

### 使用 Docker Compose 运行（包含 PostgreSQL 和 Redis）

1. **确保在 `backend` 目录下**，且该目录已有 `.env` 文件（可参考 `.env.example` 创建）。
2. **启动服务**（Docker Compose 会自动读取同目录下的 `.env` 并注入到容器）：

```bash
cd backend
docker-compose up -d
```

这将启动：
- 后端服务（端口 8080）
- Redis 缓存（端口 6379）
- Zookeeper + Kafka（端口 9092）

**说明**：`docker-compose.yml` 中已配置 `env_file: .env`，因此无需额外参数，`.env` 中的变量会同时用于：
- 在 compose 文件中做变量替换（如 `${DB_HOST}`）
- 作为容器的环境变量传入后端服务

### 仅运行后端容器

```bash
docker run -d \
  --name twixter-backend \
  -p 8080:8080 \
  -e DB_HOST=your_db_host \
  -e DB_USER=your_db_user \
  -e DB_PASSWORD=your_db_password \
  -e DB_NAME=your_db_name \
  -e DB_PORT=5432 \
  -e DB_SSLMODE=disable \
  damonleelcx/twixter.store-backend:latest
```

### 使用 .env 文件

创建 `.env` 文件并挂载：

```bash
docker run -d \
  --name twixter-backend \
  -p 8080:8080 \
  --env-file .env \
  damonleelcx/twixter.store-backend:latest
```

## 环境变量

应用需要以下环境变量：

### 数据库配置
- `DB_HOST` - 数据库主机（默认: localhost）
- `DB_USER` - 数据库用户（默认: postgres）
- `DB_PASSWORD` - 数据库密码（默认: postgres）
- `DB_NAME` - 数据库名称（默认: postgres）
- `DB_PORT` - 数据库端口（默认: 5432）
- `DB_SSLMODE` - SSL 模式（默认: disable）

### Redis 配置
- `REDIS_ADDR` - Redis 地址（默认: localhost:6379）
- `REDIS_PASSWORD` - Redis 密码（默认: 空）
- `REDIS_DB` - Redis 数据库编号（默认: 0）

在 Docker Compose 中，Redis 地址应设置为 `redis:6379`（使用服务名称）。

## 健康检查

应用启动后，可以通过以下端点检查健康状态：

```bash
curl http://localhost:8080/ping
curl http://localhost:8080/health
```

## 查看日志

```bash
docker logs -f twixter-backend
```

## 停止和清理

```bash
# 停止容器
docker stop twixter-backend

# 删除容器
docker rm twixter-backend

# 使用 Docker Compose
docker-compose down
```
