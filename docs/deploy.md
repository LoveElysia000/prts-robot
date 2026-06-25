# 部署指南

将 Discord 机器人部署到服务器。

## 前提条件

- 服务器已安装 Docker 和 Docker Compose
- 已在 [Discord Developer Portal](https://discord.com/developers/applications) 创建 Bot，获取 Token
- 已在 [DeepSeek 开放平台](https://platform.deepseek.com) 获取 API Key

## 操作流程

### 1. 在 Discord 创建 Bot

1. `https://discord.com/developers/applications` → New Application
2. Bot → Add Bot → Reset Token（复制保存）
3. Privileged Gateway Intents：三个全开
4. OAuth2 → URL Generator → 勾 `bot` + `Send Messages` + `Read Message History` → 链接邀请入服务器

### 2. 推送代码并等 Actions 构建

```bash
git push origin main
# 等 https://github.com/LoveElysia000/prts-robot/actions 变绿
```

### 3. 镜像设为 Public

`https://github.com/LoveElysia000/prts-robot/pkgs/container/prts-robot` → Package Settings → Public

### 4. 服务器部署

```bash
mkdir -p /opt/bot && cd /opt/bot

# 下载 compose
wget https://raw.githubusercontent.com/LoveElysia000/prts-robot/main/docker-compose.yml
wget https://raw.githubusercontent.com/LoveElysia000/prts-robot/main/config.example.yaml -O config.yaml

# .env
cat > .env << 'EOF'
DISCORD_BOT_TOKEN=你的BotToken
DEEPSEEK_API_KEY=sk-你的key
EOF

# 启动
docker compose pull && docker compose up -d
```

### 5. 验证

Discord 服务器里 @机器人 或私聊，回复即成功。

## 文件结构

```
/opt/bot/
├── docker-compose.yml
├── config.yaml
└── .env
```

## 更新

Push 后 Actions 自动构建。服务器上手动拉：

```bash
cd /opt/bot && docker compose pull && docker compose up -d
```

## 日志

```bash
docker compose logs bot
cat logs/bot.log
```
