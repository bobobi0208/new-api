package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const ProbeDefenseTargetAPIKeyMask = "__PROBE_DEFENSE_TARGET_API_KEY_CONFIGURED__"

type ProbeDefenseSource struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	Key         string    `json:"key" gorm:"size:64;uniqueIndex;not null"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Description string    `json:"description" gorm:"type:text"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProbeDefenseSignature struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:128;not null"`
	SourceKey string    `json:"source_key" gorm:"size:64;index;not null"`
	Protocol  string    `json:"protocol" gorm:"size:64;index;not null"`
	Topic     string    `json:"topic" gorm:"size:128;index"`
	MatchType string    `json:"match_type" gorm:"size:32;not null"`
	Patterns  string    `json:"patterns" gorm:"type:text;not null"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProbeDefenseGroupPolicy struct {
	ID           int       `json:"id" gorm:"primaryKey"`
	GroupName    string    `json:"group_name" gorm:"size:64;uniqueIndex;not null"`
	Enabled      bool      `json:"enabled" gorm:"default:false"`
	SourceKeys   string    `json:"source_keys" gorm:"type:text"`
	TargetURL    string    `json:"target_url" gorm:"size:512"`
	TargetAPIKey string    `json:"target_api_key" gorm:"type:text"`
	LogOnly      bool      `json:"log_only" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ProbeDefenseEvent struct {
	ID             int       `json:"id" gorm:"primaryKey"`
	CreatedAt      time.Time `json:"created_at" gorm:"index"`
	GroupName      string    `json:"group_name" gorm:"size:64;index"`
	ChannelID      int       `json:"channel_id" gorm:"index"`
	ModelName      string    `json:"model_name" gorm:"size:128;index"`
	Protocol       string    `json:"protocol" gorm:"size:64;index"`
	SourceKey      string    `json:"source_key" gorm:"size:64;index"`
	Topic          string    `json:"topic" gorm:"size:128;index"`
	SignatureID    int       `json:"signature_id" gorm:"index"`
	SignatureName  string    `json:"signature_name" gorm:"size:128"`
	MatchType      string    `json:"match_type" gorm:"size:32"`
	Action         string    `json:"action" gorm:"size:32;index"`
	TargetURL      string    `json:"target_url" gorm:"size:512"`
	RequestID      string    `json:"request_id" gorm:"size:128;index"`
	ContentPreview string    `json:"content_preview" gorm:"type:text"`
}

type ProbeDefenseEventQuery struct {
	GroupName string
	SourceKey string
	Topic     string
	Action    string
	Offset    int
	Limit     int
}

func NormalizeProbeDefenseKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func NormalizeProbeDefensePatterns(patterns []string) []string {
	out := make([]string, 0, len(patterns))
	seen := map[string]bool{}
	for _, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func ProbeDefenseArrayToJSON(values []string) string {
	normalized := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		key := NormalizeProbeDefenseKey(value)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		normalized = append(normalized, key)
	}
	bytes, err := common.Marshal(normalized)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

func ProbeDefenseJSONToArray(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var out []string
	if err := common.Unmarshal([]byte(value), &out); err != nil {
		return nil
	}
	return out
}

func ListProbeDefenseSources() ([]ProbeDefenseSource, error) {
	var sources []ProbeDefenseSource
	err := DB.Order("key asc").Find(&sources).Error
	return sources, err
}

func UpsertProbeDefenseSource(source *ProbeDefenseSource) error {
	if source == nil {
		return errors.New("source is nil")
	}
	source.Key = NormalizeProbeDefenseKey(source.Key)
	if source.Key == "" || strings.TrimSpace(source.Name) == "" {
		return errors.New("source key and name are required")
	}
	if source.ID > 0 {
		return DB.Model(&ProbeDefenseSource{}).Where("id = ?", source.ID).Updates(map[string]interface{}{
			"key": source.Key, "name": source.Name, "description": source.Description, "enabled": source.Enabled,
		}).Error
	}
	var existing ProbeDefenseSource
	err := DB.Where("key = ?", source.Key).First(&existing).Error
	if err == nil {
		source.ID = existing.ID
		return UpsertProbeDefenseSource(source)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return DB.Create(source).Error
}

func ListProbeDefenseSignatures(sourceKey string) ([]ProbeDefenseSignature, error) {
	var signatures []ProbeDefenseSignature
	query := DB.Order("source_key asc, id asc")
	if key := NormalizeProbeDefenseKey(sourceKey); key != "" {
		query = query.Where("source_key = ?", key)
	}
	err := query.Find(&signatures).Error
	return signatures, err
}

func UpsertProbeDefenseSignature(signature *ProbeDefenseSignature) error {
	if signature == nil {
		return errors.New("signature is nil")
	}
	signature.SourceKey = NormalizeProbeDefenseKey(signature.SourceKey)
	if signature.Name == "" || signature.SourceKey == "" || signature.Protocol == "" || signature.MatchType == "" || signature.Patterns == "" {
		return errors.New("signature name, source, protocol, match type and patterns are required")
	}
	if signature.ID > 0 {
		return DB.Model(&ProbeDefenseSignature{}).Where("id = ?", signature.ID).Updates(map[string]interface{}{
			"name": signature.Name, "source_key": signature.SourceKey, "protocol": signature.Protocol, "topic": signature.Topic, "match_type": signature.MatchType, "patterns": signature.Patterns, "enabled": signature.Enabled,
		}).Error
	}
	return DB.Create(signature).Error
}

func DeleteProbeDefenseSignature(id int) error {
	return DB.Delete(&ProbeDefenseSignature{}, id).Error
}

func ListProbeDefenseGroupPolicies(mask bool) ([]ProbeDefenseGroupPolicy, error) {
	var policies []ProbeDefenseGroupPolicy
	if err := DB.Order("group_name asc").Find(&policies).Error; err != nil {
		return nil, err
	}
	if mask {
		for i := range policies {
			policies[i] = MaskProbeDefenseGroupPolicy(&policies[i])
		}
	}
	return policies, nil
}

func GetProbeDefenseGroupPolicy(groupName string) (*ProbeDefenseGroupPolicy, error) {
	var policy ProbeDefenseGroupPolicy
	err := DB.Where("group_name = ?", strings.TrimSpace(groupName)).First(&policy).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func UpsertProbeDefenseGroupPolicy(policy *ProbeDefenseGroupPolicy) error {
	if policy == nil {
		return errors.New("policy is nil")
	}
	policy.GroupName = strings.TrimSpace(policy.GroupName)
	if policy.GroupName == "" {
		return errors.New("group name is required")
	}
	var existing ProbeDefenseGroupPolicy
	err := DB.Where("group_name = ?", policy.GroupName).First(&existing).Error
	if err == nil {
		apiKey := strings.TrimSpace(policy.TargetAPIKey)
		if apiKey == "" || apiKey == ProbeDefenseTargetAPIKeyMask {
			apiKey = existing.TargetAPIKey
		}
		return DB.Model(&ProbeDefenseGroupPolicy{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
			"enabled": policy.Enabled, "source_keys": policy.SourceKeys, "target_url": policy.TargetURL, "target_api_key": apiKey, "log_only": policy.LogOnly,
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return DB.Create(policy).Error
}

func MaskProbeDefenseGroupPolicy(policy *ProbeDefenseGroupPolicy) ProbeDefenseGroupPolicy {
	if policy == nil {
		return ProbeDefenseGroupPolicy{}
	}
	masked := *policy
	if strings.TrimSpace(masked.TargetAPIKey) != "" {
		masked.TargetAPIKey = ProbeDefenseTargetAPIKeyMask
	}
	return masked
}

func CreateProbeDefenseEvent(event *ProbeDefenseEvent) error {
	if event == nil {
		return errors.New("event is nil")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	return DB.Create(event).Error
}

func ListProbeDefenseEvents(query ProbeDefenseEventQuery) ([]ProbeDefenseEvent, int64, error) {
	limit := query.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	db := DB.Model(&ProbeDefenseEvent{})
	if query.GroupName != "" {
		db = db.Where("group_name = ?", query.GroupName)
	}
	if query.SourceKey != "" {
		db = db.Where("source_key = ?", NormalizeProbeDefenseKey(query.SourceKey))
	}
	if query.Topic != "" {
		db = db.Where("topic = ?", query.Topic)
	}
	if query.Action != "" {
		db = db.Where("action = ?", query.Action)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var events []ProbeDefenseEvent
	err := db.Order("created_at desc, id desc").Offset(query.Offset).Limit(limit).Find(&events).Error
	return events, total, err
}

func SeedDefaultProbeDefenseData() error {
	sources := []ProbeDefenseSource{
		{Key: "cctest", Name: "cctest", Description: "cctest-like probe source", Enabled: true},
		{Key: "ztest", Name: "ztest", Description: "ztest probe source", Enabled: true},
		{Key: "custom", Name: "custom", Description: "custom probe source", Enabled: true},
	}
	for i := range sources {
		if err := UpsertProbeDefenseSource(&sources[i]); err != nil {
			return err
		}
	}

	defaults := []ProbeDefenseSignature{
		defaultProbeSignature("tag echo", "cctest", "claude_messages", "tag-echo", "contains_all", []string{"return exactly", "tag-"}),
		defaultProbeSignature("einstein five houses", "cctest", "claude_messages", "logic-einstein-5houses", "contains_all", []string{"五座颜色不同的房子", "谁养鱼"}),
		defaultProbeSignature("ai news web search", "cctest", "claude_messages", "websearch-ai-news", "contains_all", []string{"ai news", "web_search"}),
		defaultProbeSignature("identity conflict", "cctest", "claude_messages", "identity-conflict", "contains_any", []string{"多重身份", "claude code", "kiro", "warp", "antigravity"}),
		defaultProbeSignature("ocr probe", "cctest", "claude_messages", "ocr", "contains_any", []string{"ocr", "短码", "pdf ocr", "image ocr"}),
	}
	for i := range defaults {
		var existing ProbeDefenseSignature
		err := DB.Where("name = ? AND source_key = ?", defaults[i].Name, defaults[i].SourceKey).First(&existing).Error
		if err == nil {
			defaults[i].ID = existing.ID
			if err := UpsertProbeDefenseSignature(&defaults[i]); err != nil {
				return err
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := UpsertProbeDefenseSignature(&defaults[i]); err != nil {
			return err
		}
	}
	return nil
}

func defaultProbeSignature(name, sourceKey, protocol, topic, matchType string, patterns []string) ProbeDefenseSignature {
	bytes, _ := common.Marshal(NormalizeProbeDefensePatterns(patterns))
	return ProbeDefenseSignature{Name: name, SourceKey: sourceKey, Protocol: protocol, Topic: topic, MatchType: matchType, Patterns: string(bytes), Enabled: true}
}
