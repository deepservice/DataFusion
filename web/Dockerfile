# DataFusion Web 前端 Dockerfile
# 多阶段构建，优化镜像大小

# 阶段 1: 构建阶段
FROM node:18-alpine AS builder

# 设置工作目录
WORKDIR /app

# 设置环境变量，禁用 TypeScript 类型检查（加快构建，类型在开发阶段已检查）
ENV SKIP_PREFLIGHT_CHECK=true
ENV EXTEND_ESLINT=true
ENV NODE_OPTIONS=--max_old_space_size=4096

# 复制 package.json 和 package-lock.json
COPY package*.json ./

# 安装依赖（包含 dev 依赖用于构建，使用 --legacy-peer-deps 处理依赖冲突）
# 注意：不使用 npm audit fix，因为它会激进地修改版本，导致依赖破坏
RUN npm install --legacy-peer-deps --omit=optional && \
    npm install ajv@^8 --save-dev --legacy-peer-deps && \
    npm ls ajv 2>/dev/null || true

# 复制源代码
COPY . .

# 移除 jsconfig.json（React Scripts 不允许同时存在 tsconfig.json 和 jsconfig.json）
RUN rm -f jsconfig.json

# 构建生产版本
RUN npm run build

# 阶段 2: 生产阶段
FROM nginx:alpine

# 复制自定义 nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 从构建阶段复制构建产物
COPY --from=builder /app/build /usr/share/nginx/html

# 暴露端口
EXPOSE 80

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost/ || exit 1

# 启动 nginx
CMD ["nginx", "-g", "daemon off;"]
# 启动 nginx
CMD ["nginx", "-g", "daemon off;"]
