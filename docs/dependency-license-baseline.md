# 依赖与 License 基线（Stage 1 建立，Stage 7.2.1 复核）

> 基线日期：2026-10-23 ｜ License 检测：模块缓存 LICENSE 文件头 + 上游声明。
> 项目 License：Apache-2.0（D-5）。策略：仅引入 Apache-2.0/MIT/BSD/ISC 等
> 宽松许可证依赖，拒绝 GPL/AGPL 传染性依赖。

## 直接依赖（运行时）

| 模块 | 版本 | License | 用途（决策） |
|---|---|---|---|
| gopkg.in/yaml.v3 | v3.0.1 | MIT & Apache-2.0（双许可） | 配置严格解析（D-2） |

## 直接依赖（计划中，按 Stage 引入）

| 模块 | License | 用途（决策） | 引入 Stage |
|---|---|---|---|
| github.com/prometheus/client_golang | Apache-2.0 | 指标 collector（D-3） | S6 |

## 直接依赖（仅测试）

| 模块 | 版本 | License | 用途（决策） |
|---|---|---|---|
| gotest.tools/v3 | v3.5.1 | Apache-2.0 | 断言最小集（D-4） |
| go.uber.org/goleak | v1.3.0 | MIT | goroutine 泄漏检测（D-9） |

## 间接依赖（当前 go.sum）

| 模块 | License | 引入方 | 阶段 |
|---|---|---|---|
| github.com/stretchr/testify | MIT | gotest.tools/v3 | 仅测试 |
| github.com/davecgh/go-spew | ISC | testify | 仅测试 |
| github.com/pmezard/go-difflib | MIT | testify | 仅测试 |
| gopkg.in/check.v1 | BSD-2-Clause | yaml.v3 | 仅测试 |

## 结论

当前全部依赖为宽松许可证，与 Apache-2.0 兼容，无传染性风险。
新增任何依赖前须更新本表并复核 License（CONTRIBUTING 约定：先提 ADR）。
