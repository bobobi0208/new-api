package reconciliation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// RunOptions 描述一次对账执行的输入。
type RunOptions struct {
	ChannelId int
	RunType   string // model.ReconciliationRunTypeBalance / Daily / Manual
	// 可选时间窗口；balance_snapshot 模式留零值
	WindowStart time.Time
	WindowEnd   time.Time
}

// RunOnce 在指定 channel 上执行一次对账，写入 ReconciliationRecord 并返回。
// 不抛出 panic；任何错误（网络、解析、DB）都包装到 ReconciliationRecord.Status=error 写入后再 return。
func RunOnce(ctx context.Context, opts RunOptions) (*model.ReconciliationRecord, error) {
	if opts.ChannelId == 0 {
		return nil, errors.New("channel_id is required")
	}
	if opts.RunType == "" {
		opts.RunType = model.ReconciliationRunTypeManual
	}

	cfg, err := model.GetReconciliationChannelConfig(opts.ChannelId)
	if err != nil {
		return nil, fmt.Errorf("load reconciliation config: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("channel %d is not configured for reconciliation", opts.ChannelId)
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("channel %d reconciliation disabled", opts.ChannelId)
	}
	if !IsKnownUpstream(cfg.UpstreamType) {
		return nil, fmt.Errorf("channel %d has unknown upstream_type=%q", opts.ChannelId, cfg.UpstreamType)
	}

	channel, err := model.GetChannelById(opts.ChannelId, true)
	if err != nil {
		return nil, fmt.Errorf("load channel: %w", err)
	}

	now := time.Now()
	record := &model.ReconciliationRecord{
		ChannelId:    opts.ChannelId,
		UpstreamType: cfg.UpstreamType,
		RunType:      opts.RunType,
		RunAt:        now.Unix(),
		WindowStart:  unixOrZero(opts.WindowStart),
		WindowEnd:    unixOrZero(opts.WindowEnd),
	}

	setting := operation_setting.GetReconciliationSetting()
	timeout := time.Duration(setting.SanitizedHttpTimeout()) * time.Second

	baseURL := cfg.BaseURL
	if baseURL == "" && channel.BaseURL != nil {
		baseURL = *channel.BaseURL
	}
	apiKey := pickAdapterKey(channel)
	isMultiKey := channel.ChannelInfo.IsMultiKey

	adapter := ResolveAdapter(cfg.UpstreamType)
	if adapter == nil {
		record.Status = model.ReconciliationStatusError
		record.Message = "no adapter for upstream_type"
		if err := model.CreateReconciliationRecord(record); err != nil {
			return nil, err
		}
		return record, nil
	}

	ac := AdapterContext{
		BaseURL:     baseURL,
		ApiKey:      apiKey,
		HttpTimeout: timeout,
		WindowStart: opts.WindowStart,
		WindowEnd:   opts.WindowEnd,
		IsMultiKey:  isMultiKey,
	}

	snapshot, err := adapter.FetchSnapshot(ctx, ac)
	if err != nil {
		record.Status = model.ReconciliationStatusError
		record.Message = truncateMessage("fetch upstream: " + err.Error())
		if e := model.CreateReconciliationRecord(record); e != nil {
			return nil, e
		}
		return record, nil
	}
	record.UpstreamUsedUSD = snapshot.UsedUSD
	record.UpstreamRemainUSD = snapshot.RemainUSD
	record.UpstreamRawJSON = truncateRaw(snapshot.RawJSON)

	if snapshot.Inconclusive {
		record.Status = model.ReconciliationStatusInconclusive
		record.Message = truncateMessage(snapshot.Message)
		if e := model.CreateReconciliationRecord(record); e != nil {
			return nil, e
		}
		return record, nil
	}

	localUsedUSD, err := computeLocalUsedUSD(opts, channel)
	if err != nil {
		record.Status = model.ReconciliationStatusError
		record.Message = truncateMessage("local sum: " + err.Error())
		if e := model.CreateReconciliationRecord(record); e != nil {
			return nil, e
		}
		return record, nil
	}
	record.LocalUsedUSD = localUsedUSD

	// 对账维度：窗口模式下用 snapshot.WindowCostUSD (sub2api) 与 localUsedUSD 比对；
	// 非窗口模式下用 snapshot.UsedUSD 累计 vs localUsedUSD 累计。
	upstreamForCompare := snapshot.UsedUSD
	if opts.RunType == model.ReconciliationRunTypeDaily && snapshot.WindowCostUSD > 0 {
		upstreamForCompare = snapshot.WindowCostUSD
	}

	delta := upstreamForCompare - localUsedUSD
	record.DeltaUSD = delta
	record.DeltaRel = SafeDeltaRel(delta, upstreamForCompare)

	absDelta := delta
	if absDelta < 0 {
		absDelta = -absDelta
	}
	if absDelta <= setting.AbsThresholdUSD || record.DeltaRel <= setting.RelThreshold {
		record.Status = model.ReconciliationStatusMatch
	} else {
		record.Status = model.ReconciliationStatusMismatch
		record.Message = truncateMessage(fmt.Sprintf("delta=%.6f USD, rel=%.4f", delta, record.DeltaRel))
	}
	if snapshot.Message != "" && record.Message == "" {
		record.Message = truncateMessage(snapshot.Message)
	}

	if e := model.CreateReconciliationRecord(record); e != nil {
		return nil, e
	}
	return record, nil
}

// computeLocalUsedUSD 根据 RunType 决定本地"已用"金额取值方式。
//   - balance_snapshot：用 Channel.UsedQuota 累计（cheap，no scan）
//   - daily_diff / manual+window：扫 logs 表按窗口聚合
//   - manual without window：fallback 到累计 UsedQuota
func computeLocalUsedUSD(opts RunOptions, channel *model.Channel) (float64, error) {
	if opts.RunType == model.ReconciliationRunTypeBalance || (opts.WindowStart.IsZero() && opts.WindowEnd.IsZero()) {
		return QuotaToUSD(float64(channel.UsedQuota)), nil
	}
	startSec := unixOrZero(opts.WindowStart)
	endSec := unixOrZero(opts.WindowEnd)
	sum, err := model.SumChannelConsumeQuota(channel.Id, startSec, endSec)
	if err != nil {
		return 0, err
	}
	return QuotaToUSD(float64(sum)), nil
}

func pickAdapterKey(channel *model.Channel) string {
	// 多 key 渠道在 Adapter 中会被标记 inconclusive；仍传入第一个 key 让 Adapter 至少能尝试，
	// 失败时也由 Adapter 自己写错误。这里只做"非空"兜底。
	if channel.Key != "" {
		// 多 key 模式下 channel.Key 是换行分隔的 key 列表，取第一行。
		for i := 0; i < len(channel.Key); i++ {
			if channel.Key[i] == '\n' {
				return channel.Key[:i]
			}
		}
		return channel.Key
	}
	return ""
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

const (
	maxMessageLen = 480 // 留余量给 varchar(512)
	maxRawJSONLen = 16 * 1024
)

func truncateMessage(s string) string {
	if len(s) <= maxMessageLen {
		return s
	}
	return s[:maxMessageLen]
}

func truncateRaw(s string) string {
	if len(s) <= maxRawJSONLen {
		return s
	}
	return s[:maxRawJSONLen]
}
