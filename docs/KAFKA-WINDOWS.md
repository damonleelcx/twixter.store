# Windows 单机运行 Kafka（本地开发）

在 Windows 上直接运行 Kafka（不用 Docker）时，可能遇到 **log retention 清理** 时报错 “The process cannot access the file because it is being used by another process”，导致 broker 将 log 目录标记为失败并退出。本文给出推荐配置和操作步骤，尽量降低该问题发生概率。

---

## 1. 推荐配置（server.properties）

在 Kafka 安装目录的 `config\server.properties` 中做如下调整。

### 1.1 使用专用日志目录（避免 C:\tmp）

Windows 上建议**不要**用 `C:\tmp\kafka-logs`，改用单独目录，并排除杀毒/索引扫描：

```properties
# 示例：放在 D 盘或用户目录下，避免临时目录被其他程序占用
log.dirs=D:\\kafka-logs
```

或（按你的实际路径替换）：

```properties
log.dirs=C:\\kafka-data\\logs
```

- 确保该目录存在且 Kafka 进程有读写权限。
- 在 Windows  Defender / 杀毒软件中**排除**该目录，减少“文件被占用”的概率。

### 1.2 适当放宽保留时间（减少清理频率）

清理越频繁，越容易在 Windows 上碰到“文件仍被占用”。可适当增大保留时间，例如 7 天：

```properties
# 保留 7 天（与项目 docker-compose 中 KAFKA_LOG_RETENTION_HOURS=168 一致）
log.retention.hours=168
```

若已有 `log.retention.ms`，可注释掉或与 `log.retention.hours` 二选一，避免冲突。

### 1.3 单机开发常用项（可选）

```properties
# 单 broker
broker.id=0
listeners=PLAINTEXT://localhost:9092
num.network.threads=3
num.io.threads=8
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400
socket.request.max.bytes=104857600
log.dirs=D:\\kafka-logs
num.partitions=1
log.retention.hours=168
log.segment.bytes=1073741824
log.retention.check.interval.ms=300000
zookeeper.connect=localhost:2181
```

---

## 2. 启动与关闭顺序（减少文件占用）

### 2.1 启动顺序

1. **先启动 Zookeeper**（第一个终端）  
   ```bat
   cd C:\path\to\kafka
   bin\windows\zookeeper-server-start.bat config\zookeeper.properties
   ```
2. **再启动 Kafka**（第二个终端）  
   ```bat
   cd C:\path\to\kafka
   bin\windows\kafka-server-start.bat config\server.properties
   ```
3. 确认 Kafka 日志无报错后，再启动本项目的 backend（`KAFKA_BROKERS=localhost:9092`）。

### 2.2 关闭顺序（重要）

为避免“文件被占用”，请**先停 Kafka，再停 Zookeeper**，并且不要强制杀进程：

1. 在运行 Kafka 的终端按 **Ctrl+C**，等待进程正常退出。
2. 再在运行 Zookeeper 的终端按 **Ctrl+C**。
3. 不要直接关窗口或 `taskkill /F`，否则下次启动容易遇到 segment 文件仍被占用。

---

## 3. 若已出现 “file is being used by another process”

1. **停止所有相关进程**  
   - 停止 Kafka（Ctrl+C）。  
   - 停止 Zookeeper（Ctrl+C）。  
   - 停止本项目的 backend 以及任何连接该 Kafka 的进程。

2. **检查占用（可选）**  
   - 在任务管理器中结束残留的 `java` 进程（确认是 Kafka/Zookeeper 再结束）。  
   - 确认杀毒、OneDrive、备份等没有在扫描 `log.dirs` 所在目录。

3. **二选一恢复方式**  
   - **方式 A**：不删数据，过几分钟再启动 Kafka（有时句柄释放会延迟）。  
   - **方式 B**：若仍失败，可**删除出问题的 topic 目录**（会丢该 topic 数据），例如：  
     - 若 `log.dirs=D:\kafka-logs`，错误在 `video-transcode-0`，可删除：  
       `D:\kafka-logs\video-transcode-0`  
     - 然后重新启动 Kafka；backend 会继续用 `localhost:9092`，topic 会在首次生产时自动创建。

4. **重新启动**  
   按上面「启动顺序」先 Zookeeper，再 Kafka，再 backend。

---

## 4. 可选：用 Docker 避免 Windows 文件锁（推荐）

若仍经常遇到“文件被占用”或 log dir 失败，建议在 Windows 上用 Docker 跑 Kafka（Linux 容器内无此文件锁问题）：

```bash
cd backend
docker-compose up -d zookeeper kafka
```

后端保持 `KAFKA_BROKERS=localhost:9092` 即可。详见 `backend/README.VIDEO_PROCESSING.md` 与 `backend/docker-compose.yml`。

---

## 小结

| 项目           | 建议 |
|----------------|------|
| **log.dirs**   | 使用专用目录（如 `D:\kafka-logs`），并在杀毒中排除。 |
| **log.retention** | 适当增大（如 `log.retention.hours=168`），减少清理频率。 |
| **关闭顺序**   | 先 Ctrl+C 停 Kafka，再停 Zookeeper，勿强制杀进程。 |
| **仍失败时**   | 停掉所有相关进程后，删除出问题的 topic 子目录再启动，或改用 Docker。 |
