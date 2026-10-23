# Feature 6.3.1 — TraceSink 与字段

> 状态：todo ｜ Epic 6.3 ｜ 粗估：2 人日

## 目标

TraceSink 实现 + 事件→trace 记录 + 脱敏器 + TraceID 贯通。

## 设计要点

- 默认 sink：结构化日志或内存（测试用）；宿主可注入自定义（OTel 适配 post-MVP）。
- 字段过滤：include 列表驱动；capture_question_text/evidence_text 默认 false 且 v1.2 不实现 true 路径（设计文件口径：诊断 query 仅经批准用不可逆 hash，本版不做）。
- 脱敏器是唯一出口，任何新字段必须过白名单。

## Stories

| Story | 名称 | 任务数 |
|---|---|---:|
| 6.3.1.1 | trace 实现 | 4 |

## 验收标准

- 脱敏扫描测试：默认配置下 trace 输出不含问题/Evidence/身份/密钥子串。

## 需求关联

F-009（P0）。
