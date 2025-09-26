# ---------------- 全局参数 ----------------
ARG GO_VERSION=1.24.5
ARG WHISPER_REF=v1.7.6
ARG BUILD_JOBS=2
ARG MAIN=.

# =========================================================
# ============== CPU 构建阶段（无 CUDA） ===================
# =========================================================
FROM ubuntu:22.04 AS cpu-builder
ARG GO_VERSION
ARG WHISPER_REF
ARG BUILD_JOBS
ARG MAIN
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    git ca-certificates build-essential cmake pkg-config curl ffmpeg \
 && rm -rf /var/lib/apt/lists/*

# 安装 Go
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -o /tmp/go.tgz \
 && tar -C /usr/local -xzf /tmp/go.tgz \
 && ln -s /usr/local/go/bin/go /usr/local/bin/go

# 拉 whisper.cpp 并构建（CPU）
WORKDIR /opt
RUN git clone --depth=1 -b ${WHISPER_REF} https://github.com/ggerganov/whisper.cpp.git
WORKDIR /opt/whisper.cpp
RUN cmake -S . -B build_go \
      -DCMAKE_BUILD_TYPE=Release \
      -DBUILD_SHARED_LIBS=OFF \
      -DGGML_CUDA=OFF \
 && cmake --build build_go --target whisper -j${BUILD_JOBS}

# 拷贝你的项目源码并编译
WORKDIR /app
COPY . .

# 代理与 VCS 设置（按需改）
ENV GOPROXY="https://goproxy.cn,direct"
ENV GOFLAGS="-buildvcs=false"

# 先拉依赖，提前暴露问题
RUN test -f go.mod || (echo "go.mod not found in /app; ls:" && ls -la && exit 1)
RUN go mod download

# 头/库路径 + 链接参数（CPU：不含 CUDA）
ENV CGO_CFLAGS="-I/opt/whisper.cpp/include -I/opt/whisper.cpp/ggml/include"
ENV CGO_LDFLAGS="\
 -L/opt/whisper.cpp/build_go/src \
 -L/opt/whisper.cpp/build_go/ggml/src \
 -Wl,--start-group \
   -lwhisper -lggml -lggml-base -lggml-cpu \
 -Wl,--end-group \
 -ldl -lpthread -lm -lstdc++ -lgomp"

RUN CGO_ENABLED=1 go build -v -x -o /app/bin/server ${MAIN}

# =========================================================
# ============== GPU 构建阶段（CUDA 12.4 devel） ===========
# =========================================================
FROM nvidia/cuda:12.4.1-devel-ubuntu22.04 AS gpu-builder
ARG GO_VERSION
ARG WHISPER_REF
ARG BUILD_JOBS
ARG MAIN
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    git ca-certificates build-essential cmake pkg-config curl ffmpeg \
 && rm -rf /var/lib/apt/lists/*

# 安装 Go
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -o /tmp/go.tgz \
 && tar -C /usr/local -xzf /tmp/go.tgz \
 && ln -s /usr/local/go/bin/go /usr/local/bin/go

# 拉 whisper.cpp 并构建（启用 CUDA）
WORKDIR /opt
RUN git clone --depth=1 -b ${WHISPER_REF} https://github.com/ggerganov/whisper.cpp.git
WORKDIR /opt/whisper.cpp
RUN cmake -S . -B build_go \
      -DCMAKE_BUILD_TYPE=Release \
      -DBUILD_SHARED_LIBS=OFF \
      -DGGML_CUDA=ON \
 && cmake --build build_go --target whisper -j${BUILD_JOBS}

# 诊断：确认 ggml-cuda 静态库是否生成在子目录
RUN ls -lh /opt/whisper.cpp/build_go/ggml/src || true && \
    find /opt/whisper.cpp/build_go -maxdepth 3 -name "libggml-cuda*.a" -ls || true

# 拷贝你的项目源码并编译
WORKDIR /app
COPY . .

ENV GOPROXY="https://goproxy.cn,direct"
ENV GOFLAGS="-buildvcs=false"
RUN test -f go.mod || (echo "go.mod not found in /app; ls:" && ls -la && exit 1)
RUN go mod download

# 头/库路径 + CUDA 链接（关键：包含 ggml-cuda 子目录）
ENV GGML_LIB="/opt/whisper.cpp/build_go/ggml/src"
ENV GGML_CUDA_LIB="/opt/whisper.cpp/build_go/ggml/src/ggml-cuda"
ENV WHISPER_LIB="/opt/whisper.cpp/build_go/src"
ENV CUDA_LIB="/usr/local/cuda/lib64"

ENV CGO_CFLAGS="-I/opt/whisper.cpp/include -I/opt/whisper.cpp/ggml/include"
ENV CGO_LDFLAGS="\
 -L${WHISPER_LIB} \
 -L${GGML_LIB} \
 -L${GGML_CUDA_LIB} \
 -L${CUDA_LIB} \
 -Wl,-rpath,${CUDA_LIB} \
 -Wl,--start-group \
   -lwhisper -lggml -lggml-cuda -lggml-base -lggml-cpu \
 -Wl,--end-group \
 -lcudart -lcublas -lcublasLt -lcuda -ldl -lpthread -lm -lstdc++ -lgomp"

RUN CGO_ENABLED=1 go build -v -x -o /app/bin/server ${MAIN}

# =========================================================
# ============== CPU 运行阶段 =============================
# =========================================================
FROM ubuntu:22.04 AS cpu
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg ca-certificates libgomp1 \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=cpu-builder /app/bin/server /app/server
RUN mkdir -p /app/models
ENV PORT=41235
EXPOSE 41235
ENTRYPOINT ["/app/server"]

# =========================================================
# ============== GPU 运行阶段（CUDA runtime） =============
# =========================================================
FROM nvidia/cuda:12.4.1-runtime-ubuntu22.04 AS gpu
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg ca-certificates libgomp1 \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=gpu-builder /app/bin/server /app/server
RUN mkdir -p /app/models
ENV LD_LIBRARY_PATH=/usr/local/cuda/lib64
ENV PORT=41235
EXPOSE 41235
ENTRYPOINT ["/app/server"]
