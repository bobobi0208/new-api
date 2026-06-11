package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupProbeDefenseTestDB(t *testing.T) {
	t.Helper()
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	require.NoError(t, db.AutoMigrate(&ProbeDefenseSource{}, &ProbeDefenseSignature{}, &ProbeDefenseGroupPolicy{}, &ProbeDefenseEvent{}))
}

func TestProbeDefensePolicyMasksAndPreservesAPIKey(t *testing.T) {
	setupProbeDefenseTestDB(t)

	require.NoError(t, UpsertProbeDefenseGroupPolicy(&ProbeDefenseGroupPolicy{GroupName: "vip", Enabled: true, SourceKeys: `["cctest"]`, TargetURL: "https://a.example/v1", TargetAPIKey: "secret-key"}))
	policy, err := GetProbeDefenseGroupPolicy("vip")
	require.NoError(t, err)
	require.Equal(t, "secret-key", policy.TargetAPIKey)

	masked := MaskProbeDefenseGroupPolicy(policy)
	require.Equal(t, ProbeDefenseTargetAPIKeyMask, masked.TargetAPIKey)

	require.NoError(t, UpsertProbeDefenseGroupPolicy(&ProbeDefenseGroupPolicy{GroupName: "vip", Enabled: true, SourceKeys: `["ztest"]`, TargetURL: "https://b.example", TargetAPIKey: ""}))
	updated, err := GetProbeDefenseGroupPolicy("vip")
	require.NoError(t, err)
	require.Equal(t, "secret-key", updated.TargetAPIKey)
	require.Equal(t, `["ztest"]`, updated.SourceKeys)
}

func TestSeedDefaultProbeDefenseDataIsIdempotent(t *testing.T) {
	setupProbeDefenseTestDB(t)

	require.NoError(t, SeedDefaultProbeDefenseData())
	require.NoError(t, SeedDefaultProbeDefenseData())

	sources, err := ListProbeDefenseSources()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(sources), 2)

	signatures, err := ListProbeDefenseSignatures("")
	require.NoError(t, err)
	require.NotEmpty(t, signatures)
}

func TestSeedDefaultProbeDefenseDataCoversKnownProbeTopics(t *testing.T) {
	setupProbeDefenseTestDB(t)

	require.NoError(t, SeedDefaultProbeDefenseData())

	signatures, err := ListProbeDefenseSignatures("")
	require.NoError(t, err)

	byTopic := make(map[string][]ProbeDefenseSignature)
	for _, signature := range signatures {
		byTopic[signature.Topic] = append(byTopic[signature.Topic], signature)
	}

	expectedTopics := []string{
		"tag-echo",
		"logic-einstein-5houses",
		"math-1plus1",
		"websearch-ai-news",
		"identity-long-thinking",
		"identity-long-with-poison",
		"image-ocr-5x7",
		"pdf-ocr-8char",
		"long-stream",
	}
	for _, topic := range expectedTopics {
		require.NotEmpty(t, byTopic[topic], "missing default probe signature topic %s", topic)
	}

	require.Contains(t, byTopic["math-1plus1"][0].Patterns, "1+1")
	require.Contains(t, byTopic["identity-long-with-poison"][0].Patterns, "1000")
	require.Contains(t, byTopic["long-stream"][0].Patterns, "max_tokens")
}

func TestCreateProbeDefenseEvent(t *testing.T) {
	setupProbeDefenseTestDB(t)

	event := &ProbeDefenseEvent{GroupName: "vip", ChannelID: 12, Protocol: "claude_messages", SourceKey: "cctest", Topic: "tag", SignatureID: 3, SignatureName: "tag echo", MatchType: "contains_all", Action: "transfer", TargetURL: "https://target", RequestID: "rid", ContentPreview: "preview"}
	require.NoError(t, CreateProbeDefenseEvent(event))

	events, total, err := ListProbeDefenseEvents(ProbeDefenseEventQuery{GroupName: "vip", SourceKey: "cctest", Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, events, 1)
	require.Equal(t, "tag echo", events[0].SignatureName)
}
