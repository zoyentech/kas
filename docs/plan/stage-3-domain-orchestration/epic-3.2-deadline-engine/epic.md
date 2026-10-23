# Epic 3.2 — 五阶段 deadline 引擎

> 状态：todo ｜ Stage 3 ｜ 关联需求：F-002（P0）｜ 粗估：3 人日

## 目标与价值

实现 docs/04 §1 的 context 规则引擎：root = min(caller, start+total)；每阶段从 root 独立派生并按阶段预算截断；阶段结束即 cancel；绝不从上一阶段 context 派生；provider transport timeout 不晚于阶段 deadline。这是"不超 root deadline"这一 P0 安全不变式的机械保障。

## 验收（Epic DoD）

- F-002：注入阻塞 provider 的测试证明取消传播及时（≤50ms 误差口径，docs/05 §3）。
- "禁止阶段间派生"有编译期/运行期防护（阶段 context 只接收 root）。
- goleak 无 context 泄漏。

## Features

| Feature | 名称 | Stories |
|---|---|---|
| 3.2.1 | context deadline 规则 | 3.2.1.1 deadline 计算；3.2.1.2 provider ctx 合规防护 |

## 依赖

- Epic 3.1（类型）；S2（TimeoutBudget）。

## 关联设计

docs/04 §1（context 规则、代码样例、启动校验）；docs/05 §2（provider 阻塞矩阵行）。
