# ── 阶段 1：编译 Go 二进制 ──
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bot ./cmd/bot

# ── 阶段 2：运行镜像（含 Python 用于 wiki 解析）──
FROM python:3.12-alpine
WORKDIR /app

# 基础依赖
RUN apk add --no-cache ca-certificates tzdata

# 复制编译产物和运行时资源
COPY --from=builder /build/bot .
COPY prompts/ ./prompts/
COPY tools/ ./tools/
RUN pip install --no-cache-dir beautifulsoup4 lxml

# 注：data/ 目录通过 docker-compose volume 挂载，不在此处 COPY
EXPOSE 8080
ENTRYPOINT ["./bot"]
