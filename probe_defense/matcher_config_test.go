package probe_defense

import "testing"

func TestMatchTextWithConfiguredRules(t *testing.T) {
	rules := []SignatureRule{
		{ID: 1, Name: "tag echo", SourceKey: "cctest", Protocol: ProtocolClaudeMessages, Topic: "tag-echo", MatchType: MatchContainsAll, Patterns: []string{"return exactly", "tag-"}, Enabled: true, SourceEnabled: true},
		{ID: 2, Name: "disabled source", SourceKey: "ztest", Protocol: ProtocolClaudeMessages, Topic: "disabled", MatchType: MatchContainsAny, Patterns: []string{"never"}, Enabled: true, SourceEnabled: false},
		{ID: 3, Name: "disabled rule", SourceKey: "cctest", Protocol: ProtocolClaudeMessages, Topic: "disabled", MatchType: MatchContainsAny, Patterns: []string{"never"}, Enabled: false, SourceEnabled: true},
	}

	result := MatchText("Please RETURN EXACTLY: TAG-ab12", ProtocolClaudeMessages, []string{"cctest", "ztest"}, rules)

	if !result.Matched {
		t.Fatalf("expected configured rule to match")
	}
	if result.SourceKey != "cctest" || result.Topic != "tag-echo" || result.SignatureID != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestMatchTextContainsAnyAndRegex(t *testing.T) {
	rules := []SignatureRule{
		{ID: 10, Name: "ocr any", SourceKey: "ztest", Protocol: ProtocolClaudeMessages, Topic: "ocr", MatchType: MatchContainsAny, Patterns: []string{"pdf ocr", "image ocr"}, Enabled: true, SourceEnabled: true},
		{ID: 11, Name: "hex tag", SourceKey: "cctest", Protocol: ProtocolClaudeMessages, Topic: "tag", MatchType: MatchRegex, Patterns: []string{`TAG-[a-f0-9]{4}`}, Enabled: true, SourceEnabled: true},
	}

	anyResult := MatchText("please run IMAGE OCR now", ProtocolClaudeMessages, []string{"ztest"}, rules)
	if !anyResult.Matched || anyResult.SignatureID != 10 {
		t.Fatalf("expected contains_any match, got %+v", anyResult)
	}

	regexResult := MatchText("Return exactly TAG-a1b2", ProtocolClaudeMessages, []string{"cctest"}, rules)
	if !regexResult.Matched || regexResult.SignatureID != 11 {
		t.Fatalf("expected regex match, got %+v", regexResult)
	}
}

func TestMatchTextDoesNotUseThreshold(t *testing.T) {
	rules := []SignatureRule{
		{ID: 20, Name: "single weak phrase", SourceKey: "custom", Protocol: ProtocolClaudeMessages, Topic: "custom", MatchType: MatchContainsAny, Patterns: []string{"one phrase"}, Enabled: true, SourceEnabled: true},
	}

	result := MatchText("contains ONE PHRASE only", ProtocolClaudeMessages, []string{"custom"}, rules)
	if !result.Matched {
		t.Fatalf("expected any enabled rule match to be enough")
	}
}
