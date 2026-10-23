# Story 4.1.1.1 — 契约类型与套件

> 状态：done ｜ 作为宿主，我希望 Evidence 契约明确且可测，以便把自己的检索服务适配进来即可获得 KAS 全部保障。

## 前置依赖

- S3（错误分类、deadline 防护）。

## 任务

#### T4.1.1.1.1 Evidence/Query/Result 类型与接口
- **目标：** 落地 docs/03 §4 全部类型。
- **产出：** 根包 `evidence.go`（或 types_evidence.go）。
- **步骤：** 1) 结构与接口定义；2) godoc 定义 ValidFrom/ValidUntil nil 语义=未知；3) EvidenceQuery 含 namespace/top_k/scope 相关字段（对齐 docs/02 evidence 节）。
- **验收标准：** 类型与 docs/03 §4 字段一一对应；nil 语义写入 godoc 并有测试固化。
- **依赖：** 无。
- **预估：** 1d

#### T4.1.1.1.2 契约测试套件
- **目标：** 可被任意 EvidenceProvider 实现引入的合规套件。
- **产出：** `internal/contract/evidence_test.go`（exported as suite 函数）。
- **步骤：** 1) 用例：正常检索、空结果、error 映射（ErrEvidenceUnavailable）、ctx 取消及时（复用阻塞 harness）、Health 报告结构；2) 文档说明接入方式。
- **验收标准：** inmem 实现接入即跑通；故意违规实现（忽略 ctx）被套件抓住。
- **依赖：** T4.1.1.1.1；T3.2.1.2.2。
- **预估：** 1d

## Story DoD

- [ ] 契约冻结；套件进入 internal/contract 供 Stage 5 与验收复用。

## 设计依据

docs/03 §4；docs/01 §3。
