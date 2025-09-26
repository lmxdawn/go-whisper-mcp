
# 主要功能

<details>
<summary><b>1. 音视频的声音转成文字</b></summary>

注意：只能是 mp4 格式和 wav 格式

</details>


## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=lmxdawn/go-whisper-mcp&type=Timeline)](https://www.star-history.com/#lmxdawn/go-whisper-mcp&Timeline)

## 使用教程

### 依赖
```shell

$ sudo apt install ffmpeg

```

### linux-cpu
```shell

# 1) 让 cgo 在任何包里都能找到头文件
$ export C_INCLUDE_PATH="$(pwd)/whisper/linux/cpu/include:$(pwd)/whisper/linux/cpu/ggml/include"

# 2) 让链接器能找到静态库（注意：ggml 的 .a 也要有）
$ export LIBRARY_PATH="$(pwd)/whisper/linux/cpu/build_go/src:$(pwd)/whisper/linux/cpu/build_go/ggml/src"

# 3) 常见系统库（顺序要靠后，避免 DSO missing）
$ export CGO_LDFLAGS="-Wl,--no-as-needed -ldl -lpthread -lstdc++ -lm"

# 4) 可选：关闭 VCS stamping（避免 128 报错）
$ go env -w GOFLAGS="-buildvcs=false"

```

### linux-gpu（目前直接打包的不支持 build 打包，本地运行请使用 cpu）

```shell

# 1) 让 cgo 在任何包里都能找到头文件
$ export C_INCLUDE_PATH="$(pwd)/whisper/linux/gpu/include:$(pwd)/whisper/linux/gpu/ggml/include"

# 2) 让链接器能找到静态库（注意：ggml 的 .a 也要有）
$ export LIBRARY_PATH="$(pwd)/whisper/linux/gpu/build_go/src:$(pwd)/whisper/linux/gpu/build_go/ggml/src"

# 3) 常见系统库（顺序要靠后，避免 DSO missing）
$ export CGO_LDFLAGS="-Wl,--no-as-needed -ldl -lpthread -lstdc++ -lm"

# 4) 可选：关闭 VCS stamping（避免 128 报错）
$ go env -w GOFLAGS="-buildvcs=false"

```

### CPU 镜像
```shell

$ docker build --target cpu -t go-whisper-mcp:cpu .
$ docker run --rm -p 14562:14562 -v "$PWD/models:/app/models" -v "$PWD/samples:/app/samples" go-whisper-mcp:cpu

```

### GPU 镜像

```shell

docker build --target gpu -t go-whisper-mcp:gpu --build-arg BUILD_JOBS=8 .
docker run --rm --gpus all -p 14562:14562 -v "$PWD/models:/app/models" -v "$PWD/samples:/app/samples" go-whisper-mcp:gpu

```