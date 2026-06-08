package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type ChannelAffinityKeySource struct {
	Type string `json:"type"` // context_int, context_string, request_header, gjson
	Key  string `json:"key,omitempty"`
	Path string `json:"path,omitempty"`
}

type ChannelAffinityRule struct {
	Name             string                     `json:"name"`
	ModelRegex       []string                   `json:"model_regex"`
	PathRegex        []string                   `json:"path_regex"`
	UserAgentInclude []string                   `json:"user_agent_include,omitempty"`
	KeySources       []ChannelAffinityKeySource `json:"key_sources"`

	ValueRegex string `json:"value_regex"`
	TTLSeconds int    `json:"ttl_seconds"`

	ParamOverrideTemplate map[string]interface{} `json:"param_override_template,omitempty"`

	SkipRetryOnFailure bool `json:"skip_retry_on_failure"`

	// Failover overrides (0 = inherit global). FailoverDisabled turns the
	// per-user failure circuit-breaker off for this rule.
	FailoverThreshold     int  `json:"failover_threshold,omitempty"`
	FailoverWindowSeconds int  `json:"failover_window_seconds,omitempty"`
	FailoverDisabled      bool `json:"failover_disabled,omitempty"`

	IncludeUsingGroup bool `json:"include_using_group"`
	IncludeModelName  bool `json:"include_model_name"`
	IncludeRuleName   bool `json:"include_rule_name"`
}

type ChannelAffinitySetting struct {
	Enabled           bool                  `json:"enabled"`
	SwitchOnSuccess   bool                  `json:"switch_on_success"`
	MaxEntries        int                   `json:"max_entries"`
	DefaultTTLSeconds int                   `json:"default_ttl_seconds"`

	// Per-user circuit breaker: when the same affinity key fails on a channel
	// more than FailureThreshold times within FailureWindowSeconds, that channel
	// stops being preferred so the next request falls back to another channel.
	FailoverEnabled      bool `json:"failover_enabled"`
	FailureThreshold     int  `json:"failure_threshold"`
	FailureWindowSeconds int  `json:"failure_window_seconds"`

	// Custom failure status codes for the affinity counter only. When non-empty,
	// these ranges override the global AutomaticRetryStatusCodeRanges and bypass
	// the global alwaysSkipRetryStatusCodes (504/524) classifier. Empty string =
	// fall back to global retry rules.
	FailureStatusCodes string `json:"failure_status_codes"`

	// Slow-response circuit breaker: when upstream TTFB (time-to-first-byte) on
	// a channel exceeds SlowResponseThresholdMs more than SlowResponseThreshold
	// times within SlowResponseWindowSeconds, that channel is suppressed for
	// the same affinity key. Independent counter from the status-code breaker
	// above — either one tripping causes a switch.
	SlowFailoverEnabled       bool `json:"slow_failover_enabled"`
	SlowResponseThresholdMs   int  `json:"slow_response_threshold_ms"`
	SlowResponseThreshold     int  `json:"slow_response_threshold"`
	SlowResponseWindowSeconds int  `json:"slow_response_window_seconds"`

	Rules []ChannelAffinityRule `json:"rules"`
}

var codexCliPassThroughHeaders = []string{
	"Originator",
	"Session_id",
	"User-Agent",
	"X-Codex-Beta-Features",
	"X-Codex-Turn-Metadata",
}

var claudeCliPassThroughHeaders = []string{
	"X-Stainless-Arch",
	"X-Stainless-Lang",
	"X-Stainless-Os",
	"X-Stainless-Package-Version",
	"X-Stainless-Retry-Count",
	"X-Stainless-Runtime",
	"X-Stainless-Runtime-Version",
	"X-Stainless-Timeout",
	"User-Agent",
	"X-App",
	"Anthropic-Beta",
	"Anthropic-Dangerous-Direct-Browser-Access",
	"Anthropic-Version",
}

func buildPassHeaderTemplate(headers []string) map[string]interface{} {
	clonedHeaders := make([]string, 0, len(headers))
	clonedHeaders = append(clonedHeaders, headers...)
	return map[string]interface{}{
		"operations": []map[string]interface{}{
			{
				"mode":        "pass_headers",
				"value":       clonedHeaders,
				"keep_origin": true,
			},
		},
	}
}

var channelAffinitySetting = ChannelAffinitySetting{
	Enabled:                   true,
	SwitchOnSuccess:           true,
	MaxEntries:                100_000,
	DefaultTTLSeconds:         3600,
	FailoverEnabled:           true,
	FailureThreshold:          3,
	FailureWindowSeconds:      60,
	FailureStatusCodes:        "",
	SlowFailoverEnabled:       false,
	SlowResponseThresholdMs:   30000,
	SlowResponseThreshold:     3,
	SlowResponseWindowSeconds: 60,
	Rules: []ChannelAffinityRule{
		{
			Name:       "codex cli trace",
			ModelRegex: []string{"^gpt-.*$"},
			PathRegex:  []string{"/v1/responses"},
			KeySources: []ChannelAffinityKeySource{
				{Type: "gjson", Path: "prompt_cache_key"},
			},
			ValueRegex:            "",
			TTLSeconds:            0,
			ParamOverrideTemplate: buildPassHeaderTemplate(codexCliPassThroughHeaders),
			SkipRetryOnFailure:    true,
			IncludeUsingGroup:     true,
			IncludeRuleName:       true,
			UserAgentInclude:      nil,
		},
		{
			Name:       "claude cli trace",
			ModelRegex: []string{"^claude-.*$"},
			PathRegex:  []string{"/v1/messages"},
			KeySources: []ChannelAffinityKeySource{
				{Type: "gjson", Path: "metadata.user_id"},
			},
			ValueRegex:            "",
			TTLSeconds:            0,
			ParamOverrideTemplate: buildPassHeaderTemplate(claudeCliPassThroughHeaders),
			SkipRetryOnFailure:    true,
			IncludeUsingGroup:     true,
			IncludeRuleName:       true,
			UserAgentInclude:      nil,
		},
	},
}

func init() {
	config.GlobalConfig.Register("channel_affinity_setting", &channelAffinitySetting)
}

func GetChannelAffinitySetting() *ChannelAffinitySetting {
	return &channelAffinitySetting
}

// FailoverEffectiveForRule resolves the effective per-user failover config for a
// rule, applying rule-level overrides on top of the global setting.
// Returns enabled=false when failover is globally off, the rule opts out, or the
// effective threshold/window is non-positive.
func (s *ChannelAffinitySetting) FailoverEffectiveForRule(rule *ChannelAffinityRule) (enabled bool, threshold int, windowSeconds int) {
	if s == nil || !s.FailoverEnabled {
		return false, 0, 0
	}
	if rule != nil && rule.FailoverDisabled {
		return false, 0, 0
	}

	threshold = s.FailureThreshold
	windowSeconds = s.FailureWindowSeconds
	if rule != nil {
		if rule.FailoverThreshold > 0 {
			threshold = rule.FailoverThreshold
		}
		if rule.FailoverWindowSeconds > 0 {
			windowSeconds = rule.FailoverWindowSeconds
		}
	}
	if windowSeconds <= 0 {
		windowSeconds = s.DefaultTTLSeconds
	}
	if threshold <= 0 || windowSeconds <= 0 {
		return false, 0, 0
	}
	return true, threshold, windowSeconds
}
