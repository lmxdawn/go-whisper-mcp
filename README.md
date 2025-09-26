# go-whisper-mcp

> 基于 **whisper.cpp** 的本地转写服务，支持 **REST API（JSON）** 与 **MCP over HTTP (Streamable HTTP)**，一键运行 **CPU / GPU** 两种模式。
> 默认端口：**28796** · 模型缓存：`./models` · 网络媒体缓存：`./whisper_media`

[![Star History Chart](https://api.star-history.com/svg?repos=lmxdawn/go-whisper-mcp\&type=Timeline)](https://www.star-history.com/#lmxdawn/go-whisper-mcp&Timeline)

---

## ✨ 功能

* 🎤 **音视频转文字**：支持 **MP4** / **WAV**（不满足 16kHz/mono 会自动用 ffmpeg 内存管道转换）
* 🚀 **本地推理**：CPU/GPU 自选，离线可用，模型自动缓存到本地
* 🌐 **两种访问方式**：

  * REST API：`POST /transcribe`（**JSON**，参数与 MCP 保持一致）
  * MCP over HTTP：`POST /mcp`（JSON-RPC）
* ⬇️ **模型自动下载**（带进度），支持常见别名：`tiny` / `base` / `small` / `medium` / `large-v3` / `large-v3-turbo` 等
* 🧵 多参数可调：`model` / `lang` / `t`（线程数）等

---

## 🧰 依赖

**必须**：`ffmpeg`（解复用/重采样）

```bash
sudo apt update && sudo apt install -y ffmpeg
```

---

## 🚀 快速开始

### 方式 A：Docker Compose（推荐）

使用下述 `docker-compose.yml`（端口 **28796**，挂载 `./models`、`./whisper_media`、`./samples`），通过 **profile** 切换 CPU/GPU：

```bash
# 仅 CPU 模式
docker compose --profile cpu up -d

# 仅 GPU 模式（Linux 需 nvidia-container-toolkit；Desktop 请开启 Use NVIDIA GPU）
docker compose --profile gpu up -d

# 停止
docker compose down
```

> 环境变量（compose 已设置）：
> `MODELS_DIR=./models`，`MEDIA_DIR=./whisper_media`（由应用读取）；卷挂载已映射到容器的 `/app/...`。

### 方式 B：Docker 镜像（手动构建）

```bash
# CPU 镜像
docker build --target cpu -t go-whisper-mcp:cpu .
docker run --rm -p 28796:28796 \
  -v "$PWD/models:/app/models" \
  -v "$PWD/whisper_media:/app/whisper_media" \
  -v "$PWD/samples:/app/samples" \
  go-whisper-mcp:cpu

# GPU 镜像
docker build --target gpu -t go-whisper-mcp:gpu --build-arg BUILD_JOBS=8 .
docker run --rm --gpus all -p 28796:28796 \
  -v "$PWD/models:/app/models" \
  -v "$PWD/whisper_media:/app/whisper_media" \
  -v "$PWD/samples:/app/samples" \
  go-whisper-mcp:gpu
```

### 方式 C：本地运行（仅供调试）

> 生产建议使用 Docker/Compose。GPU 本地编译依赖 CUDA/ggml-cuda 链接环境，优先使用 GPU 镜像。

**CPU（Linux）示例：**

```bash
# 让 cgo 找到头文件
export C_INCLUDE_PATH="$(pwd)/whisper/linux/cpu/include:$(pwd)/whisper/linux/cpu/ggml/include"
# 让链接器找到静态库
export LIBRARY_PATH="$(pwd)/whisper/linux/cpu/build_go/src:$(pwd)/whisper/linux/cpu/build_go/ggml/src"
# 系统库（顺序靠后避免 DSO missing）
export CGO_LDFLAGS="-Wl,--no-as-needed -ldl -lpthread -lstdc++ -lm"
# 可选：去掉 VCS 信息避免 128 报错
go env -w GOFLAGS="-buildvcs=false"

go build -o bin/server .
./bin/server
```

---

## 🔌 接口使用

### 1) REST API（`POST /transcribe` · **JSON**，与 MCP 参数一致）

**请求头**：`Content-Type: application/json`

**请求体字段：**

* `in_paths`：`string[]`，本地路径或 `http(s)://` 地址（支持多个；网络地址会自动下载到 `MEDIA_DIR`）
* `model`：`string`，模型别名或文件名（如 `tiny`、`small`、`large-v3`、或 `ggml-tiny.bin`）
* `lang`：`string`，语言代码（`zh`/`en`/`auto`）
* `t`：`number`，线程数（建议=CPU物理核数）

> **Docker 下的本地路径**：指容器内路径（例如挂载了 `./samples:/app/samples`，请求里用 `/app/samples/xxx.mp4` 或 `./samples/xxx.mp4` 取决于服务工作目录）。

**单文件（本地路径）**

```bash
curl -s http://127.0.0.1:28796/transcribe \
  -H 'Content-Type: application/json' \
  -d '{
    "in_paths": ["./samples/test.mp4"],
    "model": "tiny",
    "lang": "zh",
    "t": 8
  }'
```

**单文件（网络地址，自动下载到 `MEDIA_DIR`）**

```bash
curl -s http://127.0.0.1:28796/transcribe \
  -H 'Content-Type: application/json' \
  -d '{
    "in_paths": ["https://example.com/audio/demo.mp4"],
    "model": "small",
    "lang": "auto",
    "t": 6
  }'
```

**批量（本地 + 网络混合）**

```bash
curl -s http://127.0.0.1:28796/transcribe \
  -H 'Content-Type: application/json' \
  -d '{
    "in_paths": [
      "./samples/a.wav",
      "https://example.com/b.mp4"
    ],
    "model": "large-v3-turbo",
    "lang": "zh",
    "t": 8
  }'
```

**成功返回（统一风格）**

```json
{
  "success": true,
  "data": {
    "model_path": "models/ggml-tiny.bin",
    "language": "zh",
    "threads": 8,
    "duration_s": 2.483964,
    "results": [
      {
        "is_success": true,
        "duration_s": 2.163374,
        "segments": [
          { "start": "0s", "end": "2.92s", "text": "..." },
          { "start": "2.92s", "end": "6.48s", "text": "..." }
        ]
      }
    ]
  },
  "message": ""
}
```

**失败返回**

```json
{
  "error": "转换失败: xxx",
  "code": "BadRequest",
  "details": { "file": "https://example.com/bad.mp4" }
}
```

---

### 2) MCP over HTTP（`POST /mcp` · JSON-RPC）

**初始化**

```bash
curl -s http://127.0.0.1:28796/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{}}'
```

**工具列表**

```bash
curl -s http://127.0.0.1:28796/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":"2","method":"tools/list","params":{}}'
```

**调用转写（与 REST 参数一致，放在 `arguments`）**

```bash
curl -s http://127.0.0.1:28796/mcp \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc":"2.0",
    "id":"3",
    "method":"tools/call",
    "params":{
      "name":"transcribe",
      "arguments":{
        "in_paths":["./samples/test.mp4"],
        "model":"tiny",
        "lang":"zh",
        "t":8
      }
    }
  }'
```

> MCP Inspector 若出现超时，请提高客户端超时或先用小模型（`tiny/base`）验证链路。

---

## ⚙️ 运行时参数/环境变量

* `MODELS_DIR`：模型缓存目录（默认 `./models`；Compose 已挂载至 `/app/models`）
* `MEDIA_DIR`：网络媒体下载目录（默认 `./whisper_media`）
* `PORT`：服务监听端口（默认 `28796`；若修改需与 `ports` 映射一致）

---

## 🏎️ 性能建议

* 已知语言请 **不要用 `auto`**：`lang=zh` 更快更稳
* 关闭逐词时间戳（若有实现）：`SetTokenTimestamps(false)` 可显著提速
* 线程数：`t = CPU 物理核数`（或 `runtime.NumCPU()`）
* GPU：**`large-v3-turbo`** 在速度与显存之间更平衡；显存紧张用 `small/medium`
* 大文件可**分段**处理（减少端到端等待）

---

## 🔧 常见问题（FAQ）

* **访问不到端口**：确认 `PORT` 与 `ports` 映射一致；防火墙未拦截
* **GPU 无法使用**：使用 `--profile gpu`；Desktop 勾选 **Use NVIDIA GPU**；Linux 安装 `nvidia-container-toolkit`；容器内运行 `nvidia-smi` 验证
* **模型下载慢**：保留 `./models` 缓存；也可预先放模型进该目录
* **MCP 超时**：调大客户端超时；优先用小模型验证；必要时分段
* **本地编译缺库（如 `-lggml-cuda`）**：优先使用 GPU 镜像；或先跑 CPU 版打通链路

本项目遵循 [all-contributors](https://github.com/all-contributors/all-contributors) 规范。欢迎任何形式的贡献！

---

## 📄 许可

遵循原仓库及依赖的许可协议（见仓库内 LICENSE）。

---

如果本项目对你有帮助，欢迎 ⭐️ 支持与 PR 贡献！
