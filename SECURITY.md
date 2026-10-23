# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| v1.2.x  | ✅（首个支持版本，发布前遵循本策略） |
| < v1.2  | ❌ |

## Reporting a Vulnerability

请**不要**通过公开 Issue 报告安全漏洞。

使用 GitHub **Private Vulnerability Reporting**（Security 标签页 → "Report a vulnerability"）提交报告。请包含：

- 受影响版本 / commit
- 复现步骤或 PoC
- 影响评估与建议修复方向

### 响应预期

- 确认收到：2 个工作日内
- 初步评估：5 个工作日内
- 修复与披露：按严重程度协商，修复发布前不公开细节

## 安全边界（供报告者参考）

KAS 的安全设计约束（详见 `docs/design/01-architecture-prd.md` §6）：

- 问题正文、Evidence 正文、身份明文、secret 不进入指标 label、常规日志与 trace 属性
- 缓存 key 仅使用不可逆摘要；跨 scope/租户/授权 epoch 的缓存命中视为安全缺陷
- 过期、未生效或有效期未知的 Evidence 不得进入回答或缓存（fail-closed）
- 密钥由宿主 secret manager / 环境变量注入，YAML 只含引用，错误文本不含 secret
