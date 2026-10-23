// Package deadline 实现五阶段 deadline 引擎（docs/design/04 §1 context 规则）：
// root = min(caller deadline, start+total)；每阶段只从 root 派生
// context.WithDeadline，阶段结束 cancel 后再从 root 派生下一阶段；
// 绝不从上一阶段的短 deadline context 派生（编译期经 Root 类型防护）。
package deadline
