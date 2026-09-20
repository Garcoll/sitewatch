# Sitewatch

一个用 Go 编写的命令行网站检查工具。

输入一个网址，程序发送 HTTP GET 请求，并输出 HTTP 状态码和收到响应头的耗时。

## 当前功能

- 从命令行读取一个或多个网址，按顺序检查
- 单个网址请求失败后，继续检查后续网址
- 显示 HTTP 状态码
- 显示从发起请求到收到响应头的耗时
- 设置 5 秒请求超时
- 请求失败时输出错误原因
- 未提供网址时显示用法提示

## 运行环境

本项目开发时使用 Go 1.25.1。

## 使用方式

在项目目录中运行：

```powershell
go run . https://www.baidu.com
```

网址需要包含 `http://` 或 `https://`。

输出示例，实际状态码和耗时可能不同：

```text
website: https://www.baidu.com
HTTP status: 200
response headers received in: 88.0655ms
```
go run . https://www.baidu.com https://github.com
website: https://www.baidu.com
HTTP status: 200
response headers received in: 172.1859ms
website: https://github.com
HTTP status: 200
response headers received in: 656.0185ms

不提供网址时：

```powershell
go run .
```

程序会显示用法提示并结束。

## 当前限制

- 目前按顺序检查，前一个请求结束后才会开始下一个
- 只执行一次检查，不会定时重复
- 不保存历史结果
- 收到 HTTP 响应不代表网站业务功能完全正常
- 记录的耗时不是完整网页加载时间