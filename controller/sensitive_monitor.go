package controller

import (
	"strconv"

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
	Description string `json:"description"`
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
	action, _ := strconv.Atoi(c.Query("action"))
	hits, total, err := model.ListSensitiveWordHits(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), action)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(hits)
	common.ApiSuccess(c, pageInfo)
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
