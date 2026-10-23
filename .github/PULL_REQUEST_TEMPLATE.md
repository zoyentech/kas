<!-- 关联 Story / Task ID（必填，格式如 4.3.2.1 / T4.3.2.1.2） -->
- Story/Task ID:
- 关联需求（F-xxx，如有）:

## 变更说明

<!-- 做了什么、为什么；实现与设计文件冲突时先提 ADR（见 docs/adr/README.md） -->

## 测试证据

<!-- 粘贴 make test-race / lint 输出摘要；新逻辑须含单测，安全路径覆盖 ≥ 90% -->

- [ ] `make build` 通过
- [ ] `make test-race` 通过
- [ ] `make lint` 通过（或说明本地不可用、仅 CI 验证）
- [ ] 新增/变更逻辑有对应测试

## 安全自查

- [ ] 无 secret / 问题正文 / 身份明文进入日志、指标、trace 或错误文本
- [ ] 缓存 key 变更（如有）包含 scope/epoch/版本隔离维度
- [ ] 公共 API 变更已更新 godoc 注释
