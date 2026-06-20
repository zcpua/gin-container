# gin-container

Gin (Go) HTTP 服务,打包成 Docker 镜像,通过 Cloudflare Worker + Containers 运行。

## 结构

- `main.go` / `go.mod` — Gin 应用,监听 `PORT`(默认 8080)
- `Dockerfile` — 多阶段构建静态二进制
- `src/index.ts` — Worker 入口,把请求转发到容器
- `wrangler.jsonc` — Worker + Container + Durable Object 配置

## 本地开发

```sh
go run .                 # 直接跑 Gin(localhost:8080)
docker build -t gin .    # 构建镜像
npm install
npm run dev              # wrangler dev,本地起容器
```

## 部署

```sh
npm run deploy           # wrangler 构建镜像并部署到 Cloudflare
```

需要 Cloudflare 账号已开通 Containers(付费计划),且本地装有 Docker。
