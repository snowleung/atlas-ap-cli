# Atlas AP Remote Go CLI 设计

## 目标

实现一个纯 Go、无需运行时依赖的 `atlas-ap-remote` 命令行客户端，兼容
`client_dist/CLI_USAGE.md` 中的调用方式，重点支持 Windows 单文件
`atlas-ap-remote.exe` 分发。客户端只执行一次 HTTP 请求，不负责轮询。

## 用户接口

全局参数位于子命令之前：

```text
atlas-ap-remote --server <url> [--token <token>] <command> [args] [options]
```

`--server` 缺省时读取 `ATLAS_REMOTE_URL`；`--token` 缺省时读取
`ATLAS_REMOTE_TOKEN`。Token 仅用于构造 Bearer header，不写入文件、日志或
错误消息。服务地址为空时命令失败并返回稳定错误。

命令范围：

- `submit --file <path> [--cos-type] [--body-parts] [--product-name]
  [--usage-method] [--json]`
- `status <job-id> [--json]`
- `download <job-id> [--output-dir] [--keep-zip] [--json]`
- `cancel <job-id> [--json]`
- `--help`、`--version`

默认值与文档一致：`cos-type=驻留`、`body-parts=全身`、下载目录为当前目录。
命令名和可执行文件采用正确拼写 `atlas-ap-remote`。

## 架构与数据流

采用标准库分层，避免外部 CLI 框架依赖：

1. `cmd/atlas-ap-remote` 负责入口、版本和退出码。
2. `internal/cli` 负责参数解析、环境变量回退、人类输出与 JSON envelope。
3. `internal/client` 负责 HTTP 请求、认证、超时、错误映射和 multipart 上传。
4. `internal/archive` 负责 ZIP 下载落盘、大小限制、路径安全校验和解压。

服务端协议固定为：`POST /jobs`、`GET /jobs/{id}`、`GET
/jobs/{id}/download`、`POST /jobs/{id}/cancel`。submit 每次生成新的 UUID
幂等键，并随 multipart 表单发送；status/cancel 各只执行一次请求。

## 输出与错误处理

成功时 `--json` 输出单行 UTF-8 JSON，字段与文档兼容；失败时同样向标准输出
输出 `success=false`、`code`、`message`、可选 `http_status`，并返回退出码 1。
网络超时映射为 `TIMEOUT`，其他网络故障映射为 `NETWORK_ERROR`；服务端 JSON
错误优先保留其 `code/message`。人类模式将错误写到标准错误，JSON 模式保证
调用脚本仍可解析标准输出。

下载最多接受 500 MiB。先写入同目录临时文件，成功后再解压；所有 ZIP 成员
在写盘前检查绝对路径、`..` 路径和 Windows 反斜杠路径，任一不安全成员都以
`UNSAFE_ZIP_MEMBER` 拒绝，避免 zip-slip。失败时清理临时文件和新生成的 ZIP。

## 测试与交付

使用 Go 标准测试和 `httptest.Server` 验证：参数/环境变量、认证 header、四个
请求的 method/path/字段、JSON 与退出码、网络错误、单次请求行为、下载解压、
500 MiB 限制以及 ZIP 路径穿越防护。

提供 `build.ps1`，默认构建 Windows amd64：

```powershell
$env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w" -o dist/atlas-ap-remote.exe ./cmd/atlas-ap-remote
```

同时支持在 macOS/Linux 使用同等 `GOOS/GOARCH` 环境变量交叉编译；不引入 CGO，
保证生成文件可直接分发。
