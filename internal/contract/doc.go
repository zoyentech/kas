// Package contract 提供领域接口的可复用合规测试套件：实现方在自己的
// _test.go 中一行接入（见本包示例），验证 EvidenceProvider / DecisionProvider
// / EvidenceCache 对超时、取消、并发与契约错误的语义一致。
// 设计依据：docs/plan/overview/03-wbs-conventions.md §6（契约测试层）。
package contract
