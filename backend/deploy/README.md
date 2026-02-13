# backend/deploy

自托管 LLM 与部署相关资源。

## LLM 镜像（llama.cpp server）

`Dockerfile.llm` 基于 `ghcr.io/ggml-org/llama.cpp:server`，将模型目录 `./models` 拷贝进镜像，在容器内监听 8081 端口。`backend/k8s/llm-deployment.yaml` 已配置从 **Docker Hub** 拉取 `damonleelcx/twixter.store-llm:latest`（需先构建并 push，见下方步骤 2–3）。

### 你需要做的

1. **放模型**  
   **需自行下载** GGUF 模型（Dockerfile 与 llm-deployment 均不提供/不拉取模型）。将下载好的文件放到 **`backend/deploy/models/model.gguf`**（下载后若文件名不同，请重命名或复制为 `model.gguf`）。  
   `backend/deploy/models/.gitignore` 已忽略 `*.gguf`，避免误提交大文件。

   **推荐模型与下载（任选其一，与 llama.cpp server 兼容）：**

   | 用途     | 模型 | 大小  | 直接下载 |
   |----------|------|-------|----------|
   | 本地/测试 | [Llama 3.2 3B Instruct](https://huggingface.co/bartowski/Llama-3.2-3B-Instruct-GGUF) Q4_K_M | ~2 GB | [Llama-3.2-3B-Instruct-Q4_K_M.gguf](https://huggingface.co/bartowski/Llama-3.2-3B-Instruct-GGUF/resolve/main/Llama-3.2-3B-Instruct-Q4_K_M.gguf) |
   | 效果更好 | [OpenChat 3.5](https://huggingface.co/TheBloke/openchat-3.5-1210-GGUF) Q4_K_M | ~4.4 GB | [openchat-3.5-1210.Q4_K_M.gguf](https://huggingface.co/TheBloke/openchat-3.5-1210-GGUF/resolve/main/openchat-3.5-1210.Q4_K_M.gguf) |

   更多 GGUF 模型可搜：[Hugging Face GGUF 模型](https://huggingface.co/models?library=gguf)。下载完成后放入 `backend/deploy/models/` 并命名为 `model.gguf`。

2. **构建镜像**（在**仓库根目录**执行）  
   ```bash
   docker build -f backend/deploy/Dockerfile.llm -t twixter-llm:latest backend/deploy
   ```
   CMD（Windows）：
   ```cmd
   docker build -f backend\deploy\Dockerfile.llm -t twixter-llm:latest backend\deploy
   ```

2.1 **推送到 Docker Hub**（k3s/远程集群从 registry 拉取时必做）  
   构建完成后打标签并推送到 Docker Hub（仓库 `damonleelcx/twixter.store-llm`），集群即可拉取该镜像：

   ```bash
   docker tag twixter-llm:latest damonleelcx/twixter.store-llm:latest
   docker login
   docker push damonleelcx/twixter.store-llm:latest
   ```

   CMD（Windows）：
   ```cmd
   docker tag twixter-llm:latest damonleelcx/twixter.store-llm:latest
   docker login
   docker push damonleelcx/twixter.store-llm:latest
   ```

   `docker login` 会提示输入 Docker Hub 用户名与密码（或 Access Token）。推送完成后执行 `kubectl apply -f backend/k8s/llm-deployment.yaml`（或 `kubectl apply -k backend/k8s/`），必要时 `kubectl rollout restart deployment/llm`。

3. **Minikube 使用本地镜像**（二选一）  
   - 在 Minikube 的 Docker 环境里构建：  
     ```bash
     eval $(minikube docker-env)
     docker build -f backend/deploy/Dockerfile.llm -t twixter-llm:latest backend/deploy
     ```  
     CMD（Windows）：先切到 Minikube 的 Docker，再构建：
     ```cmd
     for /f "tokens=*" %i in ('minikube docker-env --shell cmd') do %i
     docker build -f backend\deploy\Dockerfile.llm -t twixter-llm:latest backend\deploy
     ```  
     （若写在 .bat 里，用 `%%i` 替代 `%i`。）  
   - 或在主机构建后导入集群：  
     ```bash
     minikube image load twixter-llm:latest
     ```  
     CMD（Windows）：同上，命令一致。
     ```cmd
     minikube image load twixter-llm:latest
     ```

3.1 **k3s / 远程集群：解决 ImagePullBackOff**  
   部署已配置从 Docker Hub 拉取 `damonleelcx/twixter.store-llm:latest`。先按 **2.1** 构建并 `docker push` 到 Docker Hub，再 `kubectl apply` / `kubectl rollout restart deployment/llm` 即可。

   若不能使用 Docker Hub（内网/无外网），可将镜像导入 k3s 节点：

   ```bash
   docker save damonleelcx/twixter.store-llm:latest -o llm.tar
   # 将 llm.tar 拷到 k3s 节点后：
   sudo k3s ctr images import llm.tar
   ```

   同一台 Linux 上既有 Docker 又有 k3s 时：`docker save damonleelcx/twixter.store-llm:latest | sudo k3s ctr images import -`。然后 `kubectl rollout restart deployment/llm`。

4. **部署或更新 LLM（k3s / kubectl）**  
   - **仅部署 LLM**（只上 llm-deployment + llm-service，在仓库根目录执行）：  
     ```bash
     kubectl apply -f backend/k8s/llm-deployment.yaml -f backend/k8s/llm-service.yaml
     ```  
     CMD（Windows）：
     ```cmd
     kubectl apply -f backend\k8s\llm-deployment.yaml -f backend\k8s\llm-service.yaml
     ```  
   - **或与整个 backend 一起部署**（含 ConfigMap、MongoDB、Backend 等）：  
     ```bash
     kubectl apply -k backend/k8s/
     ```  
     CMD（Windows）：
     ```cmd
     kubectl apply -k backend\k8s\
     ```  
   **k3s**：同上。若未配置默认 kubeconfig，先设置：  
   ```bash
   export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
   ```  
   然后执行上述 `kubectl apply`。或用 k3s 自带的 kubectl：  
   ```bash
   k3s kubectl apply -f backend/k8s/llm-deployment.yaml -f backend/k8s/llm-service.yaml
   ```

完成后 LLM Deployment 会从 Docker Hub 使用 `damonleelcx/twixter.store-llm:latest`（或本地已导入的镜像）并正常提供推理服务。

### GPU 与 CPU

`backend/k8s/llm-deployment.yaml` 已配置 **有 GPU 时使用 GPU**：

- **resources**：`nvidia.com/gpu: "1"`（requests/limits），集群中有带 [NVIDIA device plugin](https://github.com/NVIDIA/k8s-device-plugin) 的节点时，Pod 会被调度到该节点并使用 1 张 GPU。
- **tolerations**：允许调度到带 `nvidia.com/gpu:NoSchedule` 污点的节点（常见于云厂商 GPU 节点）。

**安装 NVIDIA device plugin（Minikube / k3s）**：需先在集群中安装 device plugin，节点才会暴露 `nvidia.com/gpu`。详见 **`backend/k8s/README.gpu.md`**，其中包含：
- 使用本仓库内 manifest：`kubectl apply -f backend/k8s/nvidia-device-plugin.yaml`
- Minikube 下带 GPU 启动（`minikube start --driver=docker --gpus=all`）及验证
- k3s 下前置条件与安装步骤

**无 GPU 节点时**：Pod 会一直处于 `Pending`（Unschedulable）。若集群只有 CPU 节点，需在 `llm-deployment.yaml` 中**去掉** `nvidia.com/gpu` 的 requests/limits 和上述 tolerations，即可改回纯 CPU 推理。

**有 GPU 时加速**：llama.cpp server 会自动使用 GPU（若镜像带 CUDA 支持）；如需显式控制 GPU 层数，可在 deployment 的 `args` 中增加 `-ngl`（如 `-ngl 99`），具体以镜像/llama.cpp 版本为准。

### 参考

- 仅作示例的 Dockerfile：`LLM_DOCKERFILE.example`
- 部署配置：`backend/k8s/llm-deployment.yaml`、`backend/k8s/llm-service.yaml`
- 聊天栈部署总览：`docs/DEPLOY-CHAT-K8S.md`
