package service

import (
	"errors"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

const (
	overviewWindowSeconds int64 = 30 * 60
)

// ResolveAvailabilityRange maps a "30m" / "1d" / "7d" range key to (bucketSec,
// windowSec). Unknown keys fall back to 30m.
func ResolveAvailabilityRange(rangeKey string) (bucketSec int64, windowSec int64, normalized string) {
	switch rangeKey {
	case "1d", "24h":
		return 30 * 60, 24 * 3600, "1d"
	case "7d", "week":
		return 2 * 3600, 7 * 24 * 3600, "7d"
	default:
		return 60, 30 * 60, "30m"
	}
}

// GetGroupAvailabilityOverview returns one snapshot per known group, scoped
// to the last 30 minutes. Groups with channels but zero traffic still appear
// (success_rate=nil); groups with traffic but no enabled channels also appear
// (channel_total=0).
func GetGroupAvailabilityOverview() (dto.GroupAvailabilityOverview, error) {
	now := time.Now().Unix()
	start := now - overviewWindowSeconds

	channelMap, err := model.ChannelOnlineByGroup()
	if err != nil {
		return dto.GroupAvailabilityOverview{}, err
	}
	logMap, err := model.LogAggregateByGroup(start, now)
	if err != nil {
		return dto.GroupAvailabilityOverview{}, err
	}

	groupSet := make(map[string]struct{}, len(channelMap)+len(logMap))
	for g := range channelMap {
		groupSet[g] = struct{}{}
	}
	for g := range logMap {
		groupSet[g] = struct{}{}
	}
	groups := make([]string, 0, len(groupSet))
	for g := range groupSet {
		groups = append(groups, g)
	}
	sort.Strings(groups)

	snapshots := make([]dto.GroupAvailabilitySnapshot, 0, len(groups))
	for _, g := range groups {
		ch := channelMap[g]
		lg := logMap[g]
		s := dto.GroupAvailabilitySnapshot{
			Group:         g,
			OnlineRate:    onlineRate(ch.Online, ch.Total),
			RequestCount:  lg.Success + lg.Error,
			ErrorCount:    lg.Error,
			ChannelTotal:  ch.Total,
			ChannelOnline: ch.Online,
		}
		s.SuccessRate = successRate(lg.Success, lg.Error)
		s.AvgLatencyMs = avgLatencyMs(lg.SumUse, lg.Success+lg.Error)
		snapshots = append(snapshots, s)
	}

	// Default sort: highest traffic first, then alphabetical.
	sort.SliceStable(snapshots, func(i, j int) bool {
		if snapshots[i].RequestCount != snapshots[j].RequestCount {
			return snapshots[i].RequestCount > snapshots[j].RequestCount
		}
		return snapshots[i].Group < snapshots[j].Group
	})

	return dto.GroupAvailabilityOverview{
		Groups:        snapshots,
		WindowSeconds: overviewWindowSeconds,
		GeneratedAt:   now,
	}, nil
}

// GetGroupAvailabilityTimeseries returns a per-bucket time series plus channel
// breakdown for one group and time range.
func GetGroupAvailabilityTimeseries(group string, rangeKey string) (dto.GroupAvailabilityTimeseries, error) {
	if group == "" {
		return dto.GroupAvailabilityTimeseries{}, errors.New("group is required")
	}
	bucketSec, windowSec, normalized := ResolveAvailabilityRange(rangeKey)
	now := time.Now().Unix()
	start := now - windowSec

	rawBuckets, err := model.GetGroupTimeseries(group, start, now, bucketSec)
	if err != nil {
		return dto.GroupAvailabilityTimeseries{}, err
	}
	channelAgg, err := model.GetGroupChannelAggregate(group, start, now)
	if err != nil {
		return dto.GroupAvailabilityTimeseries{}, err
	}
	channelMetas, err := model.GetChannelsInGroup(group)
	if err != nil {
		return dto.GroupAvailabilityTimeseries{}, err
	}

	points := pivotBuckets(rawBuckets, start, now, bucketSec)

	channels := mergeChannelRows(channelMetas, channelAgg)

	var totalSuccess, totalError, totalUse int64
	for _, r := range rawBuckets {
		switch r.Type {
		case model.LogTypeConsume:
			totalSuccess += r.Cnt
		case model.LogTypeError:
			totalError += r.Cnt
		}
		totalUse += r.SumUse
	}

	out := dto.GroupAvailabilityTimeseries{
		Group:        group,
		Range:        normalized,
		BucketSec:    bucketSec,
		StartSec:     start,
		EndSec:       now,
		Points:       points,
		SuccessRate:  successRate(totalSuccess, totalError),
		RequestCount: totalSuccess + totalError,
		ErrorCount:   totalError,
		AvgLatencyMs: avgLatencyMs(totalUse, totalSuccess+totalError),
		Channels:     channels,
	}

	for _, m := range channelMetas {
		out.ChannelTotal++
		if isOnline(m.Status) {
			out.ChannelOnline++
		}
	}

	// Percentiles are only computed for the 30m window — sampling 2k rows over
	// 1d or 7d is unrepresentative and the larger windows already have AVG.
	if normalized == "30m" {
		samples, err := model.GetLatencySamples(group, start, now, 2000)
		if err != nil {
			return dto.GroupAvailabilityTimeseries{}, err
		}
		out.P50Ms = model.PercentileFromSorted(samples, 50)
		out.P95Ms = model.PercentileFromSorted(samples, 95)
	}

	return out, nil
}

func pivotBuckets(rows []model.AvailabilityBucketRow, startSec, endSec, bucketSec int64) []dto.AvailabilityBucket {
	if bucketSec <= 0 {
		bucketSec = 60
	}
	// Anchor the bucket grid so that gaps render as zero-bars rather than
	// missing data.
	firstBucket := (startSec / bucketSec) * bucketSec
	count := int((endSec-firstBucket)/bucketSec) + 1
	if count <= 0 {
		return nil
	}
	out := make([]dto.AvailabilityBucket, count)
	for i := range out {
		out[i].Bucket = firstBucket + int64(i)*bucketSec
	}
	idxOf := func(b int64) int {
		i := int((b - firstBucket) / bucketSec)
		if i < 0 || i >= count {
			return -1
		}
		return i
	}
	useSum := make([]int64, count)
	useTotal := make([]int64, count)
	for _, r := range rows {
		i := idxOf(r.Bucket)
		if i < 0 {
			continue
		}
		switch r.Type {
		case model.LogTypeConsume:
			out[i].SuccessCount += r.Cnt
		case model.LogTypeError:
			out[i].ErrorCount += r.Cnt
		}
		useSum[i] += r.SumUse
		useTotal[i] += r.Cnt
	}
	for i := range out {
		if useTotal[i] > 0 {
			out[i].AvgLatencyMs = int(useSum[i] / useTotal[i])
		}
	}
	return out
}

func mergeChannelRows(metas []model.ChannelMeta, agg []model.ChannelAggregateRow) []dto.ChannelAvailabilityRow {
	type acc struct {
		Success int64
		Error   int64
	}
	bucketByCh := make(map[int]acc, len(agg))
	for _, a := range agg {
		entry := bucketByCh[a.ChannelId]
		switch a.Type {
		case model.LogTypeConsume:
			entry.Success += a.Cnt
		case model.LogTypeError:
			entry.Error += a.Cnt
		}
		bucketByCh[a.ChannelId] = entry
	}
	// Channels surface even if the abilities table doesn't have them anymore
	// but logs reference them (e.g. just-deleted channel).
	known := make(map[int]struct{}, len(metas))
	out := make([]dto.ChannelAvailabilityRow, 0, len(metas)+len(bucketByCh))
	for _, m := range metas {
		known[m.Id] = struct{}{}
		a := bucketByCh[m.Id]
		out = append(out, dto.ChannelAvailabilityRow{
			ChannelId:    m.Id,
			Name:         m.Name,
			Status:       m.Status,
			ResponseTime: m.ResponseTime,
			TestTime:     m.TestTime,
			RequestCount: a.Success + a.Error,
			ErrorCount:   a.Error,
		})
	}
	for id, a := range bucketByCh {
		if _, ok := known[id]; ok {
			continue
		}
		out = append(out, dto.ChannelAvailabilityRow{
			ChannelId:    id,
			Name:         "",
			Status:       0,
			RequestCount: a.Success + a.Error,
			ErrorCount:   a.Error,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ErrorCount != out[j].ErrorCount {
			return out[i].ErrorCount > out[j].ErrorCount
		}
		if out[i].RequestCount != out[j].RequestCount {
			return out[i].RequestCount > out[j].RequestCount
		}
		return out[i].ChannelId < out[j].ChannelId
	})
	return out
}

func successRate(success, errs int64) *float64 {
	total := success + errs
	if total <= 0 {
		return nil
	}
	r := float64(success) / float64(total)
	return &r
}

func onlineRate(online, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(online) / float64(total)
}

func avgLatencyMs(sumUse int64, total int64) int {
	if total <= 0 {
		return 0
	}
	return int(sumUse / total)
}

func isOnline(status int) bool {
	return status == common.ChannelStatusEnabled
}
