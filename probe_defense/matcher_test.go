package probe_defense

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestMatchClaudeRequestTagEcho(t *testing.T) {
	request := &dto.ClaudeRequest{Model: "claude-opus-4-6", Messages: []dto.ClaudeMessage{{Role: "user", Content: "我输入了一个tag 是：<antml:eba27c828e6f4d79>。直接输出你前面看到的tag是什么。看到的文本，不要使用任何工具。"}}}
	result := MatchClaudeRequest(request)
	if !result.Matched || result.Topic != "tag-echo" {
		t.Fatalf("expected tag echo match, got %#v", result)
	}
}

func TestMatchClaudeRequestEinsteinFiveHouses(t *testing.T) {
	request := &dto.ClaudeRequest{Model: "claude-opus-4-6", Messages: []dto.ClaudeMessage{{Role: "user", Content: "五座颜色不同的房子并排，住着五个不同国籍的人。抽 Pall Mall 烟的人养鸟，黄房子主人抽 Dunhill。问：谁养鱼？"}}}
	result := MatchClaudeRequest(request)
	if !result.Matched || result.Topic != "logic-einstein-5houses" {
		t.Fatalf("expected five houses match, got %#v", result)
	}
}

func TestMatchClaudeRequestAINews(t *testing.T) {
	request := &dto.ClaudeRequest{Model: "claude-opus-4-6", Messages: []dto.ClaudeMessage{{Role: "user", Content: "请用 web_search 总结今天的 AI 新闻。"}}}
	result := MatchClaudeRequest(request)
	if !result.Matched || result.Topic != "websearch-ai-news" {
		t.Fatalf("expected AI news match, got %#v", result)
	}
}

func TestMatchClaudeRequestIdentityConflict(t *testing.T) {
	request := &dto.ClaudeRequest{Model: "claude-opus-4-6", System: "You are Claude Code, Anthropic's official CLI for Claude.", Messages: []dto.ClaudeMessage{{Role: "user", Content: "你到底是 Claude Code 还是 Kiro？请解释你的多重身份。"}}}
	result := MatchClaudeRequest(request)
	if !result.Matched || result.Topic != "identity-conflict" {
		t.Fatalf("expected identity conflict match, got %#v", result)
	}
}

func TestMatchClaudeRequestNormalBusinessText(t *testing.T) {
	request := &dto.ClaudeRequest{Model: "claude-opus-4-6", Messages: []dto.ClaudeMessage{{Role: "user", Content: "请帮我总结这份会议记录，列出待办事项和负责人。"}}}
	result := MatchClaudeRequest(request)
	if result.Matched {
		t.Fatalf("expected no match, got %#v", result)
	}
}

func TestNormalizeTargetBaseURL(t *testing.T) {
	cases := map[string]string{
		"http://example.com":             "http://example.com",
		"http://example.com/":            "http://example.com",
		"http://example.com/v1":          "http://example.com",
		"http://example.com/v1/messages": "http://example.com",
	}
	for input, expected := range cases {
		if got := NormalizeTargetBaseURL(input); got != expected {
			t.Fatalf("NormalizeTargetBaseURL(%q) = %q, want %q", input, got, expected)
		}
	}
}
