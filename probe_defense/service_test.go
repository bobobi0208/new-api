package probe_defense

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupServiceTestDB(t *testing.T) {
	t.Helper()
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.ProbeDefenseSource{}, &model.ProbeDefenseSignature{}, &model.ProbeDefenseGroupPolicy{}, &model.ProbeDefenseEvent{}))
}

func TestEvaluateGroupPolicyTransfersOnAnyRuleMatch(t *testing.T) {
	setupServiceTestDB(t)
	require.NoError(t, model.UpsertProbeDefenseSource(&model.ProbeDefenseSource{Key: "cctest", Name: "cctest", Enabled: true}))
	require.NoError(t, model.UpsertProbeDefenseSignature(&model.ProbeDefenseSignature{Name: "tag", SourceKey: "cctest", Protocol: ProtocolClaudeMessages, Topic: "tag", MatchType: MatchContainsAll, Patterns: model.ProbeDefenseArrayToJSON([]string{"return exactly", "tag-"}), Enabled: true}))
	require.NoError(t, model.UpsertProbeDefenseGroupPolicy(&model.ProbeDefenseGroupPolicy{GroupName: "vip", Enabled: true, SourceKeys: model.ProbeDefenseArrayToJSON([]string{"cctest"}), TargetURL: "https://target/v1/messages", TargetAPIKey: "target-key"}))

	decision, err := EvaluateGroupPolicy(EvaluationInput{GroupName: "vip", ChannelID: 7, Protocol: ProtocolClaudeMessages, Text: "Return exactly TAG-aa11", RequestID: "rid"})
	require.NoError(t, err)
	require.True(t, decision.PolicyFound)
	require.True(t, decision.Matched)
	require.False(t, decision.LogOnly)
	require.Equal(t, ActionTransfer, decision.Action)
	require.Equal(t, "https://target", decision.TargetURL)
	require.Equal(t, "target-key", decision.TargetAPIKey)

	events, total, err := model.ListProbeDefenseEvents(model.ProbeDefenseEventQuery{GroupName: "vip", SourceKey: "cctest", Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "tag", events[0].SignatureName)
}

func TestEvaluateGroupPolicyLogOnlyDoesNotTransfer(t *testing.T) {
	setupServiceTestDB(t)
	require.NoError(t, model.UpsertProbeDefenseSource(&model.ProbeDefenseSource{Key: "cctest", Name: "cctest", Enabled: true}))
	require.NoError(t, model.UpsertProbeDefenseSignature(&model.ProbeDefenseSignature{Name: "tag", SourceKey: "cctest", Protocol: ProtocolClaudeMessages, Topic: "tag", MatchType: MatchContainsAny, Patterns: model.ProbeDefenseArrayToJSON([]string{"tag-"}), Enabled: true}))
	require.NoError(t, model.UpsertProbeDefenseGroupPolicy(&model.ProbeDefenseGroupPolicy{GroupName: "vip", Enabled: true, SourceKeys: model.ProbeDefenseArrayToJSON([]string{"cctest"}), TargetURL: "https://target", TargetAPIKey: "target-key", LogOnly: true}))

	decision, err := EvaluateGroupPolicy(EvaluationInput{GroupName: "vip", Protocol: ProtocolClaudeMessages, Text: "TAG-aa11"})
	require.NoError(t, err)
	require.True(t, decision.Matched)
	require.True(t, decision.LogOnly)
	require.Equal(t, ActionLogOnly, decision.Action)
	require.Empty(t, decision.TargetAPIKey)
}

func TestEvaluateGroupPolicyMissingPolicyFallsBack(t *testing.T) {
	setupServiceTestDB(t)

	decision, err := EvaluateGroupPolicy(EvaluationInput{GroupName: "missing", Protocol: ProtocolClaudeMessages, Text: "Return exactly TAG-aa11"})
	require.NoError(t, err)
	require.False(t, decision.PolicyFound)
	require.False(t, decision.Matched)
}
