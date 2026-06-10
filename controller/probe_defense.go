package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/probe_defense"
	"github.com/gin-gonic/gin"
)

type probeDefenseSourceRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

type probeDefenseSignatureRequest struct {
	Name      string   `json:"name"`
	SourceKey string   `json:"source_key"`
	Protocol  string   `json:"protocol"`
	Topic     string   `json:"topic"`
	MatchType string   `json:"match_type"`
	Patterns  []string `json:"patterns"`
	Enabled   bool     `json:"enabled"`
}

type probeDefensePolicyRequest struct {
	Enabled      bool     `json:"enabled"`
	SourceKeys   []string `json:"source_keys"`
	TargetURL    string   `json:"target_url"`
	TargetAPIKey string   `json:"target_api_key"`
	LogOnly      bool     `json:"log_only"`
}

type probeDefenseTestMatchRequest struct {
	Protocol   string   `json:"protocol"`
	Text       string   `json:"text"`
	SourceKeys []string `json:"source_keys"`
}

func GetProbeDefenseSources(c *gin.Context) {
	sources, err := model.ListProbeDefenseSources()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, sources)
}

func CreateProbeDefenseSource(c *gin.Context) {
	var req probeDefenseSourceRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	source := &model.ProbeDefenseSource{Key: req.Key, Name: strings.TrimSpace(req.Name), Description: req.Description, Enabled: req.Enabled}
	if err := model.UpsertProbeDefenseSource(source); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, source)
}

func UpdateProbeDefenseSource(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, errors.New("invalid source id"))
		return
	}
	var req probeDefenseSourceRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	source := &model.ProbeDefenseSource{ID: id, Key: req.Key, Name: strings.TrimSpace(req.Name), Description: req.Description, Enabled: req.Enabled}
	if err := model.UpsertProbeDefenseSource(source); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, source)
}

func GetProbeDefenseSignatures(c *gin.Context) {
	signatures, err := model.ListProbeDefenseSignatures(c.Query("source_key"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, signatures)
}

func CreateProbeDefenseSignature(c *gin.Context) {
	signature, err := buildProbeDefenseSignature(0, c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpsertProbeDefenseSignature(signature); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, signature)
}

func UpdateProbeDefenseSignature(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, errors.New("invalid signature id"))
		return
	}
	signature, err := buildProbeDefenseSignature(id, c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpsertProbeDefenseSignature(signature); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, signature)
}

func DeleteProbeDefenseSignature(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiError(c, errors.New("invalid signature id"))
		return
	}
	if err := model.DeleteProbeDefenseSignature(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetProbeDefensePolicies(c *gin.Context) {
	policies, err := model.ListProbeDefenseGroupPolicies(true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, policies)
}

func UpdateProbeDefensePolicy(c *gin.Context) {
	groupName := strings.TrimSpace(c.Param("group"))
	if groupName == "" {
		common.ApiError(c, errors.New("group is required"))
		return
	}
	var req probeDefensePolicyRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	policy := &model.ProbeDefenseGroupPolicy{
		GroupName:    groupName,
		Enabled:      req.Enabled,
		SourceKeys:   model.ProbeDefenseArrayToJSON(req.SourceKeys),
		TargetURL:    strings.TrimSpace(req.TargetURL),
		TargetAPIKey: strings.TrimSpace(req.TargetAPIKey),
		LogOnly:      req.LogOnly,
	}
	if err := model.UpsertProbeDefenseGroupPolicy(policy); err != nil {
		common.ApiError(c, err)
		return
	}
	saved, err := model.GetProbeDefenseGroupPolicy(groupName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	masked := model.MaskProbeDefenseGroupPolicy(saved)
	common.ApiSuccess(c, masked)
}

func GetProbeDefenseEvents(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	events, total, err := model.ListProbeDefenseEvents(model.ProbeDefenseEventQuery{
		GroupName: strings.TrimSpace(c.Query("group")),
		SourceKey: strings.TrimSpace(c.Query("source_key")),
		Topic:     strings.TrimSpace(c.Query("topic")),
		Action:    strings.TrimSpace(c.Query("action")),
		Offset:    pageInfo.GetStartIdx(),
		Limit:     pageInfo.GetPageSize(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(events)
	common.ApiSuccess(c, pageInfo)
}

func ProbeDefenseTestMatch(c *gin.Context) {
	var req probeDefenseTestMatchRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	sources, err := model.ListProbeDefenseSources()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	signatures, err := model.ListProbeDefenseSignatures("")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	protocol := strings.TrimSpace(req.Protocol)
	if protocol == "" {
		protocol = probe_defense.ProtocolClaudeMessages
	}
	result := probe_defense.MatchText(req.Text, protocol, req.SourceKeys, probe_defense.BuildSignatureRules(sources, signatures))
	common.ApiSuccess(c, result)
}

func SeedDefaultProbeDefense(c *gin.Context) {
	if err := model.SeedDefaultProbeDefenseData(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func buildProbeDefenseSignature(id int, c *gin.Context) (*model.ProbeDefenseSignature, error) {
	var req probeDefenseSignatureRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		return nil, err
	}
	matchType := strings.TrimSpace(req.MatchType)
	if matchType == "" {
		matchType = probe_defense.MatchContainsAll
	}
	if !isProbeDefenseMatchTypeAllowed(matchType) {
		return nil, errors.New("invalid match type")
	}
	patterns := model.NormalizeProbeDefensePatterns(req.Patterns)
	if len(patterns) == 0 {
		return nil, errors.New("patterns are required")
	}
	bytes, err := common.Marshal(patterns)
	if err != nil {
		return nil, err
	}
	protocol := strings.TrimSpace(req.Protocol)
	if protocol == "" {
		protocol = probe_defense.ProtocolClaudeMessages
	}
	return &model.ProbeDefenseSignature{
		ID:        id,
		Name:      strings.TrimSpace(req.Name),
		SourceKey: req.SourceKey,
		Protocol:  protocol,
		Topic:     strings.TrimSpace(req.Topic),
		MatchType: matchType,
		Patterns:  string(bytes),
		Enabled:   req.Enabled,
	}, nil
}

func isProbeDefenseMatchTypeAllowed(matchType string) bool {
	switch matchType {
	case probe_defense.MatchContainsAll, probe_defense.MatchContainsAny, probe_defense.MatchRegex:
		return true
	default:
		return false
	}
}
