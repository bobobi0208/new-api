// Package reconciliation 实现"上游对账"子系统的核心逻辑。
// 设计原则：
//   - 与现有 controller / model / relay 解耦，所有对账相关的业务逻辑都聚合在本包内；
//   - 通过 Adapter 接口抽象不同上游（new-api / sub2api）的能力差异；
//   - 仅依赖 service.GetHttpClient、common.Marshal/Unmarshal、model.Channel 等已有公共能力，
//     避免对仓库现有文件造成扩散式修改，便于和上游 main 分支保持解耦。
package reconciliation

import (
	"context"
	"time"
)

// UpstreamType 是对账系统识别的上游类别。
type UpstreamType string

const (
	UpstreamTypeNewApi  UpstreamType = "newapi"
	UpstreamTypeSub2Api UpstreamType = "sub2api"
)

// IsKnown 报告该字符串是否对应一个已实现 Adapter 的上游类型。
func IsKnownUpstream(t string) bool {
	switch UpstreamType(t) {
	case UpstreamTypeNewApi, UpstreamTypeSub2Api:
		return true
	}
	return false
}

// UpstreamSnapshot 是 Adapter 拉取后归一化的对账快照。
// 所有金额都以 USD 为单位，跨上游统一。
type UpstreamSnapshot struct {
	// UsedUSD 上游记录的累计已用金额（USD）。
	UsedUSD float64
	// RemainUSD 上游记录的剩余可用金额（USD）。new-api 上游能拿到；sub2api unrestricted 模式可能为 0。
	RemainUSD float64
	// TodayCostUSD 上游今日（其本机时区）已用金额。仅 sub2api 能精确给出；new-api 留 0。
	TodayCostUSD float64
	// WindowCostUSD 上游按请求窗口聚合的已用金额。仅当 Adapter 支持窗口查询时填充。
	WindowCostUSD float64
	// WindowStart / WindowEnd 表示本次拉取的查询窗口。零值表示未指定窗口（拉的是累计快照）。
	WindowStart time.Time
	WindowEnd   time.Time
	// ExpiresAt 上游 token 过期时间；0 表示未知或永不过期。
	ExpiresAt int64
	// UnlimitedQuota 标记上游是否标注"无限额度"（new-api 的 unlimited token）。
	UnlimitedQuota bool
	// Inconclusive 当 Adapter 判定本次结果不能用于对账（多 key 渠道、上游返回了无法解析的格式等）时置 true。
	// 此时 Runner 会将 ReconciliationRecord.Status 标为 inconclusive 而非 mismatch。
	Inconclusive bool
	// Message 给人看的简短说明，写到 ReconciliationRecord.Message 字段。
	Message string
	// RawJSON 上游响应的原始 JSON 字符串，用于排查时还原现场。
	RawJSON string
}

// AdapterContext 携带 Adapter 调用上游所需的运行时参数。
type AdapterContext struct {
	// BaseURL 上游基础 URL（无尾斜杠），如 https://api.example.com。
	BaseURL string
	// ApiKey 用于上游鉴权的 sk-xxx token；Adapter 自行拼到 Authorization 头。
	ApiKey string
	// HttpTimeout 单次请求超时；由 Runner 从 ReconciliationSetting 读取后注入。
	HttpTimeout time.Duration
	// WindowStart / WindowEnd 可选；若 Adapter 支持窗口查询且两个值均非零，则拉取窗口聚合数据。
	WindowStart time.Time
	WindowEnd   time.Time
	// IsMultiKey 标记当前 channel 是否多 key；多 key 时 Adapter 应直接返回 Inconclusive。
	IsMultiKey bool
}

// Adapter 抽象单个上游的对账数据拉取能力。
// 实现需保证：网络/解析失败一律返回 error 而不是空 snapshot，以便 Runner 区分 "error" 和 "inconclusive"。
type Adapter interface {
	UpstreamType() UpstreamType
	FetchSnapshot(ctx context.Context, ac AdapterContext) (*UpstreamSnapshot, error)
}

// ResolveAdapter 根据 upstream_type 字符串返回对应 Adapter；未识别返回 nil。
// 通过这个工厂函数把 Adapter 注册解耦于 Runner 主流程。
func ResolveAdapter(upstreamType string) Adapter {
	switch UpstreamType(upstreamType) {
	case UpstreamTypeNewApi:
		return newApiAdapter{}
	case UpstreamTypeSub2Api:
		return sub2ApiAdapter{}
	}
	return nil
}
