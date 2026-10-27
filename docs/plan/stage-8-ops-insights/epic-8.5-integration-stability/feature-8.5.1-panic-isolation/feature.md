# Feature 8.5.1 — provider panic 隔离

> 状态：done ｜ Epic 8.5 ｜ 关联：F-019 ｜ 粗估：1 人日

`deadline.Call` 的 provider goroutine 增加 recover：panic 转ErrProviderContract
（error_class=panic 事件），请求按降级路径继续，宿主进程存活。**Stage 8 内最先做。**
