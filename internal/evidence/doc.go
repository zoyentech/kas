// Package evidence 实现 Evidence 链路：有效期过滤（expired/future/unknown
// fail-closed）、权限复核与 EvidenceProvider 参考实现/故障注入设施。
// 设计依据：docs/design/01 §3、docs/design/03 §4（Evidence 合约）。
package evidence
