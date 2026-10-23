# Story 2.1.1.2 — Load/Validate API

> 状态：done ｜ 作为宿主开发者，我希望 LoadConfig/LoadConfigFile/ValidateConfig 行为与 docs/03 §1 一致，以便嵌入即用。

## 前置依赖

- T2.1.1.1.1~T2.1.1.1.3。

## 任务

#### T2.1.1.2.1 LoadConfig(io.Reader) / LoadConfigFile(path)
- **目标：** 公共加载入口（解析+基础校验+错误脱敏）。
- **产出：** 根包 `config.go` 导出两个函数；`internal/config/load.go` 实现。
- **步骤：** 1) Reader 版（供 Reload 复用）；2) File 版包装；3) 错误统一为 ErrInvalidConfig 包装链。
- **验收标准：** docs/02 示例加载成功；损坏 YAML/IO 错误分类返回；错误文本无 secret、无整份配置回显。
- **依赖：** Story 2.1.1.1。
- **预估：** 0.5d

#### T2.1.1.2.2 ValidateConfig 规则框架
- **目标：** 校验框架：规则注册、逐条执行、聚合错误（含全部违规而非首条）。
- **产出：** `internal/config/validate.go` + 规则表。
- **步骤：** 1) 规则接口（返回字段路径+问题）；2) 聚合为多行错误；3) 规则清单映射 docs/04 §1"拒绝"项并留追溯注释。
- **验收标准：** 一次输入多条违规时错误逐条列出；每条规则至少一个正/反用例。
- **依赖：** T2.1.1.2.1。
- **预估：** 1d

#### T2.1.1.2.3 错误脱敏与字段路径规范
- **目标：** 固化错误信息规范（供全部 Stage 复用）。
- **产出：** `internal/config/errors.go`（错误构造 helper）+ 规范测试。
- **步骤：** 1) 统一格式 `kas/config: <path>: <reason>`；2) 测试断言错误串不含 env 值/secret/整份 YAML；3) 规范写入包 doc.go。
- **验收标准：** 全部配置错误走同一构造器；脱敏断言进 CI。
- **依赖：** T2.1.1.2.2。
- **预估：** 0.5d

## Story DoD

- [ ] 三个 API 可用；示例配置/非法配置用例齐备（golden 错误信息测试）。

## 设计依据

docs/03 §1（API 签名、New 同步校验职责、密钥不写入 YAML 或错误文本）。
