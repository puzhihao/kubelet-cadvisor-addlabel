# ---------- 构建阶段 ----------
FROM registry.cn-shanghai.aliyuncs.com/puzhihao/golang:1.24-alpine AS builder

# 设置 Go 模块代理（国内镜像）
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

# 设置工作目录
WORKDIR /app

# 替换 Alpine 源为国内镜像（阿里云）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && apk add --no-cache git tzdata bash curl

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖（使用国内代理）
RUN go mod download

# 复制源代码
COPY . .

# 编译二进制文件（静态构建）
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kubelet-cadvisor-addlabel ./cmd/

# ---------- 运行阶段 ----------
FROM registry.cn-shanghai.aliyuncs.com/puzhihao/alpine:latest

# 替换为国内 APK 源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && apk --no-cache add ca-certificates tzdata curl bash wget

# 创建应用用户
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# 设置工作目录
WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/kubelet-cadvisor-addlabel .

# 调整权限
RUN chown -R appuser:appgroup /root/

# 切换为普通用户
USER appuser

# 暴露端口
EXPOSE 9090

# 健康检查（访问 /health）
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:9090/health || exit 1

# 启动应用
CMD ["./kubelet-cadvisor-addlabel"]
