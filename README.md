# 加入环境变量

## 依赖
```shell

$ sudo apt install ffmpeg

```

## linux-cpu
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

## linux-gpu

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