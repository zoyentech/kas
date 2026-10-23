# Feature 4.3.1 — 规范化与 cache key

> 状态：done ｜ Epic 4.3 ｜ 粗估：2 人日

## 目标

问题规范化（NFKC/trim/空白折叠/受控大小写）与七维 key 组成 + 不可逆摘要存储。

## 设计要点

- key 维度（docs/04 §2）：normalize(question)、config_version、principal_scope_fingerprint、authorization_epoch、knowledge_namespace、corpus_generation、retrieval_policy_version。
- 存储仅 key 摘要（不保留原始问题与身份明文）。
- 大小写归一只作用于配置声明的不区分大小写字段；保留否定词/数字/单位/型号符号（docs/04 §2）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 4.3.1.1 | 问题规范化 | 3 |
| 4.3.1.2 | cache key 组成 | 3 |

## 验收标准

- 属性测试固化语义保留约束；key 隔离矩阵通过。

## 需求关联

F-004（P0）。
