# 微盈通 V2 — 服务器部署指南

## 环境要求

| 组件 | 版本要求 | 用途 |
|------|---------|------|
| 操作系统 | Ubuntu 22.04+ / CentOS 8+ / Debian 12+ | — |
| Docker | 24+ | 容器运行时 |
| Docker Compose | v2+ | 服务编排 |
| 域名 | 已备案 | 微信回调需要 HTTPS |
| 内存 | ≥ 2GB | PostgreSQL + Redis + Go + Nginx |

---

## 方式一：Docker Compose 部署（推荐）

### 1. 安装 Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | bash
sudo usermod -aG docker $USER
# 重新登录使权限生效
```

### 2. 克隆仓库

```bash
git clone https://github.com/tattotatto/weiyeston.git /opt/weiyeston
cd /opt/weiyeston
```

### 3. 配置环境变量

```bash
cp .env.example .env
nano .env
```

必须填写的配置：

```ini
# 数据库
DB_USER=weiyeston
DB_PASSWORD=<生成强密码>
DB_NAME=weiyeston

# JWT (生成方法: openssl rand -hex 32)
JWT_SECRET=<64位随机字符串>

# 微信第三方平台
WECHAT_COMPONENT_APP_ID=wx...
WECHAT_COMPONENT_APP_SECRET=...
WECHAT_TOKEN=<自定义Token>
WECHAT_ENCODING_AES_KEY=<43位随机字符串>
WECHAT_SERVER_URL=https://your-domain.com

# AI (可选)
AI_PROVIDER=deepseek
AI_BASE_URL=https://api.deepseek.com/v1
AI_API_KEY=sk-...
AI_MODEL=deepseek-chat

# CORS
CORS_ORIGINS=https://your-domain.com
```

### 4. 配置 HTTPS（微信回调必须）

```bash
# 安装 certbot
sudo apt install certbot python3-certbot-nginx

# 获取证书
sudo certbot certonly --standalone -d your-domain.com

# 证书路径（certbot 默认输出路径）
# 证书: /etc/letsencrypt/live/your-domain.com/fullchain.pem
# 私钥: /etc/letsencrypt/live/your-domain.com/privkey.pem
```

编辑 `docker/nginx.conf`，在 `server { listen 80; ... }` 之前添加：

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate     /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # 其余配置同 HTTP server block（见 docker/nginx.conf）...
}

server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$host$request_uri;
}
```

然后在 `docker-compose.prod.yml` 的 nginx 服务中挂载证书：

```yaml
nginx:
  volumes:
    - ./docker/nginx.conf:/etc/nginx/conf.d/default.conf:ro
    - /etc/letsencrypt:/etc/letsencrypt:ro   # 挂载证书
    - uploads_data:/app/uploads:ro
  ports:
    - "80:80"
    - "443:443"
```

### 5. 启动服务

```bash
docker compose -f docker-compose.prod.yml up -d
```

### 6. 验证

```bash
# 健康检查
curl https://your-domain.com/api/health

# 管理后台
# 浏览器打开: https://your-domain.com/admin
```

### 7. 查看日志

```bash
docker compose -f docker-compose.prod.yml logs -f
```

---

## 方式二：手动部署（无 Docker）

### 1. 安装依赖

```bash
# PostgreSQL 16
sudo apt install postgresql-16 redis-server nginx

# 创建数据库
sudo -u postgres psql -c "CREATE USER weiyeston WITH PASSWORD 'strong-password';"
sudo -u postgres psql -c "CREATE DATABASE weiyeston OWNER weiyeston;"
```

### 2. 构建 Go 后端

```bash
cd /opt/weiyeston
go build -ldflags="-s -w" -o bin/weiyeston ./cmd/server
```

### 3. 构建前端

```bash
cd /opt/weiyeston/web/admin
npm ci --registry=https://registry.npmmirror.com
npm run build
# 产物在 dist/ 目录
```

### 4. 部署文件

```bash
sudo mkdir -p /opt/weiyeston
sudo cp bin/weiyeston /opt/weiyeston/
sudo cp -r templates /opt/weiyeston/
sudo cp -r migrations /opt/weiyeston/
sudo cp -r web/admin/dist /opt/weiyeston/web/admin
sudo cp config.prod.yaml /opt/weiyeston/config.yaml
sudo mkdir -p /opt/weiyeston/uploads
```

编辑 `/opt/weiyeston/config.yaml` 填入真实配置。

### 5. 配置 systemd

```bash
sudo cp deployment/weiyeston.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable weiyeston
sudo systemctl start weiyeston
```

### 6. 配置 Nginx

```bash
sudo cp deployment/nginx.conf /etc/nginx/sites-available/weiyeston
sudo ln -s /etc/nginx/sites-available/weiyeston /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

---

## 微信开放平台配置

部署完成后，在 [微信开放平台](https://open.weixin.qq.com) 配置：

| 配置项 | 值 |
|--------|-----|
| 授权事件接收 URL | `https://your-domain.com/wx/component/callback` |
| 消息校验 Token | 与 `WECHAT_TOKEN` 一致 |
| 消息加解密 Key | 与 `WECHAT_ENCODING_AES_KEY` 一致 |
| 授权后跳转域名 | `your-domain.com` |

---

## 升级步骤

```bash
cd /opt/weiyeston
git pull

# Docker 方式
docker compose -f docker-compose.prod.yml up -d --build

# 手动方式
go build -ldflags="-s -w" -o bin/weiyeston ./cmd/server
sudo systemctl restart weiyeston
```

---

## 备份

```bash
# 数据库备份
docker exec weiyeston-postgres pg_dump -U weiyeston weiyeston > backup_$(date +%Y%m%d).sql

# 上传文件备份
tar -czf uploads_backup_$(date +%Y%m%d).tar.gz uploads/
```

---

## 故障排查

```bash
# 查看服务日志
docker compose -f docker-compose.prod.yml logs server

# 进入容器调试
docker exec -it weiyeston-server sh

# 检查端口
netstat -tlnp | grep -E '80|443|8080|5432|6379'

# 测试数据库连接
docker exec weiyeston-postgres pg_isready -U weiyeston
```
