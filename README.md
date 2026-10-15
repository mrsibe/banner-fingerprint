# Banner Fingerprint

基于 Go 的轻量级、被动 Banner 指纹识别系统。Client 提交批量 Banner，Server 返回协议、产品、版本、操作系统提示和启发式置信度。

**不是网络扫描器**：不会连接目标 IP，不提供端口探测、Web UI、数据库、动态规则更新或复杂规则引擎。Go 标准库实现，零第三方运行依赖。

## 快速开始

需要 Go 1.23+。在两个终端分别运行：

```bash
go run ./cmd/server
```

```bash
go run ./cmd/client -input examples/input.json
# 自定义服务地址、请求超时
# go run ./cmd/client -server http://localhost:8080 -timeout 10s -input examples/input.json
```

或者使用 Docker Engine + Compose v2：

```bash
docker compose up --build
```

Compose 启动 server，健康检查通过后 client 执行七条示例并输出 JSON，随后正常退出；server 继续运行。Ctrl+C 停止，`docker compose down` 清理。默认仅向本机暴露 8080。自动化演示可用：

```bash
docker compose up --build --abort-on-container-exit --exit-code-from client
```

## 支持范围与示例

| Banner 特征 | 协议 | 产品 | 示例版本 |
| --- | --- | --- | --- |
| SSH identification / OpenSSH | SSH | OpenSSH | 8.9p1 |
| HTTP 状态行 / Server header | HTTP | nginx | 1.24.0 |
| Server: Apache | HTTP | Apache httpd | 2.4.57 |
| MySQL protocol-10 二进制握手 | MySQL | MySQL / MariaDB | 8.0.36 |
| Redis INFO / protected-mode 消息 | Redis | Redis | 7.2.4 |
| FTP greeting | FTP | ProFTPD / vsftpd | 1.3.8 |
| Server: Jetty | HTTP | Jetty | 9.4.54.v20240208 |
| Server: Microsoft-IIS | HTTP | Microsoft IIS | 10.0 |

`examples/input.json` 提供七条代表性输入，不宣称是未提供的原题原始样本。额外覆盖协议兜底、产品无版本、无匹配、冲突端口与规则优先级。

## API

唯一业务接口：`POST /api/v1/identify`。另提供部署健康检查 `GET /healthz`。

```bash
curl --fail-with-body http://localhost:8080/api/v1/identify \
  -H 'Content-Type: application/json' \
  --data '[{"ip":"192.0.2.1","port":22,"banner":"SSH-2.0-OpenSSH_8.9p1 Ubuntu-3ubuntu0.6"}]'
```

```json
[
  {
    "ip": "192.0.2.1",
    "port": 22,
    "protocol": "SSH",
    "product": "OpenSSH",
    "version": "8.9p1",
    "os_hint": "Ubuntu",
    "confidence": 0.95
  }
]
```

结果顺序与输入一致；IP/端口仅用于关联结果，不作为识别依据。未知结果字段为空字符串、置信度为 0。二进制输入使用 `banner_base64`，不能同时传 `banner`，详见 [API 契约](docs/api.md)。

## 规则和置信度

14 条自维护 JSON 规则通过 `go:embed` 编译进程序，不依赖工作目录。使用 Go RE2 正则与捕获组提取版本；MySQL 二进制走专用解析。没有导入 Nmap 规则库。

选择顺序：**优先级 → 产品/版本具体程度 → 匹配长度 → 文件顺序**。精确产品及版本为 0.95，仅产品为 0.85，仅协议特征为 0.70。这些是固定启发式等级，**不是经过校准的准确概率**。

## 开发与验证

```bash
gofmt -w cmd internal rules
go vet ./...
go test -race -cover ./...
go build ./...
# 可选的短时模糊测试
go test ./internal/fingerprint -run '^$' -fuzz FuzzIdentify -fuzztime 10s
```

GitHub Actions 在 push / pull request 时执行最低 Go 版本和最新稳定版的测试、race 检查、构建、CLI 端到端断言及 Docker Compose 集成测试。需要先自行建立 GitHub 远程仓库并推送，工作流才会运行。

## 项目结构

```text
cmd/{server,client}/     两个 Go 可执行程序
internal/fingerprint/  识别引擎及测试
internal/api/          HTTP 批量接口及测试
rules/                 内嵌 JSON 规则
examples/              七条演示输入
docs/                  API、架构、开发约定
scripts/               CI 演示结果断言（仅需 Python 标准库）
.github/workflows/     自动化测试与 Compose 验证
```

## 已知局限

- Banner 可伪造、隐藏或过时，识别结果不是安全结论；版本未知时不猜测。
- HTTP 只分析提交的文本，不主动请求；不解析 TLS、压缩或完整 HTTP 消息边界。
- OS 只给出 Ubuntu、Debian、Windows、FreeBSD、CentOS、Linux 关键词提示，不能证明真实内核或发行版。
- MySQL 只接受未加密 protocol-10 握手前缀（payload 或完整单包），不支持任意 TCP 分片重组、其他握手版本或认证流程；示例是最小识别前缀。
- 未内置认证、TLS、全局并发限流。请限于可信网络；对外部署须使用带认证、TLS 和限流的反向代理。
- 每请求最大 4 MiB、1–1000 条，每 Banner 最大 64 KiB；不是无限量批处理系统。

更多说明：[架构与设计](docs/architecture.md) · [API](docs/api.md) · [贡献约定](CONTRIBUTING.md)。
