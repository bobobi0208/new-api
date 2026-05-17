package reconciliation

import (
	"context"
	"fmt"
	"net/url"
)

type sub2ApiAdapter struct{}

func (sub2ApiAdapter) UpstreamType() UpstreamType {
	return UpstreamTypeSub2Api
}

// FetchSnapshot 调用上游 sub2api 的 GET /v1/usage 接口。
// 详细字段参见上游 internal/handler/gateway_handler.go: Usage / usageQuotaLimited / usageUnrestricted。
//
// 兼容两种 mode：
//
//	quota_limited (apiKey.Quota > 0 或有 rate limits):
//	  {
//	    "mode": "quota_limited",
//	    "quota": {"limit": <USD>, "used": <USD>, "remaining": <USD>, "unit": "USD"},
//	    "remaining": <USD>, "unit": "USD",
//	    "rate_limits": [...], "expires_at": ..., "usage": {...}, "model_stats": [...]
//	  }
//
//	unrestricted (订阅模式):
//	  {
//	    "mode": "unrestricted",
//	    "planName": "...",
//	    "remaining": <USD or null>,
//	    "subscription": {"daily_usage_usd": ..., "weekly_usage_usd": ..., "monthly_usage_usd": ..., ...},
//	    "usage": {"today": {"cost": ...}, "total": {"cost": ...}}, "model_stats": [...]
//	  }
//
// 当 AdapterContext 提供 WindowStart/End 时，附加 start_date/end_date query 参数让上游做窗口聚合。
func (sub2ApiAdapter) FetchSnapshot(ctx context.Context, ac AdapterContext) (*UpstreamSnapshot, error) {
	if ac.BaseURL == "" {
		return nil, fmt.Errorf("base_url required for sub2api adapter")
	}
	if ac.ApiKey == "" {
		return nil, fmt.Errorf("api_key required for sub2api adapter")
	}

	endpoint := joinURL(ac.BaseURL, "/v1/usage")
	if !ac.WindowStart.IsZero() && !ac.WindowEnd.IsZero() {
		q := url.Values{}
		q.Set("start_date", ac.WindowStart.Format("2006-01-02"))
		q.Set("end_date", ac.WindowEnd.Format("2006-01-02"))
		endpoint = endpoint + "?" + q.Encode()
	}

	raw, parsed, status, err := httpGetJSON(ctx, endpoint, ac.ApiKey, ac.HttpTimeout)
	if err != nil {
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

	snap := &UpstreamSnapshot{
		RawJSON:     raw,
		WindowStart: ac.WindowStart,
		WindowEnd:   ac.WindowEnd,
	}

	mode, _ := parsed["mode"].(string)

	// 顶层 quota 块（quota_limited 模式）
	if q := asMap(parsed, "quota"); q != nil {
		used, _ := asFloat(q, "used")
		remaining, _ := asFloat(q, "remaining")
		snap.UsedUSD = used
		snap.RemainUSD = remaining
	}

	// usage 块（两种模式都有）
	if usage := asMap(parsed, "usage"); usage != nil {
		if today := asMap(usage, "today"); today != nil {
			if cost, ok := asFloat(today, "cost"); ok {
				snap.TodayCostUSD = cost
			}
		}
		// 在 unrestricted 模式下没有 quota.used，用 usage.total.cost 当 used
		if snap.UsedUSD == 0 {
			if total := asMap(usage, "total"); total != nil {
				if cost, ok := asFloat(total, "cost"); ok {
					snap.UsedUSD = cost
				}
			}
		}
	}

	// 当带了窗口参数：上游 usage.today.cost 在窗口模式下实际代表窗口聚合（参见 parseUsageDateRange）
	if !ac.WindowStart.IsZero() && !ac.WindowEnd.IsZero() {
		snap.WindowCostUSD = snap.TodayCostUSD
	}

	// 订阅模式补充信息
	if mode == "unrestricted" {
		if sub := asMap(parsed, "subscription"); sub != nil {
			// 把当日/周/月已用记到 message，便于详情页显示
			daily, _ := asFloat(sub, "daily_usage_usd")
			snap.Message = fmt.Sprintf("subscription mode; daily_usage=%.6f USD", daily)
		}
	}

	// 没有任何可用字段视为不可对账
	if snap.UsedUSD == 0 && snap.RemainUSD == 0 && snap.TodayCostUSD == 0 && snap.WindowCostUSD == 0 {
		snap.Inconclusive = true
		if snap.Message == "" {
			snap.Message = "no usable fields in response"
		}
	}

	if expiresAt, ok := asFloat(parsed, "expires_at"); ok {
		snap.ExpiresAt = int64(expiresAt)
	}

	return snap, nil
}
