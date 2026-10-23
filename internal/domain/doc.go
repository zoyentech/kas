// Package domain 定义 KAS 全部公共领域类型与哨兵错误（docs/design/03）。
// 根包 kas 以类型别名暴露本包；internal 各实现包面向本包编程，避免
// 根包↔internal 的循环依赖。
package domain
