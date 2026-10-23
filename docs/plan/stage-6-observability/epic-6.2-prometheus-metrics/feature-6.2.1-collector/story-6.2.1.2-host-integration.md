# Story 6.2.1.2 — 宿主接入与增量断言

> 状态：done ǀ 作为宿主开发者，我希望指标注册即用且可测，以便零成本接入现有监控。

## 前置依赖

- Story 6.2.1.1。

## 任务

#### T6.2.1.2.1 Registerer 注入与生命周期
- **目标：** Option 注入 prometheus.Registerer；重复注册/注销（Close）行为定义。
- **产出：** 根包 Option `WithPrometheusRegisterer`；collector 注册/注销。
- **步骤：** 1) 注册到宿主 registry（默认不注册）；2) 冲突（已注册同名）报 ErrInvalidConfig；3) Close 注销自有 collector；4) godoc 示例。
- **验收标准：** 注入→采集→注销全流程用例；无 listener/无独立 server。
- **依赖：** T6.2.1.1.3。
- **预估：** 1d

#### T6.2.1.2.2 端到端指标增量断言
- **目标：** 用 fake 全链驱动请求（成功/超时/cache hit-miss/fallback/过滤），断言各指标增量正确。
- **产出：** `internal/obs/metrics/e2e_test.go`。
- **步骤：** 1) 场景化驱动（复用 S4/S5 设施）；2) `testutil.CollectAndCompare` 断言增量与 label；3) 覆盖 docs/05 §2"Metrics …增量正确，无敏感/高基数 label"行。
- **验收标准：** 场景增量全对；输出样本经敏感值扫描。
- **依赖：** T6.2.1.2.1；S4/S5 设施。
- **预估：** 1d

## Story DoD

- [ ] F-006 验收证据留档；宿主接入示例写入 godoc。

## 设计依据

docs/04 §5（"SDK 接收宿主 Registerer"）；docs/05 §2 Metrics 行。
