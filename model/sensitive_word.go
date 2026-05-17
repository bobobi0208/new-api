package model

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
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
	Category    string     `json:"category" gorm:"type:varchar(64);index;default:''"`
	Severity    int        `json:"severity" gorm:"default:3;index"`
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
	Id            int        `json:"id"`
	RuleId        int        `json:"rule_id" gorm:"index"`
	Pattern       string     `json:"pattern" gorm:"type:text"`
	IsRegex       bool       `json:"is_regex"`
	Action        int        `json:"action" gorm:"index"`
	UserId        int        `json:"user_id" gorm:"index"`
	Username      string     `json:"username" gorm:"index;default:''"`
	TokenId       int        `json:"token_id" gorm:"index"`
	TokenName     string     `json:"token_name" gorm:"index;default:''"`
	ModelName     string     `json:"model_name" gorm:"index;default:''"`
	RequestId     string     `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	Ip            string     `json:"ip" gorm:"index;default:''"`
	ChannelId     int        `json:"channel_id" gorm:"index"`
	Group         string     `json:"group" gorm:"index;default:''"`
	Path          string     `json:"path" gorm:"index;default:''"`
	PromptSnippet string     `json:"prompt_snippet" gorm:"type:text"`
	FalsePositive bool       `json:"false_positive" gorm:"default:false;index"`
	ReviewedBy    int        `json:"reviewed_by" gorm:"default:0"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	CreatedAt     time.Time  `json:"created_at" gorm:"index"`
}

func (SensitiveWordHit) TableName() string {
	return "sensitive_word_hits"
}

type SensitiveWordHitQuery struct {
	Action        int
	Keyword       string
	Username      string
	TokenName     string
	ModelName     string
	RequestId     string
	Path          string
	Group         string
	FalsePositive *bool
	StartTime     *time.Time
	EndTime       *time.Time
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

func ListSensitiveWordHits(offset, limit int, query SensitiveWordHitQuery) ([]SensitiveWordHit, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	tx := buildSensitiveWordHitQuery(query)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var hits []SensitiveWordHit
	err := tx.Order("id desc").Limit(limit).Offset(offset).Find(&hits).Error
	return hits, total, err
}

func ExportSensitiveWordHits(query SensitiveWordHitQuery) ([]SensitiveWordHit, error) {
	var hits []SensitiveWordHit
	err := buildSensitiveWordHitQuery(query).Order("id desc").Find(&hits).Error
	return hits, err
}

func buildSensitiveWordHitQuery(query SensitiveWordHitQuery) *gorm.DB {
	tx := DB.Model(&SensitiveWordHit{})
	if query.Action != 0 {
		tx = tx.Where("action = ?", query.Action)
	}
	if query.Keyword != "" {
		pattern := sensitiveContainsPattern(query.Keyword)
		tx = tx.Where(
			"(LOWER(pattern) LIKE ? ESCAPE '!' OR LOWER(prompt_snippet) LIKE ? ESCAPE '!' OR LOWER(path) LIKE ? ESCAPE '!')",
			pattern,
			pattern,
			pattern,
		)
	}
	if query.Username != "" {
		pattern := sensitiveContainsPattern(query.Username)
		if userId, err := strconv.Atoi(strings.TrimSpace(query.Username)); err == nil {
			tx = tx.Where("(LOWER(username) LIKE ? ESCAPE '!' OR user_id = ?)", pattern, userId)
		} else {
			tx = tx.Where("LOWER(username) LIKE ? ESCAPE '!'", pattern)
		}
	}
	if query.TokenName != "" {
		pattern := sensitiveContainsPattern(query.TokenName)
		if tokenId, err := strconv.Atoi(strings.TrimSpace(query.TokenName)); err == nil {
			tx = tx.Where("(LOWER(token_name) LIKE ? ESCAPE '!' OR token_id = ?)", pattern, tokenId)
		} else {
			tx = tx.Where("LOWER(token_name) LIKE ? ESCAPE '!'", pattern)
		}
	}
	if query.ModelName != "" {
		tx = tx.Where("LOWER(model_name) LIKE ? ESCAPE '!'", sensitiveContainsPattern(query.ModelName))
	}
	if query.RequestId != "" {
		tx = tx.Where("LOWER(request_id) LIKE ? ESCAPE '!'", sensitiveContainsPattern(query.RequestId))
	}
	if query.Path != "" {
		tx = tx.Where("LOWER(path) LIKE ? ESCAPE '!'", sensitiveContainsPattern(query.Path))
	}
	if query.Group != "" {
		tx = tx.Where(commonGroupCol+" = ?", strings.TrimSpace(query.Group))
	}
	if query.FalsePositive != nil {
		tx = tx.Where("false_positive = ?", *query.FalsePositive)
	}
	if query.StartTime != nil {
		tx = tx.Where("created_at >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		tx = tx.Where("created_at <= ?", *query.EndTime)
	}
	return tx
}

func sensitiveContainsPattern(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	input = strings.ReplaceAll(input, "!", "!!")
	input = strings.ReplaceAll(input, "%", "!%")
	input = strings.ReplaceAll(input, "_", "!_")
	return "%" + input + "%"
}

func ClearSensitiveWordHitsBefore(cutoff time.Time) (int64, error) {
	result := DB.Where("created_at < ?", cutoff).Delete(&SensitiveWordHit{})
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
	if err := DB.Select("pattern", "is_regex", "enabled", "action", "category", "severity", "description").Create(rule).Error; err != nil {
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
		"category":    rule.Category,
		"severity":    rule.Severity,
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
	if rule.Severity < 1 || rule.Severity > 5 {
		rule.Severity = 3
	}
	rule.Category = strings.TrimSpace(rule.Category)
	if rule.IsRegex {
		if _, err := regexp.Compile(rule.Pattern); err != nil {
			return err
		}
	}
	return nil
}

// MarkSensitiveHitFalsePositive 将一条命中记录标记为误报或撤销标记。
// value=true 标记为误报并写入审阅信息；value=false 清除误报标记与审阅人。
func MarkSensitiveHitFalsePositive(hitId, reviewerId int, value bool) error {
	if hitId <= 0 {
		return errors.New("命中记录 ID 无效")
	}
	hit := SensitiveWordHit{Id: hitId, FalsePositive: value}
	if value {
		now := time.Now()
		hit.ReviewedBy = reviewerId
		hit.ReviewedAt = &now
	} else {
		hit.ReviewedBy = 0
		hit.ReviewedAt = nil
	}
	result := DB.Model(&hit).
		Select("false_positive", "reviewed_by", "reviewed_at").
		Updates(hit)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
