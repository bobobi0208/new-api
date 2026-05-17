package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func TestGetStatusIncludesUsagePolicyEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	legalSettings := system_setting.GetLegalSettings()
	originalUserAgreement := legalSettings.UserAgreement
	originalPrivacyPolicy := legalSettings.PrivacyPolicy
	legalSettings.UserAgreement = "https://example.com/user-agreement"
	legalSettings.PrivacyPolicy = "https://example.com/privacy-policy"
	t.Cleanup(func() {
		legalSettings.UserAgreement = originalUserAgreement
		legalSettings.PrivacyPolicy = originalPrivacyPolicy
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)

	GetStatus(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected success response, got body: %s", recorder.Body.String())
	}

	rawValue, ok := response.Data["usage_policy_enabled"]
	if !ok {
		t.Fatalf("usage_policy_enabled is missing from /api/status response: %s", recorder.Body.String())
	}

	enabled, ok := rawValue.(bool)
	if !ok || !enabled {
		t.Fatalf("usage_policy_enabled = %#v, want true", rawValue)
	}

}
