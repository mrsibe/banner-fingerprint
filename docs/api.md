# HTTP API 契约

## POST /api/v1/identify

请求体是非空 JSON 数组。推荐 `Content-Type: application/json`，服务端按 JSON 解析，不依赖该请求头。字段名大小写建议严格使用下表形式。

| 字段 | 类型 | 约束 |
| --- | --- | --- |
| ip | string | IPv4 或 IPv6 字面量，不接受域名，不对其建立连接 |
| port | integer | 1–65535，只用于关联 |
| banner | string | 非空文本，与 banner_base64 二选一 |
| banner_base64 | string | 标准 RFC 4648 Base64，解码后非空，与 banner 二选一 |

未选择的 Banner 字段可省略或设为空字符串。未知字段、尾随第二份 JSON、无效 Base64、空数组、null、无效 IP/端口均拒绝。请求最大 4 MiB（含 JSON 编码开销），最多 1000 条；每条 Banner 文本 UTF-8 编码或二进制解码后最多 65536 字节。采用整批校验：任意一项无效，整批返回错误，不返回部分结果。

### 二进制 MySQL

`banner_base64` 支持原始 protocol-10 payload 或带 4 字节包头的单个完整握手包。服务端检查协议字节、NUL 终止的版本字符串、连接 ID、8 字节认证种子和其后填充字节；完整包还检查长度和 sequence ID 0。仅做服务识别所需的前缀验证，不验证完整 capability/authentication 结构。

```python
import base64
import json
payload = b'\x0a8.0.36\x00' + bytes([1, 0, 0, 0]) + b'abcdefgh\x00'
print(json.dumps([{
    'ip': '192.0.2.4', 'port': 3306,
    'banner_base64': base64.b64encode(payload).decode()
}]))
```

不合法的 Base64 是参数错误；合法 Base64 中未识别的二进制返回未知结果，不抛出协议解析错误。MariaDB 保留原始握手版本字符串（包括兼容前缀，如 `5.5.5-10.11.6-MariaDB`），不猜测或重写版本号。

### 成功响应：200

返回与输入等长、同序的数组，每项始终含以下字段：

| 字段 | 类型 | 含义 |
| --- | --- | --- |
| ip / port | string / integer | 原样回显 |
| protocol | string | SSH、HTTP、MySQL、Redis、FTP 或空字符串 |
| product | string | 产品或空字符串 |
| version | string | 捕获的版本或空字符串 |
| os_hint | string | 显式关键词提示或空字符串 |
| confidence | number | 0、0.70、0.85、0.95 的启发式分级 |

无匹配也是 200：

```json
[{"ip":"192.0.2.1","port":22,"protocol":"","product":"","version":"","os_hint":"","confidence":0}]
```

### 错误

- **400**：结构或参数不合法、批量条数或单条 Banner 超限。
- **413**：读取时发现请求体超过 4 MiB；如果此前已发现无效 JSON，也可能优先返回 400。
- **404 / 405**：未知路由 / 方法不允许，由 `net/http` 路由处理（默认文本响应）。

业务校验错误采用 JSON：`{"error":"item 0: invalid IP or port"}`。索引从 0 开始。响应不包含原始 Banner，日志也不记录请求正文。Go 标准 JSON 解码器接受重复字段（后者覆盖前者），不将本接口用作签名或安全策略判断依据。

## GET /healthz

返回 200、`{"status":"ok"}`。规则启动时编译失败则进程退出，因此健康服务的规则集已成功载入。`server -healthcheck` 专用于 Docker 默认 8080 端口，带 2 秒超时；自定义 `-addr` 时请直接请求相应地址的 `/healthz`。

## Client

```text
-input    输入文件（默认 examples/input.json）
-server   服务端 base URL（默认 http://localhost:8080）
-timeout  整个 HTTP 请求超时（默认 15s）
```

成功时 stdout 输出格式化 JSON；读取、网络、非 200 状态或响应 JSON 错误时 stderr 报错并以非零状态退出。不自动重试，不跟随重定向。默认支持系统 TLS 信任；scratch 演示镜像仅用于 Compose 内部 HTTP，没有根证书包。
