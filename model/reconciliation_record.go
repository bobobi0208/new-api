package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	ReconciliationStatusMatch        = "match"
	ReconciliationStatusMismatch     = "mismatch"
	ReconciliationStatusInconclusive = "inconclusive"
	ReconciliationStatusError        = "error"

	ReconciliationRunTypeBalance = "balance_snapshot"
	ReconciliationRunTypeDaily   = "daily_diff"
	ReconciliationRunTypeManual  = "manual"

	ReconciliationUpstreamTypeNewApi  = "newapi"
	ReconciliationUpstreamTypeSub2Api = "sub2api"
)

type ReconciliationRecord struct {
	Id                int       `json:"id"`
	ChannelId         int       `json:"channel_id" gorm:"index;not null"`
	ChannelName       string    `json:"channel_name" gorm:"-"`
	UpstreamType      string    `json:"upstream_type" gorm:"type:varchar(32);not null"`
	RunType           string    `json:"run_type" gorm:"type:varchar(32);not null;index"`
	RunAt             int64     `json:"run_at" gorm:"bigint;index"`
	WindowStart       int64     `json:"window_start" gorm:"bigint;default:0"`
	WindowEnd         int64     `json:"window_end" gorm:"bigint;default:0"`
	UpstreamUsedUSD   float64   `json:"upstream_used_usd"`
	UpstreamRemainUSD float64   `json:"upstream_remain_usd"`
	UpstreamRawJSON   string    `json:"upstream_raw_json" gorm:"type:text"`
	LocalUsedUSD      float64   `json:"local_used_usd"`
	DeltaUSD          float64   `json:"delta_usd"`
	DeltaRel          float64   `json:"delta_rel"`
	Status            string    `json:"status" gorm:"type:varchar(16);index;not null"`
	Message           string    `json:"message" gorm:"type:varchar(512);default:''"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (ReconciliationRecord) TableName() string {
	return "reconciliation_records"
}

type ReconciliationQuery struct {
	ChannelId    int
	UpstreamType string
	RunType      string
	Status       string
	StartTime    *time.Time
	EndTime      *time.Time
}

func CreateReconciliationRecord(record *ReconciliationRecord) error {
	if record == nil {
		return errors.New("record is nil")
	}
	if record.ChannelId == 0 {
		return errors.New("channel_id is required")
	}
	if record.RunType == "" || record.Status == "" {
		return errors.New("run_type and status are required")
	}
	return DB.Create(record).Error
}

func GetReconciliationRecordById(id int) (*ReconciliationRecord, error) {
	if id == 0 {
		return nil, errors.New("id is required")
	}
	var record ReconciliationRecord
	err := DB.First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func ListReconciliationRecords(offset, limit int, query ReconciliationQuery) ([]ReconciliationRecord, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	tx := buildReconciliationQuery(query)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []ReconciliationRecord
	err := tx.Order("id desc").Limit(limit).Offset(offset).Find(&records).Error
	return records, total, err
}

// ListLatestReconciliationPerChannel 返回每个 channel 最近一条记录，供"对账总览"列表使用。
// 子查询写法在 SQLite/MySQL/PostgreSQL 上 GORM 均可正确翻译。
func ListLatestReconciliationPerChannel(upstreamType string) ([]ReconciliationRecord, error) {
	sub := DB.Model(&ReconciliationRecord{}).Select("MAX(id) as max_id").Group("channel_id")
	if upstreamType != "" {
		sub = sub.Where("upstream_type = ?", upstreamType)
	}
	var records []ReconciliationRecord
	err := DB.Where("id IN (?)", sub).Order("channel_id asc").Find(&records).Error
	return records, err
}

// ListMismatchAlerts 返回当前各 channel 最近一条记录里 status=mismatch 的告警集合。
func ListMismatchAlerts() ([]ReconciliationRecord, error) {
	latest, err := ListLatestReconciliationPerChannel("")
	if err != nil {
		return nil, err
	}
	alerts := make([]ReconciliationRecord, 0, len(latest))
	for _, r := range latest {
		if r.Status == ReconciliationStatusMismatch {
			alerts = append(alerts, r)
		}
	}
	return alerts, nil
}

// DeleteReconciliationRecordsBefore 清理 retain 期外的旧记录。
func DeleteReconciliationRecordsBefore(cutoff time.Time) (int64, error) {
	result := DB.Where("created_at < ?", cutoff).Delete(&ReconciliationRecord{})
	return result.RowsAffected, result.Error
}

// SumChannelConsumeQuota 聚合指定 channel 在 [startSec, endSec] 区间内 Consume 日志的 quota 总和。
// startSec/endSec 为 Unix 秒；任一为 0 表示不施加该方向的边界。
// 单独放在本文件而非 model/log.go，以减少对 log.go 的入侵和合并冲突面。
// LogTypeConsume 在本包内为常量 2，直接使用数值避免跨文件耦合。
func SumChannelConsumeQuota(channelId int, startSec, endSec int64) (int64, error) {
	if channelId == 0 {
		return 0, errors.New("channel_id is required")
	}
	tx := LOG_DB.Model(&Log{}).Where("channel_id = ? AND type = ?", channelId, LogTypeConsume)
	if startSec > 0 {
		tx = tx.Where("created_at >= ?", startSec)
	}
	if endSec > 0 {
		tx = tx.Where("created_at < ?", endSec)
	}
	var total int64
	err := tx.Select("COALESCE(SUM(quota), 0)").Scan(&total).Error
	return total, err
}

func buildReconciliationQuery(query ReconciliationQuery) *gorm.DB {
	tx := DB.Model(&ReconciliationRecord{})
	if query.ChannelId > 0 {
		tx = tx.Where("channel_id = ?", query.ChannelId)
	}
	if query.UpstreamType != "" {
		tx = tx.Where("upstream_type = ?", query.UpstreamType)
	}
	if query.RunType != "" {
		tx = tx.Where("run_type = ?", query.RunType)
	}
	if query.Status != "" {
		tx = tx.Where("status = ?", query.Status)
	}
	if query.StartTime != nil {
		tx = tx.Where("created_at >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		tx = tx.Where("created_at <= ?", *query.EndTime)
	}
	return tx
}
