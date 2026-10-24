# Story 2.2.1.1 — 规范化与版本哈希

> 状态：done ｜ 作为运维，我希望 config_version 只随真实配置变化而变化，以便 trace/缓存/Reload 都以它对齐。

## 前置依赖

- Epic 2.1（Config 类型就绪）。

## 任务

#### T2.2.1.1.1 规范化序列化
- **目标：** 配置 → 稳定字节串（键序稳定、默认值折叠规则固定）。
- **产出：** `internal/config/canonical.go`。
- **步骤：** 1) 定义规范化结构（字段排序、duration 纳秒整数、slice 排序仅限无序集合并注明）；2) Marshal 为确定性 JSON/YAML；3) 同配置两份 YAML（字段顺序不同）规范化结果逐字节相等。
- **验收标准：** 顺序无关性测试通过；任何影响行为的字段变更都改变规范化输出。
- **依赖：** 无。
- **预估：** 1d

#### T2.2.1.1.2 secret 排除与 revision 参与
- **目标：** secret 明文排除；引用名/revision 参与。
- **产出：** canonical 的 secret 处理规则 + 属性测试。
- **步骤：** 1) env 引用名（如 KAS_JEV_API_KEY）参与；2) 模拟注入 secret 值的配置变体证明版本不变；3) 引用名变化证明版本变化。
- **验收标准：** 属性测试覆盖"明文变化→版本不变；引用变化→版本变"。
- **依赖：** T2.2.1.1.1。
- **预估：** 0.5d

#### T2.2.1.1.3 版本生成 API
- **目标：** `ConfigVersion(cfg) string`（SHA-256 前 12 hex）。
- **产出：** `internal/config/version.go`。
- **步骤：** 1) 哈希函数封装；2) godoc 说明口径（docs/01 §6）；3) 与 S4 cache key、S6 trace 的消费方式写入接口注释。
- **验收标准：** 幂等（同配置同串）；跨进程稳定（不依赖 map 遍历序/时间）。
- **依赖：** T2.2.1.1.2。
- **预估：** 0.5d

## Story DoD

- [x] 三项任务合入；属性测试（顺序无关/secret 排除/幂等）进入 CI。

## 设计依据

docs/01 §6（config_version 口径）；docs/03 §5（Reload 生成脱敏版本）。
