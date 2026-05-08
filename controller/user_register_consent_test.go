package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRegisterControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.RegisterEnabled = true
	common.PasswordRegisterEnabled = true
	common.EmailVerificationEnabled = false
	common.QuotaForNewUser = 0

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db

	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate user table: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func newRegisterContext(t *testing.T, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	payload, err := common.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx, recorder
}

type registerAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func decodeRegisterResponse(t *testing.T, recorder *httptest.ResponseRecorder) registerAPIResponse {
	t.Helper()

	var response registerAPIResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}
	return response
}

func TestRegisterRejectsMissingUsageConsent(t *testing.T) {
	setupRegisterControllerTestDB(t)

	body := map[string]any{
		"username":                "consent-user",
		"password":                "password123",
		"accepted_user_agreement": true,
		"accepted_privacy_policy": true,
		"accepted_usage_policy":   false,
	}

	ctx, recorder := newRegisterContext(t, body)
	Register(ctx)

	response := decodeRegisterResponse(t, recorder)
	if response.Success {
		t.Fatalf("expected register failure when usage consent is missing")
	}
}

func TestRegisterPersistsConsentTimestamps(t *testing.T) {
	db := setupRegisterControllerTestDB(t)

	body := map[string]any{
		"username":                "full-consent-user",
		"password":                "password123",
		"accepted_user_agreement": true,
		"accepted_privacy_policy": true,
		"accepted_usage_policy":   true,
	}

	ctx, recorder := newRegisterContext(t, body)
	Register(ctx)

	response := decodeRegisterResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected register success, got message: %s", response.Message)
	}

	var consent struct {
		UserAgreementAt int64
		PrivacyPolicyAt int64
		UsagePolicyAt   int64
	}
	if err := db.Raw(
		`SELECT accepted_user_agreement_at AS user_agreement_at,
		        accepted_privacy_policy_at AS privacy_policy_at,
		        accepted_usage_policy_at AS usage_policy_at
		   FROM users
		  WHERE username = ?`,
		"full-consent-user",
	).Scan(&consent).Error; err != nil {
		t.Fatalf("failed to query consent timestamps: %v", err)
	}

	if consent.UserAgreementAt <= 0 || consent.PrivacyPolicyAt <= 0 || consent.UsagePolicyAt <= 0 {
		t.Fatalf("expected all consent timestamps to be persisted, got %+v", consent)
	}
}
