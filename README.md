# Sitewatch

一个用 Go 编写的命令行网站检查工具。

输入一个或多个网址，程序通过 HTTP GET 请求检查它们，并输出 HTTP 状态码、请求耗时和检查汇总。

## 当前功能

- 从命令行读取一个或多个网址
- 使用有限并发检查多个网址
- 支持通过参数配置并发数量
- 支持通过参数配置请求超时时间
- 单个网址请求失败后，继续检查其他网址
- 显示 HTTP 状态码
- 显示从发起请求到收到响应头的耗时
- 区分正常状态码、异常状态码和请求失败
- 汇总检查总数及各类结果数量
- 并发执行时保持结果顺序与输入顺序一致
- 使用自动化测试验证状态码、超时、结果顺序和并发上限

## 运行环境

本项目开发时使用：

- Go 1.25.1
- Windows 11
- Git

## 基本使用

在项目目录中运行：

```powershell
go run . https://www.baidu.com
```

检查多个网址：

```powershell
go run . https://www.baidu.com https://github.com https://go.dev
```

网址需要包含 `http://` 或 `https://`。

## HTTP 实验服务

从项目根目录启动：

```powershell
go run ./work/httpdemo
```

服务监听 `127.0.0.1:8081`。保持该终端运行，在另一个终端操作。

### 添加目标

```powershell
$body = '{"url":"http://127.0.0.1:8081/hello","name":"local-hello"}'
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8081/targets -ContentType "application/json" -Body $body
```

### 查询目标

```powershell
curl.exe -sS http://127.0.0.1:8081/targets
```

### 触发检查

```powershell
curl.exe -sS -i -X POST http://127.0.0.1:8081/checks
```

接口检查请求开始时的目标快照，等待整批完成后返回 JSON。
结果按目标快照的顺序排列，包含目标 ID、名称、URL、状态码和毫秒耗时。

- 单次目标请求超时为 5 秒，每批最多并发检查 3 个目标。
- 收到 HTTP 4xx 或 5xx 响应时，记录实际状态码。
- 请求失败时，`status_code` 为 0，`error` 包含错误信息。
- 没有请求错误时，省略 `error` 字段。
- 目标保存在内存中，重启后恢复初始列表；检查历史暂不保存。
- 并发限制作用于每一批检查，尚未限制多批检查的总并发数。

命令行入口与 HTTP 实验共用 `internal/checker` 中的检查逻辑。

## 命令行参数

### `-concurrency`

设置最多同时检查的网址数量，默认值为 `3`。

```powershell
go run . -concurrency 2 https://www.baidu.com https://github.com
```

并发数量必须大于 `0`。

### `-timeout`

设置每个 HTTP 请求的超时时间，默认值为 `5s`。

```powershell
go run . -timeout 3s https://www.baidu.com
```

支持 Go 的时间单位，例如：

```text
500ms
3s
1m
```

超时时间必须大于 `0`。

同时设置并发数量和超时时间：

```powershell
go run . -concurrency 2 -timeout 3s https://www.baidu.com https://github.com
```

查看帮助：

```powershell
go run . -h
```

## 输出示例

具体状态码和耗时会根据网络情况变化：

```text
[状态正常] https://www.baidu.com | HTTP 200 | 耗时 88ms
[状态正常] https://github.com | HTTP 200 | 耗时 650ms

共检查 2 个网址：正常 2，状态异常 0，请求失败 0
整批检查耗时：651ms
```

如果请求失败：

```text
[请求失败] not-a-url | Get "not-a-url": unsupported protocol scheme "" | 耗时 1ms
```

请求失败不会阻止其他网址继续检查。

## 测试

运行全部测试：

```powershell
go test -v ./...
```

测试覆盖：

- HTTP 200 响应
- HTTP 500 响应
- 请求超时
- 并发结果保持输入顺序
- 最大并发请求数

运行静态检查：

```powershell
go vet ./...
```

## 当前限制

- 每次运行只执行一轮检查，不会定时重复
- 不保存历史检查结果
- 不提供 Web 界面或 HTTP API
- 收到 HTTP 响应不代表网站的业务功能完全正常
- 记录的耗时截止到收到响应头，不等同于完整网页加载时间
- 当前只检查 HTTP 状态码和请求是否成功
