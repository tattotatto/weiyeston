#!/usr/bin/env bash
set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

APP_NAME="weiyeston"
DEPLOY_DIR="/opt/${APP_NAME}"
VERSION="${1:-latest}"

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# 检查 root
if [[ $EUID -ne 0 ]]; then
    log_error "请使用 root 权限运行此脚本"
    exit 1
fi

log_info "开始部署 微盈通 V2 (版本: ${VERSION})"

# 1. 创建目录
log_info "创建部署目录..."
mkdir -p "${DEPLOY_DIR}"/{uploads,logs}
mkdir -p /var/log/weiyeston

# 2. 复制文件
log_info "复制二进制文件..."
cp ./bin/weiyeston "${DEPLOY_DIR}/weiyeston"
chmod +x "${DEPLOY_DIR}/weiyeston"

log_info "复制配置文件..."
cp ./config.prod.yaml "${DEPLOY_DIR}/config.yaml"

# 3. 复制前端静态文件
if [[ -d ./web/admin/dist ]]; then
    log_info "复制前端静态文件..."
    rm -rf "${DEPLOY_DIR}/web/admin"
    mkdir -p "${DEPLOY_DIR}/web/admin"
    cp -r ./web/admin/dist/* "${DEPLOY_DIR}/web/admin/"
fi

# 4. 复制模板文件
if [[ -d ./templates ]]; then
    log_info "复制模板文件..."
    cp -r ./templates "${DEPLOY_DIR}/templates"
fi

# 5. 配置 systemd
log_info "配置 systemd 服务..."
cp ./deployment/weiyeston.service /etc/systemd/system/weiyeston.service
systemctl daemon-reload

# 6. 配置 Nginx
log_info "配置 Nginx..."
cp ./deployment/nginx.conf /etc/nginx/sites-available/weiyeston
ln -sf /etc/nginx/sites-available/weiyeston /etc/nginx/sites-enabled/weiyeston
nginx -t && systemctl reload nginx

# 7. 创建系统用户（如果不存在）
if ! id -u weiyeston >/dev/null 2>&1; then
    log_info "创建 weiyeston 系统用户..."
    useradd -r -s /bin/false -d "${DEPLOY_DIR}" weiyeston
fi
chown -R weiyeston:weiyeston "${DEPLOY_DIR}"
chown -R weiyeston:weiyeston /var/log/weiyeston

# 8. 启动服务
log_info "启动服务..."
systemctl enable weiyeston
systemctl restart weiyeston

# 9. 检查状态
sleep 2
if systemctl is-active --quiet weiyeston; then
    log_info "部署成功！"
    log_info "服务状态:"
    systemctl status weiyeston --no-pager
else
    log_error "服务启动失败，请检查日志: journalctl -u weiyeston -n 50"
    exit 1
fi
