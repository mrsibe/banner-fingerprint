# 架构与设计决策

## 数据路径

```text
JSON 文件 → Go CLI → POST /api/v1/identify
                         ↓
                 大小、结构、字段校验
                         ↓
              immutable Fingerprint Engine
               ├─ Base64 → MySQL 前缀解析
               └─ 文本 → RE2 规则及捕获组
                         ↓
                  同序 JSON 结果数组
```

系统不对目标执行任何网络操作；唯一网络行为是 Client 到 Server 的 HTTP 请求。

## 最小依赖

采用 `net/http`、`regexp`、`encoding/json` 和 `go:embed`。选择 JSON 而非 YAML，省去额外解析库和依赖供应链。规则是编译时资产，启动时统一编译和校验，失败即退出；没有动态加载、数据库或共享缓存。

引擎构造后不再修改，正则可安全并发使用。处理结果局部构造，API 整批成功后才输出，避免部分写入响应导致错误格式不完整。HTTP 服务有读取、写入、空闲超时，并处理 SIGINT / SIGTERM 优雅关闭（最多等待 10 秒）。单批顺序处理，无每条 goroutine；由 net/http 处理请求并发。没有全局并发上限，对外部署须在代理层补齐。

## 匹配范围与排序

- OpenSSH 必须出现在 SSH identification-line 中，避免任意正文中的产品名误报。
- HTTP 产品要求 `Server:` 行；HTTP 状态行开头的输入，只使用第一段头部，不扫描正文，也不让正文切换成另一协议。
- 对无状态行的 HTTP header 片段，也只匹配首个空行前的内容。
- 其他产品使用协议特征，例如 FTP 的 220 greeting、Redis 的 INFO 属性。
- MySQL protocol-10 专门解析二进制，以读取 server version，而非对原始二进制盲目正则。

对所有命中规则按以下顺序取最大值：

1. `priority`：越大越优先，产品规则通常为 100，协议兜底为 10。
2. 具体程度：协议 < 产品 < 产品及版本。
3. 实际匹配字节长度。
4. 文件顺序：完全相同时先出现者胜出。

“匹配长度”只是简单确定性的 tie-break，不代表语义质量；规则质量主要依赖明确的协议上下文、边界和测试。confidence 按结果具体程度分级，不因为端口符合预期而加分。OS 提示是独立关键词提取；未知协议不单独输出 OS。HTTP 正文的 OS 关键词不参与。

## 边界

规则适用于有限产品的常见 Banner，并非完整 Nmap 探测库。HTTP 不做完整语法解析，MySQL 也不是完整协议实现。输入伪造无法被被动识别解决。后续新增产品应优先增加精确规则和反例，而不是引入庞大基础设施。

部署采用两个无 shell、非 root 的 scratch 镜像，Compose 以进程自身健康检查实现启动顺序。client 是一次性任务；无需写盘或容器 capabilities。
