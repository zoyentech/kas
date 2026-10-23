# Feature 3.3.1 — 阶段流水线

> 状态：todo ｜ Epic 3.3 ｜ 粗估：3 人日

## 目标

docs/01 §4 端到端流程的骨架实现：每阶段定义为可注入的执行单元，S4/S5 填充实际逻辑。

## 设计要点

- 阶段挂载点：preliminary（初判接口占位）、first_round（Evidence 检索挂载点）、rejudge（决策挂载点）、supplement（补检+重过滤挂载点）、reserve（组装/校验挂载点）。
- 初判 hook 接口：输入 Request+快照，输出路径决策（检索/澄清/abstain）——S5 规则实现。
- 每请求开始读一次 snapshot 并全程持有（docs/01 §3）。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 3.3.1.1 | Answer 主流程 | 3 |
| 3.3.1.2 | 事件总线 | 2 |

## 验收标准

- 骨架可空跑（全部挂载点为 no-op）返回 abstain 占位 Response。
- 阶段顺序/预算消费/事件顺序有断言。

## 需求关联

F-002（消费 deadline 引擎）、F-007（DecisionProvider 挂载形态）。
