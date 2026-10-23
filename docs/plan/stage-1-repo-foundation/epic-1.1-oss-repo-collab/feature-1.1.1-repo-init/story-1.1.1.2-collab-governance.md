# Story 1.1.1.2 — 协作与治理文件

> 状态：todo ｜ 作为外部贡献者/安全研究员，我希望有明确的贡献与漏洞报告渠道，以便合规参与。

## 前置依赖

- T1.1.1.1.1（仓库存在）。

## 任务

#### T1.1.1.2.1 贡献与行为规范文件
- **目标：** CONTRIBUTING.md 与 CODE_OF_CONDUCT.md。
- **产出：** 两个文件。
- **步骤：** 1) CONTRIBUTING 引用 overview/03 的分支/提交/DoD 约定并本地化到仓库 `docs/`；2) 采用 Contributor Covenant 行为准则。
- **验收标准：** 贡献流程（分支命名、Conventional Commits、PR 要求、CI 门禁）在 CONTRIBUTING 中完整可执行。
- **依赖：** T1.1.1.1.1。
- **预估：** 0.5d

#### T1.1.1.2.2 安全策略与漏洞报告
- **目标：** SECURITY.md（报告渠道、支持版本、响应预期）。
- **产出：** `SECURITY.md`；GitHub Security tab 开启私有漏洞报告。
- **步骤：** 1) 起草（渠道用 GitHub private vulnerability reporting）；2) 声明 v1.2 为首个支持版本。
- **验收标准：** 私有漏洞报告入口可用；SECURITY.md 与之一致。
- **依赖：** T1.1.1.2.1。
- **预估：** 0.5d

#### T1.1.1.2.3 CODEOWNERS 与 PR/Issue 模板
- **目标：** 评审路由与标准化协作入口。
- **产出：** `CODEOWNERS`、`.github/PULL_REQUEST_TEMPLATE.md`、`.github/ISSUE_TEMPLATE/`（bug/feature）。
- **步骤：** 1) CODEOWNERS 指向初始维护者；2) PR 模板含 Story/Task ID、测试证据、checklist；3) Issue 模板含复现步骤与环境。
- **验收标准：** 新 PR/Issue 自动带出模板；CODEOWNERS 对 `docs/`、`internal/` 自动请求 review。
- **依赖：** T1.1.1.2.1。
- **预估：** 0.5d

## Story DoD

- [ ] 治理文件齐备，main 分支保护（CI 绿 + ≥1 review）生效。
- [ ] 模板试用一次（任意初始 PR）验证可用。

## 设计依据

overview/03 §5（分支与提交约定）；overview/04（multica 工作流）。
