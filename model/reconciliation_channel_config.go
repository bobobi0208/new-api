package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReconciliationChannelConfig 独立于 Channel 模型存储渠道的对账配置。
// 这样设计避免修改 Channel/ChannelInfo struct，方便与上游 main 分支保持解耦、降低合并冲突。
type ReconciliationChannelConfig struct {
	Id           int       `json:"id"`
	ChannelId    int       `json:"channel_id" gorm:"uniqueIndex;not null"`
	UpstreamType string    `json:"upstream_type" gorm:"type:varchar(32);not null;default:''"`
	Enabled      bool      `json:"enabled" gorm:"default:true;index"`
	BaseURL      string    `json:"base_url" gorm:"type:varchar(512);default:''"` // 留空表示沿用 Channel.BaseURL
	Note         string    `json:"note" gorm:"type:varchar(512);default:''"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ReconciliationChannelConfig) TableName() string {
	return "reconciliation_channel_configs"
}

func GetReconciliationChannelConfig(channelId int) (*ReconciliationChannelConfig, error) {
	if channelId == 0 {
		return nil, errors.New("channel_id is required")
	}
	var cfg ReconciliationChannelConfig
	err := DB.Where("channel_id = ?", channelId).First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

// ListEnabledReconciliationChannelConfigs 返回所有启用且已标记上游类型的配置，供定时任务遍历。
func ListEnabledReconciliationChannelConfigs() ([]ReconciliationChannelConfig, error) {
	var list []ReconciliationChannelConfig
	err := DB.Where("enabled = ? AND upstream_type <> ''", true).Find(&list).Error
	return list, err
}

func ListAllReconciliationChannelConfigs() ([]ReconciliationChannelConfig, error) {
	var list []ReconciliationChannelConfig
	err := DB.Order("channel_id asc").Find(&list).Error
	return list, err
}

// UpsertReconciliationChannelConfig 创建或更新一个 channel 的配置。
// 使用 GORM 的 OnConflict 子句跨三库统一 upsert 行为。
func UpsertReconciliationChannelConfig(cfg *ReconciliationChannelConfig) error {
	if cfg == nil || cfg.ChannelId == 0 {
		return errors.New("channel_id is required")
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "channel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"upstream_type", "enabled", "base_url", "note", "updated_at",
		}),
	}).Create(cfg).Error
}

func DeleteReconciliationChannelConfig(channelId int) error {
	if channelId == 0 {
		return errors.New("channel_id is required")
	}
	return DB.Where("channel_id = ?", channelId).Delete(&ReconciliationChannelConfig{}).Error
}
