# Story 8.4.1.1 — 仪表盘页面骨架与数据层

> 状态：done ｜ 作为本机用户，我期望浏览器打开即见运行全貌并自动刷新，以便零学习成本排障。

## 前置依赖

- Story 8.3.1.2（/api/overview schema）。

## 任务

#### T8.4.1.1.1 页面骨架与轮询
- **目标：** 单文件 index.html（深色单列布局；顶部总状态卡：health/config_version/
  uptime/QPS/成功率；数据层 fetch /api/overview + 5s 轮询 + 失联横幅）。
- **产出：** `internal/diag/static/index.html` + embed 接线。
- **步骤：** 1) 原生 fetch/setInterval；2) 断连显示"数据停更"横幅（fetch 失败）；
  3) 无任何外部引用（审查无 http(s) 资源）。
- **验收标准：** httptest 断言 HTML 含关键容器与零外链；Close 后页面显示停更横幅（JS 逻辑单测难，
  以端到端手工+冒烟断言 JSON 停更代替）。
- **依赖：** 无。
- **预估：** 1d

## Story DoD

- [x] 骨架合入；数据层与横幅逻辑可演示。

## 设计依据

ADR-0003；stage-8 §范围外（无鉴权声明展示于页脚）。
