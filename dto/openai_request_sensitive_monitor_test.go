package dto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneralOpenAIRequestSensitiveMonitorTextUsesUserPromptsOnly(t *testing.T) {
	req := GeneralOpenAIRequest{
		Messages: []Message{
			{Role: "system", Content: "system policy should not be in monitor snippet"},
			{Role: "user", Content: "上一轮用户输入不应该进入本次记录"},
			{Role: "assistant", Content: "assistant output should not be in monitor snippet"},
			{Role: "user", Content: "用户输入强奸测试"},
		},
	}

	meta := req.GetTokenCountMeta()

	require.Contains(t, meta.CombineText, "assistant output should not be in monitor snippet")
	require.Equal(t, "用户输入强奸测试", meta.SensitiveMonitorText)
	require.NotContains(t, meta.SensitiveMonitorText, "上一轮用户输入")
	require.NotContains(t, meta.SensitiveMonitorText, "assistant output")
	require.NotContains(t, meta.SensitiveMonitorText, "system policy")
}
