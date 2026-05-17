package reconciliation

import (
	"context"
	"fmt"
)

type newApiAdapter struct{}

func (newApiAdapter) UpstreamType() UpstreamType {
	return UpstreamTypeNewApi
}

// FetchSnapshot 调用上游 new-api 的 GET /api/usage/token/ 接口。
//
// 上游响应格式（参见上游 controller/token.go:GetTokenUsage）：
//
//	{
//	  "code": true,
//	  "message": "ok",
//	  "data": {
//	    "object": "token_usage",
//	    "name": "...",
//	    "total_granted":   <quota int>,
//	    "total_used":      <quota int>,
//	    "total_available": <quota int>,
//	    "unlimited_quota": <bool>,
//	    "expires_at":      <int64 sec, 0 表示永不过期>
//	  }
//	}
//
// quota 是 new-api 内部整数（500_000 = $1）。本 Adapter 统一换算为 USD。
// 多 key 渠道上游接口本身就拒绝（参见上游 controller/channel-billing.go），Runner 在调用前会过滤；
// 这里再补一道判断作为兜底。
func (a newApiAdapter) FetchSnapshot(ctx context.Context, ac AdapterContext) (*UpstreamSnapshot, error) {
	if ac.IsMultiKey {
		return &UpstreamSnapshot{
			Inconclusive: true,
			Message:      "multi-key channel: upstream usage API not supported",
		}, nil
	}
	if ac.BaseURL == "" {
		return nil, fmt.Errorf("base_url required for newapi adapter")
	}
	if ac.ApiKey == "" {
		return nil, fmt.Errorf("api_key required for newapi adapter")
	}

	url := joinURL(ac.BaseURL, "/api/usage/token/")
	raw, parsed, status, err := httpGetJSON(ctx, url, ac.ApiKey, ac.HttpTimeout)
	if err != nil {
		fallback, ferr := a.fallbackSubscription(ctx, ac)
		if ferr == nil {
			return fallback, nil
		}
		return nil, fmt.Errorf("usage api request failed: %w", err)
	}
	if status >= 400 {
		return &UpstreamSnapshot{
			RawJSON:      raw,
			Inconclusive: true,
			Message:      fmt.Sprintf("upstream returned http %d", status),
		}, nil
	}
	if parsed == nil {
		return &UpstreamSnapshot{
			RawJSON:      raw,
			Inconclusive: true,
			Message:      "empty response body",
		}, nil
	}

	data := asMap(parsed, "data")
	if data == nil {
		return &UpstreamSnapshot{
			RawJSON:      raw,
			Inconclusive: true,
			Message:      "missing data field",
		}, nil
	}

	totalUsed, _ := asFloat(data, "total_used")
	totalAvail, _ := asFloat(data, "total_available")
	unlimited := asBool(data, "unlimited_quota")
	expiresAt := asInt64(data, "expires_at")

	snap := &UpstreamSnapshot{
		UsedUSD:        QuotaToUSD(totalUsed),
		RemainUSD:      QuotaToUSD(totalAvail),
		UnlimitedQuota: unlimited,
		ExpiresAt:      expiresAt,
		RawJSON:        raw,
	}
	if unlimited {
		// 无限额度：remain 没有对账意义，只能比较 used 与本地是否一致
		snap.RemainUSD = 0
		snap.Message = "unlimited_quota=true; remain ignored"
	}
	return snap, nil
}

func (newApiAdapter) fallbackSubscription(ctx context.Context, ac AdapterContext) (*UpstreamSnapshot, error) {
	url := joinURL(ac.BaseURL, "/dashboard/billing/subscription")
	raw, parsed, status, err := httpGetJSON(ctx, url, ac.ApiKey, ac.HttpTimeout)
	if err != nil {
		return nil, err
	}
	if status >= 400 || parsed == nil {
		return nil, fmt.Errorf("subscription fallback http %d", status)
	}
	hard, _ := asFloat(parsed, "hard_limit_usd")
	soft, _ := asFloat(parsed, "soft_limit_usd")
	used := 0.0
	if hard > 0 && soft >= 0 {
		used = hard - soft
	}
	return &UpstreamSnapshot{
		UsedUSD:   used,
		RemainUSD: soft,
		RawJSON:   raw,
		Message:   "fell back to /dashboard/billing/subscription",
	}, nil
}
