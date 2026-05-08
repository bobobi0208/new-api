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

func TestCheckSensitiveMonitorMatchesStandaloneRapeKeyword(t *testing.T) {
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
	})

	require.NoError(t, err)
	require.True(t, result.Matched)
	require.False(t, result.Blocked)
	require.Len(t, result.Hits, 1)
	require.Equal(t, `(?i)(强|轮)\s*奸(\s*(儿童|幼女|幼童|小学生|未成年))?`, result.Hits[0].Pattern)
	require.Equal(t, model.SensitiveWordActionMonitor, result.Hits[0].Action)
}

func TestCheckSensitiveMonitorRunsForPlaygroundChatCompletions(t *testing.T) {
	resetSensitiveMonitorTables(t)
	_, err := SeedDefaultSensitiveWords()
	require.NoError(t, err)
	require.True(t, ShouldRunSensitiveMonitor("/pg/chat/completions"))

	result, err := CheckSensitiveMonitor(SensitiveMonitorInput{
		UserId:     7,
		Username:   "alice",
		TokenId:    12,
		TokenName:  "playground-default",
		ModelName:  "gpt-test",
		RequestId:  "req-playground-rape-keyword",
		PromptText: "强 奸",
		Path:       "/pg/chat/completions",
	})

	require.NoError(t, err)
	require.True(t, result.Matched)
	require.False(t, result.Blocked)
	require.Len(t, result.Hits, 1)
	require.Equal(t, `(?i)(强|轮)\s*奸(\s*(儿童|幼女|幼童|小学生|未成年))?`, result.Hits[0].Pattern)
	require.Equal(t, model.SensitiveWordActionMonitor, result.Hits[0].Action)
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

func TestRunSensitiveMonitorHitCleanupOnceDeletesHitDetailsOnly(t *testing.T) {
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
		PromptSnippet: "输入强奸上下文",
		CreatedAt:     time.Now(),
	}).Error)

	runSensitiveMonitorHitCleanupOnce()

	var hitCount int64
	require.NoError(t, model.DB.Model(&model.SensitiveWordHit{}).Count(&hitCount).Error)
	require.EqualValues(t, 0, hitCount)

	var updatedRule model.SensitiveWord
	require.NoError(t, model.DB.First(&updatedRule, rule.Id).Error)
	require.Equal(t, 9, updatedRule.HitCount)
	require.NotNil(t, updatedRule.LastHitAt)
}
