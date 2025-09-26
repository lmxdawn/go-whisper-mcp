# go-whisper-mcp · docker-compose 使用说明

使用 **Docker Compose** 一键启动 **CPU** / **GPU** 两种运行模式。
本说明以你提供的最新 `docker-compose.yml` 为准（端口 **28796**、环境变量 **MODELS_DIR / MEDIA_DIR**、三处卷挂载）。

---

## 1) 前置条件

* **Docker Compose v2**（`docker compose version` 能正常输出）
* **CPU 模式**：无需额外依赖
* **GPU 模式**：

    * Linux：已安装 `nvidia-driver` 与 **nvidia-container-toolkit**，宿主机 `nvidia-smi` 可正常运行
    * Docker Desktop（Windows/macOS）：Settings → Resources → 勾选 **Use NVIDIA GPU**；WSL2 下需安装 NVIDIA 驱动

---

## 2) 目录与端口

* 端口：**28796**（容器与宿主一致）
* 本地目录（会挂载到容器）：

    * `./models`  → `/app/models`（模型缓存）
    * `./whisper_media` → `/app/whisper_media`（网络音视频下载缓存）
    * `./samples` → `/app/samples`（示例/待处理文件）

> 说明：`environment` 中的 `MODELS_DIR=./models`、`MEDIA_DIR=./whisper_media` 由你的应用读取并解释；卷挂载已将对应目录映射到容器内路径。

---

## 3) 快速启动

### 仅 CPU 模式

```bash
docker compose --profile cpu up -d
```

### 仅 GPU 模式

```bash
docker compose --profile gpu up -d
# 如果 CLI 较老或 IDE 校验严格，可加 --compatibility
# docker compose --compatibility --profile gpu up -d
```

### 停止与清理

```bash
docker compose down
```

---

## 4) 环境变量（按你的 compose）

```yaml
environment:
  - MODELS_DIR=./models         # 模型保存路径（应用读取的逻辑路径）
  - MEDIA_DIR=./whisper_media   # 网络资源下载保存路径（应用读取的逻辑路径）

# 仅 GPU 服务额外建议：
  - NVIDIA_VISIBLE_DEVICES=all
  - NVIDIA_DRIVER_CAPABILITIES=compute,utility
```

> 若你的程序更偏好容器内**绝对路径**，可将以上改为 `/app/models`、`/app/whisper_media`，并在代码中按需读取。

---

## 5) 验证服务是否启动

服务默认监听 **[http://127.0.0.1:28796**，提供](http://127.0.0.1:28796**，提供) **MCP over HTTP（Streamable HTTP）** 接口 `POST /mcp`。

### 最小握手（initialize）

```bash
curl -s http://127.0.0.1:28796/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{}}'
```

若返回 `result` 中包含 `protocolVersion` 与 `capabilities`，说明服务已就绪。
（若你同时暴露了 REST 接口如 `/transcribe`，也可用 `curl` 测试。）

---

## 6) 修改端口

若需改为 `18080`：

```yaml
services:
  go-whisper-mcp-xxx:
    ports:
      - "18080:18080"
    # 若你的应用需要从环境变量读取监听端口，请同时设置：
    # environment:
    #   - PORT=18080
```

更新后：

```bash
docker compose down
docker compose --profile cpu up -d   # 或 --profile gpu
```

---

## 7) 常用命令

```bash
# 查看容器状态
docker compose ps

# 查看日志（CPU / GPU）
docker compose logs -f go-whisper-mcp-cpu
docker compose logs -f go-whisper-mcp-gpu

# 进入 GPU 容器并检查 GPU
docker compose --profile gpu exec go-whisper-mcp-gpu nvidia-smi
```

---

## 8) 常见问题

**Q1：IDE（GoLand 等）提示 compose 字段未知或报红？**
A：本文件使用 `deploy.resources.reservations.devices` 声明 GPU，Compose v2 支持。
可用 `docker compose config` 渲染检查，只要 CLI 接受即可运行。

**Q2：GPU 模式无 GPU？**

* 使用 `--profile gpu` 启动
* Desktop 勾选 **Use NVIDIA GPU**；Linux 安装 nvidia-container-toolkit
* 保留 `NVIDIA_VISIBLE_DEVICES=all`、`NVIDIA_DRIVER_CAPABILITIES=compute,utility`
* 容器内执行 `nvidia-smi` 验证

**Q3：模型下载慢/失败？**

* 保留 `./models` 卷缓存，避免重复下载
* 可预先把模型放入 `./models`，容器会直接使用

**Q4：权限问题（Windows/WSL）？**

* 建议在项目根目录运行命令
* 确保 `./models`、`./whisper_media`、`./samples` 目录存在且可读写

---
