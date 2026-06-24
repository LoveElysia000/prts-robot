# 部署指南

将 QQ 机器人部署到服务器的完整步骤。

## 前提条件

- 服务器已安装 Docker 和 Docker Compose
- 已在 [QQ 开放平台](https://q.qq.com) 创建机器人，获取 AppID 和 AppSecret
- 已在 [DeepSeek 开放平台](https://platform.deepseek.com) 获取 API Key
- 服务器公网 IP，8080 端口已放行

## 操作流程

### 1. 推送代码到 GitHub

```bash
cd /path/to/robot
git remote add origin https://github.com/loveelysia000/robot.git
git push -u origin main
```

### 2. 等待 GitHub Actions 构建

打开 `https://github.com/loveelysia000/robot/actions`，等待 Build 任务变成绿色 ✅（约 3-5 分钟）。

### 3. 将镜像设为公开

打开 `https://github.com/loveelysia000/robot/pkgs/container/robot`，点击 **Package Settings** → **Change Visibility** → 选择 **Public**。

### 4. 在服务器上部署

SSH 登录服务器，执行以下命令：

```bash
# 创建目录
mkdir -p /opt/robot
cd /opt/robot

# 下载 docker-compose.yml
wget https://raw.githubusercontent.com/loveelysia000/robot/main/docker-compose.yml

# 创建 config.yaml（替换 app_id 为实际值）
cat > config.yaml << 'EOF'
qq:
  app_id: "你的AppID"
  app_secret: ""
  webhook_port: 8080

deepseek:
  api_key: ""
  base_url: "https://api.deepseek.com"
  model: "deepseek-v4-flash"
  default_system_prompt: "你是一个友好的QQ群助手"

trigger:
  mode: "hybrid"
  command_prefix: "/"

database:
  path: "./data/bot.db"
EOF

# 创建 .env（替换为实际密钥）
cat > .env << 'EOF'
QQ_APP_SECRET=你的AppSecret
DEEPSEEK_API_KEY=sk-你的APIKey
EOF

# 启动
docker compose pull
docker compose up -d

# 验证
docker compose ps
```

看到 bot 和 watchtower 都是 `Up` 即部署成功。

### 5. 配置 QQ 开放平台 Webhook

在 QQ 开放平台 → 机器人管理 → 开发设置 → Webhook 地址，填写：

```
http://你的服务器公网IP:8080/webhook
```

### 6. 验证

在群里 @机器人 发送消息，机器人回复即完成。

## 更新代码

每次 push 到 GitHub 后，Actions 自动构建新镜像。服务器上 Watchtower 每 5 分钟检查一次，发现新镜像自动拉取并重启。无需手动操作。

如需立即更新，在服务器上执行：

```bash
cd /opt/robot && docker compose pull && docker compose up -d
```

## 文件结构

服务器上只需 3 个文件：

```
/opt/robot/
├── docker-compose.yml    # 容器编排
├── config.yaml           # 机器人配置
└── .env                  # 环境变量（密钥）
```

## 查看日志

```bash
docker compose logs -f bot
```

## 常见问题

**Q: Webhook 验证失败？**
A: 确认 AppSecret 与 QQ 开放平台一致，app_secret 必须是 32 字符的 hex 字符串。

**Q: @机器人没有反应？**
A: `docker compose logs bot` 查看日志，确认 Webhook 收到了请求。检查服务器防火墙是否放行 8080 端口。

**Q: 镜像拉取失败？**
A: 确认镜像已设为 Public。如果是 Private，需要先在服务器上 `docker login ghcr.io`。

**Q: 用 1Panel 怎么部署？**
A: 1Panel → Docker → Compose → 新建，粘贴 docker-compose.yml 内容，填写环境变量后启动即可。
