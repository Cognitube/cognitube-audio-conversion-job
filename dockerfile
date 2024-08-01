# 使用官方的 Go 镜像作为构建环境
FROM golang:1.22.5 as builder

# 设置工作目录
WORKDIR /app

# 复制 go mod 和 sum 文件
COPY go.mod go.sum ./

# 下载所有依赖
RUN go mod download

# 复制源代码到容器中
COPY . .

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o myapp .

FROM debian:buster-slim

# 设置工作目录
WORKDIR /root/

# 安装 ffmpeg
RUN apt-get update && \
  apt-get install -y ffmpeg && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/*

# 从构建环境中复制构建的可执行文件到当前容器
COPY --from=builder /app/myapp .

ENV AZURE_BLOB_CONNECTION_STRING="DefaultEndpointsProtocol=https;AccountName=cognitube;AccountKey=a1XDmr4IlO9I/tcsuh1akTaGFgmp+nQEoQdA8SlFpmmn7Zi0HKeDMk3ntxWjGI/HMFpQjzBys2ZX+AStqYVfsg==;EndpointSuffix=core.windows.net"
ENV FFMPEG_PATH="/usr/bin/ffmpeg"
ENV AUDIO_CONTAINER_NAME="audio-container"
ENV KAFKA_EVENTHUB_CONNECTION_STRING="Endpoint=sb://cognitube-kafka.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=l9+PMVbv8R4LuCtQlPo5x8PIE8jZqn8O4+AEhEMQoqA="
ENV KAFKA_EVENTHUB_NAMESPACE=cognitube-kafka
ENV KEYWORD_SERVICE_URL="https://cognitube-keyword-service.thankfulfield-7c1523f0.eastus.azurecontainerapps.io/api"

# 运行应用程序
CMD ["./myapp"]
