# Story 2.2.2.2 — 原子替换语义

> 状态：todo ｜ 作为宿主，我希望 Reload 要么整体生效要么完全不生效，以便配置发布可预期、可回退。

## 前置依赖

- Story 2.2.2.1。

## 任务

#### T2.2.2.2.1 current pointer 原子替换
- **目标：** `atomic.Pointer` 保存 current snapshot；提供 `Load()`（请求开始读一次）。
- **产出：** `internal/runtime/manager.go`。
- **步骤：** 1) Store/Load 封装；2) 注释固化"请求开始读取一次并全程持有"；3) 单线程替换/读取正确性测试。
- **验收标准：** 替换瞬间并发 Load 只见旧或新二者之一，绝无混合。
- **依赖：** T2.2.2.1.1。
- **预估：** 0.5d

#### T2.2.2.2.2 Reload 五步流程
- **目标：** `Client.Reload(ctx, r)`：解析 → Validate → 准备 → 版本 → 原子替换；任一步失败保持旧值并返回错误。
- **产出：** 根包 client.go Reload；`internal/runtime/reload.go`。
- **步骤：** 1) 复用 LoadConfig/ValidateConfig/构建器；2) 每步错误原样透出（ErrInvalidConfig 分类）；3) 成功返回新 config_version。
- **验收标准：** docs/06 §5 五步顺序注释化；无效配置 Reload 后行为与版本完全不变。
- **依赖：** T2.2.2.2.1。
- **预估：** 1d

#### T2.2.2.2.3 在途/新请求快照并发测试
- **目标：** 证明"在途持旧、新请求读新"（docs/05 矩阵 Reload 并发行）。
- **产出：** `internal/runtime/reload_concurrent_test.go`。
- **步骤：** 1) 慢 provider 模拟长请求，中途 Reload；2) 断言在途 Response.ConfigVersion=旧；新请求=新；3) `-race` 下反复跑；4) 无效 Reload 并发混入。
- **验收标准：** 千次迭代无版本串线；无效 Reload 全部失败且版本不变。
- **依赖：** T2.2.2.2.2。
- **预估：** 1d

## Story DoD

- [ ] F-011 验收完成；测试进入 CI 常规集。

## 设计依据

docs/01 §6（发布顺序/失败保持）；docs/05 §2（Reload 并发两行矩阵）。
