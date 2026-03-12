# Security Group Manager

阿里云安全组动态 IP 白名单管理工具。提供 Web UI 和 CLI 客户端，允许用户将当前 IP 自动更新到阿里云 ECS 安全组规则中。

## 功能特性

- **Web UI** — 浏览器中输入用户名和密码即可一键更新 IP
- **CLI 客户端** — 支持定时轮询，适合后台持续运行
- **自动规则管理** — 自动移除旧 IP 规则，添加新 IP 规则
- **安全防护** — 密码恒定时间比较、按 IP 限流与暴力破解防护
- **单文件部署** — Web 资源内嵌到二进制文件，无需额外文件

## 项目结构

```
├── main.go                  # 服务端入口
├── config.yaml.example      # 配置文件模板
├── cmd/client/main.go       # CLI 客户端
├── internal/
│   ├── aliyun/aliyun.go     # 阿里云安全组操作
│   ├── auth/auth.go         # 认证与限流
│   ├── config/config.go     # 配置加载
│   └── server/server.go     # HTTP 服务与 API
└── web/index.html           # Web 前端页面
```

## 快速开始

### 环境要求

- Go 1.25+
- 阿里云 AccessKey（需要 ECS 安全组权限）

### 编译

```bash
# 编译服务端
go build -o security-group .

# 编译客户端
go build -o sg-client ./cmd/client
```

### 配置

复制配置模板并填入实际参数：

```bash
cp config.yaml.example config.yaml
```

```yaml
server:
  listen: "127.0.0.1:8080"

security:
  max_failures: 5        # 最大失败次数
  fail_window: 5m        # 失败计数窗口
  block_duration: 30m    # 封禁时长

aliyun:
  access_key_id: "your-access-key-id"
  access_key_secret: "your-access-key-secret"
  region_id: "cn-shenzhen"
  security_group_id: "sg-xxx"

password: "your-password"
```

### 运行服务端

```bash
./security-group -config config.yaml
```

访问 `http://127.0.0.1:8080` 即可打开 Web 界面。

### 使用 CLI 客户端

```bash
./sg-client \
  -url https://example.com/api/update \
  -username myuser \
  -password your-password \
  -interval 5m
```

参数说明：

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-url` | 服务端 API 地址 | — |
| `-username` | 用户名 | — |
| `-password` | 密码 | — |
| `-interval` | 轮询间隔 | `1m` |

## API

### POST /api/update

请求体（JSON）：

```json
{
  "username": "myuser",
  "password": "your-password"
}
```

响应：

```json
{
  "code": 0,
  "message": "IP updated successfully",
  "ip": "1.2.3.4"
}
```

状态码含义：`0` 成功 | `1` 请求错误 | `2` IP 被封禁 | `3` 服务端错误

## 反向代理

如需部署在反向代理（Nginx 等）后面，请确保正确传递客户端真实 IP：

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

## License

MIT
