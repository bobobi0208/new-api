package model

import (
	"errors"
	"regexp"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	SensitiveWordActionBlock   = 1
	SensitiveWordActionMonitor = 2
)

type SensitiveWord struct {
	Id          int        `json:"id"`
	Pattern     string     `json:"pattern" gorm:"type:text;not null"`
	IsRegex     bool       `json:"is_regex" gorm:"default:false;index"`
	Enabled     bool       `json:"enabled" gorm:"default:true;index"`
	Action      int        `json:"action" gorm:"default:2;index"`
	Description string     `json:"description" gorm:"type:text"`
	HitCount    int        `json:"hit_count" gorm:"default:0"`
	LastHitAt   *time.Time `json:"last_hit_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (SensitiveWord) TableName() string {
	return "sensitive_words"
}

type SensitiveWordHit struct {
	Id            int       `json:"id"`
	RuleId        int       `json:"rule_id" gorm:"index"`
	Pattern       string    `json:"pattern" gorm:"type:text"`
	IsRegex       bool      `json:"is_regex"`
	Action        int       `json:"action" gorm:"index"`
	UserId        int       `json:"user_id" gorm:"index"`
	Username      string    `json:"username" gorm:"index;default:''"`
	TokenId       int       `json:"token_id" gorm:"index"`
	TokenName     string    `json:"token_name" gorm:"index;default:''"`
	ModelName     string    `json:"model_name" gorm:"index;default:''"`
	RequestId     string    `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	Ip            string    `json:"ip" gorm:"index;default:''"`
	ChannelId     int       `json:"channel_id" gorm:"index"`
	Group         string    `json:"group" gorm:"index;default:''"`
	Path          string    `json:"path" gorm:"index;default:''"`
	PromptSnippet string    `json:"prompt_snippet" gorm:"type:text"`
	CreatedAt     time.Time `json:"created_at" gorm:"index"`
}

func (SensitiveWordHit) TableName() string {
	return "sensitive_word_hits"
}

func ListSensitiveWords(offset, limit int, enabled *bool) ([]SensitiveWord, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	tx := DB.Model(&SensitiveWord{})
	if enabled != nil {
		tx = tx.Where("enabled = ?", *enabled)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rules []SensitiveWord
	err := tx.Order("id asc").Limit(limit).Offset(offset).Find(&rules).Error
	return rules, total, err
}

func ListSensitiveWordHits(offset, limit int, action int) ([]SensitiveWordHit, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	tx := DB.Model(&SensitiveWordHit{})
	if action != 0 {
		tx = tx.Where("action = ?", action)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var hits []SensitiveWordHit
	err := tx.Order("id desc").Limit(limit).Offset(offset).Find(&hits).Error
	return hits, total, err
}

func ClearSensitiveWordHits() (int64, error) {
	result := DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&SensitiveWordHit{})
	return result.RowsAffected, result.Error
}

func CreateSensitiveWord(rule *SensitiveWord) error {
	if rule == nil || rule.Pattern == "" {
		return errors.New("规则内容为空")
	}
	if err := normalizeSensitiveWord(rule); err != nil {
		return err
	}
	enabled := rule.Enabled
	if err := DB.Select("pattern", "is_regex", "enabled", "action", "description").Create(rule).Error; err != nil {
		return err
	}
	if !enabled {
		rule.Enabled = false
		return DB.Model(&SensitiveWord{}).Where("id = ?", rule.Id).Update("enabled", false).Error
	}
	return nil
}

func UpdateSensitiveWord(rule *SensitiveWord) error {
	if rule == nil || rule.Id == 0 {
		return errors.New("规则 ID 为空")
	}
	if err := normalizeSensitiveWord(rule); err != nil {
		return err
	}
	return DB.Model(&SensitiveWord{}).Where("id = ?", rule.Id).Updates(map[string]any{
		"pattern":     rule.Pattern,
		"is_regex":    rule.IsRegex,
		"enabled":     rule.Enabled,
		"action":      rule.Action,
		"description": rule.Description,
	}).Error
}

func IncrementSensitiveHit(ruleId int, at time.Time) error {
	return DB.Model(&SensitiveWord{}).Where("id = ?", ruleId).Updates(map[string]any{
		"hit_count":   gorm.Expr("hit_count + ?", 1),
		"last_hit_at": at,
	}).Error
}

func DisableTokenForSensitiveHit(tokenId int) error {
	if tokenId == 0 {
		return nil
	}
	token, err := GetTokenById(tokenId)
	if err != nil {
		return err
	}
	token.Status = common.TokenStatusDisabled
	return token.SelectUpdate()
}

func normalizeSensitiveWord(rule *SensitiveWord) error {
	if rule.Action != SensitiveWordActionBlock && rule.Action != SensitiveWordActionMonitor {
		rule.Action = SensitiveWordActionMonitor
	}
	if rule.IsRegex {
		if _, err := regexp.Compile(rule.Pattern); err != nil {
			return err
		}
	}
	return nil
}
