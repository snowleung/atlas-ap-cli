# GitHub Actions Windows EXE 发布设计

## 目标

为 `atlas-ap-remote` 增加基于 GitHub Actions 的 Windows 单文件 EXE 发布流程。
用户推送符合 `v*` 格式的 Git tag 后，Actions 在 Windows runner 上构建
`atlas-ap-remote.exe`，生成 SHA256 校验文件，并自动创建 GitHub Release。

本次设计只覆盖正式 tag 发布，不增加 PR、普通分支 push 或独立的验证工作流。

## 触发与工作流架构

新增 `.github/workflows/release.yml`，仅由版本 tag 触发：

```yaml
on:
  push:
    tags:
      - "v*"
```

工作流只包含一个 Windows job，顺序执行以下步骤：

1. checkout 当前 tag 对应的代码。
2. 安装仓库要求的 Go 版本。
3. 从 `github.ref_name` 取得 tag，并移除开头的 `v`，得到发布版本。
4. 调用 `build.ps1 -Version <version>` 构建 Windows amd64 EXE。
5. 检查 EXE 存在且非空。
6. 计算 EXE 的 SHA256 校验值。
7. 使用当前 tag 创建 GitHub Release，并上传 EXE 与校验文件。

不设置 `needs: validate` 或其他前置 job。构建、校验或 Release 上传任一步骤失败，
工作流直接失败。

## 构建脚本调整

将 `build.ps1` 作为本地和 CI 的统一构建入口，增加参数：

```powershell
-Version "0.2.0"
```

`-Version` 默认值为 `dev`，本地不传参数时仍可正常构建；CI 传入去掉 `v` 前缀的
tag 版本。脚本保留现有参数：

- `-Arch`：默认 `amd64`，允许 `amd64`、`386`、`arm64`。
- `-Out`：默认 `dist`。

构建固定使用：

```text
GOOS=windows
GOARCH=amd64
CGO_ENABLED=0
-trimpath
-ldflags "-s -w -X github.com/atlas-ap/atlas-ap-remote/internal/cli.Version=<version>"
```

脚本应只执行一次 `go build`，避免当前脚本先无版本构建、再版本构建的重复工作。
输出路径仍为 `<Out>/atlas-ap-remote.exe`。

## Release 资产

CI 将构建脚本的通用输出重命名为带平台信息的正式资产：

```text
atlas-ap-remote-windows-amd64.exe
atlas-ap-remote-windows-amd64.exe.sha256
```

Release 的 tag 和名称使用当前 tag，例如：

```text
tag:  v0.2.0
name: v0.2.0
```

Release notes 使用 GitHub 的自动生成能力。正式发布默认不标记为 prerelease。
当前只发布 Windows amd64；后续如有明确需求，再扩展为 Windows arm64 矩阵构建。

## 权限与并发

工作流使用最小必要权限：

```yaml
permissions:
  contents: write
```

该权限用于创建 Release 和上传资产。运行时服务端 URL、Token 以及其他业务密钥
不进入构建环境，也不写入 EXE。

使用 tag 维度的并发组：

```yaml
concurrency:
  group: release-${{ github.ref }}
  cancel-in-progress: false
```

同一 tag 的重复发布不应静默覆盖正式资产；Release 创建或资产上传冲突时工作流
明确失败，由维护者处理重复运行问题。

## 错误处理与验收标准

工作流必须满足：

- 仅推送 `v*` tag 时运行。
- 构建失败时不会产生成功的 Release。
- 生成的 EXE 非空，且为 Windows amd64、CGO disabled 的单文件程序。
- EXE 内的 `--version` 输出去掉 `v` 前缀后的版本号。
- SHA256 文件对应最终上传的 EXE。
- Release 同时包含 EXE 和 SHA256 文件。
- 工作流日志不包含任何服务端 Token 或运行时凭据。
- 普通 push 和 PR 不触发该发布工作流。

本地可使用以下命令复现正式构建：

```powershell
.\build.ps1 -Version 0.2.0
```

## 非目标

- 不在本次工作中增加 PR / push 验证工作流。
- 不制作 MSI、安装包、压缩包或签名证书流程。
- 不引入第三方 Go 依赖。
- 不修改 CLI 业务逻辑和服务端协议。
