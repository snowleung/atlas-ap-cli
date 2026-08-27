# Health 命令设计

## 目标

为 `atlas-ap-remote` 增加 `health` 子命令，用于调用服务端存活探针：

```text
GET /health
```

成功响应按服务端 OpenAPI 定义为 JSON 对象，当前预期内容为：

```json
{"status":"ok"}
```

本次不处理 TLS 证书信任问题，不增加自定义 CA 配置，也不关闭证书校验。

## 用户接口

普通输出：

```text
atlas-ap-remote --server https://ap.atlaslabtest.com health
```

输出：

```text
status=ok
```

JSON 输出：

```text
atlas-ap-remote --server https://ap.atlaslabtest.com --token <token> health --json
```

输出：

```json
{"status":"ok","success":true}
```

`--server` 和 `--token` 继续遵循现有全局参数规则；配置 Token 时，客户端向
`/health` 请求发送 `Authorization: Bearer <token>`。未配置 Token 时不发送该 header。

## 架构

在 `internal/client` 增加 `Health` 方法，负责构造并执行 `GET /health`，复用已有的
URL 构建、HTTP 超时、认证和错误映射逻辑。

在 `internal/cli` 增加 `health` 分支和响应输出逻辑。健康响应使用结构体保存
`status` 字段；成功 envelope 在 JSON 模式中追加现有的 `success: true` 字段。
网络错误、TLS 错误和服务端非 2xx 响应沿用现有错误码与退出码。

帮助文本的顶层命令列表、README 命令列表和命令帮助同步增加 `health`。

## 测试与验收

- client 测试确认请求 method 为 `GET`、路径为 `/health`。
- client 测试确认有 Token 时发送 Bearer header，无 Token 时不发送。
- CLI 测试确认普通输出为 `status=ok`。
- CLI 测试确认 JSON 输出包含 `status` 和 `success: true`。
- CLI 测试确认服务端错误沿用现有错误 envelope。
- 帮助测试确认顶层帮助包含 `health`。
- `go test ./...`、`go vet ./...` 和 `gofmt` 通过。

## 非目标

- 不修改 TLS 校验行为。
- 不增加轮询、重试或健康状态缓存。
- 不增加其他探针路径，如 `/ready` 或 `/metrics`。
