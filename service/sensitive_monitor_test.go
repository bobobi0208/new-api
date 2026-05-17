package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func resetSensitiveMonitorTables(t *testing.T) {
	t.Helper()
	require.NoError(t, model.DB.AutoMigrate(
		&model.SensitiveWord{},
		&model.SensitiveWordHit{},
		&model.Token{},
	))
	require.NoError(t, model.DB.Exec("DELETE FROM sensitive_word_hits").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM sensitive_words").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM tokens").Error)
	resetSensitiveMonitorCacheForTest()
}

func TestCheckSensitiveMonitorRecordsMonitorHitWithoutBlocking(t *testing.T) {
	resetSensitiveMonitorTables(t)
	rule := model.SensitiveWord{
		Pattern:     "lolicon hentai",
		Action:      model.SensitiveWordActionMonitor,
		Enabled:     true,
		Description: "monitor only",
	}
	require.NoError(t, model.DB.Create(&rule).Error)

	result, err := CheckSensitiveMonitor(SensitiveMonitorInput{
		UserId:     7,
		Username:   "alice",
		TokenId:    12,
		TokenName:  "dev-token",
		ModelName:  "gpt-test",
		RequestId:  "req-monitor",
		ChannelId:  3,
		PromptText: "please draw lolicon hentai style",
		Path:       "/v1/chat/completions",
	})

	require.NoError(t, err)
	require.True(t, result.Matched)
	require.False(t, result.Blocked)
	require.Len(t, result.Hits, 1)

	var hit model.SensitiveWordHit
	require.NoError(t, model.DB.First(&hit).Error)
	require.Equal(t, rule.Id, hit.RuleId)
	require.Equal(t, model.SensitiveWordActionMonitor, hit.Action)
	require.Equal(t, "req-monitor", hit.RequestId)

	var updated model.SensitiveWord
	require.NoError(t, model.DB.First(&updated, rule.Id).Error)
	require.Equal(t, 1, updated.HitCount)
	require.NotNil(t, updated.LastHitAt)
}

func TestCheckSensitiveMonitorDisablesTokenForBlockRule(t *testing.T) {
	resetSensitiveMonitorTables(t)
	token := model.Token{
		Id:          31,
		UserId:      9,
		Key:         "block-token-key",
		Status:      common.TokenStatusEnabled,
		Name:        "prod-token",
		RemainQuota: 1000,
	}
	require.NoError(t, model.DB.Create(&token).Error)
	rule := model.SensitiveWord{
		Pattern: "bad child pattern",
		Action:  model.SensitiveWordActionBlock,
		Enabled: true,
	}
	require.NoError(t, model.DB.Create(&rule).Error)

	result, err := CheckSensitiveMonitor(SensitiveMonitorInput{
		UserId:     9,
		TokenId:    token.Id,
		TokenName:  token.Name,
		RequestId:  "req-block",
		PromptText: "contains bad child pattern",
		Path:       "/v1/responses",
	})

	require.NoError(t, err)
	require.True(t, result.Blocked)

	var updatedToken model.Token
	require.NoError(t, model.DB.First(&updatedToken, token.Id).Error)
	require.Equal(t, common.TokenStatusDisabled, updatedToken.Status)
}

func TestSeedDefaultSensitiveWordsCreatesFullExportedRuleSet(t *testing.T) {
	resetSensitiveMonitorTables(t)

	result, err := SeedDefaultSensitiveWords()

	require.NoError(t, err)
	require.Equal(t, 85, result.Created)
	require.Equal(t, 0, result.Updated)

	var total int64
	require.NoError(t, model.DB.Model(&model.SensitiveWord{}).Count(&total).Error)
	require.EqualValues(t, 85, total)

	var enabled int64
	require.NoError(t, model.DB.Model(&model.SensitiveWord{}).Where("enabled = ?", true).Count(&enabled).Error)
	require.EqualValues(t, 85, enabled)

	var regexRules int64
	require.NoError(t, model.DB.Model(&model.SensitiveWord{}).Where("is_regex = ?", true).Count(&regexRules).Error)
	require.EqualValues(t, 11, regexRules)

	var blockRules int64
	require.NoError(t, model.DB.Model(&model.SensitiveWord{}).Where("action = ?", model.SensitiveWordActionBlock).Count(&blockRules).Error)
	require.EqualValues(t, 58, blockRules)

	resultAgain, err := SeedDefaultSensitiveWords()
	require.NoError(t, err)
	require.Equal(t, 0, resultAgain.Created)
	require.Equal(t, 85, resultAgain.Updated)
}

func TestSeedDefaultSensitiveWordsSyncsExistingDefaultRule(t *testing.T) {
	resetSensitiveMonitorTables(t)

	result, err := SeedDefaultSensitiveWords()
	require.NoError(t, err)
	require.Equal(t, 85, result.Created)

	require.NoError(t, model.DB.Model(&model.SensitiveWord{}).
		Where("pattern = ?", "lolicon hentai").
		Updates(map[string]any{
			"enabled":     false,
			"is_regex":    true,
			"action":      model.SensitiveWordActionBlock,
			"description": "stale",
		}).Error)

	result, err = SeedDefaultSensitiveWords()
	require.NoError(t, err)
	require.Equal(t, 0, result.Created)
	require.Equal(t, 85, result.Updated)

	var rule model.SensitiveWord
	require.NoError(t, model.DB.Where("pattern = ?", "lolicon hentai").First(&rule).Error)
	require.True(t, rule.Enabled)
	require.False(t, rule.IsRegex)
	require.Equal(t, model.SensitiveWordActionMonitor, rule.Action)
	require.Equal(t, "【英文·ACG】涉及低龄女性角色的不适当漫画", rule.Description)
}

func TestCheckSensitiveMonitorIgnoresStandaloneRapeKeyword(t *testing.T) {
	resetSensitiveMonitorTables(t)
	_, err := SeedDefaultSensitiveWords()
	require.NoError(t, err)

	result, err := CheckSensitiveMonitor(SensitiveMonitorInput{
		UserId:     7,
		Username:   "alice",
		TokenId:    12,
		TokenName:  "dev-token",
		ModelName:  "gpt-test",
		RequestId:  "req-rape-keyword",
		PromptText: "强奸",
		Path:       "/v1/chat/completions",
		Role:       "user",
	})

	require.NoError(t, err)
	require.False(t, result.Matched)
	require.False(t, result.Blocked)
	require.Empty(t, result.Hits)
}

func TestCheckSensitiveMonitorMatchesRapeWithMinorTerm(t *testing.T) {
	resetSensitiveMonitorTables(t)
	_, err := SeedDefaultSensitiveWords()
	require.NoError(t, err)

	result, err := CheckSensitiveMonitor(SensitiveMonitorInput{
		UserId:     7,
		Username:   "alice",
		TokenId:    12,
		TokenName:  "dev-token",
		ModelName:  "gpt-test",
		RequestId:  "req-rape-minor",
		PromptText: "强 奸 幼女",
		Path:       "/v1/chat/completions",
		Role:       "user",
	})

	require.NoError(t, err)
	require.True(t, result.Matched)
	for _, hit := range result.Hits {
		require.Contains(t, hit.Pattern, "强", "命中规则应包含强字")
	}
}

func TestCheckSensitiveMonitorRunsForPlaygroundChatCompletions(t *testing.T) {
	resetSensitiveMonitorTables(t)
	_, err := SeedDefaultSensitiveWords()
	require.NoError(t, err)
	require.True(t, ShouldRunSensitiveMonitor("/pg/chat/completions"))

	token := model.Token{
		Id:          12,
		UserId:      7,
		Key:         "playground-token-key",
		Status:      common.TokenStatusEnabled,
		Name:        "playground-default",
		RemainQuota: 1000,
	}
	require.NoError(t, model.DB.Create(&token).Error)

	result, err := CheckSensitiveMonitor(SensitiveMonitorInput{
		UserId:     7,
		Username:   "alice",
		TokenId:    token.Id,
		TokenName:  token.Name,
		ModelName:  "gpt-test",
		RequestId:  "req-playground-rape-minor",
		PromptText: "强奸幼童",
		Path:       "/pg/chat/completions",
		Role:       "user",
	})

	require.NoError(t, err)
	require.True(t, result.Matched)
	require.True(t, result.Blocked, "强奸幼童 应命中 Block 子串规则")
}

func TestCheckSensitiveMonitorStoresSnippetAroundMatchedPromptOnly(t *testing.T) {
	resetSensitiveMonitorTables(t)
	rule := model.SensitiveWord{
		Pattern: "强奸",
		Action:  model.SensitiveWordActionMonitor,
		Enabled: true,
	}
	require.NoError(t, model.DB.Create(&rule).Error)

	promptText := "system\n这是一段很长的系统提示，不应该整段进入命中片段。\nuser\n前一前二前三前四前五前六前七前八前九前十前十一前十二强奸后一后二后三后四后五后六后七后八后九后十后十一后十二\nassistant\n这是一段模型输出，不应该进入命中片段。"
	result, err := CheckSensitiveMonitor(SensitiveMonitorInput{
		UserId:     7,
		TokenId:    12,
		TokenName:  "dev-token",
		ModelName:  "gpt-test",
		RequestId:  "req-snippet-context",
		PromptText: promptText,
		Path:       "/v1/chat/completions",
	})

	require.NoError(t, err)
	require.True(t, result.Matched)

	var hit model.SensitiveWordHit
	require.NoError(t, model.DB.First(&hit).Error)
	require.Equal(t, "前八前九前十前十一前十二强奸后一后二后三后四后五后六", hit.PromptSnippet)
	require.NotContains(t, hit.PromptSnippet, "system")
	require.NotContains(t, hit.PromptSnippet, "assistant")
	require.NotContains(t, hit.PromptSnippet, "模型输出")
}

func TestRunSensitiveMonitorHitCleanupOnceDeletesOnlyExpiredHitDetails(t *testing.T) {
	resetSensitiveMonitorTables(t)
	lastHitAt := time.Now()
	rule := model.SensitiveWord{
		Pattern:   "强奸",
		Action:    model.SensitiveWordActionMonitor,
		Enabled:   true,
		HitCount:  9,
		LastHitAt: &lastHitAt,
	}
	require.NoError(t, model.DB.Create(&rule).Error)
	require.NoError(t, model.DB.Create(&model.SensitiveWordHit{
		RuleId:        rule.Id,
		Pattern:       rule.Pattern,
		Action:        rule.Action,
		PromptSnippet: "24 小时以前的命中",
		CreatedAt:     time.Now().Add(-25 * time.Hour),
	}).Error)
	require.NoError(t, model.DB.Create(&model.SensitiveWordHit{
		RuleId:        rule.Id,
		Pattern:       rule.Pattern,
		Action:        rule.Action,
		PromptSnippet: "最近 24 小时内的命中",
		CreatedAt:     time.Now().Add(-23 * time.Hour),
	}).Error)

	runSensitiveMonitorHitCleanupOnce()

	var hitCount int64
	require.NoError(t, model.DB.Model(&model.SensitiveWordHit{}).Count(&hitCount).Error)
	require.EqualValues(t, 1, hitCount)

	var remainingHit model.SensitiveWordHit
	require.NoError(t, model.DB.First(&remainingHit).Error)
	require.Equal(t, "最近 24 小时内的命中", remainingHit.PromptSnippet)

	var updatedRule model.SensitiveWord
	require.NoError(t, model.DB.First(&updatedRule, rule.Id).Error)
	require.Equal(t, 9, updatedRule.HitCount)
	require.NotNil(t, updatedRule.LastHitAt)
}

func TestListSensitiveWordHitsFiltersByDetailFields(t *testing.T) {
	resetSensitiveMonitorTables(t)
	now := time.Now().Truncate(time.Second)

	require.NoError(t, model.DB.Create(&model.SensitiveWordHit{
		RuleId:        1,
		Pattern:       "target pattern",
		Action:        model.SensitiveWordActionBlock,
		UserId:        7,
		Username:      "alice",
		TokenId:       12,
		TokenName:     "prod-token",
		ModelName:     "gpt-target",
		RequestId:     "req-target",
		Path:          "/v1/chat/completions",
		PromptSnippet: "contains target keyword",
		CreatedAt:     now,
	}).Error)
	require.NoError(t, model.DB.Create(&model.SensitiveWordHit{
		RuleId:        2,
		Pattern:       "other pattern",
		Action:        model.SensitiveWordActionMonitor,
		UserId:        8,
		Username:      "bob",
		TokenId:       13,
		TokenName:     "dev-token",
		ModelName:     "gpt-other",
		RequestId:     "req-other",
		Path:          "/v1/responses",
		PromptSnippet: "clean snippet",
		CreatedAt:     now.Add(-2 * time.Hour),
	}).Error)

	startTime := now.Add(-time.Minute)
	endTime := now.Add(time.Minute)
	hits, total, err := model.ListSensitiveWordHits(0, 50, model.SensitiveWordHitQuery{
		Action:    model.SensitiveWordActionBlock,
		Keyword:   "target keyword",
		Username:  "ali",
		TokenName: "prod",
		ModelName: "target",
		RequestId: "req-target",
		StartTime: &startTime,
		EndTime:   &endTime,
	})

	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, hits, 1)
	require.Equal(t, "req-target", hits[0].RequestId)
}

func TestCheckSensitiveMonitorExemptsVictimStatement(t *testing.T) {
	resetSensitiveMonitorTables(t)
	rule := model.SensitiveWord{
		Pattern: "强奸",
		Action:  model.SensitiveWordActionMonitor,
		Enabled: true,
	}
	require.NoError(t, model.DB.Create(&rule).Error)

	result, err := CheckSensitiveMonitor(SensitiveMonitorInput{
		UserId:     7,
		TokenId:    12,
		TokenName:  "dev-token",
		RequestId:  "req-victim-statement",
		PromptText: "用户试图反抗并指控他强奸，他声称监控已被处理，并",
		Path:       "/v1/chat/completions",
		Role:       "user",
	})

	require.NoError(t, err)
	require.False(t, result.Matched, "受害者陈述语境应被豁免")
	require.False(t, result.Blocked)
	require.Empty(t, result.Hits)

	var hitCount int64
	require.NoError(t, model.DB.Model(&model.SensitiveWordHit{}).Count(&hitCount).Error)
	require.EqualValues(t, 0, hitCount, "豁免命中不应落库")

	var updated model.SensitiveWord
	require.NoError(t, model.DB.First(&updated, rule.Id).Error)
	require.Equal(t, 0, updated.HitCount, "豁免命中不应递增 hit_count")
}

func TestCheckSensitiveMonitorBlockRuleNotExemptedByNegation(t *testing.T) {
	resetSensitiveMonitorTables(t)
	token := model.Token{
		Id:          51,
		UserId:      9,
		Key:         "block-token-key-legal",
		Status:      common.TokenStatusEnabled,
		Name:        "prod-token",
		RemainQuota: 1000,
	}
	require.NoError(t, model.DB.Create(&token).Error)
	rule := model.SensitiveWord{
		Pattern: "强奸幼女",
		Action:  model.SensitiveWordActionBlock,
		Enabled: true,
	}
	require.NoError(t, model.DB.Create(&rule).Error)

	result, err := CheckSensitiveMonitor(SensitiveMonitorInput{
		UserId:     9,
		TokenId:    token.Id,
		TokenName:  token.Name,
		RequestId:  "req-block-legal",
		PromptText: "法律严禁强奸幼女，违法必究",
		Path:       "/v1/chat/completions",
		Role:       "user",
	})

	require.NoError(t, err)
	require.True(t, result.Matched)
	require.True(t, result.Blocked, "Block 动作的高危规则不应被否定语境豁免")

	var updatedToken model.Token
	require.NoError(t, model.DB.First(&updatedToken, token.Id).Error)
	require.Equal(t, common.TokenStatusDisabled, updatedToken.Status)
}

func TestCheckSensitiveMonitorAssistantRoleDowngradesBlock(t *testing.T) {
	resetSensitiveMonitorTables(t)
	token := model.Token{
		Id:          61,
		UserId:      9,
		Key:         "assistant-token-key",
		Status:      common.TokenStatusEnabled,
		Name:        "prod-token",
		RemainQuota: 1000,
	}
	require.NoError(t, model.DB.Create(&token).Error)
	rule := model.SensitiveWord{
		Pattern: "bad child pattern",
		Action:  model.SensitiveWordActionBlock,
		Enabled: true,
	}
	require.NoError(t, model.DB.Create(&rule).Error)

	result, err := CheckSensitiveMonitor(SensitiveMonitorInput{
		UserId:     9,
		TokenId:    token.Id,
		TokenName:  token.Name,
		RequestId:  "req-assistant-downgrade",
		PromptText: "contains bad child pattern in model output",
		Path:       "/v1/chat/completions",
		Role:       "assistant",
	})

	require.NoError(t, err)
	require.True(t, result.Matched)
	require.False(t, result.Blocked, "assistant 角色命中 Block 应降级")
	require.Len(t, result.Hits, 1)
	require.Equal(t, model.SensitiveWordActionMonitor, result.Hits[0].Action)

	var updatedToken model.Token
	require.NoError(t, model.DB.First(&updatedToken, token.Id).Error)
	require.Equal(t, common.TokenStatusEnabled, updatedToken.Status, "assistant 角色不应禁用 token")

	var hit model.SensitiveWordHit
	require.NoError(t, model.DB.First(&hit).Error)
	require.Equal(t, model.SensitiveWordActionMonitor, hit.Action, "落库的 action 应是降级后的 Monitor")
}

func TestMarkSensitiveHitFalsePositive(t *testing.T) {
	resetSensitiveMonitorTables(t)
	now := time.Now().Truncate(time.Second)
	hit := model.SensitiveWordHit{
		RuleId:        1,
		Pattern:       "强奸",
		Action:        model.SensitiveWordActionMonitor,
		UserId:        7,
		Username:      "alice",
		PromptSnippet: "误报样本",
		CreatedAt:     now,
	}
	require.NoError(t, model.DB.Create(&hit).Error)

	require.NoError(t, model.MarkSensitiveHitFalsePositive(hit.Id, 42, true))

	var stored model.SensitiveWordHit
	require.NoError(t, model.DB.First(&stored, hit.Id).Error)
	require.True(t, stored.FalsePositive)
	require.Equal(t, 42, stored.ReviewedBy)
	require.NotNil(t, stored.ReviewedAt)

	falseVal := false
	hits, total, err := model.ListSensitiveWordHits(0, 50, model.SensitiveWordHitQuery{FalsePositive: &falseVal})
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.Empty(t, hits)

	trueVal := true
	hits, total, err = model.ListSensitiveWordHits(0, 50, model.SensitiveWordHitQuery{FalsePositive: &trueVal})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, hits, 1)

	require.NoError(t, model.MarkSensitiveHitFalsePositive(hit.Id, 42, false))
	stored = model.SensitiveWordHit{}
	require.NoError(t, model.DB.First(&stored, hit.Id).Error)
	require.False(t, stored.FalsePositive)
	require.Equal(t, 0, stored.ReviewedBy)
	require.Nil(t, stored.ReviewedAt)
}

func TestSensitiveWordCategorySeverityRoundtrip(t *testing.T) {
	resetSensitiveMonitorTables(t)
	rule := &model.SensitiveWord{
		Pattern:     "category-roundtrip",
		Action:      model.SensitiveWordActionMonitor,
		Enabled:     true,
		Category:    "sexual_assault",
		Severity:    5,
		Description: "test rule",
	}
	require.NoError(t, model.CreateSensitiveWord(rule))

	var stored model.SensitiveWord
	require.NoError(t, model.DB.First(&stored, rule.Id).Error)
	require.Equal(t, "sexual_assault", stored.Category)
	require.Equal(t, 5, stored.Severity)

	stored.Severity = 2
	stored.Category = "violence"
	require.NoError(t, model.UpdateSensitiveWord(&stored))

	var reread model.SensitiveWord
	require.NoError(t, model.DB.First(&reread, rule.Id).Error)
	require.Equal(t, "violence", reread.Category)
	require.Equal(t, 2, reread.Severity)

	out := &model.SensitiveWord{
		Pattern:  "severity-clamp",
		Action:   model.SensitiveWordActionMonitor,
		Enabled:  true,
		Severity: 99,
	}
	require.NoError(t, model.CreateSensitiveWord(out))
	var clamped model.SensitiveWord
	require.NoError(t, model.DB.First(&clamped, out.Id).Error)
	require.Equal(t, 3, clamped.Severity, "越界 severity 应被规范为默认 3")
}
