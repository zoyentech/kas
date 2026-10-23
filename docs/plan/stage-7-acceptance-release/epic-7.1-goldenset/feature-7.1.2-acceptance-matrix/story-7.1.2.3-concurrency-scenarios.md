# Story 7.1.2.3 — 并发场景套件

> 状态：done ǀ 作为发布负责人，我希望 Reload/并发行为被场景级验证，以便热替换在生产安全。

## 前置依赖

- S2（Reload 语义）、S3（并发基线）、S6（Health）。

## 任务

#### T7.1.2.3.1 Reload 并发与无效 Reload
- **目标：** Reload 进行中：在途 response/trace 为旧版、新请求为新版；无效 Reload 报错且 config_version 不变。
- **产出：** `acceptance/matrix/reload_test.go`（场景级，复用并加压 S2 单元测试）。
- **步骤：** 1) 慢 provider + 高频 Reload 混压；2) 断言每请求版本一致性；3) 无效配置并发 Reload 全失败且版本恒定。
- **验收标准：** docs/05 §2 两行（Reload 并发/Reload 无效）全绿；`-race` 通过。
- **依赖：** 无。
- **预估：** 1d

#### T7.1.2.3.2 混合并发与指标一致
- **目标：** 并发 Answer/Health/Reload 无 snapshot 混用或缓存串线；成功/超时/cache/fallback 指标增量正确。
- **产出：** `acceptance/matrix/mixed_test.go`。
- **步骤：** 1) 混合负载 + 指标采集；2) 指标总量与请求计数对账；3) goleak 干净。
- **验收标准：** docs/05 §2 两行（并发混合/Metrics 增量）全绿。
- **依赖：** T7.1.2.3.1；S6 指标。
- **预估：** 1d

## Story DoD

- [ ] 19 行矩阵 ↔ 测试映射表完成并全部通过。

## 设计依据

docs/05 §2 并发行与 Metrics 行。
