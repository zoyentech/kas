// Package orchestrator 实现请求编排、五阶段逐级 deadline、内部事件总线
// （供 obs 消费）与 Client 并发安全生命周期（Answer/Health/Reload/Close）。
// 设计依据：docs/design/01 §3/§4、docs/design/04 §1（context 规则）。
package orchestrator
