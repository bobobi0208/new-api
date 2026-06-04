package model

import (
	"fmt"
	"sort"

	"github.com/QuantumNous/new-api/common"
)

// bucketColExpr returns a cross-database time-bucket expression for an int64
// unix-seconds column. MySQL needs FLOOR(); SQLite and PostgreSQL integer
// division already truncates so a plain divide works.
func bucketColExpr(col string, bucketSize int64) string {
	if bucketSize <= 0 {
		bucketSize = 60
	}
	if common.UsingMySQL {
		return fmt.Sprintf("FLOOR(%s / %d) * %d", col, bucketSize, bucketSize)
	}
	return fmt.Sprintf("(%s / %d) * %d", col, bucketSize, bucketSize)
}

// randExpr returns the SQL random() expression for the current database.
func randExpr() string {
	if common.UsingMySQL {
		return "RAND()"
	}
	return "RANDOM()"
}

type GroupChannelCount struct {
	GroupName string `gorm:"column:group_name"`
	Total     int    `gorm:"column:total"`
	Online    int    `gorm:"column:online"`
}

// ChannelOnlineByGroup returns each group's enabled-channel count and the
// subset that is currently online (channels.status = ChannelStatusEnabled).
func ChannelOnlineByGroup() (map[string]GroupChannelCount, error) {
	enabledExpr := fmt.Sprintf("SUM(CASE WHEN c.status = %d THEN 1 ELSE 0 END) AS online",
		common.ChannelStatusEnabled)
	selectExpr := fmt.Sprintf("a.%s AS group_name, COUNT(DISTINCT a.channel_id) AS total, %s",
		commonGroupCol, enabledExpr)

	var rows []GroupChannelCount
	err := DB.Table("abilities AS a").
		Select(selectExpr).
		Joins("JOIN channels AS c ON c.id = a.channel_id").
		Where("a.enabled = " + commonTrueVal).
		Group("a." + commonGroupCol).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]GroupChannelCount, len(rows))
	for _, r := range rows {
		out[r.GroupName] = r
	}
	return out, nil
}

type GroupLogAggregate struct {
	Success int64
	Error   int64
	SumUse  int64
}

type groupTypeAggRow struct {
	GroupName string `gorm:"column:group_name"`
	Type      int    `gorm:"column:type"`
	Cnt       int64  `gorm:"column:cnt"`
	SumUse    int64  `gorm:"column:sum_use"`
}

// LogAggregateByGroup returns success/error/use-time totals per group inside
// [startSec, endSec). success = Type=Consume, error = Type=Error.
func LogAggregateByGroup(startSec, endSec int64) (map[string]GroupLogAggregate, error) {
	selectExpr := fmt.Sprintf("%s AS group_name, type, COUNT(*) AS cnt, COALESCE(SUM(use_time), 0) AS sum_use", logGroupCol)
	var rows []groupTypeAggRow
	err := LOG_DB.Table("logs").
		Select(selectExpr).
		Where("created_at >= ? AND created_at < ?", startSec, endSec).
		Where("type IN ?", []int{LogTypeConsume, LogTypeError}).
		Group(logGroupCol + ", type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]GroupLogAggregate)
	for _, r := range rows {
		entry := out[r.GroupName]
		switch r.Type {
		case LogTypeConsume:
			entry.Success = r.Cnt
			entry.SumUse += r.SumUse
		case LogTypeError:
			entry.Error = r.Cnt
			entry.SumUse += r.SumUse
		}
		out[r.GroupName] = entry
	}
	return out, nil
}

type AvailabilityBucketRow struct {
	Bucket int64 `gorm:"column:bucket"`
	Type   int   `gorm:"column:type"`
	Cnt    int64 `gorm:"column:cnt"`
	SumUse int64 `gorm:"column:sum_use"`
}

// GetGroupTimeseries returns raw bucketed rows for one group inside
// [startSec, endSec). Pivot in the caller.
func GetGroupTimeseries(group string, startSec, endSec, bucketSec int64) ([]AvailabilityBucketRow, error) {
	bucketExpr := bucketColExpr("created_at", bucketSec)
	selectExpr := fmt.Sprintf("%s AS bucket, type, COUNT(*) AS cnt, COALESCE(SUM(use_time), 0) AS sum_use", bucketExpr)
	groupExpr := fmt.Sprintf("%s, type", bucketExpr)

	var rows []AvailabilityBucketRow
	err := LOG_DB.Table("logs").
		Select(selectExpr).
		Where("created_at >= ? AND created_at < ?", startSec, endSec).
		Where(logGroupCol+" = ?", group).
		Where("type IN ?", []int{LogTypeConsume, LogTypeError}).
		Group(groupExpr).
		Order("bucket ASC").
		Scan(&rows).Error
	return rows, err
}

type ChannelAggregateRow struct {
	ChannelId int   `gorm:"column:channel_id"`
	Type      int   `gorm:"column:type"`
	Cnt       int64 `gorm:"column:cnt"`
}

// GetGroupChannelAggregate returns success/error counts per channel inside the
// window for the given group.
func GetGroupChannelAggregate(group string, startSec, endSec int64) ([]ChannelAggregateRow, error) {
	var rows []ChannelAggregateRow
	err := LOG_DB.Table("logs").
		Select("channel_id, type, COUNT(*) AS cnt").
		Where("created_at >= ? AND created_at < ?", startSec, endSec).
		Where(logGroupCol+" = ?", group).
		Where("type IN ?", []int{LogTypeConsume, LogTypeError}).
		Where("channel_id > 0").
		Group("channel_id, type").
		Scan(&rows).Error
	return rows, err
}

type ChannelMeta struct {
	Id           int    `gorm:"column:id"`
	Name         string `gorm:"column:name"`
	Status       int    `gorm:"column:status"`
	ResponseTime int    `gorm:"column:response_time"`
	TestTime     int64  `gorm:"column:test_time"`
}

// GetChannelsInGroup returns all enabled abilities' channels for the given
// group. "Enabled" here mirrors what request routing actually sees.
func GetChannelsInGroup(group string) ([]ChannelMeta, error) {
	var rows []ChannelMeta
	err := DB.Table("abilities AS a").
		Select("DISTINCT c.id, c.name, c.status, c.response_time, c.test_time").
		Joins("JOIN channels AS c ON c.id = a.channel_id").
		Where("a."+commonGroupCol+" = ?", group).
		Where("a.enabled = " + commonTrueVal).
		Scan(&rows).Error
	return rows, err
}

// GetLatencySamples pulls a bounded random sample of Consume-type latencies
// inside the window for percentile estimation in Go. Returned slice is sorted
// ascending; empty if no rows match.
func GetLatencySamples(group string, startSec, endSec int64, limit int) ([]int, error) {
	if limit <= 0 {
		limit = 2000
	}
	var samples []int
	err := LOG_DB.Table("logs").
		Select("use_time").
		Where("created_at >= ? AND created_at < ?", startSec, endSec).
		Where(logGroupCol+" = ?", group).
		Where("type = ?", LogTypeConsume).
		Where("use_time > 0").
		Order(randExpr()).
		Limit(limit).
		Pluck("use_time", &samples).Error
	if err != nil {
		return nil, err
	}
	sort.Ints(samples)
	return samples, nil
}

// PercentileFromSorted returns the value at the given percentile (0–100) from
// an already-sorted ascending slice. Returns nil if the slice is empty.
func PercentileFromSorted(sorted []int, pct float64) *int {
	if len(sorted) == 0 {
		return nil
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	idx := int(float64(len(sorted)-1) * pct / 100.0)
	v := sorted[idx]
	return &v
}

// ListKnownGroups returns the union of groups visible in abilities and
// recent logs.
func ListKnownGroups(recentSinceSec int64) ([]string, error) {
	set := make(map[string]struct{})
	var fromAbilities []string
	if err := DB.Table("abilities").
		Distinct(commonGroupCol).
		Where("enabled = " + commonTrueVal).
		Pluck(commonGroupCol, &fromAbilities).Error; err != nil {
		return nil, err
	}
	for _, g := range fromAbilities {
		if g != "" {
			set[g] = struct{}{}
		}
	}
	var fromLogs []string
	if err := LOG_DB.Table("logs").
		Distinct(logGroupCol).
		Where("created_at >= ?", recentSinceSec).
		Pluck(logGroupCol, &fromLogs).Error; err != nil {
		return nil, err
	}
	for _, g := range fromLogs {
		if g != "" {
			set[g] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for g := range set {
		out = append(out, g)
	}
	sort.Strings(out)
	return out, nil
}
