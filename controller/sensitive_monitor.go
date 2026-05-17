package controller

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type sensitiveWordRequest struct {
	Pattern     string `json:"pattern"`
	IsRegex     bool   `json:"is_regex"`
	Enabled     bool   `json:"enabled"`
	Action      int    `json:"action"`
	Category    string `json:"category"`
	Severity    int    `json:"severity"`
	Description string `json:"description"`
}

type sensitiveHitFalsePositiveRequest struct {
	Value bool `json:"value"`
}

func GetSensitiveMonitorRules(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	var enabled *bool
	if raw := c.Query("enabled"); raw != "" {
		value := raw == "true" || raw == "1"
		enabled = &value
	}
	rules, total, err := model.ListSensitiveWords(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), enabled)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(rules)
	common.ApiSuccess(c, pageInfo)
}

func GetSensitiveMonitorHits(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	query, err := getSensitiveWordHitQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	hits, total, err := model.ListSensitiveWordHits(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(hits)
	common.ApiSuccess(c, pageInfo)
}

func ExportSensitiveMonitorHits(c *gin.Context) {
	query, err := getSensitiveWordHitQuery(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	hits, err := model.ExportSensitiveWordHits(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="sensitive-monitor-hits-%s.csv"`, time.Now().Format("20060102150405")))
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{
		"id",
		"created_at",
		"action",
		"rule_id",
		"pattern",
		"is_regex",
		"user_id",
		"username",
		"token_id",
		"token_name",
		"model_name",
		"request_id",
		"ip",
		"channel_id",
		"group",
		"path",
		"prompt_snippet",
	})
	for _, hit := range hits {
		_ = writer.Write([]string{
			strconv.Itoa(hit.Id),
			hit.CreatedAt.Format(time.RFC3339),
			strconv.Itoa(hit.Action),
			strconv.Itoa(hit.RuleId),
			hit.Pattern,
			strconv.FormatBool(hit.IsRegex),
			strconv.Itoa(hit.UserId),
			hit.Username,
			strconv.Itoa(hit.TokenId),
			hit.TokenName,
			hit.ModelName,
			hit.RequestId,
			hit.Ip,
			strconv.Itoa(hit.ChannelId),
			hit.Group,
			hit.Path,
			hit.PromptSnippet,
		})
	}
	writer.Flush()
}

func CreateSensitiveMonitorRule(c *gin.Context) {
	var req sensitiveWordRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	rule := &model.SensitiveWord{
		Pattern:     req.Pattern,
		IsRegex:     req.IsRegex,
		Enabled:     req.Enabled,
		Action:      req.Action,
		Category:    req.Category,
		Severity:    req.Severity,
		Description: req.Description,
	}
	if err := model.CreateSensitiveWord(rule); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.ReloadSensitiveMonitorRules(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rule)
}

func UpdateSensitiveMonitorRule(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req sensitiveWordRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	rule := &model.SensitiveWord{
		Id:          id,
		Pattern:     req.Pattern,
		IsRegex:     req.IsRegex,
		Enabled:     req.Enabled,
		Action:      req.Action,
		Category:    req.Category,
		Severity:    req.Severity,
		Description: req.Description,
	}
	if err := model.UpdateSensitiveWord(rule); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.ReloadSensitiveMonitorRules(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func ReloadSensitiveMonitorRules(c *gin.Context) {
	if err := service.ReloadSensitiveMonitorRules(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func SeedDefaultSensitiveMonitorRules(c *gin.Context) {
	result, err := service.SeedDefaultSensitiveWords()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

// MarkSensitiveMonitorHitFalsePositive 将一条命中标记为误报或撤销标记。
// body: {"value": true}  标记；{"value": false} 撤销
func MarkSensitiveMonitorHitFalsePositive(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, errors.New("invalid hit id"))
		return
	}
	var req sensitiveHitFalsePositiveRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	reviewerId := c.GetInt("id")
	if err := model.MarkSensitiveHitFalsePositive(id, reviewerId, req.Value); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func getSensitiveWordHitQuery(c *gin.Context) (model.SensitiveWordHitQuery, error) {
	action, _ := strconv.Atoi(c.Query("action"))
	startTime, err := parseSensitiveMonitorTime(c.Query("start_time"))
	if err != nil {
		return model.SensitiveWordHitQuery{}, err
	}
	endTime, err := parseSensitiveMonitorTime(c.Query("end_time"))
	if err != nil {
		return model.SensitiveWordHitQuery{}, err
	}
	var falsePositive *bool
	switch c.Query("false_positive") {
	case "true", "1":
		v := true
		falsePositive = &v
	case "false", "0":
		v := false
		falsePositive = &v
	}
	return model.SensitiveWordHitQuery{
		Action:        action,
		Keyword:       c.Query("keyword"),
		Username:      c.Query("username"),
		TokenName:     c.Query("token_name"),
		ModelName:     c.Query("model_name"),
		RequestId:     c.Query("request_id"),
		Path:          c.Query("path"),
		Group:         c.Query("group"),
		FalsePositive: falsePositive,
		StartTime:     startTime,
		EndTime:       endTime,
	}, nil
}

func parseSensitiveMonitorTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, errors.New("invalid sensitive monitor time filter")
}
