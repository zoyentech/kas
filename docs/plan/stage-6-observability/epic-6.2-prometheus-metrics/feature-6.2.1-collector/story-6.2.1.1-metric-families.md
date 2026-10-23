# Story 6.2.1.1 — 指标族实现

> 状态：done ǀ 作为运维，我希望指标与设计表一一对应，以便仪表盘与告警按表配置。

## 前置依赖

- S3 事件总线；D-3。

## 任务

#### T6.2.1.1.1 请求与阶段耗时族
- **目标：** requests_total（operation/outcome/action）、request_duration_seconds、stage_duration_seconds（stage/provider/outcome）、两个 deadline_exceeded 计数。
- **产出：** `internal/obs/metrics/request.go`、`stage.go`。
- **步骤：** 1) 按表定义 MetricVec（help 文案引用 docs/04 §5）；2) 事件→指标映射；3) 单测。
- **验收标准：** 每指标 label 集与表完全一致；无多余 label。
- **依赖：** 无。
- **预估：** 1d

#### T6.2.1.1.2 缓存/过滤/错误/降级族
- **目标：** evidence_cache_total、evictions_total、filtered_total、provider_errors_total、decision_fallback_total、health_status。
- **产出：** `internal/obs/metrics/cache.go` 等。
- **步骤：** 1) 同上模式；2) error_class 枚举收敛（timeout/5xx/contract/cancelled…）；3) 单测。
- **验收标准：** reason/route/result 枚举与事件层一致（常量共享包）。
- **依赖：** T6.2.1.1.1。
- **预估：** 1d

#### T6.2.1.1.3 label allowlist 与基数防护
- **目标：** collector 层强制 allowlist；禁入值清单（question/身份/Evidence ID/trace ID/错误原文/config_version/模型版本）机械检查。
- **产出：** `internal/obs/metrics/allowlist.go` + 防护测试。
- **步骤：** 1) 注册期校验 label 名单；2) 写入期校验值域（枚举类 label）；3) 测试注入非法 label 被拒且计数不受影响。
- **验收标准：** docs/04 §5 禁令逐条有防护测试（含"模型版本默认只写 trace"）。
- **依赖：** T6.2.1.1.2。
- **预估：** 1d

## Story DoD

- [ ] 十类指标齐备；allowlist 防护进入 CI。

## 设计依据

docs/04 §5 全表；docs/02 metrics 节。
