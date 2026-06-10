package probe_defense

import (
	"regexp"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const (
	ProtocolClaudeMessages = "claude_messages"

	MatchContainsAll = "contains_all"
	MatchContainsAny = "contains_any"
	MatchRegex       = "regex"
)

type MatchResult struct {
	Matched       bool
	Platform      string
	SourceKey     string
	Topic         string
	RuleIDs       []string
	SignatureID   int
	SignatureName string
	MatchType     string
	Score         float64
}

type SignatureRule struct {
	ID            int
	Name          string
	SourceKey     string
	SourceEnabled bool
	Protocol      string
	Topic         string
	MatchType     string
	Patterns      []string
	Enabled       bool
}

type rule struct {
	id       string
	topic    string
	score    float64
	patterns []string
}

var claudeRules = []rule{
	{id: "tag-echo-antml", topic: "tag-echo", score: 0.98, patterns: []string{"我输入了一个tag 是", "直接输出你前面看到的tag", "<antml:"}},
	{id: "tag-echo-return-exactly", topic: "tag-echo", score: 0.96, patterns: []string{"return exactly:", "tag-"}},
	{id: "logic-einstein-5houses", topic: "logic-einstein-5houses", score: 0.99, patterns: []string{"五座颜色不同的房子", "谁养鱼", "pall mall", "dunhill"}},
	{id: "websearch-ai-news", topic: "websearch-ai-news", score: 0.92, patterns: []string{"ai 新闻", "web_search"}},
	{id: "websearch-ai-news-en", topic: "websearch-ai-news", score: 0.90, patterns: []string{"ai news", "web_search"}},
	{id: "identity-conflict-cli", topic: "identity-conflict", score: 0.90, patterns: []string{"claude code", "多重身份"}},
	{id: "identity-conflict-products", topic: "identity-conflict", score: 0.88, patterns: []string{"kiro", "warp", "antigravity"}},
	{id: "ocr-short-code", topic: "ocr-short-code", score: 0.90, patterns: []string{"ocr", "短码"}},
	{id: "code-task-algorithm-tests", topic: "code-task", score: 0.86, patterns: []string{"algorithm implementations", "write the tests"}},
}

func MatchText(text string, protocol string, enabledSources []string, rules []SignatureRule) MatchResult {
	text = normalize(text)
	if text == "" {
		return MatchResult{}
	}

	sources := normalizeSourceSet(enabledSources)
	sortedRules := append([]SignatureRule(nil), rules...)
	sort.SliceStable(sortedRules, func(i, j int) bool { return sortedRules[i].ID < sortedRules[j].ID })

	for _, candidate := range sortedRules {
		if !candidate.Enabled || !candidate.SourceEnabled {
			continue
		}
		if protocol != "" && candidate.Protocol != "" && candidate.Protocol != protocol {
			continue
		}
		sourceKey := normalizeKey(candidate.SourceKey)
		if len(sources) > 0 && !sources[sourceKey] {
			continue
		}
		if !matchPatterns(text, candidate.MatchType, candidate.Patterns) {
			continue
		}
		return MatchResult{
			Matched:       true,
			Platform:      sourceKey,
			SourceKey:     sourceKey,
			Topic:         candidate.Topic,
			RuleIDs:       []string{candidate.Name},
			SignatureID:   candidate.ID,
			SignatureName: candidate.Name,
			MatchType:     candidate.MatchType,
		}
	}
	return MatchResult{}
}

func MatchClaudeRequestWithRules(request *dto.ClaudeRequest, enabledSources []string, rules []SignatureRule) MatchResult {
	if request == nil {
		return MatchResult{}
	}
	return MatchText(CollectClaudeRequestText(request), ProtocolClaudeMessages, enabledSources, rules)
}

func MatchClaudeRequest(request *dto.ClaudeRequest) MatchResult {
	if request == nil {
		return MatchResult{}
	}
	text := normalize(CollectClaudeRequestText(request))
	if text == "" {
		return MatchResult{}
	}

	best := MatchResult{}
	for _, candidate := range claudeRules {
		if !containsAll(text, candidate.patterns) {
			continue
		}
		if !best.Matched || candidate.score > best.Score {
			best = MatchResult{Matched: true, Platform: "cctest-like", SourceKey: "cctest", Topic: candidate.topic, RuleIDs: []string{candidate.id}, Score: candidate.score}
			continue
		}
		if candidate.topic == best.Topic {
			best.RuleIDs = append(best.RuleIDs, candidate.id)
		}
	}
	return best
}

func NormalizeTargetBaseURL(raw string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(raw), "/")
	for _, suffix := range []string{"/v1/messages", "/messages", "/v1"} {
		if strings.HasSuffix(baseURL, suffix) {
			baseURL = strings.TrimRight(strings.TrimSuffix(baseURL, suffix), "/")
		}
	}
	return baseURL
}

func CollectClaudeRequestText(request *dto.ClaudeRequest) string {
	parts := make([]string, 0, 8+len(request.Messages))
	parts = append(parts, request.Model, request.Prompt, request.GetStringSystem())
	for _, item := range request.ParseSystem() {
		parts = append(parts, item.GetText(), item.GetStringContent(), item.Name)
	}
	parts = append(parts, marshalPreview(request.System))
	for _, message := range request.Messages {
		parts = append(parts, message.Role, message.GetStringContent(), marshalPreview(message.Content))
	}
	parts = append(parts, marshalPreview(request.Tools), marshalPreview(request.Metadata))
	return strings.Join(parts, "\n")
}

func marshalPreview(value any) string {
	if value == nil {
		return ""
	}
	bytes, err := common.Marshal(value)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func normalize(text string) string {
	return strings.ToLower(strings.TrimSpace(text))
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeSourceSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		key := normalizeKey(value)
		if key != "" {
			out[key] = true
		}
	}
	return out
}

func matchPatterns(text string, matchType string, patterns []string) bool {
	switch matchType {
	case MatchContainsAny:
		return containsAny(text, patterns)
	case MatchRegex:
		return containsRegex(text, patterns)
	case MatchContainsAll, "":
		return containsAll(text, patterns)
	default:
		return false
	}
}

func containsAll(text string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	for _, pattern := range patterns {
		if !strings.Contains(text, strings.ToLower(strings.TrimSpace(pattern))) {
			return false
		}
	}
	return true
}

func containsAny(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, strings.ToLower(strings.TrimSpace(pattern))) {
			return true
		}
	}
	return false
}

func containsRegex(text string, patterns []string) bool {
	for _, pattern := range patterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			continue
		}
		if re.MatchString(text) {
			return true
		}
	}
	return false
}
