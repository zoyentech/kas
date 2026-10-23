# Feature 2.2.1 — 脱敏 config_version

> 状态：todo ｜ Epic 2.2 ｜ 粗估：1.5 人日

## 目标

从规范化配置生成稳定、脱敏、可观测的 config_version：相同配置同版本，secret 明文不参与，secret reference/revision 可参与（docs/01 §6）。

## 设计要点

- 规范化：稳定键序序列化；零值/默认值折叠规则固定。
- 哈希：SHA-256 取前 12 hex 作为版本串。
- secret 字段（env 引用名以外的任何敏感值）不进入规范化输入。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 2.2.1.1 | 规范化与版本哈希 | 3 |

## 验收标准

- 幂等性/顺序无关性/secret 排除三类属性测试通过。

## 需求关联

F-009（版本可追溯的前提）、F-004（cache key 含 config_version）。
