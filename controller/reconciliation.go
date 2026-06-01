package controller

import (
	"errors"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/reconciliation"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// GetReconciliationRecords 返回对账记录列表，支持按 channel_id / status / run_type / 时间范围筛选。
// GET /api/reconciliation/records?channel_id=&status=&run_type=&start_at=&end_at=
func GetReconciliationRecords(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	query := model.ReconciliationQuery{
		ChannelId:    atoiSafe(c.Query("channel_id")),
		UpstreamType: c.Query("upstream_type"),
		RunType:      c.Query("run_type"),
		Status:       c.Query("status"),
	}
	if s := c.Query("start_at"); s != "" {
		if t, err := strconv.ParseInt(s, 10, 64); err == nil && t > 0 {
			tt := time.Unix(t, 0)
			query.StartTime = &tt
		}
	}
	if s := c.Query("end_at"); s != "" {
		if t, err := strconv.ParseInt(s, 10, 64); err == nil && t > 0 {
			tt := time.Unix(t, 0)
			query.EndTime = &tt
		}
	}
	records, total, err := model.ListReconciliationRecords(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	attachChannelNames(records)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(records)
	common.ApiSuccess(c, pageInfo)
}

// GetReconciliationRecord 返回单条对账记录详情（含 UpstreamRawJSON）。
// GET /api/reconciliation/records/:id
func GetReconciliationRecord(c *gin.Context) {
	id := atoiSafe(c.Param("id"))
	if id == 0 {
		common.ApiError(c, errors.New("invalid id"))
		return
	}
	rec, err := model.GetReconciliationRecordById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	attachChannelName(rec)
	common.ApiSuccess(c, rec)
}

// GetReconciliationAlerts 返回当前 mismatch 状态的告警列表（每个 channel 最新一条）。
// GET /api/reconciliation/alerts
func GetReconciliationAlerts(c *gin.Context) {
	alerts, err := model.ListMismatchAlerts()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	attachChannelNames(alerts)
	common.ApiSuccess(c, alerts)
}

// GetReconciliationLatestPerChannel 返回每个 channel 的最近一条记录，列表页主视图。
// GET /api/reconciliation/latest
func GetReconciliationLatestPerChannel(c *gin.Context) {
	latest, err := model.ListLatestReconciliationPerChannel(c.Query("upstream_type"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	attachChannelNames(latest)
	common.ApiSuccess(c, latest)
}

type triggerReconciliationRequest struct {
	ChannelId   int    `json:"channel_id"`
	RunType     string `json:"run_type"`
	WindowStart int64  `json:"window_start"`
	WindowEnd   int64  `json:"window_end"`
}

// TriggerReconciliation 手动触发指定 channel 一次对账（调试和回填用）。
// POST /api/reconciliation/trigger { channel_id, run_type?, window_start?, window_end? }
func TriggerReconciliation(c *gin.Context) {
	var req triggerReconciliationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.ChannelId == 0 {
		common.ApiError(c, errors.New("channel_id is required"))
		return
	}
	if req.RunType == "" {
		req.RunType = model.ReconciliationRunTypeManual
	}
	opts := reconciliation.RunOptions{
		ChannelId: req.ChannelId,
		RunType:   req.RunType,
	}
	if req.WindowStart > 0 {
		opts.WindowStart = time.Unix(req.WindowStart, 0)
	}
	if req.WindowEnd > 0 {
		opts.WindowEnd = time.Unix(req.WindowEnd, 0)
	}
	rec, err := reconciliation.RunOnce(c.Request.Context(), opts)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rec)
}

// GetReconciliationSettings 返回当前对账系统的全局配置。
// GET /api/reconciliation/settings
func GetReconciliationSettings(c *gin.Context) {
	common.ApiSuccess(c, operation_setting.GetReconciliationSetting())
}

// UpdateReconciliationSettings 修改对账系统全局配置（写入 OptionMap 热更新）。
// PUT /api/reconciliation/settings
func UpdateReconciliationSettings(c *gin.Context) {
	var newSetting operation_setting.ReconciliationSetting
	if err := c.ShouldBindJSON(&newSetting); err != nil {
		common.ApiError(c, err)
		return
	}
	current := operation_setting.GetReconciliationSetting()
	*current = newSetting
	// 持久化到 option 表，确保重启后生效；遵循已有热更新机制走 model.UpdateOption。
	if err := model.UpdateOption("reconciliation_setting", marshalSettingJSON(newSetting)); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, current)
}

func marshalSettingJSON(s operation_setting.ReconciliationSetting) string {
	b, err := common.Marshal(s)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ListReconciliationChannelConfigs 返回所有 channel 的对账配置（前端"渠道配置"页用）。
// GET /api/reconciliation/channels
func ListReconciliationChannelConfigs(c *gin.Context) {
	list, err := model.ListAllReconciliationChannelConfigs()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, list)
}

type upsertChannelConfigRequest struct {
	ChannelId    int    `json:"channel_id"`
	UpstreamType string `json:"upstream_type"`
	Enabled      bool   `json:"enabled"`
	BaseURL      string `json:"base_url"`
	Note         string `json:"note"`
}

// UpsertReconciliationChannelConfig 创建或更新某个 channel 的对账配置。
// POST /api/reconciliation/channels
func UpsertReconciliationChannelConfigController(c *gin.Context) {
	var req upsertChannelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.ChannelId == 0 {
		common.ApiError(c, errors.New("channel_id is required"))
		return
	}
	if req.UpstreamType != "" && !reconciliation.IsKnownUpstream(req.UpstreamType) {
		common.ApiError(c, errors.New("unknown upstream_type"))
		return
	}
	cfg := &model.ReconciliationChannelConfig{
		ChannelId:    req.ChannelId,
		UpstreamType: req.UpstreamType,
		Enabled:      req.Enabled,
		BaseURL:      req.BaseURL,
		Note:         req.Note,
	}
	if err := model.UpsertReconciliationChannelConfig(cfg); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, cfg)
}

// DeleteReconciliationChannelConfigController 删除某个 channel 的对账配置。
// DELETE /api/reconciliation/channels/:channel_id
func DeleteReconciliationChannelConfigController(c *gin.Context) {
	id := atoiSafe(c.Param("channel_id"))
	if id == 0 {
		common.ApiError(c, errors.New("invalid channel_id"))
		return
	}
	if err := model.DeleteReconciliationChannelConfig(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"channel_id": id})
}

func atoiSafe(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

func attachChannelNames(records []model.ReconciliationRecord) {
	if len(records) == 0 {
		return
	}
	seen := make(map[int]struct{}, len(records))
	ids := make([]int, 0, len(records))
	for _, r := range records {
		if r.ChannelId == 0 {
			continue
		}
		if _, ok := seen[r.ChannelId]; ok {
			continue
		}
		seen[r.ChannelId] = struct{}{}
		ids = append(ids, r.ChannelId)
	}
	if len(ids) == 0 {
		return
	}
	channels, err := model.GetChannelsByIds(ids)
	if err != nil {
		return
	}
	names := make(map[int]string, len(channels))
	for _, ch := range channels {
		if ch != nil {
			names[ch.Id] = ch.Name
		}
	}
	for i := range records {
		records[i].ChannelName = names[records[i].ChannelId]
	}
}

func attachChannelName(record *model.ReconciliationRecord) {
	if record == nil || record.ChannelId == 0 {
		return
	}
	ch, err := model.GetChannelById(record.ChannelId, false)
	if err != nil || ch == nil {
		return
	}
	record.ChannelName = ch.Name
}
