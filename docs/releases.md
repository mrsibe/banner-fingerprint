# 版本发布与 CD

## v0.1.1

- 新增题目指定的 `POST /fingerprint`、`GET /health`，保留旧路径别名。
- 未识别结果统一为 `protocol: "unknown"`；空/缺失 Banner 返回 unknown，合法未知数据不导致请求失败。
- Client 和容器健康检查切换为指定接口。
- Compose 使用专用 internal 网络，仅向宿主机回环地址暴露服务端口。
- 增加真实 HTTP 契约、别名一致性、unknown、错误后健康检查与网络隔离断言。
- 收录全部 20 条题目样本及独立预期；新增 Redis PONG/错误、Pure-FTPd、MySQL 截断握手支持，并同时验证文本和 Base64 传输。
- Apache 产品名统一为题目使用的 `Apache`（原为 `Apache httpd`）。
- 仓库公开，补齐与接口行为一致的文档。

兼容性注意：未知结果的 protocol 从空字符串变为 unknown，调用者需相应调整。该版本不宣称能追溯满足原题的截止时间或会议录屏要求。

## 首版 v0.1.0

轻量级被动 Banner 指纹识别 MVP：

- 批量 HTTP API、Go CLI 和健康检查。
- 内嵌 JSON 指纹规则、优先级与版本捕获、OS 关键词提示。
- MySQL / MariaDB 二进制握手前缀解析。
- 七条演示、异常参数、未知 Banner 和误报回归测试。
- 非 root Docker Compose 部署和 GitHub Actions CI/CD。

不包含主动扫描、数据库、Web UI、动态规则更新或完整协议解析。置信度是启发式评分，不是经过校准的概率。

## 发布路径

```text
推送版本标签 vX.Y.Z
  ├─ verify：复用 CI（Go 1.23 / stable、race、CLI、Compose、七条结果）
  └─ build：跨平台编译 → 压缩包 → SHA256SUMS → Actions artifact
        ↓ 两者成功
  publish：下载产物 → 验证校验和 → 创建 draft Release → 公开发布状态
```

“公开发布状态”是相对草稿而言，不会改变仓库本身的可见性。未通过 CI 不会发布 GitHub Release。只有 publish job 拥有 `contents: write`，其余 job 只读。不发布 Docker 镜像、不访问部署服务器、不使用长期发布令牌。

## 产物

文件名为 `banner-fingerprint_vX.Y.Z_GOOS_GOARCH.tar.gz`，Windows 为 `.zip`。每个平台包含独立的 `banner-server` 和 `banner-client` 可执行程序、README、examples 与 docs。所有程序禁用 CGO。

| 系统 | 架构 |
| --- | --- |
| Linux | amd64、arm64 |
| macOS | amd64、arm64 |
| Windows | amd64、arm64 |

运行测试在 Linux amd64 完成；其他目标执行交叉编译，不宣称已验证对应机器上的运行行为。没有代码签名、公证或自动升级功能，macOS / Windows 可能显示未签名程序提示。校验和用于完整性检查，不等同于独立数字签名。

Linux 校验（先下载所有压缩包和 SHA256SUMS）：

```bash
sha256sum -c SHA256SUMS
```

只下载一个包时，可筛选其对应行再校验；macOS 使用 `shasum -a 256`，Windows 可使用 PowerShell `Get-FileHash -Algorithm SHA256` 比对。

## 维护者发布操作

先确认 main 的 CI 成功、发布内容已提交，且工作区没有需要发布的未提交改动。使用新的稳定版号（当前不支持预发布标签）：

```bash
git tag -a v0.1.1 -m "Release v0.1.1"
git push origin v0.1.1
```

tag 指向的提交是唯一发布源；不要移动已经发布的 tag，不要覆盖旧版本产物。发布说明由 GitHub 根据提交/PR 自动生成。流程文件为 `.github/workflows/release.yml`。

若构建或验证失败，在 Actions 检查日志、修复后以新标签重新发布。若已创建 draft 但发布步骤中断，先检查 draft 的附件与校验和，再由维护者决定如何完成该草稿；工作流不会静默覆盖已有同名 Release。

## 本地打包

需要 Go、bash、tar、zip、sha256sum，在仓库根目录执行：

```bash
bash scripts/package.sh v0.1.0
```

结果在 `dist/`（不入 Git），已有 dist 时脚本拒绝覆盖。此操作只构建产物，不创建 tag 或发布远程资源。
