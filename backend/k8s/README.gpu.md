# NVIDIA device plugin（Minikube / k3s）

在集群中安装 [NVIDIA k8s-device-plugin](https://github.com/NVIDIA/k8s-device-plugin)，节点才会上报 `nvidia.com/gpu`，LLM Deployment（`llm-deployment.yaml`）才能调度到 GPU 节点并使用 GPU。

## 前置条件

- **宿主机**：已安装 NVIDIA 驱动，且已安装 [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)（`nvidia-container-toolkit`），以便容器能访问 GPU。
- **Minikube**：需以支持 GPU 的方式启动（见下方 Minikube 小节）。
- **k3s**：在带 GPU 的节点上安装 k3s 并确保容器运行时能访问 GPU（见下方 k3s 小节）。

## 安装 device plugin（Minikube / k3s 通用）

本仓库已包含一份静态 manifest（基于官方 v0.14.5），在**仓库根目录**执行：

```bash
kubectl apply -f backend/k8s/nvidia-device-plugin.yaml
```

或从官方直接拉取（需网络可访问 GitHub）：

```bash
kubectl create -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v0.14.5/nvidia-device-plugin.yml
```

验证：有 GPU 的节点上应出现 `nvidia-device-plugin-daemonset` 的 Pod，且节点会上报 `nvidia.com/gpu`：

```bash
kubectl get pods -n kube-system -l name=nvidia-device-plugin-ds
kubectl get nodes -o jsonpath='{.items[*].status.allocatable.nvidia\.com/gpu}'
```

## Minikube

1. **宿主机**：安装 NVIDIA 驱动与 [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)。

2. **用 GPU 驱动启动 Minikube**（二选一）：
   - **Docker 运行时**（宿主机用 Docker 且已装 nvidia-container-toolkit）：
     ```bash
     minikube start --driver=docker --gpus=all
     ```
   - **裸机 / none 驱动**（Minikube 直接跑在带 GPU 的本机）：
     ```bash
     minikube start --driver=none
     ```
     本机需已装 kubelet/kubeadm 或使用 Minikube 的 none 文档要求的环境。

3. **安装 device plugin**：
   ```bash
   kubectl apply -f backend/k8s/nvidia-device-plugin.yaml
   ```

4. **确认**：
   ```bash
   minikube ssh -- nvidia-smi
   kubectl get nodes -o custom-columns=NAME:.metadata.name,GPU:.status.allocatable.nvidia\.com/gpu
   ```

若 Minikube 未带 GPU（例如 `minikube start` 未加 `--gpus=all`），device plugin Pod 可能报错或节点不出现 `nvidia.com/gpu`，此时 LLM 只能跑 CPU（需在 `llm-deployment.yaml` 中去掉 GPU 相关配置）。

### Minikube 里 `nvidia-smi: command not found`

说明 **Minikube 节点内没有 NVIDIA 驱动**，GPU 没有透传到 Minikube。常见原因：

- **未用 GPU 启动**：需先 `minikube stop`，再 `minikube start --driver=docker --gpus=all`（Linux + Docker）。若之前没用 `--gpus=all`，当前节点不会带 GPU。
- **Windows 宿主机**：Docker Desktop 跑在 WSL2 或 Hyper-V 里，Minikube 再跑在 Docker 里，**GPU 透传到 Minikube 一般不可用**，节点内通常没有 `nvidia-smi`。此时在 Minikube 里用不了本机显卡跑 LLM。
  - **可行做法**：在 `llm-deployment.yaml` 中去掉 `nvidia.com/gpu` 的 requests/limits 和 tolerations，让 LLM 用 **CPU** 跑；或改用 **Linux 宿主机 / 带 GPU 的远程集群** 再配 GPU。
- **驱动未装进节点**：宿主机有 NVIDIA 驱动和 nvidia-container-toolkit 时，用 `--gpus=all` 启动后节点内才可能有 `nvidia-smi`（以 Linux + Docker 为例）。

**结论**：若 `minikube ssh -- nvidia-smi` 报错，当前 Minikube 无法给 LLM 提供 GPU，请按上文去掉 LLM 的 GPU 配置改跑 CPU，避免 LLM Pod 一直 Pending。

## k3s

1. **带 GPU 的节点**：安装 NVIDIA 驱动与 [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)。

2. **安装/配置 k3s**：确保 k3s 使用的容器运行时（containerd）能加载 NVIDIA 容器运行时。通常需在节点上配置 containerd 使用 `nvidia-container-runtime`，或使用 k3s 的 [GPU / NVIDIA 文档](https://docs.k3s.io/advanced#nvidia-gpu-support)（若你的 k3s 版本支持）。

3. **安装 device plugin**：
   ```bash
   kubectl apply -f backend/k8s/nvidia-device-plugin.yaml
   ```

4. **确认**：
   ```bash
   kubectl get pods -n kube-system -l name=nvidia-device-plugin-ds
   kubectl get nodes -o custom-columns=NAME:.metadata.name,GPU:.status.allocatable.nvidia\.com/gpu
   ```

k3s 默认使用 containerd，需确保 `/var/lib/rancher/k3s/agent/etc/containerd/config.toml` 等配置中已启用 nvidia 运行时（详见 k3s 官方 GPU 说明）。

## 相关文件

- **Manifest**：`backend/k8s/nvidia-device-plugin.yaml`（本仓库内静态 YAML）
- **LLM 使用 GPU**：`backend/k8s/llm-deployment.yaml`（已配置 `nvidia.com/gpu` requests/limits 与 tolerations）
- **LLM 构建与部署**：`backend/deploy/README.md`（含「GPU 与 CPU」小节）
