# 部署指南

将 PRTS Robot 部署到服务器。

## 前提

- 服务器已安装 Docker + Docker Compose
- [Discord Developer Portal](https://discord.com/developers/applications) 已创建 Bot
- [DeepSeek](https://platform.deepseek.com) API Key

## 步骤

### 1. Discord Bot 创建

```
https://discord.com/developers/applications → New Application
 Bot → Add Bot → Reset Token（复制保存）
 Privileged Gateway Intents → 三个全开
 OAuth2 → URL Generator → bot + Send Messages + Read Message History → 邀请链接
```

### 2. 服务器部署

```bash
mkdir -p /opt/prts-robot && cd /opt/prts-robot

wget https://raw.githubusercontent.com/LoveElysia000/prts-robot/main/docker-compose.yml
wget https://raw.githubusercontent.com/LoveElysia000/prts-robot/main/config.example.yaml -O config.yaml

echo 'DISCORD_BOT_TOKEN=你的Token' > .env
echo 'DEEPSEEK_API_KEY=sk-你的Key' >> .env

docker compose pull && docker compose up -d
```

### 3. 验证

Discord 里 @机器人 或私聊。

## 文件结构

```
/opt/prts-robot/
├── docker-compose.yml
├── config.yaml
├── .env
├── data/          # SQLite + 角色文件（自动生成）
└── logs/          # 日志（自动生成）
```

## 更新

```bash
cd /opt/prts-robot
docker compose pull && docker compose up -d
```

## 日志

```bash
docker compose logs bot
cat logs/bot.log
```
