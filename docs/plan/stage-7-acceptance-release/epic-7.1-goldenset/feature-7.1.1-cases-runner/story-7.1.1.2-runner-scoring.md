# Story 7.1.1.2 — runner 与评分

> 状态：done ǀ 作为验收负责人，我希望一键跑全集并产出结构化报告，以便每轮对比可追溯。

## 前置依赖

- Story 7.1.1.1；S4/S5 设施（fake provider/语料装载）。

## 任务

#### T7.1.1.2.1 runner 主体
- **目标：** 固定 corpus generation/授权上下文/valid_at 执行全部用例，收集 Response/trace/指标快照。
- **产出：** `acceptance/cmd/runner/main.go`（或 `make acceptance`）。
- **步骤：** 1) 装载语料到 inmem provider；2) 逐例构造 Request（含 scope/valid_at）；3) 输出原始结果集（JSONL）。
- **验收标准：** 可重复运行结果稳定（除时延）；单例失败不中断全集。
- **依赖：** T7.1.1.1.1。
- **预估：** 1d

#### T7.1.1.2.2 评分器
- **目标：** docs/05 §4 全部指标计算（Evidence Recall@k/Precision@k/MRR/nDCG、回答正确/漏答/无据、引用覆盖/准确/过期率、p50/p95/p99、stage timeout、cache hit、fallback 比例）。
- **产出：** `acceptance/internal/scoring.go`。
- **步骤：** 1) 指标实现 + 单测（手工构造小金集验证公式）；2) P0 子集单独统计；3) 门槛对照（O-3 冻结值）判定 pass/fail。
- **验收标准：** 每个指标有公式单测；P0 与全集分开输出（docs/05 §4"不得只给总体均值"）。
- **依赖：** T7.1.1.2.1。
- **预估：** 1.5d

#### T7.1.1.2.3 报告产物
- **目标：** 每轮归档：GoldenSet 版本、样本数、corpus/config/model version、时间窗口、指标、人工裁定占位、缺陷项。
- **产出：** `acceptance/report/` 模板与生成器；`docs/release-notes/` 归档约定。
- **步骤：** 1) Markdown+JSON 双格式；2) 门槛对照表；3) 与 CI 集成（失败即红）。
- **验收标准：** 报告字段覆盖 docs/05 §4 清单；归档路径写入贡献文档。
- **依赖：** T7.1.1.2.2。
- **预估：** 1d

## Story DoD

- [x] `make acceptance` 端到端可用；首轮报告归档。

## 设计依据

docs/05 §3~§4。
