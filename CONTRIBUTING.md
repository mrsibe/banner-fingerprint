# 开发约定

## 本地流程

Go 1.23+，无需安装第三方库。

```bash
gofmt -w cmd internal rules
go vet ./...
go test -race -cover ./...
go build ./...
```

CI 使用最低 Go 版本及最新稳定版，并验证真实 Client/Server 和 Docker Compose 输出。变更提交前请执行可运行的本地检查；无法验证的环境限制应明确记录。

## 新增指纹

1. 修改 `rules/fingerprints.json`，为规则分配唯一 `name`。
2. 提供 `protocol`、可选 `product`、Go RE2 `regex`、可选 `version_group`、`priority`。
3. `version_group` 是从 1 开始的捕获组索引；0 或省略表示无版本。正则不支持 look-around 和 backreference。
4. 添加真实风格的正例，以及相似产品名、无版本、正文嵌入等负例。
5. 如改变支持范围，更新 README 和 API 文档。
6. 重建程序；运行中的 Server 不热更新规则。

不得根据端口猜测产品或版本。不要复制来源或许可证不清楚的大型规则库。置信度统一由引擎按识别具体程度决定，不为每个规则随意增加评分。

## 提交

使用简洁的 Conventional Commits 风格：`feat:`、`fix:`、`test:`、`docs:`、`ci:`。每个提交只包含对应范围，不提交二进制、日志、凭据或 agent 临时文件。涉及外部行为的修复应带回归测试。

## 文档入口

- [README](README.md)：运行和功能边界
- [API](docs/api.md)：输入、输出、错误和限制
- [架构](docs/architecture.md)：匹配策略及部署决策
