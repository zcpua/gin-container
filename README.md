# gin-container

Gin (Go) HTTP 服务,部署在**微信云托管**(WeChat Cloud Run),后端使用 Postgres + GORM。
实现了 cantabile 的 `/api/v2` 接口。

## 结构

- `main.go` — GORM 连 Postgres + 路由装配 `/api/v2`,监听 `PORT`(默认 8080)
- `models.go` / `repository.go` / `handlers.go` — 模型、查询层、HTTP handler
- `wechat.go` — 读取微信云托管网关注入的 `X-WX-OPENID` 鉴权
- `storage.go` — 头像上传到腾讯 COS(public-read)
- `Dockerfile` — 多阶段构建静态二进制
- `docker-compose.yml` — 本地联调(Postgres + API)

## 环境变量

| 变量 | 说明 |
| --- | --- |
| `DATABASE_URL` | Postgres 连接串(必填) |
| `PORT` | 监听端口,默认 8080 |
| `COS_BUCKET` | 头像上传的 COS bucket;不配则头像上传返回 501 |
| `COS_REGION` | COS 区域,默认 `ap-shanghai` |
| `COS_CLOUD_ENV` | 微信云环境 ID,用于生成 `cloud://` fileId |
| `COS_PUBLIC_DOMAIN` | COS 自定义公开域名 |
| `WECHAT_DEV_OPENID` | 仅本地开发:绕过网关鉴权 |

## 本地开发

```sh
docker compose up --build      # 起 Postgres + Gin,自动建表
curl localhost:8080/api/v2/composers
docker compose down            # 停;down -v 连数据卷一起删
```

## 部署

合并到 `main` 分支后,GitHub Action(`.github/workflows/deploy.yml`)会自动用
`wxcloud` CLI 部署到微信云托管。需要在仓库 Secrets 配置:

| Secret | 说明 |
| --- | --- |
| `WX_APPID` | 微信 AppID |
| `WX_PRIVATE_KEY` | 微信云托管 API 密钥 |
| `WX_ENV_ID` | 云托管环境 ID |
| `WX_SERVICE_NAME` | 云托管服务名 |
