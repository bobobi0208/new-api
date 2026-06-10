package probe_defense

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	ActionTransfer         = "transfer"
	ActionLogOnly          = "log_only"
	ActionTargetIncomplete = "target_incomplete"
)

type EvaluationInput struct {
	GroupName string
	ChannelID int
	ModelName string
	Protocol  string
	Text      string
	RequestID string
}

type EvaluationDecision struct {
	PolicyFound  bool
	Matched      bool
	LogOnly      bool
	Action       string
	TargetURL    string
	TargetAPIKey string
	Result       MatchResult
}

func EvaluateGroupPolicy(input EvaluationInput) (EvaluationDecision, error) {
	policy, err := model.GetProbeDefenseGroupPolicy(strings.TrimSpace(input.GroupName))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return EvaluationDecision{}, nil
	}
	if err != nil {
		return EvaluationDecision{}, err
	}

	decision := EvaluationDecision{PolicyFound: true, LogOnly: policy.LogOnly}
	if !policy.Enabled {
		return decision, nil
	}

	sources, err := model.ListProbeDefenseSources()
	if err != nil {
		return decision, err
	}
	signatures, err := model.ListProbeDefenseSignatures("")
	if err != nil {
		return decision, err
	}

	result := MatchText(input.Text, input.Protocol, model.ProbeDefenseJSONToArray(policy.SourceKeys), BuildSignatureRules(sources, signatures))
	if !result.Matched {
		return decision, nil
	}

	decision.Matched = true
	decision.Result = result
	decision.Action = ActionTransfer
	if policy.LogOnly {
		decision.Action = ActionLogOnly
	} else {
		decision.TargetURL = NormalizeTargetBaseURL(policy.TargetURL)
		decision.TargetAPIKey = strings.TrimSpace(policy.TargetAPIKey)
		if decision.TargetURL == "" || decision.TargetAPIKey == "" {
			decision.Action = ActionTargetIncomplete
			decision.TargetURL = ""
			decision.TargetAPIKey = ""
		}
	}

	_ = model.CreateProbeDefenseEvent(&model.ProbeDefenseEvent{
		GroupName:      strings.TrimSpace(input.GroupName),
		ChannelID:      input.ChannelID,
		ModelName:      input.ModelName,
		Protocol:       input.Protocol,
		SourceKey:      result.SourceKey,
		Topic:          result.Topic,
		SignatureID:    result.SignatureID,
		SignatureName:  result.SignatureName,
		MatchType:      result.MatchType,
		Action:         decision.Action,
		TargetURL:      decision.TargetURL,
		RequestID:      input.RequestID,
		ContentPreview: buildContentPreview(input.Text),
	})

	return decision, nil
}

func BuildSignatureRules(sources []model.ProbeDefenseSource, signatures []model.ProbeDefenseSignature) []SignatureRule {
	sourceEnabled := make(map[string]bool, len(sources))
	for _, source := range sources {
		sourceEnabled[model.NormalizeProbeDefenseKey(source.Key)] = source.Enabled
	}

	rules := make([]SignatureRule, 0, len(signatures))
	for _, signature := range signatures {
		sourceKey := model.NormalizeProbeDefenseKey(signature.SourceKey)
		rules = append(rules, SignatureRule{
			ID:            signature.ID,
			Name:          signature.Name,
			SourceKey:     sourceKey,
			SourceEnabled: sourceEnabled[sourceKey],
			Protocol:      signature.Protocol,
			Topic:         signature.Topic,
			MatchType:     signature.MatchType,
			Patterns:      model.ProbeDefenseJSONToArray(signature.Patterns),
			Enabled:       signature.Enabled,
		})
	}
	return rules
}

func buildContentPreview(text string) string {
	preview := strings.Join(strings.Fields(text), " ")
	if preview == "" {
		return ""
	}
	const maxRunes = 240
	runes := []rune(preview)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return preview
}
