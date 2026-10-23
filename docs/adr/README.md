# ADR — 架构决策记录

ADR（Architecture Decision Record）记录 KAS 的**已决**工程与架构决策：为什么这么选、否决了什么备选、影响哪些需求。

## 目录

| # | 标题 | 状态 |
|---|---|---|
| [0001](0001-scope-v1.2.md) | v1.2 范围与不做清单 | accepted |
| [0002](0002-technology-choices.md) | 技术选型（D-1~D-10） | accepted |

新 ADR 按下一个可用编号追加到本表，不修改已归档 ADR 正文；推翻旧决策时新建 ADR 并将旧的状态改为 `superseded by <n>`。

## 成稿流程

1. 复制 [0000-template.md](0000-template.md) 为 `NNNN-kebab-slug.md`，状态置 `proposed`。
2. 补全背景/决策/备选/后果/影响需求五节（15 分钟内可成稿；写不进一页说明决策还没想清楚）。
3. PR 评审通过后状态改 `accepted`（被否决则 `rejected`，保留备查）。
4. 在本 README 目录表登记。

## 与 multica 的引用方式

multica Issue/Epic 承接开发时，PR 描述与 Issue 评论引用 `ADR-NNNN`（不复制正文，保持单一权威源）。设计文件（`docs/design/`）与 ADR 冲突时：先修订设计文件并提 ADR，再改代码（见 `docs/plan/overview/03-wbs-conventions.md` §7）。
