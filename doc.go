// Package kas 是 KAS（Knowledge Answer Service）v1.2 的公共 API 层。
//
// KAS 是可嵌入宿主 Go 服务的知识问答运行时 SDK：统一请求预算、Evidence
// 检索与知识有效期筛选、决策 provider、带引用回答、安全缓存与配置热替换。
//
// 包布局（D-7）：本包只暴露公共 API；全部实现在 internal/ 下，由 New 装配。
// 完整 API 草案见 docs/design/03-go-sdk-api.md。
package kas
